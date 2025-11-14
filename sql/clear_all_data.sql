-- =====================================================
-- 清空所有数据脚本
-- =====================================================
-- 创建日期: 2025-11-08
-- 说明: 清空所有表数据，但保留表结构
-- 使用方法:
--   mysql -h 43.138.113.105 -P 13306 -u root -p hub < clear_all_data.sql
-- 或在MySQL命令行中:
--   USE hub;
--   source clear_all_data.sql
-- =====================================================

USE hub;

-- 关闭外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- =====================================================
-- 第一部分：清空用户系统表
-- =====================================================
TRUNCATE TABLE `user_auth`;
TRUNCATE TABLE `user_profile`;
TRUNCATE TABLE `password_reset_tokens`;

-- =====================================================
-- 第二部分：清空文章系统表
-- =====================================================
TRUNCATE TABLE `article_reports`;
TRUNCATE TABLE `article_comment_likes`;
TRUNCATE TABLE `article_comments`;
TRUNCATE TABLE `article_likes`;
TRUNCATE TABLE `article_tag_relations`;
TRUNCATE TABLE `article_category_relations`;
TRUNCATE TABLE `article_code_blocks`;
TRUNCATE TABLE `articles`;
TRUNCATE TABLE `article_tags`;
TRUNCATE TABLE `article_categories`;

-- =====================================================
-- 第三部分：清空聊天系统表
-- =====================================================
TRUNCATE TABLE `chat_messages`;
TRUNCATE TABLE `online_users`;

-- =====================================================
-- 第四部分：清空代码运行平台表
-- =====================================================
TRUNCATE TABLE `code_collaborations`;
TRUNCATE TABLE `code_executions`;
TRUNCATE TABLE `code_snippets`;

-- =====================================================
-- 第五部分：清空资源分享系统表
-- =====================================================
TRUNCATE TABLE `resource_comment_likes`;
TRUNCATE TABLE `resource_comments`;
TRUNCATE TABLE `resource_likes`;
TRUNCATE TABLE `resource_tags`;
TRUNCATE TABLE `resource_images`;
TRUNCATE TABLE `upload_chunks`;
TRUNCATE TABLE `resources`;
TRUNCATE TABLE `resource_categories`;

-- =====================================================
-- 第六部分：清空私信系统表
-- =====================================================
TRUNCATE TABLE `private_messages`;
TRUNCATE TABLE `private_conversations`;

-- =====================================================
-- 第七部分：清空历史记录表
-- =====================================================
TRUNCATE TABLE `user_login_history`;
TRUNCATE TABLE `user_operation_history`;
TRUNCATE TABLE `profile_change_history`;

-- =====================================================
-- 第八部分：清空统计系统表
-- =====================================================
TRUNCATE TABLE `cumulative_statistics`;
TRUNCATE TABLE `daily_metrics`;
TRUNCATE TABLE `realtime_metrics`;
TRUNCATE TABLE `user_statistics`;
TRUNCATE TABLE `api_statistics`;

-- 开启外键检查
SET FOREIGN_KEY_CHECKS = 1;

-- =====================================================
-- 重新插入初始化数据
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
('total_errors', 0, '总错误次数', 'api'),
('total_api_errors', 0, '总API错误次数', 'api'),
('total_uploads', 0, '总上传次数', 'api'),
('avg_response_time', 0, '平均响应时间（毫秒）', 'api'),
('failed_login_attempts', 0, '失败登录尝试次数', 'security'),
('blocked_ips', 0, '被封禁IP数', 'security'),
('security_alerts', 0, '安全告警次数', 'security'),
('total_password_changes', 0, '修改密码次数', 'security'),
('total_password_resets', 0, '重置密码次数', 'security'),
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
-- 清空完成
-- =====================================================

SELECT '✅ 数据清空完成！' AS Message;
SELECT '✅ 已重新插入初始化数据（分类、标签、统计）' AS Info;

-- 统计各表数据量
SELECT 
    '用户表' AS Category,
    (SELECT COUNT(*) FROM user_auth) AS user_auth,
    (SELECT COUNT(*) FROM user_profile) AS user_profile;

SELECT 
    '文章表' AS Category,
    (SELECT COUNT(*) FROM articles) AS articles,
    (SELECT COUNT(*) FROM article_categories) AS categories,
    (SELECT COUNT(*) FROM article_tags) AS tags;

SELECT 
    '资源表' AS Category,
    (SELECT COUNT(*) FROM resources) AS resources,
    (SELECT COUNT(*) FROM resource_categories) AS categories;

SELECT 
    '聊天表' AS Category,
    (SELECT COUNT(*) FROM chat_messages) AS messages,
    (SELECT COUNT(*) FROM online_users) AS online_users;

SELECT 
    '统计表' AS Category,
    (SELECT COUNT(*) FROM cumulative_statistics) AS cumulative,
    (SELECT COUNT(*) FROM realtime_metrics) AS realtime;

