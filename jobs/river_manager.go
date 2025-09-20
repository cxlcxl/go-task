package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"task-executor/config"
	"task-executor/database"
	"task-executor/logger"
	"task-executor/models"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// RiverManager River 队列管理器
type RiverManager struct {
	config       *config.Config
	db           *database.Database
	serverID     string
	client       *river.Client[pgx.Tx]
	workers      *river.Workers
	taskHandlers map[string]models.JobHandler
	queueConfigs map[string]*database.QueueConfig // 数据库中的队列配置
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	insertOnly   bool // 是否仅插入模式
}

// 定义具体的任务类型

// EmailTaskArgs 邮件任务参数
type EmailTaskArgs struct {
	TaskParams string `json:"task_params"`
}

func (EmailTaskArgs) Kind() string { return "email" }

// DataSyncTaskArgs 数据同步任务参数
type DataSyncTaskArgs struct {
	TaskParams string `json:"task_params"`
}

func (DataSyncTaskArgs) Kind() string { return "data_sync" }

// FileProcessTaskArgs 文件处理任务参数
type FileProcessTaskArgs struct {
	TaskParams string `json:"task_params"`
}

func (FileProcessTaskArgs) Kind() string { return "file_process" }

// ReportGenerateTaskArgs 报告生成任务参数
type ReportGenerateTaskArgs struct {
	TaskParams string `json:"task_params"`
}

func (ReportGenerateTaskArgs) Kind() string { return "report_generate" }

// NotificationTaskArgs 通知任务参数
type NotificationTaskArgs struct {
	TaskParams string `json:"task_params"`
}

func (NotificationTaskArgs) Kind() string { return "notification" }

// DemoTaskArgs 演示任务参数
type DemoTaskArgs struct {
	TaskParams string `json:"task_params"`
}

func (DemoTaskArgs) Kind() string { return "demo" }

// 对应的 Worker 实现

// EmailWorker 邮件工作器
type EmailWorker struct {
	river.WorkerDefaults[EmailTaskArgs]
	manager *RiverManager
}

func (w *EmailWorker) Work(ctx context.Context, job *river.Job[EmailTaskArgs]) error {
	return w.manager.executeTask(ctx, "email", job.Args.TaskParams, job.ID)
}

// DataSyncWorker 数据同步工作器
type DataSyncWorker struct {
	river.WorkerDefaults[DataSyncTaskArgs]
	manager *RiverManager
}

func (w *DataSyncWorker) Work(ctx context.Context, job *river.Job[DataSyncTaskArgs]) error {
	return w.manager.executeTask(ctx, "data_sync", job.Args.TaskParams, job.ID)
}

// FileProcessWorker 文件处理工作器
type FileProcessWorker struct {
	river.WorkerDefaults[FileProcessTaskArgs]
	manager *RiverManager
}

func (w *FileProcessWorker) Work(ctx context.Context, job *river.Job[FileProcessTaskArgs]) error {
	return w.manager.executeTask(ctx, "file_process", job.Args.TaskParams, job.ID)
}

// ReportGenerateWorker 报告生成工作器
type ReportGenerateWorker struct {
	river.WorkerDefaults[ReportGenerateTaskArgs]
	manager *RiverManager
}

func (w *ReportGenerateWorker) Work(ctx context.Context, job *river.Job[ReportGenerateTaskArgs]) error {
	return w.manager.executeTask(ctx, "report_generate", job.Args.TaskParams, job.ID)
}

// NotificationWorker 通知工作器
type NotificationWorker struct {
	river.WorkerDefaults[NotificationTaskArgs]
	manager *RiverManager
}

func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[NotificationTaskArgs]) error {
	return w.manager.executeTask(ctx, "notification", job.Args.TaskParams, job.ID)
}

// DemoWorker 演示工作器
type DemoWorker struct {
	river.WorkerDefaults[DemoTaskArgs]
	manager *RiverManager
}

func (w *DemoWorker) Work(ctx context.Context, job *river.Job[DemoTaskArgs]) error {
	return w.manager.executeTask(ctx, "demo", job.Args.TaskParams, job.ID)
}

