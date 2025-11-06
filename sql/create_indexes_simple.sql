-- =====================================================
-- 数据库性能索引优化脚本（简化版 - 兼容MySQL 5.7+）
-- =====================================================
-- 数据库: hub
-- 执行日期: 2025-10-20
-- 说明: 兼容MySQL 5.7+的索引创建脚本
-- =====================================================

USE hub;

-- =====================================================
-- 1. 文章表 (articles) 索引
-- =====================================================
SELECT '正在为articles表创建索引...' AS Status;

-- 删除可能存在的旧索引（忽略错误）
ALTER TABLE articles DROP INDEX IF EXISTS idx_articles_status_created;
ALTER TABLE articles DROP INDEX IF EXISTS idx_articles_likes_views;
ALTER TABLE articles DROP INDEX IF EXISTS idx_articles_user_status;

-- 创建新索引
CREATE INDEX idx_articles_status_created 
ON articles(status, created_at DESC);

CREATE INDEX idx_articles_likes_views 
ON articles(like_count DESC, view_count DESC, created_at DESC);

CREATE INDEX idx_articles_user_status 
ON articles(user_id, status, created_at DESC);

SELECT '✓ articles表索引创建完成' AS Result;

-- =====================================================
-- 2. 评论表 (article_comments) 索引
-- =====================================================
SELECT '正在为article_comments表创建索引...' AS Status;

ALTER TABLE article_comments DROP INDEX IF EXISTS idx_comments_article_parent_status;
ALTER TABLE article_comments DROP INDEX IF EXISTS idx_comments_user_status;
ALTER TABLE article_comments DROP INDEX IF EXISTS idx_comments_root_status;

CREATE INDEX idx_comments_article_parent_status 
ON article_comments(article_id, parent_id, status, created_at);

CREATE INDEX idx_comments_user_status 
ON article_comments(user_id, status, created_at DESC);

CREATE INDEX idx_comments_root_status 
ON article_comments(root_id, status, created_at ASC);

SELECT '✓ article_comments表索引创建完成' AS Result;

-- =====================================================
-- 3. 评论点赞表 (article_comment_likes) 索引
-- =====================================================
SELECT '正在为article_comment_likes表创建索引...' AS Status;

ALTER TABLE article_comment_likes DROP INDEX IF EXISTS idx_comment_likes_check;
ALTER TABLE article_comment_likes DROP INDEX IF EXISTS idx_comment_likes_user;

CREATE INDEX idx_comment_likes_check 
ON article_comment_likes(comment_id, user_id);

CREATE INDEX idx_comment_likes_user 
ON article_comment_likes(user_id, comment_id);

SELECT '✓ article_comment_likes表索引创建完成' AS Result;

-- =====================================================
-- 4. 文章点赞表 (article_likes) 索引
-- =====================================================
SELECT '正在为article_likes表创建索引...' AS Status;

ALTER TABLE article_likes DROP INDEX IF EXISTS idx_article_likes_check;
ALTER TABLE article_likes DROP INDEX IF EXISTS idx_article_likes_user_time;

CREATE INDEX idx_article_likes_check 
ON article_likes(article_id, user_id);

CREATE INDEX idx_article_likes_user_time 
ON article_likes(user_id, created_at DESC);

SELECT '✓ article_likes表索引创建完成' AS Result;

-- =====================================================
-- 5. 文章分类关系表 (article_category_relations) 索引
-- =====================================================
SELECT '正在为article_category_relations表创建索引...' AS Status;

ALTER TABLE article_category_relations DROP INDEX IF EXISTS idx_article_category_article;
ALTER TABLE article_category_relations DROP INDEX IF EXISTS idx_article_category_category;

CREATE INDEX idx_article_category_article 
ON article_category_relations(article_id, category_id);

CREATE INDEX idx_article_category_category 
ON article_category_relations(category_id, article_id);

SELECT '✓ article_category_relations表索引创建完成' AS Result;

-- =====================================================
-- 6. 文章标签关系表 (article_tag_relations) 索引
-- =====================================================
SELECT '正在为article_tag_relations表创建索引...' AS Status;

