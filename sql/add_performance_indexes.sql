-- =====================================================
-- 数据库性能索引优化脚本
-- =====================================================
-- 数据库: hub
-- 执行日期: 2025-10-20
-- 版本: 1.0
-- 说明: 为社区平台核心表添加性能优化索引
-- =====================================================

-- 使用hub数据库
USE hub;

-- 设置字符集
SET NAMES utf8mb4;

-- =====================================================
-- 检查数据库是否存在
-- =====================================================
SELECT 'Checking database...' AS Status;
SELECT DATABASE() AS CurrentDatabase;

-- =====================================================
-- 1. 文章表 (articles) 索引优化
-- =====================================================
SELECT '1. 优化文章表索引...' AS Status;

-- 检查表是否存在
SELECT COUNT(*) INTO @table_exists 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'hub' 
  AND TABLE_NAME = 'articles';

-- 如果表存在，添加索引
SET @sql = IF(@table_exists > 0, 
    'SELECT ''articles表存在，开始添加索引'' AS Info',
    'SELECT ''警告: articles表不存在'' AS Warning');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 1.1 文章列表查询索引（按状态和创建时间排序）
-- 用于: SELECT * FROM articles WHERE status = 1 ORDER BY created_at DESC LIMIT 20;
-- 兼容MySQL 5.7+的删除索引方式
SET @drop_index_1 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'articles' 
       AND INDEX_NAME = 'idx_articles_status_created') > 0,
    'ALTER TABLE articles DROP INDEX idx_articles_status_created',
    'SELECT ''索引不存在，跳过删除'' AS Info'
));
PREPARE stmt FROM @drop_index_1;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_articles_status_created 
ON articles(status, created_at DESC)
COMMENT '文章列表查询：按状态和时间排序';

SELECT CONCAT('✓ 已创建索引: idx_articles_status_created (', 
    COUNT(*), ' 条记录)') AS Result
FROM articles;

-- 1.2 文章热度排序索引（按点赞数和浏览数）
-- 用于: SELECT * FROM articles WHERE status = 1 ORDER BY like_count DESC, view_count DESC;
SET @drop_index_2 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'articles' 
       AND INDEX_NAME = 'idx_articles_likes_views') > 0,
    'ALTER TABLE articles DROP INDEX idx_articles_likes_views',
    'SELECT ''索引不存在，跳过删除'' AS Info'
));
PREPARE stmt FROM @drop_index_2;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_articles_likes_views 
ON articles(like_count DESC, view_count DESC, created_at DESC)
COMMENT '文章热度排序：点赞数+浏览数';

SELECT '✓ 已创建索引: idx_articles_likes_views' AS Result;

-- 1.3 用户文章列表索引
-- 用于: SELECT * FROM articles WHERE user_id = ? AND status = 1 ORDER BY created_at DESC;
SET @drop_index_3 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'articles' 
       AND INDEX_NAME = 'idx_articles_user_status') > 0,
    'ALTER TABLE articles DROP INDEX idx_articles_user_status',
    'SELECT ''索引不存在，跳过删除'' AS Info'
));
PREPARE stmt FROM @drop_index_3;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_articles_user_status 
ON articles(user_id, status, created_at DESC)
COMMENT '用户文章查询：按用户ID和状态';

SELECT '✓ 已创建索引: idx_articles_user_status' AS Result;

-- =====================================================
-- 2. 文章评论表 (article_comments) 索引优化
-- =====================================================
SELECT '2. 优化评论表索引...' AS Status;

-- 检查表是否存在
SELECT COUNT(*) INTO @table_exists 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'hub' 
  AND TABLE_NAME = 'article_comments';

-- 2.1 评论查询索引（评论树结构）
-- 用于: SELECT * FROM article_comments WHERE article_id = ? AND parent_id = 0 AND status = 1;
SET @drop_index_4 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_comments' 
       AND INDEX_NAME = 'idx_comments_article_parent_status') > 0,
    'ALTER TABLE article_comments DROP INDEX idx_comments_article_parent_status',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_4;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_comments_article_parent_status 