// executeTask 统一的任务执行逻辑
func (rm *RiverManager) executeTask(ctx context.Context, taskType, taskParams string, jobID int64) error {
	logger.Info("开始执行River任务: %s, 参数: %s", taskType, taskParams)

	// 获取任务处理器
	rm.mu.RLock()
	handler, exists := rm.taskHandlers[taskType]
	rm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("未找到任务处理器: %s", taskType)
	}

	// 创建任务上下文
	jobCtx := &models.JobContext{
		JobID:       int(jobID),
		JobParam:    taskParams,
		LogID:       int(jobID),
		LogDateTime: time.Now(),
	}

	// 执行任务
	startTime := time.Now()
	err := handler.Execute(jobCtx)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("River任务执行失败: %s, 错误: %v", taskType, err)
		return err
	}

	logger.Info("River任务执行成功: %s, 耗时: %v", taskType, duration)
	return nil
}

// NewRiverManager 创建 River 队列管理器
func NewRiverManager(cfg *config.Config, db *database.Database, serverID string) (*RiverManager, error) {
	if !cfg.River.Enable {
		return nil, fmt.Errorf("River队列未启用")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建 River workers
	workers := river.NewWorkers()

	// 创建管理器实例
	manager := &RiverManager{
		config:       cfg,
		db:           db,
		serverID:     serverID,
		workers:      workers,
		taskHandlers: make(map[string]models.JobHandler),
		queueConfigs: make(map[string]*database.QueueConfig),
		ctx:          ctx,
		cancel:       cancel,
		insertOnly:   cfg.River.InsertOnly,
	}

	// 注册所有工作器
	river.AddWorker(workers, &DemoWorker{manager: manager})
	river.AddWorker(workers, &EmailWorker{manager: manager})
	river.AddWorker(workers, &DataSyncWorker{manager: manager})
	river.AddWorker(workers, &FileProcessWorker{manager: manager})
	river.AddWorker(workers, &ReportGenerateWorker{manager: manager})
	river.AddWorker(workers, &NotificationWorker{manager: manager})
	river.AddWorker(workers, &DemoWorker{manager: manager})

	// 配置 River 客户端
	riverConfig := &river.Config{
		Workers: workers,
	}

	// 如果不是仅插入模式，从数据库加载队列配置
	if !cfg.River.InsertOnly {
		riverConfig.Queues = make(map[string]river.QueueConfig)

		// 从数据库获取队列配置
		queueConfigs, err := db.GetQueueConfigs(serverID)
		if err != nil {
			logger.Error("获取数据库队列配置失败: %v", err)
			// 使用默认配置
			riverConfig.Queues[river.QueueDefault] = river.QueueConfig{
				MaxWorkers: cfg.River.Workers,
			}
		} else {
			// 使用数据库中的配置并缓存
			for _, queueCfg := range queueConfigs {
				riverConfig.Queues[queueCfg.QueueName] = river.QueueConfig{
					MaxWorkers: queueCfg.MaxWorkers,
				}
				// 缓存队列配置
				manager.queueConfigs[queueCfg.QueueName] = &queueCfg
				logger.Info("加载数据库队列配置: %s, MaxWorkers: %d, TaskTable: %s",
					queueCfg.QueueName, queueCfg.MaxWorkers, queueCfg.TaskTable)
			}
		}

		// 如果数据库中没有配置，使用默认队列
		if len(riverConfig.Queues) == 0 {
			riverConfig.Queues[river.QueueDefault] = river.QueueConfig{
				MaxWorkers: cfg.River.Workers,
			}
			logger.Info("使用默认队列配置: MaxWorkers: %d", cfg.River.Workers)
		}
	}

	// 创建 River 客户端
	driver := riverpgxv5.New(db.GetPGXPool())
	client, err := river.NewClient(driver, riverConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建River客户端失败: %w", err)
	}

	manager.client = client

	logger.Info("River队列管理器初始化成功, 插入模式: %t", cfg.River.InsertOnly)
	return manager, nil
}

// Start 启动 River 管理器
func (rm *RiverManager) Start() error {
	if rm.insertOnly {
		logger.Info("River管理器以插入模式启动")
		return nil
	}

	logger.Info("启动River队列管理器...")

	// 启动 River 客户端
	if err := rm.client.Start(rm.ctx); err != nil {
		return fmt.Errorf("启动River客户端失败: %w", err)
	}

	logger.Info("River队列管理器启动成功")
	return nil
}

// Stop 停止 River 管理器
func (rm *RiverManager) Stop() {
	logger.Info("正在停止River队列管理器...")

	rm.cancel()

	if rm.client != nil {
		if err := rm.client.Stop(context.Background()); err != nil {
			logger.Error("停止River客户端失败: %v", err)
		}
	}

	logger.Info("River队列管理器已停止")
}

// RegisterTaskHandler 注册任务处理器
func (rm *RiverManager) RegisterTaskHandler(taskType string, handler models.JobHandler) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.taskHandlers[taskType] = handler
	logger.Info("注册River任务处理器: %s", taskType)
}

