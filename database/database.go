package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	connections map[string]*gorm.DB
	defaultDB   *gorm.DB
}

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Charset  string `json:"charset"`
}

func NewDatabase(configs map[string]Config) (*Database, error) {
	connections := make(map[string]*gorm.DB)
	var defaultDB *gorm.DB

	for name, config := range configs {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			config.Username, config.Password, config.Host, config.Port, config.Database, config.Charset)

		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, fmt.Errorf("连接数据库 %s 失败: %w", name, err)
		}

		// 设置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)

		connections[name] = db

		// 设置默认数据库（使用第一个或者名为"default"的）
		if name == "default" || defaultDB == nil {
			defaultDB = db
		}
	}

	if defaultDB == nil {
		return nil, fmt.Errorf("没有找到默认数据库连接")
	}

	return &Database{
		connections: connections,
		defaultDB:   defaultDB,
	}, nil
}

func (d *Database) GetDB() *gorm.DB {
	return d.defaultDB
}

// GetConnection 获取指定名称的数据库连接
func (d *Database) GetConnection(name string) (*gorm.DB, error) {
	if name == "" {
		return d.defaultDB, nil
	}

	db, exists := d.connections[name]
	if !exists {
		return nil, fmt.Errorf("数据库连接 %s 不存在", name)
	}
	return db, nil
}

// ParseTableName 解析表名，返回数据库名和表名
func (d *Database) ParseTableName(tableName string) (string, string, error) {
	parts := strings.SplitN(tableName, ".", 2)
	if len(parts) == 1 {
		// 没有指定数据库，使用默认数据库
		return "", parts[0], nil
	}

	dbName := parts[0]
	realTableName := parts[1]

	// 检查数据库连接是否存在
	if _, exists := d.connections[dbName]; !exists {
		return "", "", fmt.Errorf("数据库连接 %s 不存在", dbName)
	}

	return dbName, realTableName, nil
}

// AutoMigrate 自动迁移表结构（只在默认数据库中创建系统表）
func (d *Database) AutoMigrate() error {
	return d.defaultDB.AutoMigrate(&QueueConfig{}, &ServerInfo{}, &TaskHandlerRegistry{})
}

// GetQueueConfigs 获取队列配置
func (d *Database) GetQueueConfigs(serverID string) ([]QueueConfig, error) {
	var configs []QueueConfig
	result := d.defaultDB.Where("status = ? AND (server_id = ? OR server_id = '')",
		QueueStatusEnabled, serverID).Find(&configs)
	return configs, result.Error
}

// GetQueueConfigByName 根据队列名获取配置
func (d *Database) GetQueueConfigByName(queueName string) (*QueueConfig, error) {
	var config QueueConfig
	result := d.defaultDB.Where("queue_name = ? AND status = ?", queueName, QueueStatusEnabled).First(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	return &config, nil
}

// UpdateServerHeartbeat 更新服务器心跳
func (d *Database) UpdateServerHeartbeat(serverID string, serverName, ip string, port int) error {
	now := time.Now()
	return d.defaultDB.Model(&ServerInfo{}).Where("id = ?", serverID).Updates(map[string]interface{}{
		"server_name":    serverName,
		"ip":             ip,
		"port":           port,
		"status":         ServerStatusOnline,
		"last_heartbeat": now,
		"updated_at":     now,
	}).Error
}

// CreateOrUpdateServer 创建或更新服务器信息
func (d *Database) CreateOrUpdateServer(serverID, serverName, ip string, port int) error {
	server := ServerInfo{
		ID:            serverID,
		ServerName:    serverName,
		IP:            ip,
		Port:          port,
		Status:        ServerStatusOnline,
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}

	return d.defaultDB.Save(&server).Error
}

// RecoverOrphanedTasks 恢复孤立的任务（启动时调用）
func (d *Database) RecoverOrphanedTasks(serverID string) error {
	// 获取所有队列配置
	configs, err := d.GetQueueConfigs(serverID)
	if err != nil {
		return fmt.Errorf("获取队列配置失败: %w", err)
	}

	for _, config := range configs {
		// 解析表名
		dbName, realTableName, err := d.ParseTableName(config.TableName)
		if err != nil {
			continue // 跳过无效的表名
		}

		// 获取对应的数据库连接
		db, err := d.GetConnection(dbName)
		if err != nil {
			continue // 跳过无效的数据库连接
		}

		// 恢复执行中的任务为待执行状态
		updateQuery := fmt.Sprintf(`
			UPDATE %s
			SET status = ?, server_id = '', error_msg = ?, updated_at = ?
			WHERE status = ? AND (server_id = ? OR server_id = '')
		`, realTableName)

		errorMsg := "任务因服务重启而重置"
		result := db.Exec(updateQuery,
			TaskStatusPending, errorMsg, time.Now(),
			TaskStatusRunning, serverID)

		if result.Error != nil {
			return fmt.Errorf("恢复队列 %s 的孤立任务失败: %w", config.QueueName, result.Error)
		}

		if result.RowsAffected > 0 {
			fmt.Printf("队列 %s 恢复了 %d 个孤立任务\n", config.QueueName, result.RowsAffected)
		}
	}

	return nil
}

// RecoverTimeoutTasks 恢复超时的任务
func (d *Database) RecoverTimeoutTasks(timeoutMinutes int) error {
	// 获取所有队列配置
	var configs []QueueConfig
	result := d.defaultDB.Where("status = ?", QueueStatusEnabled).Find(&configs)
	if result.Error != nil {
		return result.Error
	}

	for _, config := range configs {
		// 解析表名
		dbName, realTableName, err := d.ParseTableName(config.TableName)
		if err != nil {
			continue
		}

		// 获取对应的数据库连接
		db, err := d.GetConnection(dbName)
		if err != nil {
			continue
		}

		// 恢复执行时间超过指定分钟数的任务
		updateQuery := fmt.Sprintf(`
			UPDATE %s
			SET status = ?, error_msg = ?, updated_at = ?
			WHERE status = ? AND start_time < ?
		`, realTableName)

		timeoutTime := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)
		errorMsg := fmt.Sprintf("任务执行超时（超过%d分钟）", timeoutMinutes)

		result := db.Exec(updateQuery,
			TaskStatusFailed, errorMsg, time.Now(),
			TaskStatusRunning, timeoutTime)

		if result.Error != nil {
			return fmt.Errorf("恢复队列 %s 的超时任务失败: %w", config.QueueName, result.Error)
		}

		if result.RowsAffected > 0 {
			fmt.Printf("队列 %s 恢复了 %d 个超时任务\n", config.QueueName, result.RowsAffected)
		}
	}

	return nil
}

