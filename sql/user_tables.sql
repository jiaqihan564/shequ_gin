-- =====================================================
-- 用户系统数据库表
-- =====================================================
-- 说明: 用户认证和用户资料表
-- 依赖: 无（核心表，最先执行）
-- =====================================================

USE hub;

-- ============================
-- 1. 用户认证表 (user_auth)
-- ============================
-- 存储用户登录认证相关信息

CREATE TABLE IF NOT EXISTS `user_auth` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password_hash` varchar(255) NOT NULL COMMENT '密码哈希',
  `email` varchar(100) NOT NULL COMMENT '邮箱地址',
  `role` varchar(20) NOT NULL DEFAULT 'user' COMMENT '用户角色：admin-管理员，user-普通用户',
  `auth_status` tinyint(1) NOT NULL DEFAULT 0 COMMENT '认证状态：0-未认证，1-已认证',
  `account_status` tinyint(1) NOT NULL DEFAULT 1 COMMENT '账户状态：0-禁用，1-正常，2-锁定',
  `last_login_time` datetime DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(50) DEFAULT NULL COMMENT '最后登录IP',
  `failed_login_count` int(11) NOT NULL DEFAULT 0 COMMENT '连续登录失败次数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`) COMMENT '用户名唯一索引',
  UNIQUE KEY `uk_email` (`email`) COMMENT '邮箱唯一索引',
  KEY `idx_role` (`role`) COMMENT '角色索引',
  KEY `idx_account_status` (`account_status`) COMMENT '账户状态索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户认证表';

-- ============================
-- 2. 用户资料表 (user_profile)
-- ============================
-- 存储用户扩展信息（昵称、签名、头像等）

CREATE TABLE IF NOT EXISTS `user_profile` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID（关联user_auth.id）',
  `nickname` varchar(100) DEFAULT NULL COMMENT '用户昵称',
  `bio` varchar(500) DEFAULT NULL COMMENT '个人简介',
  `avatar_url` varchar(500) DEFAULT NULL COMMENT '头像URL',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `gender` tinyint(1) DEFAULT NULL COMMENT '性别：0-未知，1-男，2-女',
  `birthday` date DEFAULT NULL COMMENT '生日',
  `province` varchar(50) DEFAULT NULL COMMENT '省份',
  `city` varchar(50) DEFAULT NULL COMMENT '城市',
  `website` varchar(200) DEFAULT NULL COMMENT '个人网站',
  `github` varchar(100) DEFAULT NULL COMMENT 'GitHub用户名',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`) COMMENT '用户ID唯一索引',
  KEY `idx_province_city` (`province`, `city`) COMMENT '地区索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户资料表';

-- ============================
-- 3. 初始化数据
-- ============================

-- 注意：管理员账号由应用程序在启动时自动创建（见 internal/bootstrap/init_admin.go）
-- 这里不插入初始数据，避免与程序逻辑冲突

-- ============================
-- 4. 验证表创建
-- ============================

SHOW TABLES LIKE 'user_%';
DESC user_auth;
DESC user_profile;

-- ============================
-- 5. 使用说明
-- ============================
-- 
-- user_auth 表：
--   - 存储用户登录认证信息
--   - username 和 email 必须唯一
--   - password_hash 使用 bcrypt 加密
--   - role 字段区分管理员和普通用户
--   - failed_login_count 用于防暴力破解
--
-- user_profile 表：
--   - 存储用户扩展资料
--   - user_id 外键关联到 user_auth.id
--   - 一对一关系：一个用户只有一条资料记录
--   - 注册时会自动创建对应的 profile 记录
--
-- 关系说明：
--   user_auth (1) ←→ (1) user_profile
--   通过 user_profile.user_id 关联
--

