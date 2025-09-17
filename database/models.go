package database

import (
	"time"
)

// QueueConfig 队列配置表
type QueueConfig struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	QueueName    string    `json:"queue_name" gorm:"uniqueIndex;size:100;not null;comment:队列名称"`
	TableName    string    `json:"table_name" gorm:"size:100;not null;comment:关联的任务表名"`
	ServerID     string    `json:"server_id" gorm:"size:50;comment:指定运行的服务器ID"`
	MaxWorkers   int       `json:"max_workers" gorm:"default:1;comment:最大并发协程数"`
	ScanInterval int       `json:"scan_interval" gorm:"default:5;comment:扫描间隔(秒)"`
	Status       int       `json:"status" gorm:"default:1;comment:状态 1:启用 0:禁用"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TaskBase 任务基础结构 - 所有任务表都应该包含这些字段
type TaskBase struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskType     string     `json:"task_type" gorm:"size:50;not null;comment:任务类型"`
	TaskParams   string     `json:"task_params" gorm:"type:text;comment:任务参数(JSON格式)"`
	Status       int        `json:"status" gorm:"default:0;index;comment:状态 0:待执行 1:执行中 2:已完成 3:失败 4:取消"`
	ScheduleTime time.Time  `json:"schedule_time" gorm:"index;comment:计划执行时间"`
	StartTime    *time.Time `json:"start_time" gorm:"comment:开始执行时间"`
	EndTime      *time.Time `json:"end_time" gorm:"comment:结束执行时间"`
	RetryCount   int        `json:"retry_count" gorm:"default:0;comment:重试次数"`
	MaxRetries   int        `json:"max_retries" gorm:"default:3;comment:最大重试次数"`
	ErrorMsg     string     `json:"error_msg" gorm:"type:text;comment:错误信息"`
	ServerID     string     `json:"server_id" gorm:"size:50;comment:执行的服务器ID"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// ServerInfo 服务器信息
type ServerInfo struct {
	ID            string    `json:"id" gorm:"primaryKey;size:50;comment:服务器唯一标识"`
	ServerName    string    `json:"server_name" gorm:"size:100;comment:服务器名称"`
	IP            string    `json:"ip" gorm:"size:15;comment:服务器IP"`
	Port          int       `json:"port" gorm:"comment:服务器端口"`
	Status        int       `json:"status" gorm:"default:1;comment:状态 1:在线 0:离线"`
	LastHeartbeat time.Time `json:"last_heartbeat" gorm:"comment:最后心跳时间"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TaskStatus 任务状态常量
const (
	TaskStatusPending   = 0 // 待执行
	TaskStatusRunning   = 1 // 执行中
	TaskStatusCompleted = 2 // 已完成
	TaskStatusFailed    = 3 // 失败
	TaskStatusCancelled = 4 // 取消
)

// QueueStatus 队列状态常量
const (
	QueueStatusDisabled = 0 // 禁用
	QueueStatusEnabled  = 1 // 启用
)

// ServerStatus 服务器状态常量
const (
	ServerStatusOffline = 0 // 离线
	ServerStatusOnline  = 1 // 在线
)
