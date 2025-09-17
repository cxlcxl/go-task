-- 创建主数据库（用于队列配置和服务器管理）
CREATE DATABASE IF NOT EXISTS xxljob_executor CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建业务数据库（用于业务任务表）
CREATE DATABASE IF NOT EXISTS business_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 切换到主数据库
USE xxljob_executor;

-- 队列配置表
CREATE TABLE IF NOT EXISTS `queue_configs` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `queue_name` varchar(100) NOT NULL COMMENT '队列名称',
  `table_name` varchar(100) NOT NULL COMMENT '关联的任务表名',
  `server_id` varchar(50) DEFAULT '' COMMENT '指定运行的服务器ID，空表示任意服务器',
  `max_workers` int(11) DEFAULT 1 COMMENT '最大并发协程数',
  `scan_interval` int(11) DEFAULT 5 COMMENT '扫描间隔(秒)',
  `status` tinyint(1) DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_queue_name` (`queue_name`),
  KEY `idx_server_status` (`server_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='队列配置表';

-- 服务器信息表
CREATE TABLE IF NOT EXISTS `server_infos` (
  `id` varchar(50) NOT NULL COMMENT '服务器唯一标识',
  `server_name` varchar(100) DEFAULT '' COMMENT '服务器名称',
  `ip` varchar(15) DEFAULT '' COMMENT '服务器IP',
  `port` int(11) DEFAULT 0 COMMENT '服务器端口',
  `status` tinyint(1) DEFAULT 1 COMMENT '状态 1:在线 0:离线',
  `last_heartbeat` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '最后心跳时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status_heartbeat` (`status`, `last_heartbeat`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器信息表';

-- 示例任务表1 - 邮件任务
CREATE TABLE IF NOT EXISTS `email_tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_type` varchar(50) NOT NULL DEFAULT 'email' COMMENT '任务类型',
  `task_params` text COMMENT '任务参数(JSON格式)',
  `status` tinyint(1) DEFAULT 0 COMMENT '状态 0:待执行 1:执行中 2:已完成 3:失败 4:取消',
  `schedule_time` datetime NOT NULL COMMENT '计划执行时间',
  `start_time` datetime DEFAULT NULL COMMENT '开始执行时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束执行时间',
  `retry_count` int(11) DEFAULT 0 COMMENT '重试次数',
  `max_retries` int(11) DEFAULT 3 COMMENT '最大重试次数',
  `error_msg` text COMMENT '错误信息',
  `server_id` varchar(50) DEFAULT '' COMMENT '执行的服务器ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status_schedule` (`status`, `schedule_time`),
  KEY `idx_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮件任务表';

-- 示例任务表2 - 数据同步任务
CREATE TABLE IF NOT EXISTS `data_sync_tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_type` varchar(50) NOT NULL DEFAULT 'data_sync' COMMENT '任务类型',
  `task_params` text COMMENT '任务参数(JSON格式)',
  `status` tinyint(1) DEFAULT 0 COMMENT '状态 0:待执行 1:执行中 2:已完成 3:失败 4:取消',
  `schedule_time` datetime NOT NULL COMMENT '计划执行时间',
  `start_time` datetime DEFAULT NULL COMMENT '开始执行时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束执行时间',
  `retry_count` int(11) DEFAULT 0 COMMENT '重试次数',
  `max_retries` int(11) DEFAULT 3 COMMENT '最大重试次数',
  `error_msg` text COMMENT '错误信息',
  `server_id` varchar(50) DEFAULT '' COMMENT '执行的服务器ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status_schedule` (`status`, `schedule_time`),
  KEY `idx_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据同步任务表';

-- 示例任务表3 - 报表生成任务
CREATE TABLE IF NOT EXISTS `report_tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_type` varchar(50) NOT NULL DEFAULT 'report_generate' COMMENT '任务类型',
  `task_params` text COMMENT '任务参数(JSON格式)',
  `status` tinyint(1) DEFAULT 0 COMMENT '状态 0:待执行 1:执行中 2:已完成 3:失败 4:取消',
  `schedule_time` datetime NOT NULL COMMENT '计划执行时间',
  `start_time` datetime DEFAULT NULL COMMENT '开始执行时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束执行时间',
  `retry_count` int(11) DEFAULT 0 COMMENT '重试次数',
  `max_retries` int(11) DEFAULT 3 COMMENT '最大重试次数',
  `error_msg` text COMMENT '错误信息',
  `server_id` varchar(50) DEFAULT '' COMMENT '执行的服务器ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status_schedule` (`status`, `schedule_time`),
  KEY `idx_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='报表生成任务表';

-- 插入队列配置示例数据（支持多数据库表名）
INSERT INTO `queue_configs` (`queue_name`, `table_name`, `server_id`, `max_workers`, `scan_interval`, `status`) VALUES
('email_queue', 'default.email_tasks', 'server-001', 5, 3, 1),
('data_sync_queue', 'business.data_sync_tasks', 'server-002', 3, 5, 1),
('report_queue', 'default.report_tasks', '', 2, 10, 1),
('business_queue', 'business.business_tasks', 'server-003', 4, 5, 1);

-- 插入示例任务数据
INSERT INTO `email_tasks` (`task_params`, `schedule_time`) VALUES
('{"to":["user@example.com"],"subject":"测试邮件","content":"这是一封测试邮件","type":"html"}', NOW()),
('{"to":["admin@example.com"],"subject":"系统通知","content":"系统维护通知","type":"text"}', DATE_ADD(NOW(), INTERVAL 5 MINUTE));

INSERT INTO `data_sync_tasks` (`task_params`, `schedule_time`) VALUES
('{"source_db":"db1","target_db":"db2","table_name":"users","sync_type":"incremental","batch_size":1000}', NOW()),
('{"source_db":"db1","target_db":"db2","table_name":"orders","sync_type":"full","batch_size":500}', DATE_ADD(NOW(), INTERVAL 10 MINUTE));

INSERT INTO `report_tasks` (`task_params`, `schedule_time`) VALUES
('{"report_type":"daily","date_range":"yesterday","format":"pdf","recipients":["manager@example.com"]}', NOW()),
('{"report_type":"weekly","date_range":"last_week","format":"excel","recipients":["admin@example.com","boss@example.com"]}', DATE_ADD(NOW(), INTERVAL 15 MINUTE));

-- 切换到业务数据库创建业务任务表
USE business_db;

-- 业务数据同步任务表
CREATE TABLE IF NOT EXISTS `data_sync_tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `task_type` varchar(50) NOT NULL DEFAULT 'data_sync',
  `task_params` text,
  `status` tinyint(1) DEFAULT 0,
  `schedule_time` datetime NOT NULL,
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 业务任务表
CREATE TABLE IF NOT EXISTS `business_tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `task_type` varchar(50) NOT NULL DEFAULT 'business',
  `task_params` text,
  `status` tinyint(1) DEFAULT 0,
  `schedule_time` datetime NOT NULL,
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入业务数据库示例任务
INSERT INTO `data_sync_tasks` (`task_params`, `schedule_time`) VALUES
('{"源数据库":"db1","目标数据库":"db2","表名":"users"}', NOW()),
('{"源数据库":"db1","目标数据库":"db2","表名":"orders"}', DATE_ADD(NOW(), INTERVAL 10 MINUTE));

INSERT INTO `business_tasks` (`task_params`, `schedule_time`) VALUES
('{"业务类型":"order_process","订单ID":"12345"}', NOW()),
('{"业务类型":"user_notification","用户ID":"67890"}', DATE_ADD(NOW(), INTERVAL 5 MINUTE));
