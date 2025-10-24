# 数据库优化指南

## 📊 性能优化策略

### 1. 连接池优化

当前配置（`config.yaml`）：
```yaml
database:
  max_open_conns: 150      # 最大打开连接数
  max_idle_conns: 50       # 最大空闲连接数
  conn_max_lifetime: 30m   # 连接最大生命周期
  conn_max_idle_time: 5m   # 空闲连接超时
```

**优化建议：**
- **max_open_conns**: 根据服务器CPU核心数调整，建议 = CPU核心数 × 2 到 CPU核心数 × 4
- **max_idle_conns**: 设置为 max_open_conns 的 30-50%
- **conn_max_lifetime**: 设置为 30分钟，防止长时间连接导致的内存泄漏
- **conn_max_idle_time**: 5分钟，快速释放不活跃连接

### 2. 查询优化

#### 使用 Prepared Statements
```go
// ✅ 推荐：使用缓存的prepared statement
result, err := db.ExecWithCache(ctx, query, args...)

// ❌ 避免：每次都创建新的语句
result, err := db.DB.ExecContext(ctx, query, args...)
```

#### 避免 N+1 查询问题
```go
// ❌ 错误：N+1 查询
for _, userID := range userIDs {
    user, _ := GetUserByID(ctx, userID)  // 每次循环都查询一次数据库
}

// ✅ 正确：批量查询
users, _ := GetUsersByIDs(ctx, userIDs)  // 一次查询获取所有数据
```

#### 使用覆盖索引
```sql
-- ✅ 创建覆盖索引（包含所有查询列）
CREATE INDEX idx_article_list ON articles(category_id, created_at DESC, id, title);

-- 查询将不需要回表
SELECT id, title FROM articles WHERE category_id = 1 ORDER BY created_at DESC;
```

### 3. 索引优化

#### 索引创建原则
1. **高频查询字段**：WHERE、ORDER BY、JOIN 字段
2. **唯一性高的列**：区分度高的列优先索引
3. **复合索引顺序**：WHERE > ORDER BY > SELECT
4. **避免过度索引**：每个索引都占用空间，影响写入性能

#### 索引使用示例
```sql
-- 单列索引
CREATE INDEX idx_user_email ON user_auth(email);

-- 复合索引（左前缀原则）
CREATE INDEX idx_article_query ON articles(category_id, created_at DESC, status);

-- 可以使用以下查询：
-- WHERE category_id = ?
-- WHERE category_id = ? AND created_at > ?
-- WHERE category_id = ? AND created_at > ? AND status = ?

-- 不能使用索引的查询：
-- WHERE created_at > ?
-- WHERE status = ?
```

#### 查看索引使用情况
```sql
-- 查看未使用的索引
SELECT * FROM sys.schema_unused_indexes WHERE object_schema = 'hub';

-- 查看索引大小
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    ROUND(stat_value * @@innodb_page_size / 1024 / 1024, 2) AS size_mb
FROM mysql.innodb_index_stats
WHERE database_name = 'hub' AND stat_name = 'size'
ORDER BY size_mb DESC;
```

### 4. 查询性能分析

#### 使用 EXPLAIN 分析查询
```sql
EXPLAIN SELECT * FROM articles 
WHERE category_id = 1 
ORDER BY created_at DESC 
LIMIT 10;

-- 关注字段：
-- type: ALL(全表扫描) < index < range < ref < const
-- rows: 扫描行数越少越好
-- Extra: Using index（覆盖索引）最优
```

#### 慢查询监控
```sql
-- 启用慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 1;  -- 记录超过1秒的查询

-- 查看慢查询
SELECT * FROM mysql.slow_log 
ORDER BY query_time DESC 
LIMIT 10;
```

### 5. 批量操作优化

#### 批量插入
```go
// ✅ 使用批量插入
bp := utils.NewBatchProcessor(db.DB, 100)
values := [][]interface{}{
    {1, "user1", "email1@test.com"},
    {2, "user2", "email2@test.com"},
    // ... 更多数据
}
bp.BatchInsert(ctx, "users", []string{"id", "username", "email"}, values)

// ❌ 避免循环单条插入
for _, user := range users {
    db.Exec("INSERT INTO users ...")  // 慢！
}
```

#### 批量更新
```go
// ✅ 使用 IN 批量更新
UPDATE articles 
SET view_count = view_count + 1 
WHERE id IN (1, 2, 3, 4, 5);

// ❌ 避免多次单独更新
UPDATE articles SET view_count = view_count + 1 WHERE id = 1;
UPDATE articles SET view_count = view_count + 1 WHERE id = 2;
```

### 6. 事务优化

