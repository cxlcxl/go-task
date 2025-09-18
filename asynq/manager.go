package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"xxljob-go-executor/config"
	"xxljob-go-executor/logger"
	"xxljob-go-executor/models"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
)

// AsynqManager Asynq 管理器
type AsynqManager struct {
	config       *config.Config
	client       *asynq.Client
	server       *asynq.Server
	mux          *asynq.ServeMux
	inspector    *asynq.Inspector
	redisClient  *redis.Client
	taskHandlers map[string]models.JobHandler
	mu           sync.RWMutex
}

// TaskPayload 任务载荷
type TaskPayload struct {
	TaskType   string                 `json:"task_type"`
	TaskParams string                 `json:"task_params"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewAsynqManager 创建新的Asynq管理器
func NewAsynqManager(cfg *config.Config) (*AsynqManager, error) {
	if !cfg.Asynq.Enable {
		return nil, fmt.Errorf("Asynq功能未启用")
	}

	// 创建Redis连接选项
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 创建Asynq客户端
	client := asynq.NewClient(redisOpt)

	// 创建服务器配置
	serverConfig := asynq.Config{
		Concurrency: cfg.Asynq.Concurrency,
		Queues:      cfg.Asynq.Queues,
		Logger:      &AsynqLogger{},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			logger.Error("任务执行失败: %s, 错误: %v", task.Type(), err)
		}),
		// 启用严格优先级模式
		StrictPriority: true,
		// 设置关闭超时
		ShutdownTimeout: 5 * time.Second,
		// 设置健康检查
		HealthCheckFunc: func(error) {
			logger.Debug("Asynq健康检查通过")
		},
		// 启用任务重试
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			// 指数退避重试策略: 1s, 2s, 4s, 8s, 16s...
			return time.Duration(1<<uint(n)) * time.Second
		},
	}

	// 创建Asynq服务器
	server := asynq.NewServer(redisOpt, serverConfig)

	// 创建多路复用器
	mux := asynq.NewServeMux()

	// 创建检查器
	inspector := asynq.NewInspector(redisOpt)

	// 创建Redis客户端（用于额外操作）
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	logger.Info("Asynq管理器初始化成功，Redis连接: %s", cfg.Redis.Addr)

	return &AsynqManager{
		config:       cfg,
		client:       client,
		server:       server,
		mux:          mux,
		inspector:    inspector,
		redisClient:  redisClient,
		taskHandlers: make(map[string]models.JobHandler),
	}, nil
}

// RegisterTaskHandler 注册任务处理器
func (am *AsynqManager) RegisterTaskHandler(taskType string, handler models.JobHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.taskHandlers[taskType] = handler

	// 在Asynq中注册处理器
	am.mux.HandleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		return am.executeTask(ctx, task, handler)
	})

	logger.Info("注册Asynq任务处理器: %s", taskType)
}

// executeTask 执行任务
func (am *AsynqManager) executeTask(ctx context.Context, task *asynq.Task, handler models.JobHandler) error {
	// 解析任务载荷
	var payload TaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务载荷失败: %w", err)
	}

	// 创建任务上下文
	jobCtx := &models.JobContext{
		JobID:       int(time.Now().Unix()), // 使用时间戳作为临时ID
		JobParam:    payload.TaskParams,
		LogID:       int(time.Now().Unix()),
		LogDateTime: time.Now(),
	}

	logger.Info("开始执行Asynq任务: %s, 参数: %s", payload.TaskType, payload.TaskParams)

	// 执行任务
	if err := handler.Execute(jobCtx); err != nil {
		logger.Error("Asynq任务执行失败: %s, 错误: %v", payload.TaskType, err)
		return err
	}

	logger.Info("Asynq任务执行成功: %s", payload.TaskType)
	return nil
}

// EnqueueTask 入队任务
func (am *AsynqManager) EnqueueTask(taskType, taskParams string, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	payload := TaskPayload{
		TaskType:   taskType,
		TaskParams: taskParams,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)

	info, err := am.client.Enqueue(task, opts...)
	if err != nil {
		logger.Error("任务入队失败: %s, 错误: %v", taskType, err)
		return nil, err
	}

	logger.Info("任务入队成功: %s, 队列: %s, ID: %s", taskType, info.Queue, info.ID)
	return info, nil
}

// EnqueueTaskIn 延迟入队任务
func (am *AsynqManager) EnqueueTaskIn(taskType, taskParams string, delay time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	payload := TaskPayload{
		TaskType:   taskType,
		TaskParams: taskParams,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)
	opts = append(opts, asynq.ProcessIn(delay))

	info, err := am.client.Enqueue(task, opts...)
	if err != nil {
		logger.Error("延迟任务入队失败: %s, 错误: %v", taskType, err)
		return nil, err
	}

	logger.Info("延迟任务入队成功: %s, 延迟: %v, ID: %s", taskType, delay, info.ID)
	return info, nil
}

// Start 启动Asynq服务器
func (am *AsynqManager) Start() error {
	// 配置中间件
	am.setupMiddleware()

	logger.Info("启动Asynq服务器...")

	// 启动服务器
	go func() {
		if err := am.server.Run(am.mux); err != nil {
			logger.Error("Asynq服务器运行失败: %v", err)
		}
	}()

	logger.Info("Asynq服务器启动成功，并发数: %d", am.config.Asynq.Concurrency)
	return nil
}

// Stop 停止Asynq服务器
func (am *AsynqManager) Stop() {
	logger.Info("正在关闭Asynq服务器...")

	am.server.Shutdown()
	am.client.Close()
	am.redisClient.Close()

	logger.Info("Asynq服务器已关闭")
}

// setupMiddleware 设置中间件
func (am *AsynqManager) setupMiddleware() {
	// 日志中间件
	am.mux.Use(LoggingMiddleware())

	// 恢复中间件
	am.mux.Use(RecoveryMiddleware())

	// 指标中间件
	am.mux.Use(MetricsMiddleware())
}

// GetStats 获取统计信息
func (am *AsynqManager) GetStats() (map[string]interface{}, error) {
	// 在 Asynq v0.24.1 中，我们可以使用 inspector 的其他方法来获取统计信息
	stats := make(map[string]interface{})

	// 获取队列信息
	queues, err := am.inspector.Queues()
	if err != nil {
		return nil, err
	}

	stats["queues"] = queues
	return stats, nil
}

// GetTaskInfo 获取任务信息
func (am *AsynqManager) GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error) {
	return am.inspector.GetTaskInfo(queue, taskID)
}

// ListTasks 列出任务
func (am *AsynqManager) ListTasks(queue string, state asynq.TaskState) ([]*asynq.TaskInfo, error) {
	// 在 Asynq v0.24.1 中，我们使用不同的方法来列出任务
	switch state {
	case asynq.TaskStatePending:
		return am.inspector.ListPendingTasks(queue)
	case asynq.TaskStateActive:
		return am.inspector.ListActiveTasks(queue)
	case asynq.TaskStateScheduled:
		return am.inspector.ListScheduledTasks(queue)
	case asynq.TaskStateRetry:
		return am.inspector.ListRetryTasks(queue)
	case asynq.TaskStateArchived:
		return am.inspector.ListArchivedTasks(queue)
	default:
		return am.inspector.ListPendingTasks(queue)
	}
}

// CancelTask 取消任务
func (am *AsynqManager) CancelTask(queue, taskID string) error {
	// 在 Asynq v0.24.1 中，我们使用 ArchiveTask 来取消任务
	return am.inspector.ArchiveTask(queue, taskID)
}

// DeleteTask 删除任务
func (am *AsynqManager) DeleteTask(queue, taskID string) error {
	return am.inspector.DeleteTask(queue, taskID)
}

// AsynqLogger Asynq日志适配器
type AsynqLogger struct{}

func (l *AsynqLogger) Debug(args ...interface{}) {
	logger.Debug("[Asynq] %v", args...)
}

func (l *AsynqLogger) Info(args ...interface{}) {
	logger.Info("[Asynq] %v", args...)
}

func (l *AsynqLogger) Warn(args ...interface{}) {
	logger.Error("[Asynq] WARN: %v", args...)
}

func (l *AsynqLogger) Error(args ...interface{}) {
	logger.Error("[Asynq] ERROR: %v", args...)
}

func (l *AsynqLogger) Fatal(args ...interface{}) {
	logger.Error("[Asynq] FATAL: %v", args...)
	log.Fatal(args...)
}
