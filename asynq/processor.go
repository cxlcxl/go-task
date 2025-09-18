package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"xxljob-go-executor/database"
	"xxljob-go-executor/logger"
	"xxljob-go-executor/models"

	"github.com/hibiken/asynq"
)

// TaskProcessor 任务处理器适配层
type TaskProcessor struct {
	asynqManager *AsynqManager
	db           *database.Database
	serverID     string
}

// NewTaskProcessor 创建新的任务处理器
func NewTaskProcessor(asynqManager *AsynqManager, db *database.Database, serverID string) *TaskProcessor {
	return &TaskProcessor{
		asynqManager: asynqManager,
		db:           db,
		serverID:     serverID,
	}
}

// ProcessDatabaseTasks 处理数据库中的任务（迁移到Asynq）
func (tp *TaskProcessor) ProcessDatabaseTasks() error {
	logger.Info("开始处理数据库任务迁移到Asynq...")

	// 获取队列配置
	configs, err := tp.db.GetQueueConfigs(tp.serverID)
	if err != nil {
		return fmt.Errorf("获取队列配置失败: %w", err)
	}

	for _, config := range configs {
		if err := tp.processSingleQueue(config); err != nil {
			logger.Error("处理队列失败: %s, 错误: %v", config.QueueName, err)
			continue
		}
	}

	return nil
}

// processSingleQueue 处理单个队列的任务
func (tp *TaskProcessor) processSingleQueue(config database.QueueConfig) error {
	// 获取待处理任务
	tasks, err := tp.db.GetPendingTasks(config.TableName, 100) // 批量获取100个任务
	if err != nil {
		return fmt.Errorf("获取待处理任务失败: %w", err)
	}

	if len(tasks) == 0 {
		return nil // 没有待处理任务
	}

	logger.Info("队列 %s 发现 %d 个待处理任务，开始迁移到Asynq", config.QueueName, len(tasks))

	for _, taskData := range tasks {
		if err := tp.migrateTaskToAsynq(config, taskData); err != nil {
			logger.Error("迁移任务失败: %v", err)
			continue
		}
	}

	return nil
}

// migrateTaskToAsynq 将数据库任务迁移到Asynq
func (tp *TaskProcessor) migrateTaskToAsynq(config database.QueueConfig, taskData map[string]interface{}) error {
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

	// 更新数据库任务状态为"迁移中"
	if err := tp.db.UpdateTaskStatus(config.TableName, taskID, database.TaskStatusRunning, tp.serverID, "迁移到Asynq中"); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 创建Asynq任务选项
	opts := []asynq.Option{
		asynq.Queue(config.QueueName),
		asynq.MaxRetry(maxRetries),
		asynq.Timeout(30 * time.Second),
	}

	// 如果有计划时间且是未来时间，则设置延迟
	if !scheduleTime.IsZero() && scheduleTime.After(time.Now()) {
		delay := time.Until(scheduleTime)
		opts = append(opts, asynq.ProcessIn(delay))
	}

	// 创建任务元数据
	metadata := map[string]interface{}{
		"original_task_id": taskID,
		"table_name":       config.TableName,
		"migrated_at":      time.Now(),
	}

	// 将元数据添加到任务参数中
	enrichedParams := map[string]interface{}{
		"params":   taskParams,
		"metadata": metadata,
	}

	paramsJSON, _ := json.Marshal(enrichedParams)

	// 入队到Asynq
	taskInfo, err := tp.asynqManager.EnqueueTask(taskType, string(paramsJSON), opts...)
	if err != nil {
		// 恢复数据库任务状态
		tp.db.UpdateTaskStatus(config.TableName, taskID, database.TaskStatusPending, "", "Asynq入队失败: "+err.Error())
		return fmt.Errorf("Asynq入队失败: %w", err)
	}

	// 更新数据库任务状态为"已迁移"
	if err := tp.db.UpdateTaskStatus(config.TableName, taskID, database.TaskStatusCompleted, tp.serverID,
		fmt.Sprintf("已迁移到Asynq，任务ID: %s", taskInfo.ID)); err != nil {
		logger.Error("更新任务状态失败: %v", err)
	}

	logger.Info("任务迁移成功: DB任务ID=%d -> Asynq任务ID=%s", taskID, taskInfo.ID)
	return nil
}

// TaskWrapper 任务包装器，用于适配现有的JobHandler接口
type TaskWrapper struct {
	handler models.JobHandler
	db      *database.Database
}

// NewTaskWrapper 创建任务包装器
func NewTaskWrapper(handler models.JobHandler, db *database.Database) *TaskWrapper {
	return &TaskWrapper{
		handler: handler,
		db:      db,
	}
}

// ProcessTask 处理Asynq任务
func (tw *TaskWrapper) ProcessTask(ctx context.Context, task *asynq.Task) error {
	// 解析任务载荷
	var payload TaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务载荷失败: %w", err)
	}

	// 解析参数（可能包含元数据）
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(payload.TaskParams), &params); err != nil {
		// 如果解析失败，直接使用原始参数
		params = map[string]interface{}{
			"params": payload.TaskParams,
		}
	}

	// 提取实际参数
	actualParams := payload.TaskParams
	if p, ok := params["params"].(string); ok {
		actualParams = p
	}

	// 创建任务上下文
	jobCtx := &models.JobContext{
		JobID:       int(time.Now().Unix()), // 使用时间戳作为临时ID
		JobParam:    actualParams,
		LogID:       int(time.Now().Unix()),
		LogDateTime: time.Now(),
		// 可以从metadata中提取更多信息
	}

	// 如果有元数据，可以从中获取原始任务ID
	if metadata, ok := params["metadata"].(map[string]interface{}); ok {
		if originalID, ok := metadata["original_task_id"].(float64); ok {
			jobCtx.JobID = int(originalID)
		}
		if tableName, ok := metadata["table_name"].(string); ok {
			logger.Debug("处理来自表 %s 的迁移任务", tableName)
		}
	}

	logger.Info("开始执行任务: %s, JobID: %d", payload.TaskType, jobCtx.JobID)

	// 执行任务
	return tw.handler.Execute(jobCtx)
}

// RegisterTaskHandlers 批量注册任务处理器到Asynq
func (tp *TaskProcessor) RegisterTaskHandlers(handlers map[string]models.JobHandler) {
	for taskType, handler := range handlers {
		wrapper := NewTaskWrapper(handler, tp.db)

		// 在Asynq中注册处理器
		tp.asynqManager.mux.HandleFunc(taskType, wrapper.ProcessTask)

		logger.Info("已注册任务处理器到Asynq: %s", taskType)
	}
}

// SchedulePeriodicMigration 定期从数据库迁移任务到Asynq
func (tp *TaskProcessor) SchedulePeriodicMigration(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := tp.ProcessDatabaseTasks(); err != nil {
					logger.Error("定期任务迁移失败: %v", err)
				}
			}
		}
	}()

	logger.Info("已启动定期任务迁移，间隔: %v", interval)
}