ON article_comments(article_id, parent_id, status, created_at)
COMMENT '评论查询：文章ID+父评论ID+状态';

SELECT CONCAT('✓ 已创建索引: idx_comments_article_parent_status (', 
    COUNT(*), ' 条记录)') AS Result
FROM article_comments;

-- 2.2 用户评论查询索引
-- 用于: SELECT * FROM article_comments WHERE user_id = ? AND status = 1;
SET @drop_index_5 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_comments' 
       AND INDEX_NAME = 'idx_comments_user_status') > 0,
    'ALTER TABLE article_comments DROP INDEX idx_comments_user_status',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_5;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_comments_user_status 
ON article_comments(user_id, status, created_at DESC)
COMMENT '用户评论查询';

SELECT '✓ 已创建索引: idx_comments_user_status' AS Result;

-- 2.3 根评论查询索引
-- 用于: SELECT * FROM article_comments WHERE root_id = ? AND status = 1;
SET @drop_index_6 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_comments' 
       AND INDEX_NAME = 'idx_comments_root_status') > 0,
    'ALTER TABLE article_comments DROP INDEX idx_comments_root_status',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_6;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_comments_root_status 
ON article_comments(root_id, status, created_at ASC)
COMMENT '根评论查询：获取评论回复';

SELECT '✓ 已创建索引: idx_comments_root_status' AS Result;

-- =====================================================
-- 3. 评论点赞表 (article_comment_likes) 索引优化
-- =====================================================
SELECT '3. 优化评论点赞表索引...' AS Status;

-- 3.1 检查用户是否点赞评论
-- 用于: SELECT * FROM article_comment_likes WHERE comment_id = ? AND user_id = ?;
SET @drop_index_7 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_comment_likes' 
       AND INDEX_NAME = 'idx_comment_likes_check') > 0,
    'ALTER TABLE article_comment_likes DROP INDEX idx_comment_likes_check',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_7;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_comment_likes_check 
ON article_comment_likes(comment_id, user_id)
COMMENT '检查评论点赞状态';

SELECT '✓ 已创建索引: idx_comment_likes_check' AS Result;

-- 3.2 批量查询评论点赞状态
-- 用于: SELECT comment_id FROM article_comment_likes WHERE comment_id IN (?,?,?) AND user_id = ?;
SET @drop_index_8 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_comment_likes' 
       AND INDEX_NAME = 'idx_comment_likes_user') > 0,
    'ALTER TABLE article_comment_likes DROP INDEX idx_comment_likes_user',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_8;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_comment_likes_user 
ON article_comment_likes(user_id, comment_id)
COMMENT '批量查询用户点赞的评论';

SELECT '✓ 已创建索引: idx_comment_likes_user' AS Result;

-- =====================================================
-- 4. 文章点赞表 (article_likes) 索引优化
-- =====================================================
SELECT '4. 优化文章点赞表索引...' AS Status;

-- 4.1 检查用户是否点赞文章
-- 用于: SELECT * FROM article_likes WHERE article_id = ? AND user_id = ?;
SET @drop_index_9 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_likes' 
       AND INDEX_NAME = 'idx_article_likes_check') > 0,
    'ALTER TABLE article_likes DROP INDEX idx_article_likes_check',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_9;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_article_likes_check 
ON article_likes(article_id, user_id)
COMMENT '检查文章点赞状态';

SELECT '✓ 已创建索引: idx_article_likes_check' AS Result;

-- 4.2 用户点赞的文章列表
-- 用于: SELECT article_id FROM article_likes WHERE user_id = ? ORDER BY created_at DESC;
SET @drop_index_10 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_likes' 
       AND INDEX_NAME = 'idx_article_likes_user_time') > 0,
    'ALTER TABLE article_likes DROP INDEX idx_article_likes_user_time',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_10;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_article_likes_user_time 
ON article_likes(user_id, created_at DESC)
COMMENT '用户点赞文章列表';

SELECT '✓ 已创建索引: idx_article_likes_user_time' AS Result;

-- =====================================================
-- 5. 文章分类关系表 (article_category_relations) 索引优化
-- =====================================================
SELECT '5. 优化文章分类关系表索引...' AS Status;