// EnqueueTask 入队任务到指定队列
func (rm *RiverManager) EnqueueTask(taskType, taskParams string) error {
	return rm.EnqueueTaskToQueue(taskType, taskParams, "")
}

// EnqueueTaskToQueue 入队任务到指定队列
func (rm *RiverManager) EnqueueTaskToQueue(taskType, taskParams, queueName string) error {
	// 如果没有指定队列，根据任务类型自动选择队列
	if queueName == "" {
		queueName = rm.getQueueForTaskType(taskType)
	}

	// 创建插入选项，指定队列
	insertOpts := &river.InsertOpts{
		Queue: queueName,
	}

	var err error

	switch taskType {
	case "email":
		_, err = rm.client.Insert(rm.ctx, EmailTaskArgs{TaskParams: taskParams}, insertOpts)
	case "data_sync":
		_, err = rm.client.Insert(rm.ctx, DataSyncTaskArgs{TaskParams: taskParams}, insertOpts)
	case "file_process":
		_, err = rm.client.Insert(rm.ctx, FileProcessTaskArgs{TaskParams: taskParams}, insertOpts)
	case "report_generate":
		_, err = rm.client.Insert(rm.ctx, ReportGenerateTaskArgs{TaskParams: taskParams}, insertOpts)
	case "notification":
		_, err = rm.client.Insert(rm.ctx, NotificationTaskArgs{TaskParams: taskParams}, insertOpts)
	case "demo":
		_, err = rm.client.Insert(rm.ctx, DemoTaskArgs{TaskParams: taskParams}, insertOpts)
	default:
		// 对于未知类型，使用 demo 作为默认
		_, err = rm.client.Insert(rm.ctx, DemoTaskArgs{TaskParams: taskParams}, insertOpts)
	}

	if err != nil {
		logger.Error("任务入队失败: %s (队列: %s), 错误: %v", taskType, queueName, err)
		return err
	}

	logger.Info("任务入队成功: %s (队列: %s)", taskType, queueName)
	return nil
}

// getQueueForTaskType 根据任务类型获取对应的队列名称
func (rm *RiverManager) getQueueForTaskType(taskType string) string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 从数据库配置中查找匹配的队列
	for queueName, queueConfig := range rm.queueConfigs {
		// 如果配置了TaskTable且包含任务类型，则使用该队列
		if queueConfig.TaskTable != "" {
			// 检查TaskTable是否包含TaskType相关的表名
			if rm.isTaskTypeMatchQueue(taskType, queueConfig.TaskTable) {
				return queueName
			}
		}
	}

	// 默认映射策略（作为备选）
	queueMapping := map[string]string{
		"email":           "email",
		"report_generate": "report",
		"notification":    "high_priority",
		"data_sync":       "default",
		"file_process":    "default",
		"demo":            "low_priority",
	}

	if queueName, exists := queueMapping[taskType]; exists {
		// 检查队列是否在数据库配置中存在
		if _, configured := rm.queueConfigs[queueName]; configured {
			return queueName
		}
	}

	// 默认使用 default 队列
	return river.QueueDefault
}

