package jobs

import (
	"task-executor/define"
	"task-executor/jobs/task"
	"task-executor/jobs/xxljob"
)

var (
	TaskHandles = map[string]define.TaskHandler{
		"email_send":        task.EmailSenderTask,
		"advertiser_report": task.AdvertiserReportTask,
	}
	XXLJobHandlers = map[string]define.JobHandler{
		"email_send":                   xxljob.NewEmailJobHandler(),
		"gdt_advertiser_token_refresh": xxljob.NewEmailJobHandler(),
	}
)
