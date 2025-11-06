-- =====================================================
-- 安全添加性能优化索引
-- =====================================================
-- 用途: 检查索引是否存在，不存在才创建（避免重复键错误）
-- 执行: mysql -u root -p hub < add_indexes_safely.sql
-- =====================================================

USE hub;

-- ========================================
-- 辅助函数：安全创建索引
-- ========================================

DELIMITER $$

DROP PROCEDURE IF EXISTS add_index_if_not_exists$$
CREATE PROCEDURE add_index_if_not_exists(
    IN p_table_name VARCHAR(64),
    IN p_index_name VARCHAR(64),
    IN p_index_definition TEXT
)
BEGIN
    DECLARE index_count INT;
    
    -- 检查索引是否存在
    SELECT COUNT(*) INTO index_count
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
        AND table_name = p_table_name
        AND index_name = p_index_name;
    
    -- 如果索引不存在，则创建
    IF index_count = 0 THEN
        SET @sql = CONCAT('CREATE INDEX ', p_index_name, ' ON ', p_table_name, ' ', p_index_definition);
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        SELECT CONCAT('✓ 索引 ', p_index_name, ' 创建成功') AS result;
    ELSE
        SELECT CONCAT('⚠ 索引 ', p_index_name, ' 已存在，跳过') AS result;
    END IF;
END$$

DELIMITER ;

-- ========================================
-- 用户认证表 (user_auth) 索引
-- ========================================
SELECT '正在为 user_auth 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'user_auth',
    'idx_account_status_login',
    '(account_status, last_login_time DESC) COMMENT ''优化活跃用户查询'''
);

CALL add_index_if_not_exists(
    'user_auth',
    'idx_failed_login_monitoring',
    '(failed_login_count, last_login_time) COMMENT ''监控异常登录'''
);

-- ========================================
-- 用户资料表 (user_profile) 索引
-- ========================================
SELECT '正在为 user_profile 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'user_profile',
    'idx_profile_updated',
    '(user_id, updated_at DESC) COMMENT ''用户资料更新查询'''
);

-- ========================================
-- 登录历史表 (user_login_history) 索引
-- ========================================
SELECT '正在为 user_login_history 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'user_login_history',
    'idx_login_user_time',
    '(user_id, login_time DESC) COMMENT ''用户登录历史查询'''
);

CALL add_index_if_not_exists(
    'user_login_history',
    'idx_login_ip_time',
    '(login_ip, login_time DESC) COMMENT ''IP登录历史查询'''
);

CALL add_index_if_not_exists(
    'user_login_history',
    'idx_login_location',
    '(province, city, login_time DESC) COMMENT ''地理位置分析'''
);

CALL add_index_if_not_exists(
    'user_login_history',
    'idx_login_user_status_time',
    '(user_id, login_status, login_time DESC) COMMENT ''用户登录状态查询'''
);

-- ========================================
-- 聊天消息表 (chat_messages) 索引
-- ========================================
SELECT '正在为 chat_messages 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'chat_messages',
    'idx_chat_time_id',
    '(created_at DESC, id DESC) COMMENT ''聊天消息时间查询'''
);

CALL add_index_if_not_exists(
    'chat_messages',
    'idx_chat_user_time',
    '(user_id, created_at DESC) COMMENT ''用户消息查询'''
);

-- ========================================
-- 文章表 (articles) 索引
-- ========================================
SELECT '正在为 articles 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'articles',
    'idx_article_user_time',
    '(user_id, created_at DESC) COMMENT ''用户文章时间查询'''
);

CALL add_index_if_not_exists(
    'articles',
    'idx_article_status_time',
    '(status, created_at DESC) COMMENT ''文章状态时间查询'''
);

CALL add_index_if_not_exists(
    'articles',
    'idx_article_hot_sort',
    '(like_count DESC, view_count DESC, comment_count DESC) COMMENT ''热门文章排序'''
);

-- ========================================
-- 文章分类关联表 (article_category_relations)
-- ========================================
SELECT '正在为 article_category_relations 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'article_category_relations',
    'idx_category_article_time',
    '(category_id, article_id) COMMENT ''分类文章查询'''
);