// isTaskTypeMatchQueue 检查任务类型是否匹配队列的TaskTable配置
func (rm *RiverManager) isTaskTypeMatchQueue(taskType, taskTable string) bool {
	// 解析 taskTable 格式: connection_name.table_name
	// 如: default.email_queue, default.report_queue
	if taskTable == "" {
		return false
	}

	// 简单的匹配逻辑：检查表名是否包含任务类型
	// 例如: email 任务匹配 email_queue表
	// report_generate 任务匹配 report_queue表
	taskTypeMapping := map[string][]string{
		"email":           {"email"},
		"report_generate": {"report"},
		"notification":    {"notification", "notify"},
		"data_sync":       {"sync", "data"},
		"file_process":    {"file"},
		"demo":            {"demo", "test"},
	}

	if keywords, exists := taskTypeMapping[taskType]; exists {
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(taskTable), keyword) {
				return true
			}
		}
	}

	return false
}

// GetStats 获取统计信息
func (rm *RiverManager) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats["type"] = "river"
	stats["server_id"] = rm.serverID
	stats["insert_only"] = rm.insertOnly
	stats["registered_handlers"] = len(rm.taskHandlers)

	// 添加队列配置信息
	queues := make(map[string]interface{})
	for queueName, queueConfig := range rm.queueConfigs {
		queues[queueName] = map[string]interface{}{
			"max_workers": queueConfig.MaxWorkers,
			"priority":    queueConfig.Priority,
			"task_table":  queueConfig.TaskTable,
			"server_id":   queueConfig.ServerID,
			"status":      queueConfig.Status,
		}
	}
	stats["queues"] = queues

	return stats, nil
}

// GetQueueStats 获取队列实时统计信息
func (rm *RiverManager) GetQueueStats() (map[string]interface{}, error) {
	if rm.client == nil {
		return nil, fmt.Errorf("River客户端未初始化")
	}

	// 这里可以添加查询River队列状态的逻辑
	// 由于River的API限制，我们主要返回配置信息
	stats := make(map[string]interface{})

	// 获取已配置的队列信息
	queues := make([]map[string]interface{}, 0)
	for queueName, queueConfig := range rm.queueConfigs {
		queueInfo := map[string]interface{}{
			"name":        queueName,
			"max_workers": queueConfig.MaxWorkers,
			"priority":    queueConfig.Priority,
			"task_table":  queueConfig.TaskTable,
			"server_id":   queueConfig.ServerID,
			"status":      "active",
		}
		queues = append(queues, queueInfo)
	}

	stats["queues"] = queues
	stats["total_queues"] = len(rm.queueConfigs)
	stats["insert_only_mode"] = rm.insertOnly

	return stats, nil
}

// ListAvailableQueues 列出所有可用的队列
func (rm *RiverManager) ListAvailableQueues() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	queues := make([]string, 0, len(rm.queueConfigs))
	for queueName := range rm.queueConfigs {
		queues = append(queues, queueName)
	}
	return queues
}

// RefreshQueueConfigs 从数据库刷新队列配置
func (rm *RiverManager) RefreshQueueConfigs() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 从数据库重新加载配置
	queueConfigs, err := rm.db.GetQueueConfigs(rm.serverID)
	if err != nil {
		return fmt.Errorf("刷新队列配置失败: %w", err)
	}

	// 清空旧配置
	for k := range rm.queueConfigs {
		delete(rm.queueConfigs, k)
	}

	// 加载新配置
	for _, queueCfg := range queueConfigs {
		rm.queueConfigs[queueCfg.QueueName] = &queueCfg
		logger.Info("刷新队列配置: %s, MaxWorkers: %d, TaskTable: %s",
			queueCfg.QueueName, queueCfg.MaxWorkers, queueCfg.TaskTable)
	}

	return nil
}

// GetQueueConfig 获取指定队列的配置
func (rm *RiverManager) GetQueueConfig(queueName string) (*database.QueueConfig, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	config, exists := rm.queueConfigs[queueName]
	if !exists {
		return nil, false
	}

	// 返回副本以防止外部修改
	configCopy := *config
	return &configCopy, true
}
