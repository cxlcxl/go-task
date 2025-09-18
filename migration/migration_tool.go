package migration
package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"xxljob-go-executor/asynq"
	"xxljob-go-executor/config"
	"xxljob-go-executor/database"
	"xxljob-go-executor/logger"
)

// MigrationTool 数据库任务到Asynq的迁移工具
type MigrationTool struct {
	config       *config.Config
	db           *database.Database
	asynqManager *asynq.AsynqManager
	serverID     string
}

// MigrationStats 迁移统计信息
type MigrationStats struct {
	TotalTasks     int       `json:"total_tasks"`
	MigratedTasks  int       `json:"migrated_tasks"`
	FailedTasks    int       `json:"failed_tasks"`
	SkippedTasks   int       `json:"skipped_tasks"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Duration       string    `json:"duration"`
	FailedTasksDetails []MigrationError `json:"failed_tasks_details,omitempty"`
}

// MigrationError 迁移错误详情
type MigrationError struct {
	TaskID    int64  `json:"task_id"`
	TaskType  string `json:"task_type"`
	Error     string `json:"error"`
	TableName string `json:"table_name"`
}

// NewMigrationTool 创建迁移工具
func NewMigrationTool(cfg *config.Config, db *database.Database, serverID string) (*MigrationTool, error) {
	// 创建Asynq管理器
	asynqMgr, err := asynq.NewAsynqManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建Asynq管理器失败: %w", err)
	}

	return &MigrationTool{
		config:       cfg,
		db:           db,
		asynqManager: asynqMgr,
		serverID:     serverID,
	}, nil
}

// MigrateAllTasks 迁移所有待处理任务
func (mt *MigrationTool) MigrateAllTasks(ctx context.Context) (*MigrationStats, error) {
	stats := &MigrationStats{
		StartTime:          time.Now(),
		FailedTasksDetails: make([]MigrationError, 0),
	}

	logger.Info("开始全量任务迁移...")

	// 获取所有队列配置
	configs, err := mt.db.GetQueueConfigs(mt.serverID)
	if err != nil {
		return stats, fmt.Errorf("获取队列配置失败: %w", err)
	}

	// 遍历每个队列配置
	for _, config := range configs {
		if err := ctx.Err(); err != nil {
			logger.Error("迁移被取消: %v", err)
			break
		}

		queueStats, err := mt.migrateQueueTasks(ctx, config)
		if err != nil {
			logger.Error("迁移队列 %s 失败: %v", config.QueueName, err)
			continue
		}

		// 合并统计信息
		stats.TotalTasks += queueStats.TotalTasks
		stats.MigratedTasks += queueStats.MigratedTasks
		stats.FailedTasks += queueStats.FailedTasks
		stats.SkippedTasks += queueStats.SkippedTasks
		stats.FailedTasksDetails = append(stats.FailedTasksDetails, queueStats.FailedTasksDetails...)
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime).String()

	logger.Info("任务迁移完成 - 总计: %d, 成功: %d, 失败: %d, 跳过: %d",
		stats.TotalTasks, stats.MigratedTasks, stats.FailedTasks, stats.SkippedTasks)

	return stats, nil
}

// migrateQueueTasks 迁移单个队列的任务
func (mt *MigrationTool) migrateQueueTasks(ctx context.Context, config database.QueueConfig) (*MigrationStats, error) {
	stats := &MigrationStats{
		StartTime:          time.Now(),
		FailedTasksDetails: make([]MigrationError, 0),
	}

	logger.Info("开始迁移队列: %s (表: %s)", config.QueueName, config.TableName)

	// 分批获取任务（避免内存占用过大）
	batchSize := 100
	offset := 0

	for {
		if err := ctx.Err(); err != nil {
			return stats, err
		}

		// 获取一批待处理任务
		tasks, err := mt.getBatchTasks(config.TableName, offset, batchSize)
		if err != nil {
			return stats, fmt.Errorf("获取任务批次失败: %w", err)
		}

		if len(tasks) == 0 {
			break // 没有更多任务
		}

		// 迁移这批任务
		for _, taskData := range tasks {
			if err := ctx.Err(); err != nil {
				return stats, err
			}

			stats.TotalTasks++

			err := mt.migrateSingleTask(config, taskData)
			if err != nil {
				stats.FailedTasks++

				// 记录失败详情
				taskID, _ := taskData["id"].(int64)
				taskType, _ := taskData["task_type"].(string)

				stats.FailedTasksDetails = append(stats.FailedTasksDetails, MigrationError{
					TaskID:    taskID,
					TaskType:  taskType,
					Error:     err.Error(),
					TableName: config.TableName,
				})

				logger.Error("任务迁移失败: ID=%d, 错误=%v", taskID, err)
				continue
			}

			stats.MigratedTasks++
		}

		offset += len(tasks)

		// 添加少许延迟避免对数据库造成过大压力
		time.Sleep(10 * time.Millisecond)
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime).String()

	logger.Info("队列 %s 迁移完成 - 总计: %d, 成功: %d, 失败: %d",
		config.QueueName, stats.TotalTasks, stats.MigratedTasks, stats.FailedTasks)

	return stats, nil
}

// getBatchTasks 分批获取任务
func (mt *MigrationTool) getBatchTasks(tableName string, offset, limit int) ([]map[string]interface{}, error) {
	// 这里需要实现从数据库分批获取任务的逻辑
	// 为了简化示例，这里返回空结果
	// 实际实现中需要执行SQL查询: SELECT * FROM {tableName} WHERE status = 0 LIMIT {limit} OFFSET {offset}
	return []map[string]interface{}{}, nil
}

// migrateSingleTask 迁移单个任务
func (mt *MigrationTool) migrateSingleTask(config database.QueueConfig, taskData map[string]interface{}) error {
	// 解析任务数据
	taskID, ok := taskData["id"].(int64)
	if !ok {
		return fmt.Errorf("无效的任务ID")
	}

	taskType, ok := taskData["task_type"].(string)
	if !ok {
		return fmt.Errorf("无效的任务类型")
	}

	taskParams, _ := taskData["task_params"].(string)
	scheduleTime, _ := taskData["schedule_time"].(time.Time)
	maxRetries, _ := taskData["max_retries"].(int)

	// 检查任务状态，只迁移待处理的任务
	status, _ := taskData["status"].(int)
	if status != database.TaskStatusPending {
		return fmt.Errorf("任务状态不是待处理: %d", status)
	}

	// 更新数据库任务状态为"迁移中"
	if err := mt.db.UpdateTaskStatus(config.TableName, taskID, database.TaskStatusRunning, mt.serverID, "迁移到Asynq中"); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 创建任务元数据
	metadata := map[string]interface{}{
		"original_task_id": taskID,
		"table_name":       config.TableName,
		"migrated_at":      time.Now(),
		"migrated_by":      mt.serverID,
		"queue_name":       config.QueueName,
	}

	// 构建丰富的任务参数
	enrichedParams := map[string]interface{}{
		"params":   taskParams,
		"metadata": metadata,
	}

	paramsJSON, _ := json.Marshal(enrichedParams)

	// 计算延迟时间
	var delay time.Duration
	if !scheduleTime.IsZero() && scheduleTime.After(time.Now()) {
		delay = time.Until(scheduleTime)
	}

	// 入队到Asynq
	var taskInfo interface{}
	var err error

	if delay > 0 {
		taskInfo, err = mt.asynqManager.EnqueueTaskIn(taskType, string(paramsJSON), delay)
	} else {
		taskInfo, err = mt.asynqManager.EnqueueTask(taskType, string(paramsJSON))
	}

	if err != nil {
		// 恢复数据库任务状态
		mt.db.UpdateTaskStatus(config.TableName, taskID, database.TaskStatusPending, "", "Asynq入队失败: "+err.Error())
		return fmt.Errorf("Asynq入队失败: %w", err)
	}

	// 更新数据库任务状态为"已迁移"
	taskInfoStr := "已迁移到Asynq"
	if taskInfo != nil {
		taskInfoStr = fmt.Sprintf("已迁移到Asynq，延迟: %v", delay)
	}

	if err := mt.db.UpdateTaskStatus(config.TableName, taskID, database.TaskStatusCompleted, mt.serverID, taskInfoStr); err != nil {
		logger.Error("更新任务状态失败: %v", err)
	}

	return nil
}

// ValidateMigration 验证迁移结果
func (mt *MigrationTool) ValidateMigration() (*MigrationValidationReport, error) {
	report := &MigrationValidationReport{
		CheckTime: time.Now(),
		Checks:    make([]ValidationCheck, 0),
	}

	logger.Info("开始验证迁移结果...")

	// 检查1: 统计待迁移任务数量
	pendingCount, err := mt.countPendingTasks()
	if err != nil {
		report.Checks = append(report.Checks, ValidationCheck{
			Name:   "pending_tasks_count",
			Status: "error",
			Error:  err.Error(),
		})
	} else {
		report.Checks = append(report.Checks, ValidationCheck{
			Name:   "pending_tasks_count",
			Status: "success",
			Count:  pendingCount,
			Message: fmt.Sprintf("待迁移任务数量: %d", pendingCount),
		})
	}

	// 检查2: Asynq连接状态
	asynqStats, err := mt.asynqManager.GetStats()
	if err != nil {
		report.Checks = append(report.Checks, ValidationCheck{
			Name:   "asynq_connection",
			Status: "error",
			Error:  err.Error(),
		})
	} else {
		report.Checks = append(report.Checks, ValidationCheck{
			Name:    "asynq_connection",
			Status:  "success",
			Data:    asynqStats,
			Message: "Asynq连接正常",
		})
	}

	// 检查3: 队列健康状态
	// 这里可以添加更多检查逻辑

	logger.Info("迁移验证完成")
	return report, nil
}

// countPendingTasks 统计待迁移任务数量
func (mt *MigrationTool) countPendingTasks() (int, error) {
	configs, err := mt.db.GetQueueConfigs(mt.serverID)
	if err != nil {
		return 0, err
	}

	totalPending := 0
	for _, config := range configs {
		// 这里需要实现计算每个表中待处理任务数量的逻辑
		// 实际实现中需要执行SQL查询: SELECT COUNT(*) FROM {tableName} WHERE status = 0
		// 为了简化示例，这里返回0
	}

	return totalPending, nil
}

// MigrationValidationReport 迁移验证报告
type MigrationValidationReport struct {
	CheckTime time.Time         `json:"check_time"`
	Checks    []ValidationCheck `json:"checks"`
}

// ValidationCheck 验证检查项
type ValidationCheck struct {
	Name    string      `json:"name"`
	Status  string      `json:"status"` // success, warning, error
	Count   int         `json:"count,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Close 关闭迁移工具
func (mt *MigrationTool) Close() {
	if mt.asynqManager != nil {
		mt.asynqManager.Stop()
	}
}

// RunMigrationCommand 运行迁移命令行工具
func RunMigrationCommand(configPath string) {
	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	dbConfigs := make(map[string]database.Config)
	for name, dbCfg := range cfg.Database {
		dbConfigs[name] = database.Config{
			Host:     dbCfg.Host,
			Port:     dbCfg.Port,
			Username: dbCfg.Username,
			Password: dbCfg.Password,
			Database: dbCfg.Database,
			Charset:  dbCfg.Charset,
		}
	}

	db, err := database.NewDatabase(dbConfigs)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 创建迁移工具
	migrationTool, err := NewMigrationTool(cfg, db, cfg.Executor.ServerID)
	if err != nil {
		log.Fatalf("创建迁移工具失败: %v", err)
	}
	defer migrationTool.Close()

	// 执行迁移
	ctx := context.Background()
	stats, err := migrationTool.MigrateAllTasks(ctx)
	if err != nil {
		log.Fatalf("迁移失败: %v", err)
	}

	// 输出统计信息
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Printf("迁移完成统计:\n%s\n", string(statsJSON))

	// 验证迁移结果
	report, err := migrationTool.ValidateMigration()
	if err != nil {
		log.Printf("验证迁移结果失败: %v", err)
	} else {
		reportJSON, _ := json.MarshalIndent(report, "", "  ")
		fmt.Printf("验证报告:\n%s\n", string(reportJSON))
	}
}