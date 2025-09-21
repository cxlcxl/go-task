package define

import (
	"task-executor/jobs/task"
	"task-executor/jobs/xxljob"
	"task-executor/models"
)

var (
	TaskHandles = map[string]models.TaskHandler{
		"email_send":        task.EmailSenderTask,
		"advertiser_report": task.AdvertiserReportTask,
	}
	XXLJobHandlers = map[string]models.JobHandler{
		"email_send": xxljob.NewEmailJobHandler(),
	}
)
