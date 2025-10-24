-- ========================================
-- 性能优化索引 - 针对高频查询添加优化索引
-- ========================================
-- ⚠️ 重要使用说明：
-- 1. MySQL不支持 CREATE INDEX IF NOT EXISTS 语法
-- 2. 如果索引已存在，执行会报错（可以忽略）
-- 3. 执行前建议先取消注释 DROP INDEX 语句（如果需要重建索引）
-- 4. 在开发环境先测试索引效果
-- 5. 使用 EXPLAIN 分析查询计划
-- 6. 生产环境在低峰期添加索引
-- 7. 大表添加索引建议使用：CREATE INDEX ... ALGORITHM=INPLACE, LOCK=NONE
--
-- 安全执行方式：
-- 方法1: 逐条执行，忽略已存在错误
-- 方法2: 先检查索引是否存在
--   SELECT COUNT(*) FROM information_schema.STATISTICS 
--   WHERE TABLE_SCHEMA = 'hub' AND TABLE_NAME = 'user_auth' 
--   AND INDEX_NAME = 'idx_account_status_login';
--
-- 删除索引（如需重建）：
--   DROP INDEX idx_account_status_login ON user_auth;
-- ========================================

USE hub;

-- ========================================
-- 用户认证表 (user_auth) 索引优化
-- ========================================

-- 登录查询优化（用户名查询）
-- 已有主键：id
-- 已有唯一索引：username, email

-- 检查索引是否存在，如果已存在则跳过
-- DROP INDEX IF EXISTS idx_account_status_login ON user_auth;
CREATE INDEX idx_account_status_login 
ON user_auth(account_status, last_login_time DESC)
COMMENT '优化活跃用户查询';

-- 失败登录监控
-- DROP INDEX IF EXISTS idx_failed_login_monitoring ON user_auth;
CREATE INDEX idx_failed_login_monitoring 
ON user_auth(failed_login_count, last_login_time)
COMMENT '监控异常登录';

-- ========================================
-- 用户资料表 (user_profile) 索引优化
-- ========================================

-- 用户资料查询
-- DROP INDEX IF EXISTS idx_profile_updated ON user_profile;
CREATE INDEX idx_profile_updated 
ON user_profile(user_id, updated_at DESC)
COMMENT '用户资料更新查询';

-- ========================================
-- 登录历史表 (login_history) 索引优化
-- ========================================

-- 按用户ID和时间查询登录历史
-- DROP INDEX IF EXISTS idx_login_user_time ON login_history;
CREATE INDEX idx_login_user_time 
ON login_history(user_id, login_time DESC)
COMMENT '用户登录历史查询';

-- 按IP地址查询登录历史（安全审计）
-- DROP INDEX IF EXISTS idx_login_ip_time ON login_history;
CREATE INDEX idx_login_ip_time 
ON login_history(login_ip, login_time DESC)
COMMENT 'IP登录历史查询';

-- 地理位置分析
-- DROP INDEX IF EXISTS idx_login_location ON login_history;
CREATE INDEX idx_login_location 
ON login_history(province, city, login_time DESC)
COMMENT '地理位置分析';

-- 复合索引：用户+状态+时间（覆盖索引）
-- DROP INDEX IF EXISTS idx_login_user_status_time ON login_history;
CREATE INDEX idx_login_user_status_time 
ON login_history(user_id, login_status, login_time DESC)
COMMENT '用户登录状态查询（覆盖索引）';

-- ========================================
-- 聊天消息表 (chat_messages) 索引优化
-- ========================================

-- 获取最新消息（分页查询）
-- DROP INDEX IF EXISTS idx_chat_time_id ON chat_messages;
CREATE INDEX idx_chat_time_id 
ON chat_messages(created_at DESC, id DESC)
COMMENT '聊天消息时间查询';

-- 按用户查询消息
-- DROP INDEX IF EXISTS idx_chat_user_time ON chat_messages;
CREATE INDEX idx_chat_user_time 
ON chat_messages(user_id, created_at DESC)
COMMENT '用户消息查询';

-- 全文搜索（MySQL 5.7+）
-- ALTER TABLE chat_messages ADD FULLTEXT INDEX ft_content (content) WITH PARSER ngram;

-- ========================================
-- 文章表 (articles) 索引优化
-- ========================================
-- 注意：articles 表本身已有基础索引 (idx_user_id, idx_status, idx_created_at, idx_hot)
-- 只添加复合索引优化查询

