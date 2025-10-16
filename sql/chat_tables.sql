-- 聊天室相关表
USE hub;

-- 表1：聊天消息表
CREATE TABLE IF NOT EXISTS `chat_messages` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '消息ID',
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `nickname` varchar(100) DEFAULT NULL COMMENT '用户昵称',
  `avatar` varchar(500) DEFAULT NULL COMMENT '用户头像URL',
  `content` varchar(500) NOT NULL COMMENT '消息内容',
  `message_type` tinyint(1) DEFAULT 1 COMMENT '消息类型：1-普通消息，2-系统消息',
  `send_time` datetime NOT NULL COMMENT '发送时间',
  `ip_address` varchar(50) DEFAULT NULL COMMENT '发送IP',
  `status` tinyint(1) DEFAULT 1 COMMENT '状态：0-已删除，1-正常',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`) COMMENT '用户ID索引',
  KEY `idx_send_time` (`send_time`) COMMENT '发送时间索引',
  KEY `idx_status` (`status`) COMMENT '状态索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='聊天消息表';

-- 表2：在线用户表（可选，用于统计）
CREATE TABLE IF NOT EXISTS `online_users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `last_heartbeat` datetime NOT NULL COMMENT '最后心跳时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`) COMMENT '用户ID唯一索引',
  KEY `idx_heartbeat` (`last_heartbeat`) COMMENT '心跳时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='在线用户表';

-- 验证表创建成功
SHOW TABLES LIKE '%chat%';
SHOW TABLES LIKE '%online%';
DESC chat_messages;
DESC online_users;

