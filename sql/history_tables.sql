-- 历史记录表
USE hub;

-- 表1：用户登录历史
CREATE TABLE IF NOT EXISTS `user_login_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `login_time` datetime NOT NULL COMMENT '登录时间',
  `login_ip` varchar(50) DEFAULT NULL COMMENT '登录IP地址',
  `user_agent` varchar(500) DEFAULT NULL COMMENT '浏览器UA信息',
  `login_status` tinyint(1) NOT NULL DEFAULT 1 COMMENT '登录状态：0-失败，1-成功',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`) COMMENT '按用户查询优化',
  KEY `idx_login_time` (`login_time`) COMMENT '按时间查询优化',
  KEY `idx_username` (`username`) COMMENT '按用户名查询优化'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户登录历史记录表';

-- 表2：用户操作历史
CREATE TABLE IF NOT EXISTS `user_operation_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `operation_type` varchar(50) NOT NULL COMMENT '操作类型（如：修改资料、修改密码、上传头像）',
  `operation_desc` varchar(500) DEFAULT NULL COMMENT '操作描述',
  `operation_time` datetime NOT NULL COMMENT '操作时间',
  `ip_address` varchar(50) DEFAULT NULL COMMENT '操作IP地址',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`) COMMENT '按用户查询优化',
  KEY `idx_operation_type` (`operation_type`) COMMENT '按操作类型查询优化',
  KEY `idx_operation_time` (`operation_time`) COMMENT '按时间查询优化'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户操作历史记录表';

-- 表3：个人资料修改历史
CREATE TABLE IF NOT EXISTS `profile_change_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `field_name` varchar(50) NOT NULL COMMENT '修改的字段名（nickname/bio/avatar等）',
  `old_value` text DEFAULT NULL COMMENT '修改前的值',
  `new_value` text DEFAULT NULL COMMENT '修改后的值',
  `change_time` datetime NOT NULL COMMENT '修改时间',
  `ip_address` varchar(50) DEFAULT NULL COMMENT '操作IP地址',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`) COMMENT '按用户查询优化',
  KEY `idx_field_name` (`field_name`) COMMENT '按字段名查询优化',
  KEY `idx_change_time` (`change_time`) COMMENT '按时间查询优化'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='个人资料修改历史记录表';

-- 验证表创建成功
SHOW TABLES LIKE '%history%';
DESC user_login_history;
DESC user_operation_history;
DESC profile_change_history;

