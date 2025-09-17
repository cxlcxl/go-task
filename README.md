## 功能概述

本系统实现了基于数据库的分布式任务队列，支持以下核心功能：

1. **自定义队列并发控制** - 为每个队列配置独立的协程并发数
2. **数据库驱动的任务调度** - 基于数据库表扫描的常驻任务调度系统
3. **服务器隔离** - 支持指定队列只在特定服务器上运行
4. **动态扩展** - 支持动态调整队列与服务器的关联关系

## 快速开始

### 1. 数据库初始化

```bash
# 导入数据库结构和示例数据
mysql -u root -p < init.sql
```

### 2. 配置修改

编辑 `config.json` 文件：

```json
{
  "database": {
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "your_password",
    "database": "xxljob_executor",
    "charset": "utf8mb4"
  },
  "executor": {
    "server_id": "server-001"
  },
  "queue": {
    "enable": true,
    "config_sync_interval": 10
  }
}
```

### 3. 启动服务

```bash
# 安装依赖
go mod tidy

# 启动服务
go run main.go
```

## 核心概念

### 队列配置表 (queue_configs)

| 字段 | 类型 | 说明 |
|------|------|------|
| queue_name | varchar(100) | 队列名称，唯一标识 |
| table_name | varchar(100) | 关联的任务表名 |
| server_id | varchar(50) | 指定运行的服务器ID，空表示任意服务器 |
| max_workers | int | 最大并发协程数 |
| scan_interval | int | 扫描间隔(秒) |
| status | tinyint | 状态 1:启用 0:禁用 |

### 任务表结构

所有任务表都必须包含以下基础字段：

```sql
CREATE TABLE `your_task_table` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `task_type` varchar(50) NOT NULL COMMENT '任务类型',
  `task_params` text COMMENT '任务参数(JSON格式)',
  `status` tinyint(1) DEFAULT 0 COMMENT '0:待执行 1:执行中 2:已完成 3:失败 4:取消',
  `schedule_time` datetime NOT NULL COMMENT '计划执行时间',
  `start_time` datetime DEFAULT NULL,
  `end_time` datetime DEFAULT NULL,
  `retry_count` int(11) DEFAULT 0,
  `max_retries` int(11) DEFAULT 3,
  `error_msg` text,
  `server_id` varchar(50) DEFAULT '',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_status_schedule` (`status`, `schedule_time`)
);
```

## 管理接口

### 队列管理

#### 创建队列配置
```bash
curl -X POST http://localhost:9999/admin/queue/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue_name": "new_queue",
    "table_name": "new_tasks",
    "server_id": "server-001",
    "max_workers": 5,
    "scan_interval": 3,
    "status": 1
  }'
```

#### 获取队列列表
```bash
curl http://localhost:9999/admin/queue/list
```

#### 更新队列配置
```bash
curl -X PUT "http://localhost:9999/admin/queue/update?id=1" \
  -H "Content-Type: application/json" \
  -d '{
    "max_workers": 10,
    "server_id": "server-002"
  }'
```

#### 删除队列配置
```bash
curl -X DELETE "http://localhost:9999/admin/queue/delete?id=1"
```

### 服务器管理

#### 查看服务器状态
```bash
curl http://localhost:9999/admin/server/list
```

### 任务管理

#### 创建任务
```bash
# 创建邮件任务
curl -X POST "http://localhost:9999/admin/task/create?table=email_tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "email",
    "task_params": {
      "to": ["user@example.com"],
      "subject": "测试邮件",
      "content": "这是一封测试邮件",
      "type": "html"
    },
    "schedule_time": "2024-01-01 10:00:00",
    "max_retries": 3
  }'

# 创建数据同步任务
curl -X POST "http://localhost:9999/admin/task/create?table=data_sync_tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "data_sync",
    "task_params": {
      "source_db": "db1",
      "target_db": "db2",
      "table_name": "users",
      "sync_type": "incremental",
      "batch_size": 1000
    },
    "schedule_time": "2024-01-01 02:00:00",
    "max_retries": 2
  }'
```

## 使用场景示例

### 场景1：多服务器队列隔离

假设您有3台服务器，希望不同类型的任务在不同服务器上执行：

```sql
-- 服务器1: 专门处理邮件任务
INSERT INTO queue_configs (queue_name, table_name, server_id, max_workers, scan_interval)
VALUES ('email_queue', 'email_tasks', 'server-001', 10, 2);

-- 服务器2: 专门处理数据同步任务
INSERT INTO queue_configs (queue_name, table_name, server_id, max_workers, scan_interval)
VALUES ('sync_queue', 'data_sync_tasks', 'server-002', 5, 5);

-- 服务器3: 处理报表生成任务
INSERT INTO queue_configs (queue_name, table_name, server_id, max_workers, scan_interval)
VALUES ('report_queue', 'report_tasks', 'server-003', 3, 10);
```

### 场景2：动态调整负载

当某台服务器负载过高时，可以动态调整队列分配：

```sql
-- 将邮件队列从server-001转移到server-002
UPDATE queue_configs SET server_id = 'server-002' WHERE queue_name = 'email_queue';

-- 或者允许任意服务器处理
UPDATE queue_configs SET server_id = '' WHERE queue_name = 'email_queue';
```

### 场景3：控制并发数

根据任务类型和服务器性能调整并发数：

```sql
-- 高并发邮件发送
UPDATE queue_configs SET max_workers = 20 WHERE queue_name = 'email_queue';

-- 低并发数据同步（避免数据库压力）
UPDATE queue_configs SET max_workers = 2 WHERE queue_name = 'sync_queue';
```

## 自定义任务处理器

### 1. 实现JobHandler接口

```go
type CustomTaskHandler struct{}

func (h *CustomTaskHandler) Execute(ctx *models.JobContext) error {
    // 解析任务参数
    var params YourParamsStruct
    if err := json.Unmarshal([]byte(ctx.JobParam), &params); err != nil {
        return fmt.Errorf("解析参数失败: %w", err)
    }

    // 执行具体业务逻辑
    // ...

    return nil
}
```

### 2. 注册处理器

在 `main.go` 的 `registerQueueTaskHandlers` 函数中添加：

```go
queueManager.RegisterTaskHandler("your_task_type", &CustomTaskHandler{})
```

## 监控和日志

### 日志位置
- 主日志：`./logs/executor-YYYY-MM-DD.log`
- 任务日志：`./logs/job-{logID}-YYYY-MM-DD.log`

### 监控指标
- 队列状态：通过管理接口查看
- 服务器状态：心跳监控
- 任务执行状态：数据库记录

## 注意事项

1. **数据库连接** - 确保数据库连接配置正确
2. **表结构** - 所有任务表必须包含基础字段
3. **服务器ID** - 每个服务器实例必须有唯一的server_id
4. **并发控制** - 根据服务器性能合理设置max_workers
5. **扫描间隔** - 平衡实时性和数据库压力
6. **错误处理** - 合理设置重试次数和错误处理逻辑

## 故障排查

### 常见问题

1. **队列不工作** - 检查queue_configs表中的status字段和server_id匹配
2. **任务不执行** - 确认schedule_time和当前时间的关系
3. **数据库连接失败** - 检查配置文件中的数据库连接信息
4. **任务处理器未找到** - 确认task_type与注册的处理器名称匹配