-- 5.1 查询文章的分类
-- 用于: SELECT * FROM article_category_relations WHERE article_id = ?;
SET @drop_index_11 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_category_relations' 
       AND INDEX_NAME = 'idx_article_category_article') > 0,
    'ALTER TABLE article_category_relations DROP INDEX idx_article_category_article',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_11;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_article_category_article 
ON article_category_relations(article_id, category_id)
COMMENT '查询文章的分类';

SELECT '✓ 已创建索引: idx_article_category_article' AS Result;

-- 5.2 查询分类下的文章
-- 用于: SELECT article_id FROM article_category_relations WHERE category_id = ?;
SET @drop_index_12 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_category_relations' 
       AND INDEX_NAME = 'idx_article_category_category') > 0,
    'ALTER TABLE article_category_relations DROP INDEX idx_article_category_category',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_12;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_article_category_category 
ON article_category_relations(category_id, article_id)
COMMENT '查询分类下的文章';

SELECT '✓ 已创建索引: idx_article_category_category' AS Result;

-- =====================================================
-- 6. 文章标签关系表 (article_tag_relations) 索引优化
-- =====================================================
SELECT '6. 优化文章标签关系表索引...' AS Status;

-- 6.1 查询文章的标签
-- 用于: SELECT * FROM article_tag_relations WHERE article_id = ?;
SET @drop_index_13 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_tag_relations' 
       AND INDEX_NAME = 'idx_article_tag_article') > 0,
    'ALTER TABLE article_tag_relations DROP INDEX idx_article_tag_article',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_13;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_article_tag_article 
ON article_tag_relations(article_id, tag_id)
COMMENT '查询文章的标签';

SELECT '✓ 已创建索引: idx_article_tag_article' AS Result;

-- 6.2 查询标签下的文章
-- 用于: SELECT article_id FROM article_tag_relations WHERE tag_id = ?;
SET @drop_index_14 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'article_tag_relations' 
       AND INDEX_NAME = 'idx_article_tag_tag') > 0,
    'ALTER TABLE article_tag_relations DROP INDEX idx_article_tag_tag',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_14;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_article_tag_tag 
ON article_tag_relations(tag_id, article_id)
COMMENT '查询标签下的文章';

SELECT '✓ 已创建索引: idx_article_tag_tag' AS Result;

-- =====================================================
-- 7. 聊天消息表 (chat_messages) 索引优化
-- =====================================================
SELECT '7. 优化聊天消息表索引...' AS Status;

-- 检查表是否存在
SELECT COUNT(*) INTO @table_exists 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'hub' 
  AND TABLE_NAME = 'chat_messages';

-- 7.1 获取最新消息
-- 用于: SELECT * FROM chat_messages WHERE status = 1 ORDER BY id DESC LIMIT 50;
SET @drop_index_15 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'chat_messages' 
       AND INDEX_NAME = 'idx_chat_status_id_desc') > 0,
    'ALTER TABLE chat_messages DROP INDEX idx_chat_status_id_desc',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_15;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_chat_status_id_desc 
ON chat_messages(status, id DESC)
COMMENT '获取最新聊天消息';

SELECT CONCAT('✓ 已创建索引: idx_chat_status_id_desc (', 
    COUNT(*), ' 条记录)') AS Result
FROM chat_messages;

-- 7.2 获取新消息（轮询）
-- 用于: SELECT * FROM chat_messages WHERE status = 1 AND id > ? ORDER BY id ASC;
SET @drop_index_16 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'chat_messages' 
       AND INDEX_NAME = 'idx_chat_status_id_asc') > 0,
    'ALTER TABLE chat_messages DROP INDEX idx_chat_status_id_asc',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_16;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_chat_status_id_asc 
ON chat_messages(status, id ASC)
COMMENT '获取指定ID之后的新消息';

SELECT '✓ 已创建索引: idx_chat_status_id_asc' AS Result;

