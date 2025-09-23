package task

import (
	"task-executor/database"
)

// HttpParams 请求的固定参数
type HttpParams struct {
}

func HttpRequestTask(db *database.Database, data *database.TaskData, table database.QueueTable) func() {
	return func() {
		_ = db.SetTaskStart(&table, data.ID)

		_ = db.SetTaskFinish(&table, data.ID)
	}
}
