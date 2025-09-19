# River 队列系统部署文档

## 简介

本系统已经从 MySQL+Asynq 架构迁移到 PostgreSQL+River 架构，提供更强大的队列处理能力和更好的可靠性。

## 主要特性

- **基于PostgreSQL**: 利用PostgreSQL的ACID特性确保任务的可靠性
- **高性能**: River专为高吞吐量设计，支持并发处理
- **任务重试**: 自动重试失败的任务，支持指数退避策略
- **任务监控**: 提供完整的Web管理界面
- **动态配置**: 支持运行时动态调整队列配置
- **任务统计**: 详细的任务执行统计和报告

## 快速开始

### 1. 数据库初始化

首先创建PostgreSQL数据库并执行初始化脚本：

```bash
# 创建数据库
createdb river_queue

# 执行初始化脚本
psql -d river_queue -f init-river-postgresql.sql

# 运行River迁移（创建River核心表）
go run cmd/river/migrate.go up --database-url "postgres://user:password@localhost:5432/river_queue"
```

### 2. 配置文件

复制配置示例文件并修改：

```bash
cp config-river.example.yaml config.yaml
```

修改 `config.yaml` 中的数据库连接信息和其他配置。

### 3. 启动服务

```bash
# 编译并启动
go build -o river-executor
./river-executor
```

## 配置说明

### PostgreSQL配置

```yaml
postgresql:
  host: "localhost"          # 数据库主机
  port: 5432                # 数据库端口
  username: "postgres"       # 用户名
  password: "your_password"  # 密码
  database: "river_queue"    # 数据库名
  ssl_mode: "disable"        # SSL模式
  max_conns: 100            # 最大连接数
  min_conns: 10             # 最小连接数
```

### River队列配置

```yaml
river:
  enable: true                          # 启用River队列
  workers: 50                           # 默认工作协程数
  poll_only: false                      # 仅轮询模式
  insert_only: false                    # 仅插入模式
  rescue_stuck_jobs_after: "1h"         # 救援卡住的任务
  max_attempts: 3                       # 默认重试次数

  queues:
    default:
      max_workers: 20                   # 队列最大工作协程
      priority: 1                       # 队列优先级
```

## 任务管理

### 任务入队

通过HTTP API入队任务：

```bash
curl -X POST http://localhost:9999/admin/queue/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "email",
    "task_params": "{\"to\":\"user@example.com\",\"subject\":\"Test\"}",
    "queue": "email",
    "priority": 5
  }'
```

### 任务查询

```bash
# 获取队列统计
curl http://localhost:9999/admin/queue/stats

# 获取任务列表
curl http://localhost:9999/admin/queue/jobs?queue=default&state=available

# 获取仪表板数据
curl http://localhost:9999/admin/queue/dashboard
```

### 任务取消

```bash
curl -X POST http://localhost:9999/admin/queue/cancel \
  -H "Content-Type: application/json" \
  -d '{"job_id": 12345}'
```

## 任务处理器管理

### 注册处理器

```bash
curl -X POST http://localhost:9999/admin/handler/register \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "custom_task",
    "handler_name": "CustomTaskHandler",
    "handler_type": "http",
    "endpoint_url": "http://localhost:8080/handle",
    "timeout": 60,
    "retry_count": 3,
    "description": "自定义任务处理器"
  }'
```

### 处理器类型

- **local**: 本地处理器（直接调用Go函数）
- **http**: HTTP处理器（调用HTTP接口）
- **grpc**: gRPC处理器（调用gRPC服务）

## 监控和日志

### 日志配置

日志文件存储在配置的 `log_path` 目录中，按日期轮转。

### 监控指标

- 任务执行统计
- 队列长度监控
- 处理器健康状态
- 错误率统计

## 部署建议

### 生产环境部署

1. **数据库优化**:
   ```sql
   -- 优化PostgreSQL配置
   ALTER SYSTEM SET shared_buffers = '256MB';
   ALTER SYSTEM SET max_connections = 200;
   ALTER SYSTEM SET checkpoint_completion_target = 0.9;
   ```

2. **集群部署**:
   - 部署2-3个调度器实例
   - 使用不同的 `server_id`
   - 共享相同的PostgreSQL数据库

3. **监控设置**:
   - 监控PostgreSQL性能
   - 监控队列长度
   - 设置告警规则

### Docker部署

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o river-executor

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/river-executor .
COPY --from=builder /app/config-river.example.yaml ./config.yaml
CMD ["./river-executor"]
```

## 故障排除

### 常见问题

1. **连接失败**:
   - 检查PostgreSQL连接信息
   - 确认防火墙设置
   - 验证SSL配置

2. **任务不执行**:
   - 检查处理器注册状态
   - 验证队列配置
   - 查看错误日志

3. **性能问题**:
   - 调整工作协程数
   - 优化数据库索引
   - 增加连接池大小

### 日志分析

```bash
# 查看最近的错误日志
tail -f logs/executor.log | grep ERROR

# 查看任务执行统计
curl http://localhost:9999/admin/queue/dashboard
```

## API 参考

### 队列管理 API

- `GET /admin/queue/stats` - 获取统计信息
- `GET /admin/queue/jobs` - 获取任务列表
- `POST /admin/queue/enqueue` - 入队任务
- `POST /admin/queue/cancel` - 取消任务
- `GET /admin/queue/dashboard` - 获取仪表板

### 处理器管理 API

- `GET /admin/handler/list` - 列出处理器
- `POST /admin/handler/register` - 注册处理器
- `PUT /admin/handler/update` - 更新处理器
- `DELETE /admin/handler/delete` - 删除处理器

## 迁移指南

从旧的MySQL+Asynq系统迁移到River系统，请参考迁移工具使用说明。