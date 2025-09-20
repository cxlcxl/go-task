package jobs

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/syncmap"
	"sync"
	"task-executor/config"
	"task-executor/database"
	"task-executor/defines"
	"task-executor/logger"
	"time"
)

type Manager struct {
	config       *config.Config
	db           *database.Database
	serverID     string
	queueConfigs map[string]*database.QueueConfig // 数据库中的队列配置
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	pools        syncmap.Map
	deleteChan   chan int64
}

// NewManager 创建队列管理器
func NewManager(cfg *config.Config, db *database.Database, serverID string) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	// 创建管理器实例
	manager := &Manager{
		config:       cfg,
		db:           db,
		serverID:     serverID,
		queueConfigs: make(map[string]*database.QueueConfig),
		ctx:          ctx,
		cancel:       cancel,
		pools:        syncmap.Map{},
		deleteChan:   make(chan int64),
	}
	return manager, nil
}

// Start 启动管理器
func (rm *Manager) Start() error {
	logger.Info("启动队列管理器...")
	go rm.watchQueueConfigs()

	return nil
}

func (rm *Manager) NewQueue(queueConfig database.QueueConfig, attempt int) {
	handler, exists := defines.TaskHandles[queueConfig.QueueName]
	if !exists {
		logger.Error(fmt.Sprintf("任务处理器未注册: %s", queueConfig.QueueName))
		return
	}
	var (
		pool *ants.Pool
		err  error
	)
	pool, err = ants.NewPool(queueConfig.MaxWorkers)
	if err != nil {
		logger.Error(fmt.Sprintf("协程池启动失败，当前第 %d 次重试，错误信息: %s", attempt, err.Error()))
		// 指数级休眠时间
		time.Sleep(time.Duration(1<<attempt) * time.Second)
		rm.NewQueue(queueConfig, attempt+1)
		return
	}
	rm.pools.Store(queueConfig.QueueName, pool)
	logger.Info(fmt.Sprintf("协程池启动成功，队列名称: %s", queueConfig.QueueName))

	queueTable := rm.db.ParseTableName(queueConfig.TaskTable)
	for {
		freeWorkers := pool.Free()
		if freeWorkers == 0 {
			time.Sleep(time.Second * time.Duration(queueConfig.FrequencyInterval))
			continue
		}

		tasks := rm.db.GetPendingTasks(queueTable, queueConfig.QueueName, rm.serverID, freeWorkers)
		if len(tasks) == 0 {
			time.Sleep(time.Second * time.Duration(queueConfig.FrequencyInterval))
			continue
		}

		for _, task := range tasks {
			err = pool.Submit(handler(rm.db, task, *queueTable))
			if err != nil {
				logger.Error(fmt.Sprintf("任务处理失败，错误信息: %s", err.Error()))
			}
			logger.Info(fmt.Sprintf("任务处理成功: %d", task.ID))
		}
	}
}

func (rm *Manager) watchQueueConfigs() {
	rm.ListenQueue()

	go rm.RemoveDisabledQueue()

	ticker := time.NewTicker(time.Minute * 1)
	for {
		select {
		case <-ticker.C:
			rm.ListenQueue()
		}
	}
}

func (rm *Manager) ListenQueue() {
	configs, err := rm.db.GetQueueConfigs(rm.serverID)
	if err != nil {
		logger.Error(fmt.Sprintf("队列配置获取失败，错误信息: %s", err.Error()))
		return
	}

	for _, queueConfig := range configs {
		value, ok := rm.pools.Load(queueConfig.QueueName)
		// 没有协程池，新增的队列配置
		if !ok {
			if queueConfig.Status == database.QueueStatusEnabled {
				logger.Info(fmt.Sprintf("补充协程池，队列名称: %s", queueConfig.QueueName))
				go rm.NewQueue(queueConfig, 0)
				continue
			}
		} else {
			if queueConfig.Status == database.QueueStatusDisabled {
				logger.Info(fmt.Sprintf("队列已禁用，正在释放协程池，队列名称: %s", queueConfig.QueueName))
				value.(*ants.Pool).Release()
				rm.pools.Delete(queueConfig.QueueName)
				rm.deleteChan <- queueConfig.ID
				continue
			}
			if queueConfig.MaxWorkers != value.(*ants.Pool).Cap() {
				value.(*ants.Pool).Tune(queueConfig.MaxWorkers)
			}
		}
	}
}

// Stop 停管理器
func (rm *Manager) Stop() {
	logger.Info("正在停止队列管理器...")

	rm.cancel()

	rm.pools.Range(func(key, value interface{}) bool {
		value.(*ants.Pool).Release()
		return true
	})

	logger.Info("队列管理器已停止")
}

func (rm *Manager) RemoveDisabledQueue() {
	for {
		select {
		case id := <-rm.deleteChan:
			rm.db.RemoveDisabledQueue([]int64{id})
		}
	}
}
