package database

import (
	"context"
	"fmt"
	"task-executor/config"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db      *gorm.DB
	pgxPool *pgxpool.Pool // 用于 River
	ctx     context.Context
	dbs     map[string]*gorm.DB
}

func NewDatabase(configs map[string]config.PgSQL) (*Database, error) {
	_db := &Database{
		ctx: context.Background(),
		dbs: make(map[string]*gorm.DB),
	}
	for connectName, cfg := range configs {
		// 构建 PostgreSQL DSN
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Host, cfg.Username, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode,
		)

		// 创建 GORM 连接
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, fmt.Errorf("连接PostgreSQL数据库失败: %w", err)
		}

		// 设置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}

		// 设置连接池参数
		sqlDB.SetMaxIdleConns(cfg.MinConns)
		sqlDB.SetMaxOpenConns(cfg.MaxConns)
		sqlDB.SetConnMaxLifetime(time.Hour)

		if connectName == DbConnectDefault {
			_db.db = db
		}
		_db.dbs[connectName] = db
	}

	// 创建 pgx 连接池（为 River 使用）
	//pgxDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
	//	config.Username, config.Password, config.Host, config.Port, config.Database, config.SSLMode)
	//
	//pgxConfig, err := pgxpool.ParseConfig(pgxDSN)
	//if err != nil {
	//	return nil, fmt.Errorf("解析pgx配置失败: %w", err)
	//}
	//
	//// 设置 pgx 连接池参数
	//pgxConfig.MaxConns = int32(maxConns)
	//pgxConfig.MinConns = int32(minConns)
	//pgxConfig.MaxConnLifetime = time.Hour
	//pgxConfig.MaxConnIdleTime = 30 * time.Minute
	//
	//pgxPool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	//if err != nil {
	//	return nil, fmt.Errorf("创建pgx连接池失败: %w", err)
	//}
	//
	//// 测试连接
	//if err := pgxPool.Ping(ctx); err != nil {
	//	return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	//}

	return _db, nil
}

func (d *Database) GetDB() *gorm.DB {
	return d.db
}

func (d *Database) GetConnect(queueTable *QueueTable) (*gorm.DB, error) {
	if db, exists := d.dbs[queueTable.ConnectName]; exists {
		return db.Table(queueTable.TableName), nil
	} else {
		return nil, fmt.Errorf("数据库连接不存在: %s", queueTable.ConnectName)
	}
}

// GetPGXPool 获取 pgx 连接池（为 River 使用）
func (d *Database) GetPGXPool() *pgxpool.Pool {
	return d.pgxPool
}

// GetContext 获取上下文
func (d *Database) GetContext() context.Context {
	return d.ctx
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.pgxPool != nil {
		d.pgxPool.Close()
	}
	return nil
}

// AutoMigrate 自动迁移表结构
func (d *Database) AutoMigrate() error {
	return d.db.AutoMigrate(&QueueConfig{}, &ServerInfo{}, &TaskHandlerRegistry{}, &JobMetadata{}, &JobStatistics{})
}

// UpdateServerHeartbeat 更新服务器心跳
func (d *Database) UpdateServerHeartbeat(serverID string, serverName, ip string, port int) error {
	now := time.Now()
	return d.db.Model(&ServerInfo{}).Where("id = ?", serverID).Updates(map[string]interface{}{
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

	return d.db.Save(&server).Error
}

// GetTaskHandlers 获取所有启用的任务处理器
func (d *Database) GetTaskHandlers() ([]TaskHandlerRegistry, error) {
	var handlers []TaskHandlerRegistry
	result := d.db.Where("status = ?", HandlerStatusEnabled).Find(&handlers)
	return handlers, result.Error
}

// GetTaskHandlerByType 根据任务类型获取处理器
func (d *Database) GetTaskHandlerByType(taskType string) (*TaskHandlerRegistry, error) {
	var handler TaskHandlerRegistry
	result := d.db.Where("task_type = ? AND status = ?", taskType, HandlerStatusEnabled).First(&handler)
	if result.Error != nil {
		return nil, result.Error
	}
	return &handler, nil
}

// RegisterTaskHandler 注册任务处理器
func (d *Database) RegisterTaskHandler(handler *TaskHandlerRegistry) error {
	// 检查是否已存在相同类型的处理器
	var existing TaskHandlerRegistry
	result := d.db.Where("task_type = ?", handler.TaskType).First(&existing)

	if result.Error == nil {
		// 已存在，更新
		handler.ID = existing.ID
		handler.UpdatedAt = time.Now()
		return d.db.Save(handler).Error
	} else {
		// 不存在，创建新的
		handler.CreatedAt = time.Now()
		handler.UpdatedAt = time.Now()
		handler.LastHeartbeat = time.Now()
		return d.db.Create(handler).Error
	}
}

// UpdateHandlerHeartbeat 更新处理器心跳
func (d *Database) UpdateHandlerHeartbeat(taskType string) error {
	now := time.Now()
	return d.db.Model(&TaskHandlerRegistry{}).Where("task_type = ?", taskType).Update("last_heartbeat", now).Error
}

// DeleteTaskHandler 删除任务处理器
func (d *Database) DeleteTaskHandler(taskType string) error {
	return d.db.Where("task_type = ?", taskType).Delete(&TaskHandlerRegistry{}).Error
}

// DisableTaskHandler 禁用任务处理器
func (d *Database) DisableTaskHandler(taskType string) error {
	return d.db.Model(&TaskHandlerRegistry{}).Where("task_type = ?", taskType).Update("status", HandlerStatusDisabled).Error
}

// CreateJobMetadata 创建任务元数据
func (d *Database) CreateJobMetadata(metadata *JobMetadata) error {
	metadata.CreatedAt = time.Now()
	metadata.UpdatedAt = time.Now()
	return d.db.Create(metadata).Error
}

// GetJobMetadata 获取任务元数据
func (d *Database) GetJobMetadata(riverJobID int64) (*JobMetadata, error) {
	var metadata JobMetadata
	result := d.db.Where("river_job_id = ?", riverJobID).First(&metadata)
	if result.Error != nil {
		return nil, result.Error
	}
	return &metadata, nil
}

// UpdateJobStatistics 更新任务统计
func (d *Database) UpdateJobStatistics(date time.Time, queueName, taskType string, total, success, failed int, avgDuration int) error {
	stats := JobStatistics{
		Date:          date,
		QueueName:     queueName,
		TaskType:      taskType,
		TotalCount:    total,
		SuccessCount:  success,
		FailedCount:   failed,
		AvgDurationMs: avgDuration,
		UpdatedAt:     time.Now(),
	}

	// 使用 ON CONFLICT 更新或插入
	return d.db.Save(&stats).Error
}

// GetJobStatistics 获取任务统计
func (d *Database) GetJobStatistics(startDate, endDate time.Time, queueName, taskType string) ([]JobStatistics, error) {
	var stats []JobStatistics
	query := d.db.Where("date >= ? AND date <= ?", startDate, endDate)

	if queueName != "" {
		query = query.Where("queue_name = ?", queueName)
	}
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}

	result := query.Order("date DESC").Find(&stats)
	return stats, result.Error
}
