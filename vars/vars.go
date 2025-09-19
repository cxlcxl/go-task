package vars

import (
	"task-executor/jobs"
	"task-executor/models"
)

var (
	QueueTaskHandles = map[string]models.QueueTaskHandler{
		"email": jobs.EmailTaskExecutor,
	}
)