#### 事务大小控制
```go
// ✅ 小事务，快速提交
err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
    // 只包含必要的操作
    _, err := tx.ExecContext(ctx, "INSERT ...")
    return err
})

// ❌ 避免大事务锁定大量数据
tx.Begin()
for i := 0; i < 10000; i++ {
    tx.Exec(...)  // 长时间持有锁
}
tx.Commit()
```

### 7. 缓存策略

#### 多级缓存
```
1. 应用内存缓存（10秒-1分钟）
   └─> 热点数据、配置信息

2. Redis缓存（1分钟-1小时）
   └─> 用户信息、文章列表

3. 数据库查询缓存
   └─> 复杂聚合查询结果
```

#### 缓存实现
```go
// 使用带缓存的批量查询
result, err := bp.CachedBatchGet(
    ctx,
    cache,
    "user:",       // 缓存key前缀
    query,
    userIDs,
    5*time.Minute, // TTL
    scanFunc,
)
```

### 8. 数据库配置优化

#### MySQL配置建议（my.cnf）
```ini
[mysqld]
# InnoDB缓冲池（建议设置为总内存的70-80%）
innodb_buffer_pool_size = 2G

# InnoDB日志文件大小
innodb_log_file_size = 512M

# 查询缓存（MySQL 8.0已移除）
# query_cache_size = 256M
# query_cache_type = 1

# 最大连接数
max_connections = 500

# 排序缓冲区
sort_buffer_size = 2M

# 临时表大小
tmp_table_size = 64M
max_heap_table_size = 64M

# 线程缓存
thread_cache_size = 100

# 表打开缓存
table_open_cache = 4096
```

### 9. 监控和维护

#### 定期维护任务
```sql
-- 1. 分析表（更新统计信息）
ANALYZE TABLE articles, user_auth, chat_messages;

-- 2. 优化表（整理碎片）
OPTIMIZE TABLE articles, user_auth, chat_messages;

-- 3. 检查表碎片
SELECT 
    TABLE_NAME,
    ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) AS total_mb,
    ROUND(DATA_FREE / 1024 / 1024, 2) AS free_mb,
    ROUND(DATA_FREE / (DATA_LENGTH + INDEX_LENGTH) * 100, 2) AS fragmentation_pct
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'hub' AND DATA_FREE > 0
ORDER BY fragmentation_pct DESC;
```

#### 关键指标监控
```go
// 监控连接池状态
stats := db.GetStats()
logger.Info("数据库连接池状态",
    "openConnections", stats.OpenConnections,
    "inUse", stats.InUse,
    "idle", stats.Idle,
    "waitCount", stats.WaitCount,
    "waitDuration", stats.WaitDuration)
```

### 10. 查询优化清单

- [ ] 所有WHERE子句字段都有索引
- [ ] 复合索引遵循左前缀原则
- [ ] 避免SELECT *，只查询需要的列
- [ ] 使用LIMIT限制返回行数
- [ ] 避免在WHERE中使用函数
- [ ] 使用JOIN代替子查询
- [ ] 使用EXISTS代替IN（大数据集）
- [ ] 批量操作代替循环单条操作
- [ ] 适当使用缓存减少查询
- [ ] 定期ANALYZE TABLE更新统计信息

## 📈 性能基准测试

### 测试场景
1. **单条查询**: < 10ms
2. **批量查询(100条)**: < 50ms
3. **复杂JOIN**: < 100ms
4. **批量插入(1000条)**: < 500ms
5. **事务提交**: < 20ms

### 压测工具
- `ab` (Apache Bench)
- `wrk`
- `hey`
- `JMeter`

```bash
# 使用 wrk 压测
wrk -t12 -c400 -d30s http://localhost:3001/api/articles

# 使用 ab 压测
ab -n 10000 -c 100 http://localhost:3001/api/articles
```

## 🔧 故障排查

### 慢查询排查
```sql
-- 查看当前正在执行的查询
SHOW FULL PROCESSLIST;

-- 杀死长时间运行的查询
KILL <process_id>;

-- 查看锁等待
SELECT * FROM sys.innodb_lock_waits;
```

### 连接数过多
```sql
-- 查看当前连接数
SHOW STATUS LIKE 'Threads_connected';

-- 查看最大连接数
SHOW VARIABLES LIKE 'max_connections';

-- 查看连接详情
SELECT * FROM information_schema.PROCESSLIST;
```

## 📚 参考资源

- [MySQL官方性能优化指南](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [High Performance MySQL](https://www.oreilly.com/library/view/high-performance-mysql/9781449332471/)
- [Percona Toolkit](https://www.percona.com/software/database-tools/percona-toolkit)
- [MySQL Slow Query Analyzer](https://github.com/box/Anemometer)