ALTER TABLE article_tag_relations DROP INDEX IF EXISTS idx_article_tag_article;
ALTER TABLE article_tag_relations DROP INDEX IF EXISTS idx_article_tag_tag;

CREATE INDEX idx_article_tag_article 
ON article_tag_relations(article_id, tag_id);

CREATE INDEX idx_article_tag_tag 
ON article_tag_relations(tag_id, article_id);

SELECT '✓ article_tag_relations表索引创建完成' AS Result;

-- =====================================================
-- 7. 聊天消息表 (chat_messages) 索引
-- =====================================================
SELECT '正在为chat_messages表创建索引...' AS Status;

ALTER TABLE chat_messages DROP INDEX IF EXISTS idx_chat_status_id_desc;
ALTER TABLE chat_messages DROP INDEX IF EXISTS idx_chat_status_id_asc;
ALTER TABLE chat_messages DROP INDEX IF EXISTS idx_chat_user_id;

CREATE INDEX idx_chat_status_id_desc 
ON chat_messages(status, id DESC);

CREATE INDEX idx_chat_status_id_asc 
ON chat_messages(status, id ASC);

CREATE INDEX idx_chat_user_id 
ON chat_messages(user_id, status, id DESC);

SELECT '✓ chat_messages表索引创建完成' AS Result;

-- =====================================================
-- 8. 在线用户表 (online_users) 索引
-- =====================================================
SELECT '正在为online_users表创建索引...' AS Status;

ALTER TABLE online_users DROP INDEX IF EXISTS idx_online_heartbeat;

CREATE INDEX idx_online_heartbeat 
ON online_users(last_heartbeat DESC);

SELECT '✓ online_users表索引创建完成' AS Result;

-- =====================================================
-- 9. 用户认证表 (user_auth) 索引
-- =====================================================
SELECT '正在为user_auth表创建索引...' AS Status;

ALTER TABLE user_auth DROP INDEX IF EXISTS idx_user_auth_email;

CREATE INDEX idx_user_auth_email 
ON user_auth(email);

SELECT '✓ user_auth表索引创建完成' AS Result;

-- =====================================================
-- 索引创建完成统计
-- =====================================================
SELECT '========================================' AS '';
SELECT '✓ 所有索引创建完成！' AS Success;
SELECT '========================================' AS '';

-- 显示创建的索引
SELECT 
    TABLE_NAME AS '表名',
    INDEX_NAME AS '索引名',
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ', ') AS '索引列',
    INDEX_TYPE AS '类型'
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'hub'
  AND INDEX_NAME LIKE 'idx_%'
  AND TABLE_NAME IN (
    'articles', 
    'article_comments', 
    'article_comment_likes',
    'article_likes',
    'article_category_relations',
    'article_tag_relations',
    'chat_messages',
    'online_users',
    'user_auth'
  )
GROUP BY TABLE_NAME, INDEX_NAME, INDEX_TYPE
ORDER BY TABLE_NAME, INDEX_NAME;

-- 显示各表统计
SELECT 
    TABLE_NAME AS '表名',
    COUNT(DISTINCT INDEX_NAME) AS '索引数量',
    TABLE_ROWS AS '数据行数',
    ROUND(DATA_LENGTH / 1024 / 1024, 2) AS '数据大小_MB',
    ROUND(INDEX_LENGTH / 1024 / 1024, 2) AS '索引大小_MB'
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'hub'
  AND TABLE_NAME IN (
    'articles', 
    'article_comments', 
    'article_comment_likes',
    'article_likes',
    'article_category_relations',
    'article_tag_relations',
    'chat_messages',
    'online_users',
    'user_auth'
  )
GROUP BY TABLE_NAME, TABLE_ROWS, DATA_LENGTH, INDEX_LENGTH
ORDER BY TABLE_NAME;

SELECT '========================================' AS '';
SELECT '验证索引（EXPLAIN示例）' AS '';
SELECT '========================================' AS '';

-- 验证示例
EXPLAIN SELECT * FROM articles WHERE status = 1 ORDER BY created_at DESC LIMIT 20;

SELECT '========================================' AS '';
SELECT '✓ 索引创建和验证完成！' AS Success;
SELECT '提示: 请检查上方EXPLAIN输出，确认使用了索引' AS Tip;
SELECT '========================================' AS '';

