package database

import (
	"strings"
	"time"
)

var (
	QueueConfigTableName = "queue_configs"
)

// QueueConfig 队列配置表
type QueueConfig struct {
	ID                int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	QueueName         string    `json:"queue_name" gorm:"size:100;not null;comment:队列名称"`
	MaxWorkers        int       `json:"max_workers" gorm:"default:1;comment:最大并发协程数"`
	FrequencyInterval int       `json:"frequency_interval" gorm:"default:3;comment:扫表间隔(s)"`
	Priority          int       `json:"priority" gorm:"default:1;comment:队列优先级"`
	TaskTable         string    `json:"task_table" gorm:"size:200;comment:任务表 格式:连接名.表名"`
	ServerID          string    `json:"server_id" gorm:"size:50;comment:指定运行的服务器ID"`
	Status            int       `json:"status" gorm:"default:1;comment:状态 1:启用 0:禁用"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	IsDelete          int8      `json:"is_delete" gorm:"default:0;comment:是否删除"`
}

// GetQueueConfigs 获取队列配置
func (d *Database) GetQueueConfigs(serverID string) ([]QueueConfig, error) {
	var configs []QueueConfig
	result := d.db.Table(QueueConfigTableName).
		Where("status = ? AND (server_id = ? OR server_id = 'all')", QueueStatusEnabled, serverID).
		Where("is_delete = ?", 0).
		Find(&configs)
	return configs, result.Error
}

// RemoveDisabledQueue 软删除禁用的队列
func (d *Database) RemoveDisabledQueue(ids []int64) {
	d.db.Table(QueueConfigTableName).Where("id in ?", ids).UpdateColumn("is_delete", 1)
}

// GetQueueConfigByName 根据队列名获取配置
func (d *Database) GetQueueConfigByName(queueName string) (*QueueConfig, error) {
	var config QueueConfig
	result := d.db.Table(QueueConfigTableName).
		Where("queue_name = ? AND status = ?", queueName, QueueStatusEnabled).First(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	return &config, nil
}

// ParseTableName 解析表名
func (d *Database) ParseTableName(taskTable string) *QueueTable {
	n := strings.SplitN(taskTable, ".", 2)
	if len(n) == 1 {
		return &QueueTable{
			ConnectName: DbConnectDefault,
			TableName:   n[0],
		}
	}

	return &QueueTable{
		ConnectName: n[0],
		TableName:   n[1],
	}
}
