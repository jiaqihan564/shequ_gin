-- =====================================================
-- 数据库完整初始化脚本
-- =====================================================
-- 创建日期: 2025-11-06
-- 说明: 整合所有表结构、索引和初始数据
-- 使用方法:
--   mysql -u root -p < init_all_tables.sql
-- 或在MySQL命令行中:
--   source init_all_tables.sql
-- =====================================================

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `hub` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE hub;

-- =====================================================
-- 第一部分：用户系统表
-- =====================================================

-- 1. 用户认证表
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

-- 2. 用户资料表
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

-- 3. 密码重置token表
CREATE TABLE IF NOT EXISTS `password_reset_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `email` varchar(255) NOT NULL COMMENT '邮箱地址',
  `token` varchar(64) NOT NULL COMMENT '重置token',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `used` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否已使用(0:未使用, 1:已使用)',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_email` (`email`),
  KEY `idx_token` (`token`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='密码重置token表';

-- =====================================================
-- 第二部分：文章系统表
-- =====================================================

-- 4. 文章主表
CREATE TABLE IF NOT EXISTS `articles` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '文章ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '作者ID',
  `title` VARCHAR(200) NOT NULL COMMENT '文章标题',
  `description` VARCHAR(500) DEFAULT NULL COMMENT '文章描述/摘要',
  `content` TEXT NOT NULL COMMENT '文章正文（Markdown格式）',
  `status` TINYINT(1) DEFAULT 1 COMMENT '状态：0-草稿，1-已发布，2-已删除',
  `view_count` INT(11) DEFAULT 0 COMMENT '浏览次数',
  `like_count` INT(11) DEFAULT 0 COMMENT '点赞数',
  `comment_count` INT(11) DEFAULT 0 COMMENT '评论数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_hot` (`like_count`, `view_count`, `comment_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章表';

-- 5. 文章代码块表
CREATE TABLE IF NOT EXISTS `article_code_blocks` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '代码块ID',
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `language` VARCHAR(50) NOT NULL COMMENT '编程语言（如go, javascript, python）',
  `code_content` TEXT NOT NULL COMMENT '代码内容',
  `description` VARCHAR(200) DEFAULT NULL COMMENT '代码说明',
  `order_index` INT(11) DEFAULT 0 COMMENT '排序顺序',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_article_id` (`article_id`),
  KEY `idx_language` (`language`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章代码块表';

-- 6. 文章分类表
CREATE TABLE IF NOT EXISTS `article_categories` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '分类ID',
  `name` VARCHAR(50) NOT NULL COMMENT '分类名称',
  `slug` VARCHAR(50) NOT NULL COMMENT '分类slug（用于URL）',
  `description` VARCHAR(200) DEFAULT NULL COMMENT '分类描述',
  `parent_id` BIGINT(20) DEFAULT 0 COMMENT '父分类ID（0表示顶级分类）',
  `article_count` INT(11) DEFAULT 0 COMMENT '文章数量',
  `sort_order` INT(11) DEFAULT 0 COMMENT '排序顺序',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章分类表';

-- 7. 文章标签表
CREATE TABLE IF NOT EXISTS `article_tags` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '标签ID',
  `name` VARCHAR(50) NOT NULL COMMENT '标签名称',
  `slug` VARCHAR(50) NOT NULL COMMENT '标签slug',
  `article_count` INT(11) DEFAULT 0 COMMENT '文章数量',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`),
  KEY `idx_article_count` (`article_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章标签表';

-- 8. 文章-分类关联表
CREATE TABLE IF NOT EXISTS `article_category_relations` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `category_id` BIGINT(20) NOT NULL COMMENT '分类ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_article_category` (`article_id`, `category_id`),
  KEY `idx_category_id` (`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章分类关联表';

-- 9. 文章-标签关联表
CREATE TABLE IF NOT EXISTS `article_tag_relations` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `tag_id` BIGINT(20) NOT NULL COMMENT '标签ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_article_tag` (`article_id`, `tag_id`),
  KEY `idx_tag_id` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章标签关联表';

-- 10. 文章点赞表
CREATE TABLE IF NOT EXISTS `article_likes` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_article_user` (`article_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章点赞表';

-- 11. 文章评论表
CREATE TABLE IF NOT EXISTS `article_comments` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '评论ID',
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '评论用户ID',
  `parent_id` BIGINT(20) DEFAULT 0 COMMENT '父评论ID（0表示一级评论）',
  `root_id` BIGINT(20) DEFAULT 0 COMMENT '根评论ID（用于快速查询评论树）',
  `reply_to_user_id` int(10) UNSIGNED DEFAULT NULL COMMENT '回复的用户ID',
  `content` TEXT NOT NULL COMMENT '评论内容',
  `like_count` INT(11) DEFAULT 0 COMMENT '点赞数',
  `reply_count` INT(11) DEFAULT 0 COMMENT '回复数',
  `status` TINYINT(1) DEFAULT 1 COMMENT '状态：0-已删除，1-正常，2-已折叠',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '评论时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_article_id` (`article_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_root_id` (`root_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章评论表';

-- 12. 评论点赞表
CREATE TABLE IF NOT EXISTS `article_comment_likes` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `comment_id` BIGINT(20) NOT NULL COMMENT '评论ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_comment_user` (`comment_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论点赞表';

-- 13. 举报表
CREATE TABLE IF NOT EXISTS `article_reports` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '举报ID',
  `article_id` BIGINT(20) DEFAULT NULL COMMENT '文章ID',
  `comment_id` BIGINT(20) DEFAULT NULL COMMENT '评论ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '举报用户ID',
  `reason` VARCHAR(500) NOT NULL COMMENT '举报原因',
  `status` TINYINT(1) DEFAULT 0 COMMENT '状态：0-待处理，1-已处理，2-已驳回',
  `handler_id` int(10) UNSIGNED DEFAULT NULL COMMENT '处理人ID',
  `handler_note` VARCHAR(500) DEFAULT NULL COMMENT '处理备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '举报时间',
  `handled_at` DATETIME DEFAULT NULL COMMENT '处理时间',
  PRIMARY KEY (`id`),
  KEY `idx_article_id` (`article_id`),
  KEY `idx_comment_id` (`comment_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='举报表';

-- =====================================================
-- 第三部分：聊天系统表
-- =====================================================

-- 14. 聊天消息表
CREATE TABLE IF NOT EXISTS `chat_messages` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '消息ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
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

-- 15. 在线用户表
CREATE TABLE IF NOT EXISTS `online_users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `last_heartbeat` datetime NOT NULL COMMENT '最后心跳时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`) COMMENT '用户ID唯一索引',
  KEY `idx_heartbeat` (`last_heartbeat`) COMMENT '心跳时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='在线用户表';

-- =====================================================
-- 第四部分：代码运行平台表
-- =====================================================

-- 16. 代码片段表
CREATE TABLE IF NOT EXISTS `code_snippets` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '创建者ID',
  `title` VARCHAR(255) NOT NULL DEFAULT 'Untitled' COMMENT '代码标题',
  `language` VARCHAR(50) NOT NULL COMMENT '编程语言: python, javascript, java, cpp等',
  `code` TEXT NOT NULL COMMENT '代码内容',
  `description` TEXT COMMENT '代码描述',
  `is_public` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否公开: 0-私有, 1-公开',
  `share_token` VARCHAR(64) UNIQUE COMMENT '分享令牌（唯一）',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  INDEX idx_user_id (user_id),
  INDEX idx_language (language),
  INDEX idx_share_token (share_token),
  INDEX idx_created_at (created_at),
  INDEX idx_is_public (is_public)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码片段表';

-- 17. 代码执行记录表
CREATE TABLE IF NOT EXISTS `code_executions` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `snippet_id` BIGINT UNSIGNED COMMENT '代码片段ID（可空，临时执行无snippet_id）',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '执行者ID',
  `language` VARCHAR(50) NOT NULL COMMENT '编程语言',
  `code` TEXT NOT NULL COMMENT '执行的代码',
  `stdin` TEXT COMMENT '标准输入',
  `output` TEXT COMMENT '执行输出',
  `error` TEXT COMMENT '错误信息',
  `execution_time` INT COMMENT '执行耗时（毫秒）',
  `memory_usage` BIGINT COMMENT '内存使用（字节）',
  `status` VARCHAR(20) NOT NULL COMMENT '执行状态: success, error, timeout',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_user_id (user_id),
  INDEX idx_snippet_id (snippet_id),
  INDEX idx_language (language),
  INDEX idx_status (status),
  INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码执行记录表';

-- 18. 协作会话表
CREATE TABLE IF NOT EXISTS `code_collaborations` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `snippet_id` BIGINT UNSIGNED NOT NULL COMMENT '代码片段ID',
  `session_token` VARCHAR(64) NOT NULL UNIQUE COMMENT '会话令牌',
  `active_users` JSON COMMENT '当前在线用户列表',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `expires_at` TIMESTAMP NOT NULL COMMENT '过期时间',
  INDEX idx_snippet_id (snippet_id),
  INDEX idx_session_token (session_token),
  INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码协作会话表';

-- =====================================================
-- 第五部分：资源分享系统表
-- =====================================================

-- 19. 资源主表
CREATE TABLE IF NOT EXISTS `resources` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '资源ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '上传者ID',
  `title` varchar(200) NOT NULL COMMENT '资源标题',
  `description` text COMMENT '简短描述',
  `document` text COMMENT '详细文档（支持Markdown）',
  `category_id` int(11) DEFAULT NULL COMMENT '分类ID',
  `file_name` varchar(255) NOT NULL COMMENT '文件原始名称',
  `file_size` bigint(20) NOT NULL COMMENT '文件大小（字节）',
  `file_type` varchar(100) DEFAULT NULL COMMENT '文件类型(MIME)',
  `file_extension` varchar(20) DEFAULT NULL COMMENT '文件扩展名',
  `storage_path` varchar(500) NOT NULL COMMENT 'MinIO存储路径',
  `download_count` int(11) DEFAULT 0 COMMENT '下载次数',
  `view_count` int(11) DEFAULT 0 COMMENT '浏览次数',
  `like_count` int(11) DEFAULT 0 COMMENT '点赞数',
  `comment_count` int(11) DEFAULT 0 COMMENT '评论数',
  `status` tinyint(1) DEFAULT 1 COMMENT '状态：0-已删除，1-正常，2-审核中',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`) COMMENT '上传者索引',
  KEY `idx_category` (`category_id`) COMMENT '分类索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源文件表';

-- 20. 资源图片表
CREATE TABLE IF NOT EXISTS `resource_images` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '图片ID',
  `resource_id` bigint(20) NOT NULL COMMENT '资源ID',
  `image_url` varchar(500) NOT NULL COMMENT '图片URL',
  `image_order` int(11) DEFAULT 0 COMMENT '图片顺序',
  `is_cover` tinyint(1) DEFAULT 0 COMMENT '是否封面图：0-否，1-是',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_resource` (`resource_id`) COMMENT '资源索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源预览图片表';

-- 21. 资源分类表
CREATE TABLE IF NOT EXISTS `resource_categories` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '分类ID',
  `name` varchar(50) NOT NULL COMMENT '分类名称',
  `slug` varchar(50) NOT NULL COMMENT 'URL标识',
  `description` text COMMENT '分类描述',
  `resource_count` int(11) DEFAULT 0 COMMENT '资源数量',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`) COMMENT 'URL标识唯一索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源分类表';

-- 22. 资源标签表
CREATE TABLE IF NOT EXISTS `resource_tags` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '标签ID',
  `resource_id` bigint(20) NOT NULL COMMENT '资源ID',
  `tag_name` varchar(50) NOT NULL COMMENT '标签名称',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_resource` (`resource_id`) COMMENT '资源索引',
  KEY `idx_tag` (`tag_name`) COMMENT '标签索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源标签表';

-- 23. 断点续传记录表
CREATE TABLE IF NOT EXISTS `upload_chunks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `upload_id` varchar(64) NOT NULL COMMENT '上传任务ID（文件MD5）',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
  `file_name` varchar(255) NOT NULL COMMENT '文件名',
  `file_size` bigint(20) NOT NULL COMMENT '总大小（字节）',
  `chunk_size` int(11) NOT NULL COMMENT '分片大小（字节）',
  `total_chunks` int(11) NOT NULL COMMENT '总分片数',
  `uploaded_chunks` text COMMENT '已上传分片列表（JSON数组）',
  `storage_path` varchar(500) DEFAULT NULL COMMENT '合并后的存储路径',
  `status` tinyint(1) DEFAULT 0 COMMENT '状态：0-上传中，1-已完成，2-已取消',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_upload_id` (`upload_id`) COMMENT '上传ID唯一索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户索引',
  KEY `idx_expires` (`expires_at`) COMMENT '过期时间索引',
  KEY `idx_status` (`status`) COMMENT '状态索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='断点续传记录表';

-- 24. 资源点赞表
CREATE TABLE IF NOT EXISTS `resource_likes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '点赞ID',
  `resource_id` bigint(20) NOT NULL COMMENT '资源ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resource_user` (`resource_id`, `user_id`) COMMENT '资源用户唯一索引',
  KEY `idx_user` (`user_id`) COMMENT '用户索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源点赞表';

-- 25. 资源评论表
CREATE TABLE IF NOT EXISTS `resource_comments` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '评论ID',
  `resource_id` BIGINT(20) NOT NULL COMMENT '资源ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '评论用户ID',
  `parent_id` BIGINT(20) DEFAULT 0 COMMENT '父评论ID（0表示一级评论）',
  `root_id` BIGINT(20) DEFAULT 0 COMMENT '根评论ID（用于快速查询评论树）',
  `reply_to_user_id` int(10) UNSIGNED DEFAULT NULL COMMENT '回复的用户ID',
  `content` TEXT NOT NULL COMMENT '评论内容',
  `like_count` INT(11) DEFAULT 0 COMMENT '点赞数',
  `reply_count` INT(11) DEFAULT 0 COMMENT '回复数',
  `status` TINYINT(1) DEFAULT 1 COMMENT '状态：0-已删除，1-正常，2-已折叠',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '评论时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_resource_id` (`resource_id`) COMMENT '资源索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户索引',
  KEY `idx_parent_id` (`parent_id`) COMMENT '父评论索引',
  KEY `idx_root_id` (`root_id`) COMMENT '根评论索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源评论表';

-- 26. 资源评论点赞表
CREATE TABLE IF NOT EXISTS `resource_comment_likes` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `comment_id` BIGINT(20) NOT NULL COMMENT '评论ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_comment_user` (`comment_id`, `user_id`) COMMENT '评论用户唯一索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源评论点赞表';

-- =====================================================
-- 第六部分：私信系统表
-- =====================================================

-- 27. 私信会话表
CREATE TABLE IF NOT EXISTS `private_conversations` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '会话ID',
  `user1_id` int(10) UNSIGNED NOT NULL COMMENT '用户1 ID（较小的ID）',
  `user2_id` int(10) UNSIGNED NOT NULL COMMENT '用户2 ID（较大的ID）',
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

-- 28. 私信消息表
CREATE TABLE IF NOT EXISTS `private_messages` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '消息ID',
  `conversation_id` bigint(20) NOT NULL COMMENT '会话ID',
  `sender_id` int(10) UNSIGNED NOT NULL COMMENT '发送者ID',
  `receiver_id` int(10) UNSIGNED NOT NULL COMMENT '接收者ID',
  `content` text NOT NULL COMMENT '消息内容',
  `is_read` tinyint(1) DEFAULT 0 COMMENT '是否已读：0-未读，1-已读',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发送时间',
  PRIMARY KEY (`id`),
  KEY `idx_conversation` (`conversation_id`) COMMENT '会话索引',
  KEY `idx_sender` (`sender_id`) COMMENT '发送者索引',
  KEY `idx_receiver` (`receiver_id`) COMMENT '接收者索引',
  KEY `idx_created_at` (`created_at`) COMMENT '时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='私信消息表';

-- =====================================================
-- 第七部分：历史记录表
-- =====================================================

-- 29. 用户登录历史
CREATE TABLE IF NOT EXISTS `user_login_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
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

-- 30. 用户操作历史
CREATE TABLE IF NOT EXISTS `user_operation_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
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

-- 31. 个人资料修改历史
CREATE TABLE IF NOT EXISTS `profile_change_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` int(10) UNSIGNED NOT NULL COMMENT '用户ID',
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

-- =====================================================
-- 第八部分：统计系统表
-- =====================================================

-- 32. 累计统计表
CREATE TABLE IF NOT EXISTS `cumulative_statistics` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `stat_key` varchar(100) NOT NULL COMMENT '统计项键名（唯一标识）',
  `stat_value` bigint(20) NOT NULL DEFAULT 0 COMMENT '统计值',
  `stat_desc` varchar(200) DEFAULT NULL COMMENT '统计项描述',
  `category` varchar(50) NOT NULL COMMENT '分类：user-用户，api-接口，security-安全，content-内容',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_stat_key` (`stat_key`) COMMENT '统计键唯一索引',
  KEY `idx_category` (`category`) COMMENT '分类索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='累计统计表';

-- 33. 每日指标表
CREATE TABLE IF NOT EXISTS `daily_metrics` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `date` date NOT NULL COMMENT '日期',
  `active_users` int(11) NOT NULL DEFAULT 0 COMMENT '活跃用户数',
  `avg_response_time` decimal(10,2) NOT NULL DEFAULT 0.00 COMMENT '平均响应时间（毫秒）',
  `success_rate` decimal(5,2) NOT NULL DEFAULT 0.00 COMMENT '成功率（百分比）',
  `peak_concurrent` int(11) NOT NULL DEFAULT 0 COMMENT '峰值并发数',
  `most_popular_endpoint` varchar(255) DEFAULT NULL COMMENT '最热门的API端点',
  `new_users` int(11) NOT NULL DEFAULT 0 COMMENT '新增用户数',
  `total_requests` int(11) NOT NULL DEFAULT 0 COMMENT '总请求数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date` (`date`) COMMENT '日期唯一索引',
  KEY `idx_active_users` (`active_users`) COMMENT '活跃用户索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每日指标表';

-- 34. 实时指标表
CREATE TABLE IF NOT EXISTS `realtime_metrics` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `metric_key` varchar(100) NOT NULL COMMENT '指标键名（唯一标识）',
  `metric_value` varchar(500) NOT NULL COMMENT '指标值（JSON或纯文本）',
  `metric_desc` varchar(200) DEFAULT NULL COMMENT '指标描述',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_metric_key` (`metric_key`) COMMENT '指标键唯一索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='实时指标表';

-- 35. 用户统计表（按天）
CREATE TABLE IF NOT EXISTS `user_statistics` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `date` date NOT NULL COMMENT '统计日期',
  `login_count` int(11) NOT NULL DEFAULT 0 COMMENT '当天登录次数',
  `register_count` int(11) NOT NULL DEFAULT 0 COMMENT '当天注册次数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date` (`date`) COMMENT '确保每天只有一条记录'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户注册登录统计表（按天）';

-- 36. API统计表（按天+接口）
CREATE TABLE IF NOT EXISTS `api_statistics` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `date` date NOT NULL COMMENT '统计日期',
  `endpoint` varchar(255) NOT NULL COMMENT 'API接口路径',
  `method` varchar(10) NOT NULL COMMENT 'HTTP请求方法',
  `success_count` int(11) NOT NULL DEFAULT 0 COMMENT '成功请求数(2xx)',
  `error_count` int(11) NOT NULL DEFAULT 0 COMMENT '失败请求数(4xx,5xx)',
  `total_count` int(11) NOT NULL DEFAULT 0 COMMENT '总请求数',
  `avg_latency_ms` decimal(10,2) NOT NULL DEFAULT 0.00 COMMENT '平均响应时间(毫秒)',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date_endpoint_method` (`date`, `endpoint`, `method`) COMMENT '确保每天每个接口只有一条记录',
  KEY `idx_date` (`date`) COMMENT '按日期查询优化',
  KEY `idx_endpoint` (`endpoint`) COMMENT '按接口查询优化'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API接口访问统计表（按天）';

-- =====================================================
-- 第九部分：性能优化索引
-- =====================================================
-- 说明：兼容所有 MySQL 版本的索引创建方式
-- 如果索引已存在会报错，但不影响其他语句执行

-- 文章表性能索引
CREATE INDEX idx_articles_status_created ON articles(status, created_at DESC);
CREATE INDEX idx_articles_likes_views ON articles(like_count DESC, view_count DESC, created_at DESC);
CREATE INDEX idx_articles_user_status_created ON articles(user_id, status, created_at DESC);

-- 评论表性能索引
CREATE INDEX idx_comments_article_parent_status ON article_comments(article_id, parent_id, status, created_at);
CREATE INDEX idx_comment_likes_user ON article_comment_likes(user_id, comment_id);
CREATE INDEX idx_comment_likes_comment ON article_comment_likes(comment_id, user_id);

-- 聊天消息表性能索引
CREATE INDEX idx_chat_status_id ON chat_messages(status, id DESC);
CREATE INDEX idx_chat_status_id_asc ON chat_messages(status, id ASC);
CREATE INDEX idx_chat_user_id ON chat_messages(user_id, id DESC);

-- 在线用户表性能索引
CREATE INDEX idx_online_heartbeat ON online_users(last_heartbeat);

-- 文章分类关系表性能索引
CREATE INDEX idx_article_category_article ON article_category_relations(article_id, category_id);
CREATE INDEX idx_article_category_category ON article_category_relations(category_id, article_id);

-- 文章标签关系表性能索引
CREATE INDEX idx_article_tag_article ON article_tag_relations(article_id, tag_id);
CREATE INDEX idx_article_tag_tag ON article_tag_relations(tag_id, article_id);

-- 文章点赞表性能索引
CREATE INDEX idx_article_likes_user ON article_likes(user_id, article_id);
CREATE INDEX idx_article_likes_article ON article_likes(article_id, user_id);

-- 私信表性能索引
CREATE INDEX idx_private_messages_conversation ON private_messages(conversation_id, created_at DESC);
CREATE INDEX idx_private_messages_receiver ON private_messages(receiver_id, is_read, created_at DESC);

-- 注意：user_auth表已有uk_email唯一索引，无需再创建普通索引
-- 如果以上索引已存在，会报错但不影响数据库使用

-- =====================================================
-- 第十部分：初始化数据
-- =====================================================

-- 插入默认文章分类
INSERT INTO `article_categories` (`name`, `slug`, `description`, `parent_id`, `sort_order`) VALUES
('前端开发', 'frontend', '前端相关技术文章', 0, 1),
('后端开发', 'backend', '后端相关技术文章', 0, 2),
('移动开发', 'mobile', '移动端开发技术', 0, 3),
('数据库', 'database', '数据库相关技术', 0, 4),
('算法与数据结构', 'algorithm', '算法和数据结构相关', 0, 5),
('DevOps', 'devops', '运维和部署相关', 0, 6),
('架构设计', 'architecture', '系统架构设计', 0, 7),
('其他', 'others', '其他技术文章', 0, 99)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- 插入默认文章标签
INSERT INTO `article_tags` (`name`, `slug`) VALUES
('JavaScript', 'javascript'),
('TypeScript', 'typescript'),
('Vue.js', 'vue'),
('React', 'react'),
('Go', 'go'),
('Python', 'python'),
('Java', 'java'),
('MySQL', 'mysql'),
('Redis', 'redis'),
('Docker', 'docker'),
('Kubernetes', 'kubernetes'),
('微服务', 'microservices'),
('性能优化', 'performance'),
('代码重构', 'refactoring'),
('最佳实践', 'best-practices')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- 插入默认资源分类
INSERT INTO `resource_categories` (`name`, `slug`, `description`, `created_at`) VALUES
('软件工具', 'software', '各类实用软件和开发工具', NOW()),
('源码项目', 'source-code', '开源项目和代码示例', NOW()),
('设计素材', 'design', '图片、图标、UI套件等设计资源', NOW()),
('文档教程', 'tutorial', '教程文档和学习资料', NOW()),
('其他资源', 'others', '其他类型的资源文件', NOW())
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- 插入累计统计初始数据
INSERT INTO `cumulative_statistics` (`stat_key`, `stat_value`, `stat_desc`, `category`) VALUES
('total_users', 0, '总用户数', 'user'),
('total_logins', 0, '总登录次数', 'user'),
('total_registrations', 0, '总注册用户数', 'user'),
('active_users_today', 0, '今日活跃用户数', 'user'),
('total_api_calls', 0, '总API调用次数', 'api'),
('total_api_errors', 0, '总API错误次数', 'api'),
('avg_response_time', 0, '平均响应时间（毫秒）', 'api'),
('failed_login_attempts', 0, '失败登录尝试次数', 'security'),
('blocked_ips', 0, '被封禁的IP数', 'security'),
('security_alerts', 0, '安全告警次数', 'security'),
('total_articles', 0, '总文章数', 'content'),
('total_code_snippets', 0, '总代码片段数', 'content'),
('total_resources', 0, '总资源数', 'content'),
('total_comments', 0, '总评论数', 'content'),
('total_chat_messages', 0, '总聊天消息数', 'content')
ON DUPLICATE KEY UPDATE `stat_desc` = VALUES(`stat_desc`);

-- 插入实时指标初始数据
INSERT INTO `realtime_metrics` (`metric_key`, `metric_value`, `metric_desc`) VALUES
('online_users', '0', '当前在线用户数'),
('current_qps', '0', '当前QPS（每秒查询数）'),
('system_cpu', '0.0', '系统CPU使用率（百分比）'),
('system_memory', '0.0', '系统内存使用率（百分比）')
ON DUPLICATE KEY UPDATE `metric_value` = VALUES(`metric_value`);

-- =====================================================
-- 初始化完成
-- =====================================================

SELECT '数据库初始化完成！' AS Message;
SELECT CONCAT('共创建 ', COUNT(*), ' 个表') AS TableCount FROM information_schema.TABLES WHERE TABLE_SCHEMA = 'hub';

-- 显示所有创建的表
SELECT TABLE_NAME, TABLE_COMMENT 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'hub' 
ORDER BY TABLE_NAME;

