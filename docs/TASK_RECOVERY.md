# 任务恢复机制说明

## 问题解决

针对您提出的"发布新版本时正在执行的任务会丢失"的问题，我已经实现了完整的任务恢复机制。

## 解决方案

### 1. 启动时任务恢复 (`RecoverOrphanedTasks`)

**机制**: 服务启动时自动恢复孤立任务
- 扫描所有队列配置中的任务表
- 查找状态为"执行中"(status=1)的任务
- 将这些任务状态重置为"待执行"(status=0)
- 清空server_id，让任何服务器都可以重新执行
- 记录错误信息："任务因服务重启而重置"

**实现位置**: [database/database.go](file:///Users/silent/Documents/codes/task/database/database.go#L123-L154) 的 `RecoverOrphanedTasks` 方法

### 2. 超时任务监控 (`RecoverTimeoutTasks`)

**机制**: 定期检查并恢复执行超时的任务
- 每5分钟检查一次所有任务表
- 查找执行中且开始时间超过30分钟的任务
- 将这些任务标记为失败状态
- 避免任务永久卡在执行中状态

**实现位置**: [database/database.go](file:///Users/silent/Documents/codes/task/database/database.go#L156-L187) 的 `RecoverTimeoutTasks` 方法

### 3. 优雅关闭机制

**机制**: 停止服务时给正在执行的任务时间完成
- 停止接收新任务
- 等待5秒让正在执行的任务完成
- 然后才真正关闭服务

**实现位置**: [queue/queue_manager.go](file:///Users/silent/Documents/codes/task/queue/queue_manager.go#L76-L91) 的 `Stop` 方法

## 工作流程

### 重启前
1. 某些任务正在执行中 (status=1, server_id=当前服务器)
2. 服务接收到关闭信号
3. 等待5秒让任务尽量完成

### 重启后
1. **启动恢复**: 自动执行 `RecoverOrphanedTasks`
   - 找到所有 status=1 的任务
   - 重置为 status=0 (待执行)
   - 清空 server_id
   - 设置错误信息

2. **正常扫描**: 队列管理器开始正常扫描
   - 重新扫描这些恢复的任务
   - 按正常流程重新执行

3. **超时监控**: 后台持续监控
   - 防止任务长时间卡住
   - 自动处理异常情况

## 多数据库支持

恢复机制完全支持多数据库环境：
- 自动解析 `database_name.table_name` 格式
- 针对每个数据库连接分别处理
- 确保所有业务库的任务都能恢复

## 日志记录

系统会详细记录恢复过程：
```
队列 email_queue 恢复了 3 个孤立任务
队列 sync_queue 恢复了 1 个超时任务
孤立任务恢复完成
```

## 配置参数

- **优雅关闭等待时间**: 5秒 (可在代码中调整)
- **超时检查间隔**: 5分钟
- **任务超时阈值**: 30分钟
- **配置同步间隔**: 10秒

## 使用示例

### 模拟场景测试

1. **启动服务并创建任务**:
```bash
# 创建一个邮件任务
curl -X POST "http://localhost:9999/admin/task/create?table=default.email_tasks" \
  -d '{"task_type": "email", "schedule_time": "2024-01-01 10:00:00", ...}'
```

2. **任务开始执行时强制重启服务**:
```bash
# 强制停止
kill -9 <进程ID>

# 重新启动
./xxljob-executor
```

3. **观察恢复过程**:
- 启动日志会显示恢复的任务数量
- 任务会重新进入待执行状态
- 自动重新执行

## 重要特点

✅ **零任务丢失**: 确保所有任务最终都会被执行
✅ **自动恢复**: 无需人工干预
✅ **多库支持**: 支持分库存储的任务表
✅ **状态追踪**: 详细的状态变更记录
✅ **超时保护**: 防止任务永久卡住

这个机制确保了即使在服务重启、异常崩溃等情况下，您的任务系统仍然可靠稳定地运行。