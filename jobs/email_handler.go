package jobs

import (
	"fmt"
	"task-executor/database"
)

func EmailTaskExecutor(data database.TaskData, table database.QueueTable) func() {
	return func() {
		fmt.Println("执行任务: ", data, table)
	}
}
