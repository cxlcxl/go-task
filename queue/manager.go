package queue

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/syncmap"
	"sync"
	"task-executor/config"
	"task-executor/database"
	"task-executor/logger"
	"task-executor/vars"
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
	handler, exists := vars.QueueTaskHandles[queueConfig.QueueName]
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
	c, err := rm.db.GetConnect(queueTable)
	if err != nil {
		logger.Error(fmt.Sprintf("获取数据库连接失败，当前第 %d 次重试，错误信息: %s", attempt, err.Error()))
		time.Sleep(time.Duration(1<<attempt) * time.Second)
		rm.NewQueue(queueConfig, attempt+1)
		return
	}

	for {
		tasks := rm.db.GetPendingTasks(c, queueTable.TableName, queueConfig.QueueName, pool.Free())
		if len(tasks) == 0 {
			time.Sleep(time.Second)
		}

		for _, task := range tasks {
			err = pool.Submit(handler(task, *queueTable))
			if err != nil {
				logger.Error(fmt.Sprintf("任务处理失败，错误信息: %s", err.Error()))
			}
		}
	}
}

func (rm *Manager) watchQueueConfigs() {
	rm.ListenQueue()

	ticker := time.NewTicker(time.Minute * 2)
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
		if !ok && queueConfig.Status == database.QueueStatusEnabled {
			go rm.NewQueue(queueConfig, 0)
		}
		if ok && queueConfig.Status == database.QueueStatusDisabled {
			value.(*ants.Pool).Release()
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
