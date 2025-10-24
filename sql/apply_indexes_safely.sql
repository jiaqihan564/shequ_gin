-- ========================================
-- 安全应用索引脚本
-- ========================================
-- 本脚本会检查索引是否存在，只创建不存在的索引
-- 使用方法：mysql -u root -p hub < apply_indexes_safely.sql
-- ========================================

USE hub;

-- 创建临时存储过程来安全创建索引
DELIMITER $$

DROP PROCEDURE IF EXISTS CreateIndexIfNotExists$$
CREATE PROCEDURE CreateIndexIfNotExists(
    IN tableName VARCHAR(128),
    IN indexName VARCHAR(128),
    IN indexColumns TEXT,
    IN indexComment TEXT,
    IN isUnique BOOLEAN
)
BEGIN
    DECLARE tableExists INT DEFAULT 0;
    DECLARE indexExists INT DEFAULT 0;
    
    -- 先检查表是否存在
    SELECT COUNT(*) INTO tableExists
    FROM information_schema.TABLES
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = tableName;
    
    IF tableExists = 0 THEN
        SELECT CONCAT('⚠ 表不存在，跳过: ', tableName) AS Result;
    ELSE
        -- 检查索引是否存在
        SELECT COUNT(*) INTO indexExists
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = tableName
          AND INDEX_NAME = indexName;
        
        -- 如果索引不存在，创建它
        IF indexExists = 0 THEN
            SET @sql = CONCAT(
                'CREATE ',
                IF(isUnique, 'UNIQUE ', ''),
                'INDEX ', indexName,
                ' ON ', tableName,
                '(', indexColumns, ')',
                ' COMMENT ''', indexComment, ''''
            );
            
            PREPARE stmt FROM @sql;
            EXECUTE stmt;
            DEALLOCATE PREPARE stmt;
            
            SELECT CONCAT('✓ 创建索引: ', indexName, ' on ', tableName) AS Result;
        ELSE
            SELECT CONCAT('⊙ 索引已存在: ', indexName, ' on ', tableName) AS Result;
        END IF;
    END IF;
END$$

DELIMITER ;

-- ========================================
-- 应用所有索引
-- ========================================

-- 用户认证表
CALL CreateIndexIfNotExists('user_auth', 'idx_account_status_login', 'account_status, last_login_time DESC', '优化活跃用户查询', FALSE);
CALL CreateIndexIfNotExists('user_auth', 'idx_failed_login_monitoring', 'failed_login_count, last_login_time', '监控异常登录', FALSE);

-- 用户资料表
CALL CreateIndexIfNotExists('user_profile', 'idx_profile_updated', 'user_id, updated_at DESC', '用户资料更新查询', FALSE);

-- 登录历史表
CALL CreateIndexIfNotExists('login_history', 'idx_login_user_time', 'user_id, login_time DESC', '用户登录历史查询', FALSE);
CALL CreateIndexIfNotExists('login_history', 'idx_login_ip_time', 'login_ip, login_time DESC', 'IP登录历史查询', FALSE);
CALL CreateIndexIfNotExists('login_history', 'idx_login_location', 'province, city, login_time DESC', '地理位置分析', FALSE);
CALL CreateIndexIfNotExists('login_history', 'idx_login_user_status_time', 'user_id, login_status, login_time DESC', '用户登录状态查询', FALSE);

-- 聊天消息表
CALL CreateIndexIfNotExists('chat_messages', 'idx_chat_time_id', 'created_at DESC, id DESC', '聊天消息时间查询', FALSE);
CALL CreateIndexIfNotExists('chat_messages', 'idx_chat_user_time', 'user_id, created_at DESC', '用户消息查询', FALSE);

-- 文章表（注意：articles表本身已有基础索引）
CALL CreateIndexIfNotExists('articles', 'idx_article_user_time', 'user_id, created_at DESC', '用户文章查询', FALSE);
CALL CreateIndexIfNotExists('articles', 'idx_article_status_time', 'status, created_at DESC', '文章状态查询', FALSE);
-- 热门文章索引（idx_hot 已存在于表定义中）

-- 文章分类关联表
CALL CreateIndexIfNotExists('article_category_relations', 'idx_category_article_time', 'category_id, article_id', '分类文章查询', FALSE);

-- 文章标签关联表
CALL CreateIndexIfNotExists('article_tag_relations', 'idx_tag_article', 'tag_id, article_id', '标签文章查询', FALSE);

-- 评论表
CALL CreateIndexIfNotExists('article_comments', 'idx_comment_article_time', 'article_id, created_at DESC', '文章评论查询', FALSE);
CALL CreateIndexIfNotExists('article_comments', 'idx_comment_user_time', 'user_id, created_at DESC', '用户评论查询', FALSE);
CALL CreateIndexIfNotExists('article_comments', 'idx_comment_parent', 'parent_id, created_at DESC', '子评论查询', FALSE);

-- 资源表（已有基础索引：idx_user_id, idx_category, idx_status, idx_created_at）
CALL CreateIndexIfNotExists('resources', 'idx_resource_category_time', 'category_id, created_at DESC', '分类资源查询', FALSE);
CALL CreateIndexIfNotExists('resources', 'idx_resource_user_time', 'user_id, created_at DESC', '用户资源查询', FALSE);
CALL CreateIndexIfNotExists('resources', 'idx_resource_status_time', 'status, created_at DESC', '资源审核查询', FALSE);
CALL CreateIndexIfNotExists('resources', 'idx_resource_hot', 'download_count DESC, like_count DESC', '热门资源排序', FALSE);
CALL CreateIndexIfNotExists('resources', 'idx_resource_file_type', 'file_type, created_at DESC', '文件类型查询', FALSE);

-- 代码片段表
CALL CreateIndexIfNotExists('code_snippets', 'idx_snippet_user_time', 'user_id, created_at DESC', '用户代码片段查询', FALSE);
CALL CreateIndexIfNotExists('code_snippets', 'idx_snippet_language_time', 'language, created_at DESC', '语言代码片段查询', FALSE);
CALL CreateIndexIfNotExists('code_snippets', 'idx_snippet_public_time', 'is_public, created_at DESC', '公开代码片段查询', FALSE);
CALL CreateIndexIfNotExists('code_snippets', 'idx_snippet_share_token', 'share_token', '分享令牌查询', TRUE);

-- 私信表
CALL CreateIndexIfNotExists('private_messages', 'idx_message_sender_receiver_time', 'sender_id, receiver_id, created_at DESC', '发送消息查询', FALSE);
CALL CreateIndexIfNotExists('private_messages', 'idx_message_receiver_sender_time', 'receiver_id, sender_id, created_at DESC', '接收消息查询', FALSE);
CALL CreateIndexIfNotExists('private_messages', 'idx_message_unread', 'receiver_id, is_read, created_at DESC', '未读消息查询', FALSE);

-- 统计表
CALL CreateIndexIfNotExists('statistics', 'idx_stats_date', 'stat_date DESC', '日期统计查询', FALSE);
CALL CreateIndexIfNotExists('statistics', 'idx_stats_api_date', 'api_path, stat_date DESC', 'API统计查询', FALSE);

-- 累计统计表
CALL CreateIndexIfNotExists('cumulative_stats', 'idx_cumulative_date', 'stat_date DESC', '累计统计日期查询', FALSE);

-- 清理存储过程
DROP PROCEDURE IF EXISTS CreateIndexIfNotExists;

SELECT '========================================' AS '';
SELECT '✓ 索引创建完成！' AS '';
SELECT '========================================' AS '';
SELECT '请运行以下命令查看索引状态：' AS '';
SELECT 'SHOW INDEX FROM table_name;' AS '';
SELECT '========================================' AS '';

