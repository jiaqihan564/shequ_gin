-- 私信系统数据库表

-- 私信会话表
CREATE TABLE IF NOT EXISTS `private_conversations` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '会话ID',
  `user1_id` bigint(20) NOT NULL COMMENT '用户1 ID（较小的ID）',
  `user2_id` bigint(20) NOT NULL COMMENT '用户2 ID（较大的ID）',
  `last_message_id` bigint(20) DEFAULT NULL COMMENT '最后一条消息ID',
  `last_message_content` varchar(500) DEFAULT NULL COMMENT '最后一条消息内容',
  `last_message_time` datetime DEFAULT NULL COMMENT '最后消息时间',
  `user1_unread` int(11) DEFAULT 0 COMMENT '用户1未读消息数',
  `user2_unread` int(11) DEFAULT 0 COMMENT '用户2未读消息数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_users` (`user1_id`, `user2_id`) COMMENT '用户组合唯一索引',
  KEY `idx_user1` (`user1_id`) COMMENT '用户1索引',
  KEY `idx_user2` (`user2_id`) COMMENT '用户2索引',
  KEY `idx_last_time` (`last_message_time`) COMMENT '最后消息时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='私信会话表';

-- 私信消息表
CREATE TABLE IF NOT EXISTS `private_messages` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '消息ID',
  `conversation_id` bigint(20) NOT NULL COMMENT '会话ID',
  `sender_id` bigint(20) NOT NULL COMMENT '发送者ID',
  `receiver_id` bigint(20) NOT NULL COMMENT '接收者ID',
  `content` text NOT NULL COMMENT '消息内容',
  `is_read` tinyint(1) DEFAULT 0 COMMENT '是否已读：0-未读，1-已读',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发送时间',
  PRIMARY KEY (`id`),
  KEY `idx_conversation` (`conversation_id`) COMMENT '会话索引',
  KEY `idx_sender` (`sender_id`) COMMENT '发送者索引',
  KEY `idx_receiver` (`receiver_id`) COMMENT '接收者索引',
  KEY `idx_created_at` (`created_at`) COMMENT '时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='私信消息表';

-- 验证表创建
SHOW TABLES LIKE '%private%';
DESC private_conversations;
DESC private_messages;

