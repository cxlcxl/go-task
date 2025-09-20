package database

import (
	"fmt"
	"task-executor/logger"
	"time"
)

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

func (d *Database) SetTaskFail(queueTable *QueueTable, id int64) (err error) {
	c, _ := d.GetConnect(queueTable)
	err = c.Table(queueTable.TableName).Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":    TaskStatusFail,
			"end_time": time.Now(),
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
