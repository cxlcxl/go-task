package database

import (
	"fmt"
	"gorm.io/datatypes"
	"task-executor/logger"
	"time"
)

type TaskData struct {
	ID           int64          `json:"id" gorm:"column:id;primaryKey;autoIncrement:true"`
	BatchNo      string         `json:"batch_no" gorm:"column:batch_no;type:varchar(80);not null"`
	QueueName    string         `json:"queue_name" gorm:"column:queue_name;type:varchar(80);not null"`
	TaskParams   datatypes.JSON `json:"task_params" gorm:"column:task_params;type:jsonb;not null;default:'{}'"`
	State        string         `json:"state" gorm:"column:state;type:varchar(20);not null"`
	ScheduleTime time.Time      `json:"schedule_time" gorm:"column:schedule_time;type:timestamp;not null;default:'2000-01-01 00:00:00'"`
	StartTime    time.Time      `json:"start_time" gorm:"column:start_time;type:timestamp;not null;default:'2000-01-01 00:00:00'"`
	EndTime      time.Time      `json:"end_time" gorm:"column:end_time;type:timestamp;not null;default:'2000-01-01 00:00:00'"`
	CostTime     int32          `json:"cost_time" gorm:"column:cost_time;type:integer;not null;default:0"`
	RetryCount   int32          `json:"retry_count" gorm:"column:retry_count;type:integer;not null;default:0"`
	MaxRetries   int32          `json:"max_retries" gorm:"column:max_retries;type:integer;not null;default:0"`
	ExecResult   datatypes.JSON `json:"exec_result" gorm:"column:exec_result;type:jsonb;not null;default:'{}'"`
	ServerID     string         `json:"server_id" gorm:"column:server_id;type:varchar(50);not null;default:''"`
	TaskType     string         `json:"task_type" gorm:"column:task_type;type:varchar(30);not null;default:'default'"`
	CreatedAt    time.Time      `json:"created_at" gorm:"column:created_at;type:timestamp;default:now()"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"column:updated_at;type:timestamp;default:current_timestamp"`
}

func (d *Database) GetPendingTasks(queueTable *QueueTable, queueName, serverId string, limit int) (tasks []*TaskData) {
	// 获取第一个连接用于更新操作
	uc, err := d.GetConnect(queueTable)
	if err != nil {
		logger.Error(fmt.Sprintf("获取数据库连接失败，错误信息: %s", err.Error()))
		return
	}

	// 先抢占任务，再查询，保证多机器部署时不会抢到相同的任务去执行
	affects := uc.Table(queueTable.TableName).Where("state = ? and queue_name = ?", TaskStatusInit, queueName).
		//Where("server_id = ''").
		Where("schedule_time <= ?", time.Now()).
		Order("schedule_time ASC").
		Limit(limit).
		Updates(map[string]interface{}{"state": TaskStatusPending, "server_id": serverId}).RowsAffected
	if affects == 0 {
		return
	}

	// 获取第二个独立连接用于查询操作，避免条件累积
	// SELECT * FROM "task_email"
	//WHERE (state = 'init' and queue_name = 'email_send_queue') AND schedule_time <= '2025-09-20 13:37:39.157' AND
	//(state = 'pending' and queue_name = 'email_send_queue' and server_id = 'task-01') AND
	//schedule_time <= '2025-09-20 13:37:39.214' ORDER BY schedule_time ASC,schedule_time ASC LIMIT 5
	sc, _ := d.GetConnect(queueTable)
	sc.Table(queueTable.TableName).Debug().
		Where("state = ? and queue_name = ? and server_id = ?", TaskStatusPending, queueName, serverId).
		Where("schedule_time <= ?", time.Now()).
		Order("schedule_time ASC").
		Limit(limit).Find(&tasks)
	return
}

func (d *Database) SetTaskStart(queueTable *QueueTable, id int64) (err error) {
	c, _ := d.GetConnect(queueTable)
	err = c.Table(queueTable.TableName).Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":      TaskStatusRunning,
			"start_time": time.Now(),
		}).Error

	return
}

func (d *Database) SetTaskFinish(queueTable *QueueTable, id int64) (err error) {
	c, _ := d.GetConnect(queueTable)
	err = c.Table(queueTable.TableName).Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":    TaskStatusFinish,
			"end_time": time.Now(),
		}).Error

	return
}

func (d *Database) SetTaskFail(queueTable *QueueTable, id int64, failMsg datatypes.JSON) (err error) {
	c, _ := d.GetConnect(queueTable)
	err = c.Table(queueTable.TableName).Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":       TaskStatusFail,
			"end_time":    time.Now(),
			"exec_result": failMsg,
		}).Error

	return
}

func (d *Database) SetTaskCancel(queueTable *QueueTable, id int64) (err error) {
	c, _ := d.GetConnect(queueTable)
	err = c.Table(queueTable.TableName).Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":    TaskStatusCancel,
			"end_time": time.Now(),
		}).Error

	return
}

func (d *Database) CreateTasks(queueTable *QueueTable, tasks []*TaskData) (err error) {
	c, err := d.GetConnect(queueTable)
	if err != nil {
		return
	}
	return c.Table(queueTable.TableName).Create(tasks).Error
}