-- 用户文章时间查询
-- DROP INDEX IF EXISTS idx_article_user_time ON articles;
CREATE INDEX idx_article_user_time 
ON articles(user_id, created_at DESC)
COMMENT '用户文章查询';

-- 文章状态时间查询
-- DROP INDEX IF EXISTS idx_article_status_time ON articles;
CREATE INDEX idx_article_status_time 
ON articles(status, created_at DESC)
COMMENT '文章状态查询';

-- ========================================
-- 文章关联表索引优化
-- ========================================

-- 文章分类关联表（多对多）
-- DROP INDEX IF EXISTS idx_category_article_time ON article_category_relations;
CREATE INDEX idx_category_article_time 
ON article_category_relations(category_id, article_id)
COMMENT '按分类查询文章';

-- 文章标签关联表（多对多）
-- DROP INDEX IF EXISTS idx_tag_article ON article_tag_relations;
CREATE INDEX idx_tag_article 
ON article_tag_relations(tag_id, article_id)
COMMENT '按标签查询文章';

-- 文章搜索（全文索引 - 可选，需要MySQL 5.7+）
-- ALTER TABLE articles ADD FULLTEXT INDEX ft_article_search (title, content) WITH PARSER ngram;

-- ========================================
-- 评论表 (article_comments) 索引优化
-- ========================================

-- 按文章获取评论
-- DROP INDEX IF EXISTS idx_comment_article_time ON article_comments;
CREATE INDEX idx_comment_article_time 
ON article_comments(article_id, created_at DESC)
COMMENT '文章评论查询';

-- 按用户获取评论
-- DROP INDEX IF EXISTS idx_comment_user_time ON article_comments;
CREATE INDEX idx_comment_user_time 
ON article_comments(user_id, created_at DESC)
COMMENT '用户评论查询';

-- 父评论查询
-- DROP INDEX IF EXISTS idx_comment_parent ON article_comments;
CREATE INDEX idx_comment_parent 
ON article_comments(parent_id, created_at DESC)
COMMENT '子评论查询';

-- ========================================
-- 资源表 (resources) 索引优化
-- ========================================
-- 注意：resources 表已有基础索引 (idx_user_id, idx_category, idx_status, idx_created_at)

-- 资源分类时间查询
-- DROP INDEX IF EXISTS idx_resource_category_time ON resources;
CREATE INDEX idx_resource_category_time 
ON resources(category_id, created_at DESC)
COMMENT '分类资源查询';

-- 用户资源查询
-- DROP INDEX IF EXISTS idx_resource_user_time ON resources;
CREATE INDEX idx_resource_user_time 
ON resources(user_id, created_at DESC)
COMMENT '用户资源查询';

-- 资源审核状态查询
-- DROP INDEX IF EXISTS idx_resource_status_time ON resources;
CREATE INDEX idx_resource_status_time 
ON resources(status, created_at DESC)
COMMENT '资源审核查询';

-- 热门资源排序
-- DROP INDEX IF EXISTS idx_resource_hot ON resources;
CREATE INDEX idx_resource_hot 
ON resources(download_count DESC, like_count DESC)
COMMENT '热门资源排序';

-- 文件类型查询
-- DROP INDEX IF EXISTS idx_resource_file_type ON resources;
CREATE INDEX idx_resource_file_type 
ON resources(file_type, created_at DESC)
COMMENT '文件类型查询';

-- ========================================
-- 代码片段表 (code_snippets) 索引优化
-- ========================================

-- 按用户查询代码片段
-- DROP INDEX IF EXISTS idx_snippet_user_time ON code_snippets;
CREATE INDEX idx_snippet_user_time 
ON code_snippets(user_id, created_at DESC)
COMMENT '用户代码片段查询';

-- 按语言查询
-- DROP INDEX IF EXISTS idx_snippet_language_time ON code_snippets;
CREATE INDEX idx_snippet_language_time 
ON code_snippets(language, created_at DESC)
COMMENT '语言代码片段查询';

-- 公开代码片段查询
-- DROP INDEX IF EXISTS idx_snippet_public_time ON code_snippets;
CREATE INDEX idx_snippet_public_time 
ON code_snippets(is_public, created_at DESC)
COMMENT '公开代码片段查询';

-- 分享令牌查询（唯一索引）
-- DROP INDEX IF EXISTS idx_snippet_share_token ON code_snippets;
CREATE UNIQUE INDEX idx_snippet_share_token 
ON code_snippets(share_token)
COMMENT '分享令牌查询';

