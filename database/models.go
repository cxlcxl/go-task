package database

import (
	"gorm.io/datatypes"
	"time"
)

const (
	DbConnectDefault = "default"
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

// JobMetadata 任务元数据表
type JobMetadata struct {
	ID           int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	RiverJobID   *int64                 `json:"river_job_id" gorm:"comment:关联到 River 的 river_job.id"`
	BusinessID   string                 `json:"business_id" gorm:"size:100;comment:业务ID"`
	JobGroup     string                 `json:"job_group" gorm:"size:50;comment:任务分组"`
	SourceSystem string                 `json:"source_system" gorm:"size:50;comment:来源系统"`
	Metadata     map[string]interface{} `json:"metadata" gorm:"type:jsonb;comment:额外的元数据"`
	CreatedAt    time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// JobStatistics 任务执行统计表
type JobStatistics struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Date          time.Time `json:"date" gorm:"type:date;not null;comment:统计日期"`
	QueueName     string    `json:"queue_name" gorm:"size:100;not null;comment:队列名称"`
	TaskType      string    `json:"task_type" gorm:"size:100;not null;comment:任务类型"`
	TotalCount    int       `json:"total_count" gorm:"default:0;comment:总数"`
	SuccessCount  int       `json:"success_count" gorm:"default:0;comment:成功数"`
	FailedCount   int       `json:"failed_count" gorm:"default:0;comment:失败数"`
	AvgDurationMs int       `json:"avg_duration_ms" gorm:"default:0;comment:平均执行时间(毫秒)"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (JobStatistics) TableName() string {
	return "job_statistics"
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

// TaskStatus 任务状态常量 (为了兼容性保留)
const (
	TaskStatusInit    = "init"    // 待执行
	TaskStatusPending = "pending" // 已锁定
	TaskStatusRunning = "running" // 执行中
	TaskStatusFinish  = "finish"  // 已完成
	TaskStatusFail    = "fail"    // 失败
	TaskStatusCancel  = "cancel"  // 取消
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

// TaskHandlerRegistry 任务处理器注册表
type TaskHandlerRegistry struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskType      string    `json:"task_type" gorm:"size:100;not null;comment:任务类型"`
	HandlerName   string    `json:"handler_name" gorm:"size:100;not null;comment:处理器名称"`
	EndpointURL   string    `json:"endpoint_url" gorm:"size:500;comment:处理器端点URL"`
	HandlerType   string    `json:"handler_type" gorm:"size:20;default:'http';comment:处理器类型 http/grpc/local"`
	Version       string    `json:"version" gorm:"size:20;comment:处理器版本"`
	Status        int       `json:"status" gorm:"default:1;comment:状态 1:启用 0:禁用"`
	Timeout       int       `json:"timeout" gorm:"default:30;comment:超时时间(秒)"`
	RetryCount    int       `json:"retry_count" gorm:"default:3;comment:重试次数"`
	Description   string    `json:"description" gorm:"size:500;comment:处理器描述"`
	ConfigData    string    `json:"config_data" gorm:"type:text;comment:配置数据(JSON格式)"`
	LastHeartbeat time.Time `json:"last_heartbeat" gorm:"comment:最后心跳时间"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// HandlerStatus 处理器状态常量
const (
	HandlerStatusDisabled = 0 // 禁用
	HandlerStatusEnabled  = 1 // 启用
)

// HandlerType 处理器类型常量
const (
	HandlerTypeHTTP  = "http"  // HTTP接口
	HandlerTypeGRPC  = "grpc"  // gRPC接口
	HandlerTypeLocal = "local" // 本地处理器
)

type QueueTable struct {
	ConnectName string
	TableName   string
}

type TaskData struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement:true"`
	BatchNo      string         `gorm:"column:batch_no;type:varchar(80);not null"`
	QueueName    string         `gorm:"column:queue_name;type:varchar(80);not null"`
	TaskParams   datatypes.JSON `gorm:"column:task_params;type:jsonb;not null;default:'{}'"`
	State        string         `gorm:"column:state;type:varchar(20);not null"`
	ScheduleTime time.Time      `gorm:"column:schedule_time;type:timestamp;not null;default:'2000-01-01 00:00:00'"`
	StartTime    time.Time      `gorm:"column:start_time;type:timestamp;not null;default:'2000-01-01 00:00:00'"`
	EndTime      time.Time      `gorm:"column:end_time;type:timestamp;not null;default:'2000-01-01 00:00:00'"`
	CostTime     int32          `gorm:"column:cost_time;type:integer;not null;default:0"`
	RetryCount   int32          `gorm:"column:retry_count;type:integer;not null;default:0"`
	MaxRetries   int32          `gorm:"column:max_retries;type:integer;not null;default:0"`
	ExecResult   datatypes.JSON `gorm:"column:exec_result;type:jsonb;not null;default:'{}'"`
	ServerID     string         `gorm:"column:server_id;type:varchar(50);not null;default:''"`
	TaskType     string         `gorm:"column:task_type;type:varchar(30);not null;default:'default'"`
	CreatedAt    time.Time      `gorm:"column:created_at;type:timestamp;default:now()"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;type:timestamp;default:current_timestamp"`
}
