-- 资源评论系统数据库表

-- 1. 资源评论表
CREATE TABLE IF NOT EXISTS `resource_comments` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '评论ID',
  `resource_id` BIGINT(20) NOT NULL COMMENT '资源ID',
  `user_id` BIGINT(20) NOT NULL COMMENT '评论用户ID',
  `parent_id` BIGINT(20) DEFAULT 0 COMMENT '父评论ID（0表示一级评论）',
  `root_id` BIGINT(20) DEFAULT 0 COMMENT '根评论ID（用于快速查询评论树）',
  `reply_to_user_id` BIGINT(20) DEFAULT NULL COMMENT '回复的用户ID',
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

-- 2. 资源评论点赞表
CREATE TABLE IF NOT EXISTS `resource_comment_likes` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `comment_id` BIGINT(20) NOT NULL COMMENT '评论ID',
  `user_id` BIGINT(20) NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_comment_user` (`comment_id`, `user_id`) COMMENT '评论用户唯一索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源评论点赞表';

-- 3. 为 resources 表添加评论数字段
-- 注意: 如果字段已存在会报错，可以忽略该错误
SET @query = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `resources` ADD COLUMN `comment_count` INT(11) DEFAULT 0 COMMENT ''评论数'' AFTER `like_count`;',
    'SELECT ''comment_count 列已存在，跳过添加'' AS message;'
  )
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = 'hub'
    AND TABLE_NAME = 'resources'
    AND COLUMN_NAME = 'comment_count'
);

PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 验证表创建
SHOW TABLES LIKE '%resource_comment%';
DESC resource_comments;
DESC resource_comment_likes;
DESC resources;