// GetPendingTasks 获取待执行的任务
func (d *Database) GetPendingTasks(tableName string, limit int) ([]map[string]interface{}, error) {
	var tasks []map[string]interface{}

	// 解析表名
	dbName, realTableName, err := d.ParseTableName(tableName)
	if err != nil {
		return nil, err
	}

	// 获取对应的数据库连接
	db, err := d.GetConnection(dbName)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = ? AND schedule_time <= ?
		ORDER BY schedule_time ASC
		LIMIT ?
	`, realTableName)

	result := db.Raw(query, TaskStatusPending, time.Now(), limit).Scan(&tasks)
	return tasks, result.Error
}

// UpdateTaskStatus 更新任务状态
func (d *Database) UpdateTaskStatus(tableName string, taskID int64, status int, serverID string, errorMsg string) error {
	// 解析表名
	dbName, realTableName, err := d.ParseTableName(tableName)
	if err != nil {
		return err
	}

	// 获取对应的数据库连接
	db, err := d.GetConnection(dbName)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status":     status,
		"server_id":  serverID,
		"updated_at": time.Now(),
	}

	if status == TaskStatusRunning {
		updates["start_time"] = time.Now()
	} else if status == TaskStatusCompleted || status == TaskStatusFailed {
		updates["end_time"] = time.Now()
	}

	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	query := fmt.Sprintf("UPDATE %s SET ", realTableName)
	var setParts []string
	var values []interface{}

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = ?", key))
		values = append(values, value)
	}

	query += strings.Join(setParts, ", ") + " WHERE id = ?"
	values = append(values, taskID)

	return db.Exec(query, values...).Error
}

// IncrementRetryCount 增加重试次数
func (d *Database) IncrementRetryCount(tableName string, taskID int64) error {
	// 解析表名
	dbName, realTableName, err := d.ParseTableName(tableName)
	if err != nil {
		return err
	}

	// 获取对应的数据库连接
	db, err := d.GetConnection(dbName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE %s SET retry_count = retry_count + 1, updated_at = ? WHERE id = ?", realTableName)
	return db.Exec(query, time.Now(), taskID).Error
}

// GetTaskHandlers 获取所有启用的任务处理器
func (d *Database) GetTaskHandlers() ([]TaskHandlerRegistry, error) {
	var handlers []TaskHandlerRegistry
	result := d.defaultDB.Where("status = ?", HandlerStatusEnabled).Find(&handlers)
	return handlers, result.Error
}

// GetTaskHandlerByType 根据任务类型获取处理器
func (d *Database) GetTaskHandlerByType(taskType string) (*TaskHandlerRegistry, error) {
	var handler TaskHandlerRegistry
	result := d.defaultDB.Where("task_type = ? AND status = ?", taskType, HandlerStatusEnabled).First(&handler)
	if result.Error != nil {
		return nil, result.Error
	}
	return &handler, nil
}

// RegisterTaskHandler 注册任务处理器
func (d *Database) RegisterTaskHandler(handler *TaskHandlerRegistry) error {
	// 检查是否已存在相同类型的处理器
	var existing TaskHandlerRegistry
	result := d.defaultDB.Where("task_type = ?", handler.TaskType).First(&existing)

	if result.Error == nil {
		// 已存在，更新
		handler.ID = existing.ID
		handler.UpdatedAt = time.Now()
		return d.defaultDB.Save(handler).Error
	} else {
		// 不存在，创建新的
		handler.CreatedAt = time.Now()
		handler.UpdatedAt = time.Now()
		handler.LastHeartbeat = time.Now()
		return d.defaultDB.Create(handler).Error
	}
}

// UpdateHandlerHeartbeat 更新处理器心跳
func (d *Database) UpdateHandlerHeartbeat(taskType string) error {
	now := time.Now()
	return d.defaultDB.Model(&TaskHandlerRegistry{}).Where("task_type = ?", taskType).Update("last_heartbeat", now).Error
}

// DeleteTaskHandler 删除任务处理器
func (d *Database) DeleteTaskHandler(taskType string) error {
	return d.defaultDB.Where("task_type = ?", taskType).Delete(&TaskHandlerRegistry{}).Error
}

// DisableTaskHandler 禁用任务处理器
func (d *Database) DisableTaskHandler(taskType string) error {
	return d.defaultDB.Model(&TaskHandlerRegistry{}).Where("task_type = ?", taskType).Update("status", HandlerStatusDisabled).Error
}
