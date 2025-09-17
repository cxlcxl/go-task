package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"xxljob-go-executor/database"
	"xxljob-go-executor/logger"
	"xxljob-go-executor/models"
)

// QueueManager 队列管理器
type QueueManager struct {
	db           *database.Database
	serverID     string
	queues       map[string]*Queue
	taskHandlers map[string]models.JobHandler
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// Queue 队列结构
type Queue struct {
	Name         string
	TableName    string
	MaxWorkers   int
	ScanInterval time.Duration
	workerPool   chan struct{}
	isRunning    bool
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// TaskItem 任务项
type TaskItem struct {
	ID           int64                  `json:"id"`
	TaskType     string                 `json:"task_type"`
	TaskParams   string                 `json:"task_params"`
	ScheduleTime time.Time              `json:"schedule_time"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	RawData      map[string]interface{} `json:"raw_data"`
}

func NewQueueManager(db *database.Database, serverID string) *QueueManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &QueueManager{
		db:           db,
		serverID:     serverID,
		queues:       make(map[string]*Queue),
		taskHandlers: make(map[string]models.JobHandler),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// RegisterTaskHandler 注册任务处理器
func (qm *QueueManager) RegisterTaskHandler(taskType string, handler models.JobHandler) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	qm.taskHandlers[taskType] = handler
	logger.Info("注册任务处理器: %s", taskType)
}

// Start 启动队列管理器
func (qm *QueueManager) Start() error {
	logger.Info("启动队列管理器，服务器ID: %s", qm.serverID)

	// 恢复孤立的任务（重启时执行）
	if err := qm.db.RecoverOrphanedTasks(qm.serverID); err != nil {
		logger.Error("恢复孤立任务失败: %v", err)
		// 不阻止启动，只记录错误
	} else {
		logger.Info("孤立任务恢复完成")
	}

	// 启动队列配置监控协程
	go qm.monitorQueueConfigs()

	// 注册服务器信息
	if err := qm.registerServer(); err != nil {
		return fmt.Errorf("注册服务器失败: %w", err)
	}

	// 启动心跳协程
	go qm.startHeartbeat()

	// 启动超时任务监控
	go qm.startTimeoutMonitor()

	return nil
}

// Stop 停止队列管理器
func (qm *QueueManager) Stop() {
	logger.Info("停止队列管理器")
	qm.cancel()

	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	// 停止所有队列
	for _, queue := range qm.queues {
		queue.stop()
	}

	// 等待一段时间让正在执行的任务完成
	logger.Info("等待正在执行的任务完成...")
	time.Sleep(5 * time.Second) // 等待5秒

	logger.Info("队列管理器已停止")
}

// startTimeoutMonitor 启动超时任务监控
func (qm *QueueManager) startTimeoutMonitor() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			if err := qm.db.RecoverTimeoutTasks(30); err != nil { // 30分钟超时
				logger.Error("恢复超时任务失败: %v", err)
			}
		}
	}
}

// monitorQueueConfigs 监控队列配置变化
func (qm *QueueManager) monitorQueueConfigs() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次配置变化
	defer ticker.Stop()

	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			qm.updateQueues()
		}
	}
}

// updateQueues 更新队列配置
func (qm *QueueManager) updateQueues() {
	configs, err := qm.db.GetQueueConfigs(qm.serverID)
	if err != nil {
		logger.Error("获取队列配置失败: %v", err)
		return
	}

	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	// 记录当前活跃的队列
	activeQueues := make(map[string]bool)

	for _, config := range configs {
		activeQueues[config.QueueName] = true

		// 检查队列是否存在，不存在则创建
		if queue, exists := qm.queues[config.QueueName]; !exists {
			qm.createQueue(config)
		} else {
			// 检查配置是否有变化
			if queue.MaxWorkers != config.MaxWorkers ||
				queue.ScanInterval != time.Duration(config.ScanInterval)*time.Second ||
				queue.TableName != config.TableName {
				// 停止旧队列，创建新队列
				queue.stop()
				qm.createQueue(config)
				logger.Info("队列配置已更新: %s", config.QueueName)
			}
		}
	}

	// 停止不再活跃的队列
	for queueName, queue := range qm.queues {
		if !activeQueues[queueName] {
			queue.stop()
			delete(qm.queues, queueName)
			logger.Info("队列已停止: %s", queueName)
		}
	}
}

// createQueue 创建队列
func (qm *QueueManager) createQueue(config database.QueueConfig) {
	ctx, cancel := context.WithCancel(qm.ctx)

	queue := &Queue{
		Name:         config.QueueName,
		TableName:    config.TableName,
		MaxWorkers:   config.MaxWorkers,
		ScanInterval: time.Duration(config.ScanInterval) * time.Second,
		workerPool:   make(chan struct{}, config.MaxWorkers),
		isRunning:    true,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 初始化工作池
	for i := 0; i < config.MaxWorkers; i++ {
		queue.workerPool <- struct{}{}
	}

	qm.queues[config.QueueName] = queue

	// 启动队列扫描协程
	go qm.runQueue(queue)

	logger.Info("队列已创建并启动: %s, 表名: %s, 最大并发数: %d, 扫描间隔: %v",
		config.QueueName, config.TableName, config.MaxWorkers, queue.ScanInterval)
}

// runQueue 运行队列扫描
func (qm *QueueManager) runQueue(queue *Queue) {
	ticker := time.NewTicker(queue.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-queue.ctx.Done():
			return
		case <-ticker.C:
			qm.scanAndExecuteTasks(queue)
		}
	}
}

// scanAndExecuteTasks 扫描并执行任务
func (qm *QueueManager) scanAndExecuteTasks(queue *Queue) {
	// 获取可用的工作协程数量
	availableWorkers := len(queue.workerPool)
	if availableWorkers == 0 {
		return // 没有可用的工作协程
	}

	// 获取待执行的任务
	tasks, err := qm.db.GetPendingTasks(queue.TableName, availableWorkers)
	if err != nil {
		logger.Error("获取待执行任务失败，队列: %s, 错误: %v", queue.Name, err)
		return
	}

	if len(tasks) == 0 {
		return // 没有待执行的任务
	}

	logger.Debug("队列 %s 发现 %d 个待执行任务", queue.Name, len(tasks))

	// 为每个任务分配工作协程
	for _, taskData := range tasks {
		select {
		case <-queue.workerPool: // 获取工作协程
			go qm.executeTask(queue, taskData)
		default:
			// 没有可用的工作协程，跳出循环
			break
		}
	}
}

// executeTask 执行任务
func (qm *QueueManager) executeTask(queue *Queue, taskData map[string]interface{}) {
	defer func() {
		// 归还工作协程到池中
		queue.workerPool <- struct{}{}
	}()

	// 解析任务数据
	task, err := qm.parseTaskData(taskData)
	if err != nil {
		logger.Error("解析任务数据失败: %v", err)
		return
	}

	// 更新任务状态为执行中
	if err := qm.db.UpdateTaskStatus(queue.TableName, task.ID, database.TaskStatusRunning, qm.serverID, ""); err != nil {
		logger.Error("更新任务状态失败: %v", err)
		return
	}

	logger.Info("开始执行任务，队列: %s, 任务ID: %d, 类型: %s", queue.Name, task.ID, task.TaskType)

	// 获取任务处理器
	qm.mutex.RLock()
	handler, exists := qm.taskHandlers[task.TaskType]
	qm.mutex.RUnlock()

	if !exists {
		errorMsg := fmt.Sprintf("未找到任务处理器: %s", task.TaskType)
		logger.Error(errorMsg)
		qm.db.UpdateTaskStatus(queue.TableName, task.ID, database.TaskStatusFailed, qm.serverID, errorMsg)
		return
	}

	// 创建任务上下文
	jobCtx := &models.JobContext{
		JobID:       int(task.ID),
		JobParam:    task.TaskParams,
		LogID:       int(task.ID),
		LogDateTime: task.ScheduleTime,
	}

	// 执行任务
	if err := handler.Execute(jobCtx); err != nil {
		logger.Error("任务执行失败，队列: %s, 任务ID: %d, 错误: %v", queue.Name, task.ID, err)

		// 检查是否需要重试
		if task.RetryCount < task.MaxRetries {
			// 增加重试次数，状态重置为待执行
			qm.db.IncrementRetryCount(queue.TableName, task.ID)
			qm.db.UpdateTaskStatus(queue.TableName, task.ID, database.TaskStatusPending, "", err.Error())
			logger.Info("任务将重试，队列: %s, 任务ID: %d, 重试次数: %d/%d",
				queue.Name, task.ID, task.RetryCount+1, task.MaxRetries)
		} else {
			// 重试次数已达上限，标记为失败
			qm.db.UpdateTaskStatus(queue.TableName, task.ID, database.TaskStatusFailed, qm.serverID, err.Error())
		}
	} else {
		// 任务执行成功
		qm.db.UpdateTaskStatus(queue.TableName, task.ID, database.TaskStatusCompleted, qm.serverID, "")
		logger.Info("任务执行成功，队列: %s, 任务ID: %d", queue.Name, task.ID)
	}
}

// parseTaskData 解析任务数据
func (qm *QueueManager) parseTaskData(data map[string]interface{}) (*TaskItem, error) {
	task := &TaskItem{
		RawData: data,
	}

	if id, ok := data["id"].(int64); ok {
		task.ID = id
	} else {
		return nil, fmt.Errorf("无效的任务ID")
	}

	if taskType, ok := data["task_type"].(string); ok {
		task.TaskType = taskType
	} else {
		return nil, fmt.Errorf("无效的任务类型")
	}

	if params, ok := data["task_params"].(string); ok {
		task.TaskParams = params
	}

	if scheduleTime, ok := data["schedule_time"].(time.Time); ok {
		task.ScheduleTime = scheduleTime
	}

	if retryCount, ok := data["retry_count"].(int); ok {
		task.RetryCount = retryCount
	}

	if maxRetries, ok := data["max_retries"].(int); ok {
		task.MaxRetries = maxRetries
	}

	return task, nil
}

// registerServer 注册服务器
func (qm *QueueManager) registerServer() error {
	return qm.db.CreateOrUpdateServer(qm.serverID, fmt.Sprintf("server-%s", qm.serverID), "localhost", 9999)
}

// startHeartbeat 启动心跳
func (qm *QueueManager) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			if err := qm.db.UpdateServerHeartbeat(qm.serverID, fmt.Sprintf("server-%s", qm.serverID), "localhost", 9999); err != nil {
				logger.Error("更新服务器心跳失败: %v", err)
			}
		}
	}
}

// stop 停止队列
func (q *Queue) stop() {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.isRunning {
		q.isRunning = false
		q.cancel()
	}
}
