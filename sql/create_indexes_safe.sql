-- =====================================================
-- 安全创建索引脚本（兼容所有MySQL版本）
-- =====================================================
-- 说明：此脚本会检查索引是否存在，如果不存在才创建
-- 适用于任何 MySQL 5.x 及以上版本
-- =====================================================

USE hub;

DELIMITER $$

-- 创建安全索引创建存储过程
DROP PROCEDURE IF EXISTS CreateIndexIfNotExists$$
CREATE PROCEDURE CreateIndexIfNotExists(
    IN tableName VARCHAR(128),
    IN indexName VARCHAR(128),
    IN indexColumns TEXT
)
BEGIN
    DECLARE indexExists INT DEFAULT 0;
    
    -- 检查索引是否存在
    SELECT COUNT(*) INTO indexExists
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = tableName
      AND INDEX_NAME = indexName;
    
    -- 如果索引不存在，则创建
    IF indexExists = 0 THEN
        SET @sql = CONCAT('CREATE INDEX ', indexName, ' ON ', tableName, '(', indexColumns, ')');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        SELECT CONCAT('✓ 索引 ', indexName, ' 创建成功') AS Result;
    ELSE
        SELECT CONCAT('- 索引 ', indexName, ' 已存在，跳过') AS Result;
    END IF;
END$$

DELIMITER ;

-- =====================================================
-- 开始创建性能优化索引
-- =====================================================

-- 文章表性能索引
CALL CreateIndexIfNotExists('articles', 'idx_articles_status_created', 'status, created_at DESC');
CALL CreateIndexIfNotExists('articles', 'idx_articles_likes_views', 'like_count DESC, view_count DESC, created_at DESC');
CALL CreateIndexIfNotExists('articles', 'idx_articles_user_status_created', 'user_id, status, created_at DESC');

-- 评论表性能索引
CALL CreateIndexIfNotExists('article_comments', 'idx_comments_article_parent_status', 'article_id, parent_id, status, created_at');
CALL CreateIndexIfNotExists('article_comment_likes', 'idx_comment_likes_user', 'user_id, comment_id');
CALL CreateIndexIfNotExists('article_comment_likes', 'idx_comment_likes_comment', 'comment_id, user_id');

-- 聊天消息表性能索引
CALL CreateIndexIfNotExists('chat_messages', 'idx_chat_status_id', 'status, id DESC');
CALL CreateIndexIfNotExists('chat_messages', 'idx_chat_status_id_asc', 'status, id ASC');
CALL CreateIndexIfNotExists('chat_messages', 'idx_chat_user_id', 'user_id, id DESC');

-- 在线用户表性能索引
CALL CreateIndexIfNotExists('online_users', 'idx_online_heartbeat', 'last_heartbeat');

-- 文章分类关系表性能索引
CALL CreateIndexIfNotExists('article_category_relations', 'idx_article_category_article', 'article_id, category_id');
CALL CreateIndexIfNotExists('article_category_relations', 'idx_article_category_category', 'category_id, article_id');

-- 文章标签关系表性能索引
CALL CreateIndexIfNotExists('article_tag_relations', 'idx_article_tag_article', 'article_id, tag_id');
CALL CreateIndexIfNotExists('article_tag_relations', 'idx_article_tag_tag', 'tag_id, article_id');

-- 文章点赞表性能索引
CALL CreateIndexIfNotExists('article_likes', 'idx_article_likes_user', 'user_id, article_id');
CALL CreateIndexIfNotExists('article_likes', 'idx_article_likes_article', 'article_id, user_id');

-- 私信表性能索引
CALL CreateIndexIfNotExists('private_messages', 'idx_private_messages_conversation', 'conversation_id, created_at DESC');
CALL CreateIndexIfNotExists('private_messages', 'idx_private_messages_receiver', 'receiver_id, is_read, created_at DESC');

-- =====================================================
-- 清理临时存储过程
-- =====================================================
DROP PROCEDURE IF EXISTS CreateIndexIfNotExists;

-- =====================================================
-- 验证已创建的索引
-- =====================================================
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) AS COLUMNS
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
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
    'private_messages'
  )
GROUP BY TABLE_NAME, INDEX_NAME
ORDER BY TABLE_NAME, INDEX_NAME;

SELECT '✓ 所有性能索引创建完成！' AS Status;

