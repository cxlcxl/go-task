package define

import (
	"task-executor/executor"
	"task-executor/jobs"
	"task-executor/jobs/task"
	"task-executor/jobs/xxljob"
)

var (
	TaskHandles = map[string]jobs.TaskHandler{
		"email_send":        task.EmailSenderTask,
		"advertiser_report": task.AdvertiserReportTask,
	}
	XXLJobHandlers = map[string]executor.JobHandler{
		"email_send": xxljob.NewEmailJobHandler(),
	}
)
