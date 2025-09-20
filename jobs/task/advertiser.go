package task

import "task-executor/database"

func AdvertiserReportTask(db *database.Database, data *database.TaskData, table database.QueueTable) func() {
	return func() {

	}
}
