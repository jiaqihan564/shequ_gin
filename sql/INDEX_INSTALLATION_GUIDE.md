# 数据库索引安装指南

## ⚠️ MySQL版本兼容性问题

如果执行SQL脚本时遇到以下错误：
```
1064 - You have an error in your SQL syntax... near 'IF EXISTS idx_xxx ON table_name'
```

**原因**: `DROP INDEX IF EXISTS` 语法仅在 MySQL 8.0.1+ 版本支持。

---

## 🚀 解决方案

### 方案1: 使用简化版脚本（推荐）⭐

```bash
# 使用兼容MySQL 5.7+的简化版本
mysql -u root -p hub < create_indexes_simple.sql
```

**特点**:
- ✅ 兼容MySQL 5.7+
- ✅ 自动忽略不存在的索引
- ✅ 快速执行
- ✅ 无需检查

---

### 方案2: 手动删除旧索引（如果索引已存在）

```sql
USE hub;

-- 查看现有索引
SHOW INDEX FROM articles WHERE Key_name LIKE 'idx_%';

-- 手动删除旧索引（如果存在）
ALTER TABLE articles DROP INDEX idx_articles_status_created;
ALTER TABLE articles DROP INDEX idx_articles_likes_views;
ALTER TABLE articles DROP INDEX idx_articles_user_status;
-- 继续删除其他表的索引...

-- 然后执行创建脚本
SOURCE create_indexes_simple.sql;
```

---

### 方案3: 升级MySQL版本

如果可能，升级到MySQL 8.0+以获得更好的性能和特性支持。

```bash
# 检查当前MySQL版本
mysql --version

# 或在MySQL中
SELECT VERSION();
```

---

## 📋 推荐的执行步骤

### 步骤1: 备份数据库（必需）

```bash
mysqldump -u root -p hub > hub_backup_$(date +%Y%m%d_%H%M%S).sql
```

### 步骤2: 选择合适的SQL文件

| 文件 | MySQL版本 | 特点 |
|-----|----------|------|
| `create_indexes_simple.sql` | 5.7+ | ⭐ 推荐，简洁兼容 |
| `performance_indexes.sql` | 5.7+ | 基础版本 |
| `add_performance_indexes.sql` | 8.0+ | 详细版（可能有兼容问题） |

### 步骤3: 执行SQL脚本

```bash
# 推荐：使用简化版
cd shequ_gin/sql
mysql -u root -p hub < create_indexes_simple.sql
```

**预期输出**:
```
正在为articles表创建索引...
✓ articles表索引创建完成
正在为article_comments表创建索引...
✓ article_comments表索引创建完成
...
✓ 所有索引创建完成！
```

### 步骤4: 验证索引

```sql
-- 查看所有创建的索引
SELECT 
    TABLE_NAME AS '表名',
    INDEX_NAME AS '索引名',
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ', ') AS '索引列'
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'hub'
  AND INDEX_NAME LIKE 'idx_%'
GROUP BY TABLE_NAME, INDEX_NAME;

-- 验证查询使用索引
EXPLAIN SELECT * FROM articles WHERE status = 1 ORDER BY created_at DESC LIMIT 20;
-- 检查"key"列应该显示"idx_articles_status_created"
```

---

## ❌ 如果出现错误

### 错误1: "Duplicate key name"

**错误信息**:
```
ERROR 1061 (42000): Duplicate key name 'idx_articles_status_created'
```

**解决**:
```sql
-- 索引已存在，删除后重新创建
ALTER TABLE articles DROP INDEX idx_articles_status_created;
CREATE INDEX idx_articles_status_created ON articles(status, created_at DESC);
```

**或者**: 跳过这个索引（已经存在说明已优化）

---

### 错误2: "Can't DROP 'idx_xxx'; check that column/key exists"

**解决**:
```sql
-- 忽略这个错误，索引本来就不存在
-- 继续执行后面的CREATE INDEX语句
```

---

### 错误3: 表不存在

**错误信息**:
```
ERROR 1146 (42S02): Table 'hub.articles' doesn't exist
```

**解决**:
1. 检查是否使用了正确的数据库：
   ```sql
   SELECT DATABASE();  -- 应该显示'hub'
   USE hub;
   ```

2. 检查表是否存在：
   ```sql
   SHOW TABLES LIKE 'articles';
   ```

3. 如果表不存在，先创建表，再创建索引

---

## 🎯 快速命令汇总

```bash
# 1. 备份数据库
mysqldump -u root -p hub > backup.sql

# 2. 执行索引脚本（推荐）
mysql -u root -p hub < create_indexes_simple.sql

# 3. 验证索引
mysql -u root -p hub -e "SHOW INDEX FROM articles WHERE Key_name LIKE 'idx_%';"

# 4. 测试查询性能
mysql -u root -p hub -e "EXPLAIN SELECT * FROM articles WHERE status = 1 ORDER BY created_at DESC LIMIT 20;"
```

---

## 📊 索引列表

执行脚本后，将创建以下索引：

### articles表 (3个)
- `idx_articles_status_created` - 文章列表查询
- `idx_articles_likes_views` - 热度排序
- `idx_articles_user_status` - 用户文章列表

### article_comments表 (3个)
- `idx_comments_article_parent_status` - 评论树查询
- `idx_comments_user_status` - 用户评论查询
- `idx_comments_root_status` - 根评论查询

### article_comment_likes表 (2个)
- `idx_comment_likes_check` - 点赞状态检查
- `idx_comment_likes_user` - 用户点赞查询

### article_likes表 (2个)
- `idx_article_likes_check` - 点赞状态检查
- `idx_article_likes_user_time` - 用户点赞历史

### article_category_relations表 (2个)
- `idx_article_category_article` - 文章分类关系
- `idx_article_category_category` - 分类文章关系

### article_tag_relations表 (2个)
- `idx_article_tag_article` - 文章标签关系
- `idx_article_tag_tag` - 标签文章关系

### chat_messages表 (3个)
- `idx_chat_status_id_desc` - 最新消息查询
- `idx_chat_status_id_asc` - 新消息查询
- `idx_chat_user_id` - 用户消息查询

### online_users表 (1个)
- `idx_online_heartbeat` - 在线用户查询

### user_auth表 (1个)
- `idx_user_auth_email` - 邮箱查询

**总计**: 19个索引

---

## ✅ 验证清单

执行完成后检查：

- [ ] 执行无报错（可以忽略"索引不存在"的警告）
- [ ] 使用 `SHOW INDEX` 确认索引已创建
- [ ] 使用 `EXPLAIN` 确认查询使用索引
- [ ] 重启后端服务
- [ ] 测试API响应时间是否降低

---

## 🎉 完成

如果所有步骤顺利完成，你应该看到：

- ✅ 19个索引成功创建
- ✅ 查询使用索引（EXPLAIN显示）
- ✅ API响应时间降低30-50%
- ✅ 数据库负载降低

**恭喜！索引优化完成！** 🎊

---

## 📞 需要帮助？

- 查看详细文档: [README_INDEXES.md](./README_INDEXES.md)
- 查看优化指南: [OPTIMIZATION_GUIDE.md](../OPTIMIZATION_GUIDE.md)
- 查看快速开始: [QUICK_START_OPTIMIZATION.md](../QUICK_START_OPTIMIZATION.md)

---

© 2025 社区平台 - 数据库索引安装指南

