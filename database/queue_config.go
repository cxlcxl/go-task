package database

import "strings"

// GetQueueConfigs 获取队列配置
func (d *Database) GetQueueConfigs(serverID string) ([]QueueConfig, error) {
	var configs []QueueConfig
	result := d.db.
		Where("status = ? AND (server_id = ? OR server_id = 'all')", QueueStatusEnabled, serverID).
		Find(&configs)
	return configs, result.Error
}

// GetQueueConfigByName 根据队列名获取配置
func (d *Database) GetQueueConfigByName(queueName string) (*QueueConfig, error) {
	var config QueueConfig
	result := d.db.Where("queue_name = ? AND status = ?", queueName, QueueStatusEnabled).First(&config)
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