-- 7.3 用户消息查询
-- 用于: SELECT * FROM chat_messages WHERE user_id = ? AND status = 1 ORDER BY id DESC;
SET @drop_index_17 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'chat_messages' 
       AND INDEX_NAME = 'idx_chat_user_id') > 0,
    'ALTER TABLE chat_messages DROP INDEX idx_chat_user_id',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_17;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_chat_user_id 
ON chat_messages(user_id, status, id DESC)
COMMENT '查询用户的聊天消息';

SELECT '✓ 已创建索引: idx_chat_user_id' AS Result;

-- =====================================================
-- 8. 在线用户表 (online_users) 索引优化
-- =====================================================
SELECT '8. 优化在线用户表索引...' AS Status;

-- 检查表是否存在
SELECT COUNT(*) INTO @table_exists 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'hub' 
  AND TABLE_NAME = 'online_users';

-- 8.1 获取在线用户数
-- 用于: SELECT COUNT(*) FROM online_users WHERE last_heartbeat >= ?;
SET @drop_index_18 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'online_users' 
       AND INDEX_NAME = 'idx_online_heartbeat') > 0,
    'ALTER TABLE online_users DROP INDEX idx_online_heartbeat',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_18;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_online_heartbeat 
ON online_users(last_heartbeat DESC)
COMMENT '在线用户心跳时间查询';

SELECT CONCAT('✓ 已创建索引: idx_online_heartbeat (', 
    COUNT(*), ' 条记录)') AS Result
FROM online_users;

-- =====================================================
-- 9. 私信表 (private_messages) 索引优化
-- =====================================================
SELECT '9. 优化私信表索引...' AS Status;

-- 检查表是否存在
SELECT COUNT(*) INTO @table_exists 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'hub' 
  AND TABLE_NAME = 'private_messages';

SET @sql = IF(@table_exists > 0, 
    'SELECT ''private_messages表存在'' AS Info',
    'SELECT ''跳过: private_messages表不存在'' AS Info');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 9.1 会话消息查询
-- 用于: SELECT * FROM private_messages WHERE conversation_id = ? ORDER BY created_at DESC;
-- 先尝试删除旧索引（如果存在）
SET @drop_sql = IF(@table_exists > 0, 
    (SELECT IF(
        (SELECT COUNT(*) FROM information_schema.STATISTICS 
         WHERE TABLE_SCHEMA = DATABASE() 
           AND TABLE_NAME = 'private_messages' 
           AND INDEX_NAME = 'idx_private_conversation') > 0,
        'ALTER TABLE private_messages DROP INDEX idx_private_conversation',
        'SELECT ''索引不存在'' AS Info'
    )),
    'SELECT ''表不存在'' AS Info');
PREPARE stmt FROM @drop_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(@table_exists > 0, 
    'CREATE INDEX idx_private_conversation ON private_messages(conversation_id, created_at DESC)',
    'SELECT ''跳过'' AS Result');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT '✓ 已创建索引: idx_private_conversation (如果表存在)' AS Result;

-- 9.2 接收者未读消息
-- 用于: SELECT * FROM private_messages WHERE receiver_id = ? AND is_read = 0;
SET @drop_sql2 = IF(@table_exists > 0, 
    (SELECT IF(
        (SELECT COUNT(*) FROM information_schema.STATISTICS 
         WHERE TABLE_SCHEMA = DATABASE() 
           AND TABLE_NAME = 'private_messages' 
           AND INDEX_NAME = 'idx_private_receiver_unread') > 0,
        'ALTER TABLE private_messages DROP INDEX idx_private_receiver_unread',
        'SELECT ''索引不存在'' AS Info'
    )),
    'SELECT ''表不存在'' AS Info');
PREPARE stmt FROM @drop_sql2;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(@table_exists > 0, 
    'CREATE INDEX idx_private_receiver_unread ON private_messages(receiver_id, is_read, created_at DESC)',
    'SELECT ''跳过'' AS Result');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT '✓ 已创建索引: idx_private_receiver_unread (如果表存在)' AS Result;

-- =====================================================
-- 10. 用户认证表 (user_auth) 索引优化
-- =====================================================
SELECT '10. 优化用户认证表索引...' AS Status;