-- ========================================
-- 文章标签关联表 (article_tag_relations)
-- ========================================
SELECT '正在为 article_tag_relations 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'article_tag_relations',
    'idx_tag_article',
    '(tag_id, article_id) COMMENT ''标签文章查询'''
);

-- ========================================
-- 文章评论表 (article_comments) 索引
-- ========================================
SELECT '正在为 article_comments 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'article_comments',
    'idx_comment_article_time',
    '(article_id, created_at DESC) COMMENT ''文章评论时间查询'''
);

CALL add_index_if_not_exists(
    'article_comments',
    'idx_comment_user_time',
    '(user_id, created_at DESC) COMMENT ''用户评论查询'''
);

CALL add_index_if_not_exists(
    'article_comments',
    'idx_comment_parent',
    '(parent_id, created_at DESC) COMMENT ''父评论查询'''
);

-- ========================================
-- 资源表 (resources) 索引
-- ========================================
SELECT '正在为 resources 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'resources',
    'idx_resource_category_time',
    '(category_id, created_at DESC) COMMENT ''资源分类时间查询'''
);

CALL add_index_if_not_exists(
    'resources',
    'idx_resource_user_time',
    '(user_id, created_at DESC) COMMENT ''用户资源查询'''
);

CALL add_index_if_not_exists(
    'resources',
    'idx_resource_status_time',
    '(status, created_at DESC) COMMENT ''资源审核状态查询'''
);

CALL add_index_if_not_exists(
    'resources',
    'idx_resource_hot',
    '(download_count DESC, like_count DESC) COMMENT ''热门资源排序'''
);

CALL add_index_if_not_exists(
    'resources',
    'idx_resource_file_type',
    '(file_type, created_at DESC) COMMENT ''文件类型查询'''
);

-- ========================================
-- 代码片段表 (code_snippets) 索引
-- ========================================
SELECT '正在为 code_snippets 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'code_snippets',
    'idx_snippet_user_time',
    '(user_id, created_at DESC) COMMENT ''用户代码片段查询'''
);

CALL add_index_if_not_exists(
    'code_snippets',
    'idx_snippet_language_time',
    '(language, created_at DESC) COMMENT ''语言代码片段查询'''
);

CALL add_index_if_not_exists(
    'code_snippets',
    'idx_snippet_public_time',
    '(is_public, created_at DESC) COMMENT ''公开代码片段查询'''
);

-- ========================================
-- 私信表 (private_messages) 索引
-- ========================================
SELECT '正在为 private_messages 表创建索引...' AS status;

CALL add_index_if_not_exists(
    'private_messages',
    'idx_message_sender_receiver_time',
    '(sender_id, receiver_id, created_at DESC) COMMENT ''发送者接收者时间查询'''
);

CALL add_index_if_not_exists(
    'private_messages',
    'idx_message_receiver_sender_time',
    '(receiver_id, sender_id, created_at DESC) COMMENT ''接收者发送者时间查询'''
);

CALL add_index_if_not_exists(
    'private_messages',
    'idx_message_unread',
    '(receiver_id, is_read, created_at DESC) COMMENT ''未读消息查询'''
);

-- ========================================
-- 清理辅助存储过程
-- ========================================
DROP PROCEDURE IF EXISTS add_index_if_not_exists;

-- ========================================
-- 完成
-- ========================================
SELECT '========================================' AS '';
SELECT '✓ 性能索引创建完成！' AS Success;
SELECT '========================================' AS '';

-- 显示创建的索引统计
SELECT 
    TABLE_NAME AS '表名',
    COUNT(DISTINCT INDEX_NAME) AS '索引数量'
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'hub'
    AND TABLE_NAME IN (
        'user_auth', 'user_profile', 'user_login_history',
        'chat_messages', 'articles', 'article_category_relations',
        'article_tag_relations', 'article_comments', 'resources',
        'code_snippets', 'private_messages'
    )
GROUP BY TABLE_NAME
ORDER BY TABLE_NAME;

