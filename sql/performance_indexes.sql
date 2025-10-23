-- =====================================================
-- 性能优化索引
-- =====================================================
-- 创建日期: 2025-10-20
-- 说明: 添加关键查询路径的复合索引，优化查询性能
-- MySQL版本要求: 5.7.7+
--
-- ⚠️ 如果遇到语法错误，请使用: create_indexes_simple.sql
-- ⚠️ 该版本使用 CREATE INDEX IF NOT EXISTS 语法（MySQL 5.7.7+支持）
--
-- 使用方法:
-- mysql -u root -p hub < performance_indexes.sql

-- 1. 文章表索引优化
-- =====================================================

-- 文章列表查询优化（按状态和创建时间排序）
-- 用于: GET /api/articles?sort_by=latest
CREATE INDEX IF NOT EXISTS idx_articles_status_created 
ON articles(status, created_at DESC);

-- 文章热度排序优化（按点赞数和浏览数）
-- 用于: GET /api/articles?sort_by=hot
CREATE INDEX IF NOT EXISTS idx_articles_likes_views 
ON articles(like_count DESC, view_count DESC, created_at DESC);

-- 用户文章列表查询优化
-- 用于: GET /api/articles?user_id=xxx
CREATE INDEX IF NOT EXISTS idx_articles_user_status_created
ON articles(user_id, status, created_at DESC);

-- 2. 评论表索引优化
-- =====================================================

-- 评论树查询优化（文章ID、父评论ID、状态、时间）
-- 用于: GET /api/articles/:id/comments
CREATE INDEX IF NOT EXISTS idx_comments_article_parent_status 
ON article_comments(article_id, parent_id, status, created_at);

-- 评论点赞查询优化
-- 用于: 检查用户是否点赞评论
CREATE INDEX IF NOT EXISTS idx_comment_likes_user 
ON article_comment_likes(user_id, comment_id);

-- 评论点赞查询优化（反向）
-- 用于: 批量查询评论的点赞状态
CREATE INDEX IF NOT EXISTS idx_comment_likes_comment 
ON article_comment_likes(comment_id, user_id);

-- 3. 聊天消息表索引优化
-- =====================================================

-- 聊天消息查询优化（状态和ID）
-- 用于: GET /api/chat/messages
CREATE INDEX IF NOT EXISTS idx_chat_status_id 
ON chat_messages(status, id DESC);

-- 获取新消息查询优化
-- 用于: GET /api/chat/messages/new?after_id=xxx
CREATE INDEX IF NOT EXISTS idx_chat_status_id_asc 
ON chat_messages(status, id ASC);

-- 用户消息查询优化
-- 用于: 删除消息时验证权限
CREATE INDEX IF NOT EXISTS idx_chat_user_id 
ON chat_messages(user_id, id DESC);

-- 4. 在线用户表索引优化
-- =====================================================

-- 在线用户心跳时间索引
-- 用于: GET /api/chat/online-count
CREATE INDEX IF NOT EXISTS idx_online_heartbeat 
ON online_users(last_heartbeat);

-- 5. 文章分类关系表索引优化
-- =====================================================

-- 分类查文章优化
-- 用于: GET /api/articles?category_id=xxx
CREATE INDEX IF NOT EXISTS idx_article_category_article 
ON article_category_relations(article_id, category_id);

-- 文章查分类优化
-- 用于: 批量获取文章的分类信息
CREATE INDEX IF NOT EXISTS idx_article_category_category 
ON article_category_relations(category_id, article_id);

-- 6. 文章标签关系表索引优化
-- =====================================================

-- 标签查文章优化
-- 用于: GET /api/articles?tag_id=xxx
CREATE INDEX IF NOT EXISTS idx_article_tag_article 
ON article_tag_relations(article_id, tag_id);

-- 文章查标签优化
-- 用于: 批量获取文章的标签信息
CREATE INDEX IF NOT EXISTS idx_article_tag_tag 
ON article_tag_relations(tag_id, article_id);

-- 7. 文章点赞表索引优化
-- =====================================================

-- 检查用户是否点赞文章
-- 用于: 获取文章详情时检查当前用户点赞状态
CREATE INDEX IF NOT EXISTS idx_article_likes_user 
ON article_likes(user_id, article_id);

-- 统计文章点赞用户
-- 用于: 批量查询文章点赞状态
CREATE INDEX IF NOT EXISTS idx_article_likes_article 
ON article_likes(article_id, user_id);

-- 8. 私信表索引优化（如果存在）
-- =====================================================

-- 会话消息查询优化
CREATE INDEX IF NOT EXISTS idx_private_messages_conversation 
ON private_messages(conversation_id, created_at DESC);

-- 用户会话查询优化
CREATE INDEX IF NOT EXISTS idx_private_messages_receiver 
ON private_messages(receiver_id, is_read, created_at DESC);

-- 9. 用户认证表索引优化
-- =====================================================

-- 邮箱查询优化（用于忘记密码等功能）
CREATE INDEX IF NOT EXISTS idx_user_auth_email 
ON user_auth(email);

-- =====================================================
-- 索引创建完成
-- =====================================================

-- 查看所有新创建的索引
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) AS COLUMNS,
    INDEX_TYPE
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
    'private_messages',
    'user_auth'
  )
GROUP BY TABLE_NAME, INDEX_NAME, INDEX_TYPE
ORDER BY TABLE_NAME, INDEX_NAME;

-- 使用说明：
-- 1. 在开发/测试环境执行此脚本前，先备份数据库
-- 2. 在生产环境执行前，选择低峰期执行
-- 3. 对于大表，索引创建可能需要较长时间
-- 4. 执行后使用 EXPLAIN 验证查询是否使用了新索引
-- 5. 监控索引使用情况，删除未使用的索引

-- 示例 EXPLAIN 查询：
-- EXPLAIN SELECT * FROM articles WHERE status = 1 ORDER BY created_at DESC LIMIT 20;
-- EXPLAIN SELECT * FROM chat_messages WHERE status = 1 AND id > 1000 ORDER BY id ASC LIMIT 50;

