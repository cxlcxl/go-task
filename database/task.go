package database

import (
	"time"
)

// GenerateSubTasks 生产子任务
func (d *Database) GenerateSubTasks(queueName string) ([]map[string]interface{}, error) {
	queueConfig, err := d.GetQueueConfigByName(queueName)
	if err != nil {
		return nil, err
	}

	var tasks []map[string]interface{}

	// 解析表名
	dbName, realTableName, err := d.ParseTableName(queueConfig.TableName)
	if err != nil {
		return nil, err
	}

	// 获取对应的数据库连接
	db, err := d.GetConnection(dbName)
	if err != nil {
		return nil, err
	}

	task := &TaskBase{
		TaskType:     "",
		TaskParams:   "",
		Status:       TaskStatusPending,
		ScheduleTime: time.Time{},
		StartTime:    nil,
		EndTime:      nil,
		RetryCount:   0,
		MaxRetries:   0,
		ErrorMsg:     "",
		ServerID:     "",
		CreatedAt:    time.Time{},
		UpdatedAt:    time.Time{},
	}

	result := db.Table(realTableName).Create(task)
	return tasks, result.Error
}
