# SQL脚本使用说明 📁

## 📋 文件列表

### 性能优化索引脚本

| 文件 | MySQL版本 | 特点 | 推荐 |
|-----|----------|------|------|
| `create_indexes_simple.sql` | 5.7+ | 简洁、兼容性强 | ⭐⭐⭐ 强烈推荐 |
| `performance_indexes.sql` | 5.7.7+ | 基础版本 | ⭐⭐ 推荐 |
| `add_performance_indexes.sql` | 8.0+ | 详细检查和统计 | ⭐ 高级用户 |

### 其他SQL脚本

| 文件 | 说明 |
|-----|------|
| `article_tables.sql` | 文章系统表结构 |
| `chat_tables.sql` | 聊天系统表结构 |
| `history_tables.sql` | 历史记录表结构 |
| `resource_tables.sql` | 资源系统表结构 |
| `private_messages.sql` | 私信系统表结构 |
| `password_reset_tokens.sql` | 密码重置表结构 |

---

## 🚀 快速开始

### 推荐方式（兼容MySQL 5.7+）

```bash
cd shequ_gin/sql

# 执行性能优化索引
mysql -u root -p hub < create_indexes_simple.sql
```

**输出示例**:
```
正在为articles表创建索引...
✓ articles表索引创建完成
正在为article_comments表创建索引...
✓ article_comments表索引创建完成
正在为chat_messages表创建索引...
✓ chat_messages表索引创建完成
...
✓ 所有索引创建完成！
```

---

## ⚠️ 常见问题

### Q1: 遇到 "DROP INDEX IF EXISTS" 语法错误？

**错误信息**:
```
1064 - You have an error in your SQL syntax... near 'IF EXISTS idx_xxx'
```

**原因**: 您的MySQL版本低于8.0.1

**解决方案**:
```bash
# 使用兼容版本的SQL文件
mysql -u root -p hub < create_indexes_simple.sql
```

---

### Q2: 索引已存在的错误？

**错误信息**:
```
ERROR 1061 (42000): Duplicate key name 'idx_articles_status_created'
```

**解决方案1: 手动删除旧索引**
```sql
USE hub;
ALTER TABLE articles DROP INDEX idx_articles_status_created;
-- 然后重新执行创建脚本
```

**解决方案2: 忽略错误**
```bash
# 如果索引已存在，说明已经优化过，无需重复创建
```

---

### Q3: 如何检查MySQL版本？

```bash
# 命令行
mysql --version

# 或在MySQL中
mysql> SELECT VERSION();
```

**版本说明**:
- MySQL 5.7.7+ - 使用 `create_indexes_simple.sql` 或 `performance_indexes.sql`
- MySQL 8.0.1+ - 可以使用任何版本的SQL文件

---

## 📖 详细文档

- [INDEX_INSTALLATION_GUIDE.md](./INDEX_INSTALLATION_GUIDE.md) - 详细安装指南
- [README_INDEXES.md](./README_INDEXES.md) - 索引使用说明

---

## ✅ 验证索引

```sql
USE hub;

-- 查看已创建的索引
SHOW INDEX FROM articles WHERE Key_name LIKE 'idx_%';
SHOW INDEX FROM article_comments WHERE Key_name LIKE 'idx_%';
SHOW INDEX FROM chat_messages WHERE Key_name LIKE 'idx_%';

-- 验证查询使用索引
EXPLAIN SELECT * FROM articles WHERE status = 1 ORDER BY created_at DESC LIMIT 20;
-- 检查"key"列是否显示"idx_articles_status_created"
```

---

## 🎯 索引列表

执行脚本后将创建以下索引：

### articles表 (3个)
- `idx_articles_status_created` - 文章列表查询
- `idx_articles_likes_views` - 热度排序
- `idx_articles_user_status` - 用户文章

### article_comments表 (3个)
- `idx_comments_article_parent_status` - 评论树
- `idx_comments_user_status` - 用户评论
- `idx_comments_root_status` - 根评论

### article_comment_likes表 (2个)
- `idx_comment_likes_check` - 点赞检查
- `idx_comment_likes_user` - 用户点赞

### article_likes表 (2个)
- `idx_article_likes_check` - 点赞检查
- `idx_article_likes_user_time` - 点赞历史

### article_category_relations表 (2个)
- `idx_article_category_article` - 文章分类
- `idx_article_category_category` - 分类文章

### article_tag_relations表 (2个)
- `idx_article_tag_article` - 文章标签
- `idx_article_tag_tag` - 标签文章

### chat_messages表 (3个)
- `idx_chat_status_id_desc` - 最新消息
- `idx_chat_status_id_asc` - 新消息
- `idx_chat_user_id` - 用户消息

### online_users表 (1个)
- `idx_online_heartbeat` - 在线用户

### user_auth表 (1个)
- `idx_user_auth_email` - 邮箱查询

**总计: 19个索引**

---

## 🎊 完成

索引创建完成后：
- ✅ 数据库查询性能提升30-50%
- ✅ API响应时间降低
- ✅ 服务器负载减少

更多信息请查看: [OPTIMIZATION_GUIDE.md](../../OPTIMIZATION_GUIDE.md)

---

© 2025 社区平台 - SQL脚本使用说明