-- ========================================
-- 私信表 (private_messages) 索引优化
-- ========================================

-- 会话查询（发送者+接收者）
-- DROP INDEX IF EXISTS idx_message_sender_receiver_time ON private_messages;
CREATE INDEX idx_message_sender_receiver_time 
ON private_messages(sender_id, receiver_id, created_at DESC)
COMMENT '发送消息查询';

-- DROP INDEX IF EXISTS idx_message_receiver_sender_time ON private_messages;
CREATE INDEX idx_message_receiver_sender_time 
ON private_messages(receiver_id, sender_id, created_at DESC)
COMMENT '接收消息查询';

-- 未读消息查询
-- DROP INDEX IF EXISTS idx_message_unread ON private_messages;
CREATE INDEX idx_message_unread 
ON private_messages(receiver_id, is_read, created_at DESC)
COMMENT '未读消息查询';

-- ========================================
-- 统计表 (statistics) 索引优化
-- ========================================

-- 统计数据查询
-- DROP INDEX IF EXISTS idx_stats_date ON statistics;
CREATE INDEX idx_stats_date 
ON statistics(stat_date DESC)
COMMENT '日期统计查询';

-- DROP INDEX IF EXISTS idx_stats_api_date ON statistics;
CREATE INDEX idx_stats_api_date 
ON statistics(api_path, stat_date DESC)
COMMENT 'API统计查询';

-- ========================================
-- 累计统计表 (cumulative_stats) 索引优化
-- ========================================

-- 日期范围查询
-- DROP INDEX IF EXISTS idx_cumulative_date ON cumulative_stats;
CREATE INDEX idx_cumulative_date 
ON cumulative_stats(stat_date DESC)
COMMENT '累计统计日期查询';

-- ========================================
-- 性能监控查询示例
-- ========================================

-- 1. 查看慢查询日志
-- SET GLOBAL slow_query_log = 'ON';
-- SET GLOBAL long_query_time = 1; -- 记录超过1秒的查询

-- 2. 查看索引使用情况
-- SELECT * FROM sys.schema_unused_indexes WHERE object_schema = 'hub';

-- 3. 查看表的统计信息
-- ANALYZE TABLE user_auth, articles, chat_messages;

-- 4. 优化表
-- OPTIMIZE TABLE user_auth, articles, chat_messages;

-- ========================================
-- 查询优化建议
-- ========================================

/*
1. 使用EXPLAIN分析查询计划
   EXPLAIN SELECT * FROM articles WHERE category_id = 1 ORDER BY created_at DESC LIMIT 10;

2. 避免SELECT *，只查询需要的列

3. 使用覆盖索引（index covering）
   SELECT id, title, created_at FROM articles WHERE category_id = 1;

4. 避免在WHERE子句中使用函数
   错误: WHERE DATE(created_at) = '2024-01-01'
   正确: WHERE created_at >= '2024-01-01' AND created_at < '2024-01-02'

5. 使用LIMIT分页，避免一次查询大量数据

6. 使用批量操作代替循环单条操作
   INSERT INTO table VALUES (1), (2), (3);

7. 定期分析和优化表
   ANALYZE TABLE table_name;
   OPTIMIZE TABLE table_name;

8. 监控慢查询日志
   SELECT * FROM mysql.slow_log ORDER BY query_time DESC LIMIT 10;

9. 使用Redis等缓存减少数据库压力

10. 考虑读写分离和分库分表（数据量大时）
*/

-- ========================================
-- 索引创建完成后的维护建议
-- ========================================

/*
1. 定期检查索引使用情况
   SELECT * FROM sys.schema_unused_indexes WHERE object_schema = 'hub';

2. 定期更新表统计信息
   ANALYZE TABLE table_name;

3. 监控索引大小
   SELECT 
       TABLE_NAME,
       INDEX_NAME,
       ROUND(stat_value * @@innodb_page_size / 1024 / 1024, 2) AS size_mb
   FROM mysql.innodb_index_stats
   WHERE database_name = 'hub' AND stat_name = 'size'
   ORDER BY size_mb DESC;

4. 删除冗余索引
   -- 使用工具识别冗余索引，谨慎删除

5. 定期检查表碎片
   SELECT 
       TABLE_NAME,
       ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) AS total_mb,
       ROUND(DATA_FREE / 1024 / 1024, 2) AS free_mb
   FROM information_schema.TABLES
   WHERE TABLE_SCHEMA = 'hub' AND DATA_FREE > 0
   ORDER BY free_mb DESC;
*/

