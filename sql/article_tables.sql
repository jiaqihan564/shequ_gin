-- 文章系统相关表
USE hub;

-- ============================
-- 1. 核心表结构
-- ============================

-- 文章主表
CREATE TABLE IF NOT EXISTS `articles` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '文章ID',
  `user_id` BIGINT(20) NOT NULL COMMENT '作者ID',
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

-- 文章代码块表
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

-- 文章分类表
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

-- 文章标签表
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

-- 文章-分类关联表
CREATE TABLE IF NOT EXISTS `article_category_relations` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `category_id` BIGINT(20) NOT NULL COMMENT '分类ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_article_category` (`article_id`, `category_id`),
  KEY `idx_category_id` (`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章分类关联表';

-- 文章-标签关联表
CREATE TABLE IF NOT EXISTS `article_tag_relations` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `tag_id` BIGINT(20) NOT NULL COMMENT '标签ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_article_tag` (`article_id`, `tag_id`),
  KEY `idx_tag_id` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章标签关联表';

-- ============================
-- 2. 点赞评论系统
-- ============================

-- 文章点赞表
CREATE TABLE IF NOT EXISTS `article_likes` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
  `user_id` BIGINT(20) NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_article_user` (`article_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章点赞表';

-- 文章评论表（支持嵌套）
CREATE TABLE IF NOT EXISTS `article_comments` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '评论ID',
  `article_id` BIGINT(20) NOT NULL COMMENT '文章ID',
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
  KEY `idx_article_id` (`article_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_root_id` (`root_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章评论表';

-- 评论点赞表
CREATE TABLE IF NOT EXISTS `article_comment_likes` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `comment_id` BIGINT(20) NOT NULL COMMENT '评论ID',
  `user_id` BIGINT(20) NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_comment_user` (`comment_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论点赞表';

-- ============================
-- 3. 举报系统
-- ============================

-- 举报表
CREATE TABLE IF NOT EXISTS `article_reports` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '举报ID',
  `article_id` BIGINT(20) DEFAULT NULL COMMENT '文章ID',
  `comment_id` BIGINT(20) DEFAULT NULL COMMENT '评论ID',
  `user_id` BIGINT(20) NOT NULL COMMENT '举报用户ID',
  `reason` VARCHAR(500) NOT NULL COMMENT '举报原因',
  `status` TINYINT(1) DEFAULT 0 COMMENT '状态：0-待处理，1-已处理，2-已驳回',
  `handler_id` BIGINT(20) DEFAULT NULL COMMENT '处理人ID',
  `handler_note` VARCHAR(500) DEFAULT NULL COMMENT '处理备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '举报时间',
  `handled_at` DATETIME DEFAULT NULL COMMENT '处理时间',
  PRIMARY KEY (`id`),
  KEY `idx_article_id` (`article_id`),
  KEY `idx_comment_id` (`comment_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='举报表';

-- ============================
-- 4. 初始化数据
-- ============================

-- 插入默认分类
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

-- 插入默认标签
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

-- ============================
-- 5. 验证表创建
-- ============================

-- 显示所有文章相关表
SHOW TABLES LIKE 'article%';

-- 显示表结构
DESC articles;
DESC article_code_blocks;
DESC article_categories;
DESC article_tags;
DESC article_comments;
DESC article_likes;
DESC article_reports;

