package task

import (
	"task-executor/database"
	"task-executor/logger"
	"time"
)

func EmailSenderTask(db *database.Database, data *database.TaskData, table database.QueueTable) func() {
	return func() {
		_ = db.SetTaskStart(&table, data.ID)

		logger.Info("------------ 开始执行任务: ", data, table)
		time.Sleep(3 * time.Second)
		logger.Info("------------ 执行完成任务: ", data, table)
		_ = db.SetTaskFinish(&table, data.ID)
	}
}
