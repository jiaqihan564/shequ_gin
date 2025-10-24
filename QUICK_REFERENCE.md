# 后端性能优化快速参考

## 🚀 性能优化原则（记住这些！）

### 1. 数据库查询三原则
```go
// ❌ 错误示例
for _, id := range ids {
    user := GetUser(id)  // N+1查询
}

// ✅ 正确示例
users := BatchGetUsers(ids)  // 1次批量查询
```

### 2. 批量操作原则
```go
// ❌ 错误示例
for _, item := range items {
    db.Exec("INSERT ...")  // 循环插入
}

// ✅ 正确示例
INSERT INTO table VALUES (?, ?), (?, ?), ...  // 批量插入
```

### 3. 并发控制原则
```go
// ❌ 错误示例
go func() {
    doSomething()  // 裸goroutine，可能泄漏
}()

// ✅ 正确示例
utils.SubmitTask("task_id", func(ctx context.Context) error {
    return doSomething()
}, 5*time.Second)
```

### 4. 缓存使用原则
```go
// ✅ 热点数据必须缓存
categories := cacheSvc.GetArticleCategories(ctx)  // 缓存1小时

// ✅ 合理设置TTL
- 静态数据（分类、标签）：30分钟-1小时
- 动态数据（文章列表）：2-5分钟  
- 实时数据（在线人数）：10秒
```

### 5. 对象池原则
```go
// ✅ 大量临时对象使用对象池
buf := utils.GetBuffer()
defer utils.PutBuffer(buf)
// ... 使用buf ...
```

## 📋 常用代码片段

### 批量插入
```go
values := []string{}
args := []interface{}{}
for _, item := range items {
    values = append(values, "(?, ?, ?)")
    args = append(args, item.A, item.B, item.C)
}
query := "INSERT INTO table (a, b, c) VALUES " + strings.Join(values, ", ")
db.ExecContext(ctx, query, args...)
```

### 并行查询
```go
countChan := make(chan int, 1)
rowsChan := make(chan *sql.Rows, 1)

go func() {
    var count int
    db.QueryRow(countQuery).Scan(&count)
    countChan <- count
}()

go func() {
    rows, _ := db.Query(listQuery)
    rowsChan <- rows
}()

count := <-countChan
rows := <-rowsChan
defer rows.Close()
```

### Worker Pool任务
```go
taskID := fmt.Sprintf("task_%d_%d", id, time.Now().Unix())
err := utils.SubmitTask(taskID, func(ctx context.Context) error {
    return doAsyncWork(ctx, id)
}, 5*time.Second)
```

### 事务处理
```go
err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
    // 业务逻辑
    _, err := tx.ExecContext(ctx, query1, args1...)
    if err != nil {
        return err  // 自动回滚
    }
    
    _, err = tx.ExecContext(ctx, query2, args2...)
    return err  // 成功自动提交，失败自动回滚
})
```

### 批量查询解决N+1
```go
// 第一步：收集ID
articleIDs := make([]uint, 0)
for rows.Next() {
    var article Article
    rows.Scan(&article)
    articleIDs = append(articleIDs, article.ID)
}

// 第二步：批量查询关联数据
if len(articleIDs) > 0 {
    placeholders := strings.Repeat("?,", len(articleIDs))
    placeholders = placeholders[:len(placeholders)-1]
    
    query := fmt.Sprintf(
        "SELECT article_id, tag_name FROM tags WHERE article_id IN (%s)",
        placeholders)
    
    args := make([]interface{}, len(articleIDs))
    for i, id := range articleIDs {
        args[i] = id
    }
    
    rows, _ := db.QueryContext(ctx, query, args...)
    // 处理结果...
}
```

### LRU缓存
```go
cache := utils.NewLRUCache(utils.LRUCacheConfig{
    Capacity:   1000,
    MaxMemory:  10 * 1024 * 1024,  // 10MB
    DefaultTTL: 5 * time.Minute,
})

// 设置
cache.SetWithTTL("key", value, 10*time.Minute)

// 获取
if value, ok := cache.Get("key"); ok {
    return value
}
```

## ⚡ 性能检查清单

在提交代码前检查：

- [ ] 有循环查询吗？→ 改为批量查询
- [ ] 有裸goroutine吗？→ 改为Worker Pool
- [ ] 查询有索引吗？→ 添加必要索引
- [ ] 热点数据有缓存吗？→ 添加缓存
- [ ] 事务尽可能小吗？→ 只包含必要操作
- [ ] 查询有超时吗？→ 使用WithTimeout
- [ ] 有重复查询吗？→ 合并或缓存
- [ ] 大对象有复用吗？→ 使用对象池

## 🔍 性能问题排查

### 1. 响应慢
```bash
# 查看慢查询
curl http://localhost:3001/metrics/slow-queries

# 查看性能指标
curl http://localhost:3001/metrics/performance

# 查看数据库连接池
tail -f log/app.log | grep "数据库连接池"
```

### 2. 内存增长
```bash
# 查看goroutine数量
curl http://localhost:3001/metrics/worker-pool

# 查看缓存状态
curl http://localhost:3001/metrics/cache

# Go pprof
go tool pprof http://localhost:3001/debug/pprof/heap
```

### 3. 数据库问题
```sql
-- 查看慢查询
SELECT * FROM mysql.slow_log ORDER BY query_time DESC LIMIT 10;

-- 查看连接数
SHOW STATUS LIKE 'Threads_connected';

-- 查看表状态
SHOW TABLE STATUS LIKE 'articles';
```

## 💡 最佳实践

### DO ✅
- 使用prepared statements
- 使用批量操作
- 使用Worker Pool
- 使用对象池
- 使用缓存
- 添加索引
- 设置超时
- 记录慢查询

### DON'T ❌
- 不要循环查询数据库
- 不要创建裸goroutine
- 不要忘记关闭连接
- 不要SELECT *（只查需要的列）
- 不要在WHERE中使用函数
- 不要忘记事务回滚
- 不要忽略错误
- 不要过度优化

## 📖 相关文档

- [DATABASE_OPTIMIZATION_GUIDE.md](DATABASE_OPTIMIZATION_GUIDE.md) - 数据库优化详细指南
- [BACKEND_PERFORMANCE_OPTIMIZATIONS.md](BACKEND_PERFORMANCE_OPTIMIZATIONS.md) - 性能优化总结
- [PERFORMANCE_OPTIMIZATIONS_CHECKLIST.md](PERFORMANCE_OPTIMIZATIONS_CHECKLIST.md) - 优化完成清单

---

**Keep it fast! 🚄**