-- 10.1 邮箱登录查询
-- 用于: SELECT * FROM user_auth WHERE email = ?;
SET @drop_index_19 = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
       AND TABLE_NAME = 'user_auth' 
       AND INDEX_NAME = 'idx_user_auth_email') > 0,
    'ALTER TABLE user_auth DROP INDEX idx_user_auth_email',
    'SELECT ''索引不存在'' AS Info'
));
PREPARE stmt FROM @drop_index_19;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE INDEX idx_user_auth_email 
ON user_auth(email)
COMMENT '邮箱查询（用于登录和重置密码）';

SELECT CONCAT('✓ 已创建索引: idx_user_auth_email (', 
    COUNT(*), ' 条记录)') AS Result
FROM user_auth;

-- =====================================================
-- 索引创建完成，显示统计信息
-- =====================================================
SELECT '========================================' AS '';
SELECT '索引创建完成！' AS Status;
SELECT '========================================' AS '';

-- 显示所有新创建的索引
SELECT 
    'hub' AS 数据库,
    TABLE_NAME AS 表名,
    INDEX_NAME AS 索引名,
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ', ') AS 索引列,
    INDEX_TYPE AS 索引类型,
    CASE NON_UNIQUE 
        WHEN 0 THEN '唯一' 
        ELSE '非唯一' 
    END AS 索引性质
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
    'private_messages',
    'user_auth'
  )
GROUP BY TABLE_NAME, INDEX_NAME, INDEX_TYPE, NON_UNIQUE
ORDER BY TABLE_NAME, INDEX_NAME;

-- 显示各表的基本统计
SELECT 
    TABLE_NAME AS '表名',
    TABLE_ROWS AS '预估行数',
    ROUND(DATA_LENGTH / 1024 / 1024, 2) AS '数据大小_MB',
    ROUND(INDEX_LENGTH / 1024 / 1024, 2) AS '索引大小_MB',
    ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) AS '总大小_MB'
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
    'private_messages',
    'user_auth'
  )
ORDER BY TABLE_NAME;

-- 显示各表的索引数量（单独查询）
SELECT 
    TABLE_NAME AS '表名',
    COUNT(DISTINCT INDEX_NAME) AS '索引数量'
FROM information_schema.STATISTICS
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
    'private_messages',
    'user_auth'
  )
GROUP BY TABLE_NAME
ORDER BY TABLE_NAME;

-- =====================================================
-- 验证索引是否生效
-- =====================================================
SELECT '========================================' AS '';
SELECT '验证索引效果（EXPLAIN示例）' AS Status;
SELECT '========================================' AS '';

-- 示例1: 文章列表查询
EXPLAIN SELECT * FROM articles 
WHERE status = 1 
ORDER BY created_at DESC 
LIMIT 20;

-- 示例2: 评论查询
EXPLAIN SELECT * FROM article_comments 
WHERE article_id = 1 
  AND parent_id = 0 
  AND status = 1 
ORDER BY created_at DESC;

-- 示例3: 聊天消息查询
EXPLAIN SELECT * FROM chat_messages 
WHERE status = 1 
ORDER BY id DESC 
LIMIT 50;

SELECT '========================================' AS '';
SELECT '✓ 所有索引已成功创建！' AS Success;
SELECT '✓ 请检查上方EXPLAIN输出，确认查询使用了索引' AS Tip;
SELECT '✓ 在"key"列应该看到"idx_"开头的索引名' AS Tip;
SELECT '========================================' AS '';

-- =====================================================
-- 性能优化建议
-- =====================================================
SELECT '性能优化建议：' AS '';
SELECT '1. 定期运行 ANALYZE TABLE 更新统计信息' AS Recommendation;
SELECT '2. 监控慢查询日志，及时发现性能问题' AS Recommendation;
SELECT '3. 对于大表，考虑分区表策略' AS Recommendation;
SELECT '4. 定期清理无用数据，保持表体积合理' AS Recommendation;
SELECT '5. 使用 EXPLAIN 分析查询计划，确认索引使用' AS Recommendation;

-- 完成

