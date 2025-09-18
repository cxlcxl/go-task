package asynq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"xxljob-go-executor/config"
	"xxljob-go-executor/models"

	"github.com/hibiken/asynq"
)

// MockJobHandler 模拟任务处理器
type MockJobHandler struct {
	executed bool
	error    error
}

func (m *MockJobHandler) Execute(ctx *models.JobContext) error {
	m.executed = true
	return m.error
}

// TestAsynqManager 测试Asynq管理器
func TestAsynqManager(t *testing.T) {
	// 跳过测试如果没有Redis环境
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建测试配置
	cfg := &config.Config{
		Redis: struct {
			Addr     string `yaml:"addr"`
			Password string `yaml:"password"`
			DB       int    `yaml:"db"`
		}{
			Addr:     "localhost:6379",
			Password: "",
			DB:       1, // 使用测试数据库
		},
		Asynq: struct {
			Enable      bool           `yaml:"enable"`
			Concurrency int            `yaml:"concurrency"`
			Queues      map[string]int `yaml:"queues"`
		}{
			Enable:      true,
			Concurrency: 5,
			Queues: map[string]int{
				"test": 1,
			},
		},
	}

	// 创建Asynq管理器
	manager, err := NewAsynqManager(cfg)
	if err != nil {
		t.Fatalf("创建Asynq管理器失败: %v", err)
	}
	defer manager.Stop()

	// 测试注册任务处理器
	mockHandler := &MockJobHandler{}
	manager.RegisterTaskHandler("test_task", mockHandler)

	// 启动服务器
	err = manager.Start()
	if err != nil {
		t.Fatalf("启动Asynq服务器失败: %v", err)
	}

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试入队任务
	taskInfo, err := manager.EnqueueTask("test_task", `{"test": "data"}`, asynq.Queue("test"))
	if err != nil {
		t.Fatalf("任务入队失败: %v", err)
	}

	if taskInfo.ID == "" {
		t.Error("任务ID不应为空")
	}

	// 等待任务执行
	time.Sleep(500 * time.Millisecond)

	// 验证任务是否被执行
	if !mockHandler.executed {
		t.Error("任务未被执行")
	}

	// 测试获取统计信息
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("获取统计信息失败: %v", err)
	}

	if stats == nil {
		t.Error("统计信息不应为空")
	}
}

// TestTaskWrapper 测试任务包装器
func TestTaskWrapper(t *testing.T) {
	mockHandler := &MockJobHandler{}
	wrapper := NewTaskWrapper(mockHandler, nil)

	// 创建测试任务
	payload := TaskPayload{
		TaskType:   "test_task",
		TaskParams: `{"test": "data"}`,
	}

	task := asynq.NewTask("test_task", payload)
	ctx := context.Background()

	// 执行任务
	err := wrapper.ProcessTask(ctx, task)
	if err != nil {
		t.Fatalf("任务执行失败: %v", err)
	}

	// 验证处理器被调用
	if !mockHandler.executed {
		t.Error("处理器未被调用")
	}
}

// TestTaskWrapperError 测试任务包装器错误处理
func TestTaskWrapperError(t *testing.T) {
	expectedError := "test error"
	mockHandler := &MockJobHandler{
		error: fmt.Errorf(expectedError),
	}
	wrapper := NewTaskWrapper(mockHandler, nil)

	payload := TaskPayload{
		TaskType:   "test_task",
		TaskParams: `{"test": "data"}`,
	}

	task := asynq.NewTask("test_task", payload)
	ctx := context.Background()

	// 执行任务
	err := wrapper.ProcessTask(ctx, task)
	if err == nil {
		t.Fatal("期望任务执行失败")
	}

	if err.Error() != expectedError {
		t.Errorf("期望错误: %s, 实际错误: %s", expectedError, err.Error())
	}
}

