package define

import "task-executor/database"

const (
	// TaskTypeDefault 任务类型，default 执行器方法，http 发送 http 请求
	TaskTypeDefault = "default"
	TaskTypeHTTP    = "http"
)

// TaskTypeHttpParam http 任务参数
type TaskTypeHttpParam struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout"`
	Retries int               `json:"retries"`
}

type TaskHandler func(*database.Database, *database.TaskData, database.QueueTable) func()
