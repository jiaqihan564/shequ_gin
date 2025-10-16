-- 资源分享系统数据库表

-- 1. 资源主表
CREATE TABLE IF NOT EXISTS `resources` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '资源ID',
  `user_id` bigint(20) NOT NULL COMMENT '上传者ID',
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
  `status` tinyint(1) DEFAULT 1 COMMENT '状态：0-已删除，1-正常，2-审核中',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`) COMMENT '上传者索引',
  KEY `idx_category` (`category_id`) COMMENT '分类索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源文件表';

-- 2. 资源图片表
CREATE TABLE IF NOT EXISTS `resource_images` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '图片ID',
  `resource_id` bigint(20) NOT NULL COMMENT '资源ID',
  `image_url` varchar(500) NOT NULL COMMENT '图片URL',
  `image_order` int(11) DEFAULT 0 COMMENT '图片顺序',
  `is_cover` tinyint(1) DEFAULT 0 COMMENT '是否封面图：0-否，1-是',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_resource` (`resource_id`) COMMENT '资源索引',
  CONSTRAINT `fk_resource_images` FOREIGN KEY (`resource_id`) REFERENCES `resources` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源预览图片表';

-- 3. 资源分类表
CREATE TABLE IF NOT EXISTS `resource_categories` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '分类ID',
  `name` varchar(50) NOT NULL COMMENT '分类名称',
  `slug` varchar(50) NOT NULL COMMENT 'URL标识',
  `description` text COMMENT '分类描述',
  `resource_count` int(11) DEFAULT 0 COMMENT '资源数量',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`) COMMENT 'URL标识唯一索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源分类表';

-- 初始分类数据
INSERT INTO `resource_categories` (`name`, `slug`, `description`, `created_at`) VALUES
('软件工具', 'software', '各类实用软件和开发工具', NOW()),
('源码项目', 'source-code', '开源项目和代码示例', NOW()),
('设计素材', 'design', '图片、图标、UI套件等设计资源', NOW()),
('文档教程', 'tutorial', '教程文档和学习资料', NOW()),
('其他资源', 'others', '其他类型的资源文件', NOW())
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- 4. 资源标签表
CREATE TABLE IF NOT EXISTS `resource_tags` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '标签ID',
  `resource_id` bigint(20) NOT NULL COMMENT '资源ID',
  `tag_name` varchar(50) NOT NULL COMMENT '标签名称',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_resource` (`resource_id`) COMMENT '资源索引',
  KEY `idx_tag` (`tag_name`) COMMENT '标签索引',
  CONSTRAINT `fk_resource_tags` FOREIGN KEY (`resource_id`) REFERENCES `resources` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源标签表';

-- 5. 断点续传记录表
CREATE TABLE IF NOT EXISTS `upload_chunks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `upload_id` varchar(64) NOT NULL COMMENT '上传任务ID（文件MD5）',
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `file_name` varchar(255) NOT NULL COMMENT '文件名',
  `file_size` bigint(20) NOT NULL COMMENT '总大小（字节）',
  `chunk_size` int(11) NOT NULL COMMENT '分片大小（字节）',
  `total_chunks` int(11) NOT NULL COMMENT '总分片数',
  `uploaded_chunks` text COMMENT '已上传分片列表（JSON数组）',
  `storage_path` varchar(500) DEFAULT NULL COMMENT '合并后的存储路径',
  `status` tinyint(1) DEFAULT 0 COMMENT '状态：0-上传中，1-已完成，2-已取消',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_upload_id` (`upload_id`) COMMENT '上传ID唯一索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户索引',
  KEY `idx_expires` (`expires_at`) COMMENT '过期时间索引',
  KEY `idx_status` (`status`) COMMENT '状态索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='断点续传记录表';

-- 6. 资源点赞表
CREATE TABLE IF NOT EXISTS `resource_likes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '点赞ID',
  `resource_id` bigint(20) NOT NULL COMMENT '资源ID',
  `user_id` bigint(20) NOT NULL COMMENT '用户ID',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resource_user` (`resource_id`, `user_id`) COMMENT '资源用户唯一索引',
  KEY `idx_user` (`user_id`) COMMENT '用户索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源点赞表';

-- 验证表创建
SHOW TABLES LIKE '%resource%';
SHOW TABLES LIKE '%upload_chunks%';