// TestMiddleware 测试中间件
func TestMiddleware(t *testing.T) {
	t.Run("LoggingMiddleware", func(t *testing.T) {
		executed := false

		handler := asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			executed = true
			return nil
		})

		middleware := LoggingMiddleware()
		wrappedHandler := middleware(handler)

		task := asynq.NewTask("test_task", nil)
		ctx := context.Background()

		err := wrappedHandler.ProcessTask(ctx, task)
		if err != nil {
			t.Fatalf("中间件执行失败: %v", err)
		}

		if !executed {
			t.Error("处理器未被执行")
		}
	})

	t.Run("RecoveryMiddleware", func(t *testing.T) {
		handler := asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			panic("test panic")
		})

		middleware := RecoveryMiddleware()
		wrappedHandler := middleware(handler)

		task := asynq.NewTask("test_task", nil)
		ctx := context.Background()

		err := wrappedHandler.ProcessTask(ctx, task)
		if err != asynq.SkipRetry {
			t.Errorf("期望错误: %v, 实际错误: %v", asynq.SkipRetry, err)
		}
	})
}

// BenchmarkTaskExecution 性能测试
func BenchmarkTaskExecution(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过性能测试")
	}

	cfg := &config.Config{
		Redis: struct {
			Addr     string `yaml:"addr"`
			Password string `yaml:"password"`
			DB       int    `yaml:"db"`
		}{
			Addr:     "localhost:6379",
			Password: "",
			DB:       1,
		},
		Asynq: struct {
			Enable      bool           `yaml:"enable"`
			Concurrency int            `yaml:"concurrency"`
			Queues      map[string]int `yaml:"queues"`
		}{
			Enable:      true,
			Concurrency: 10,
			Queues: map[string]int{
				"benchmark": 5,
			},
		},
	}

	manager, err := NewAsynqManager(cfg)
	if err != nil {
		b.Fatalf("创建Asynq管理器失败: %v", err)
	}
	defer manager.Stop()

	// 注册快速处理器
	fastHandler := &MockJobHandler{}
	manager.RegisterTaskHandler("fast_task", fastHandler)

	err = manager.Start()
	if err != nil {
		b.Fatalf("启动Asynq服务器失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := manager.EnqueueTask("fast_task", `{"benchmark": true}`, asynq.Queue("benchmark"))
			if err != nil {
				b.Errorf("任务入队失败: %v", err)
			}
		}
	})
}

// TestEnqueueTaskIn 测试延迟任务
func TestEnqueueTaskIn(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	cfg := &config.Config{
		Redis: struct {
			Addr     string `yaml:"addr"`
			Password string `yaml:"password"`
			DB       int    `yaml:"db"`
		}{
			Addr:     "localhost:6379",
			Password: "",
			DB:       1,
		},
		Asynq: struct {
			Enable      bool           `yaml:"enable"`
			Concurrency int            `yaml:"concurrency"`
			Queues      map[string]int `yaml:"queues"`
		}{
			Enable:      true,
			Concurrency: 5,
			Queues: map[string]int{
				"delayed": 1,
			},
		},
	}

	manager, err := NewAsynqManager(cfg)
	if err != nil {
		t.Fatalf("创建Asynq管理器失败: %v", err)
	}
	defer manager.Stop()

	mockHandler := &MockJobHandler{}
	manager.RegisterTaskHandler("delayed_task", mockHandler)

	err = manager.Start()
	if err != nil {
		t.Fatalf("启动Asynq服务器失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// 入队延迟1秒的任务
	delay := 1 * time.Second
	taskInfo, err := manager.EnqueueTaskIn("delayed_task", `{"delayed": true}`, delay, asynq.Queue("delayed"))
	if err != nil {
		t.Fatalf("延迟任务入队失败: %v", err)
	}

	if taskInfo.ID == "" {
		t.Error("任务ID不应为空")
	}

	// 立即检查，任务不应该被执行
	if mockHandler.executed {
		t.Error("延迟任务不应该立即执行")
	}

	// 等待延迟时间
	time.Sleep(delay + 200*time.Millisecond)

	// 现在任务应该被执行
	if !mockHandler.executed {
		t.Error("延迟任务应该被执行")
	}
}
