package asynq

import (
	"context"
	"runtime/debug"
	"time"

	"xxljob-go-executor/logger"

	"github.com/hibiken/asynq"
)

// LoggingMiddleware 日志中间件
func LoggingMiddleware() asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			start := time.Now()

			logger.Info("开始处理任务: %s, Payload: %s", task.Type(), string(task.Payload()))

			err := h.ProcessTask(ctx, task)

			duration := time.Since(start)
			if err != nil {
				logger.Error("任务处理失败: %s, 耗时: %v, 错误: %v",
					task.Type(), duration, err)
			} else {
				logger.Info("任务处理成功: %s, 耗时: %v",
					task.Type(), duration)
			}

			return err
		})
	}
}

// RecoveryMiddleware 恢复中间件，防止panic导致整个worker崩溃
func RecoveryMiddleware() asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("任务执行发生panic: %s, panic: %v, stack: %s",
						task.Type(), r, debug.Stack())
					err = asynq.SkipRetry
				}
			}()

			return h.ProcessTask(ctx, task)
		})
	}
}

// MetricsMiddleware 指标中间件
func MetricsMiddleware() asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			start := time.Now()

			err := h.ProcessTask(ctx, task)

			// 记录指标
			duration := time.Since(start)
			taskType := task.Type()

			// 可以在这里集成到监控系统（如Prometheus）
			logger.Debug("任务指标 - 类型: %s, 耗时: %v, 成功: %t",
				taskType, duration, err == nil)

			return err
		})
	}
}

// RetryMiddleware 自定义重试中间件
func RetryMiddleware(maxRetries int) asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			err := h.ProcessTask(ctx, task)

			if err != nil {
				// 在 Asynq v0.24.1 中，我们无法直接获取重试次数
				// 可以通过task的payload或其他方式来跟踪重试次数
				logger.Info("任务执行失败，将重试: %s, 错误: %v",
					task.Type(), err)
			}

			return err
		})
	}
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware(timeout time.Duration) asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- h.ProcessTask(ctx, task)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				logger.Error("任务执行超时: %s, 超时时间: %v",
					task.Type(), timeout)
				return ctx.Err()
			}
		})
	}
}
