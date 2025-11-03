package services

import (
	"container/list"
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	_ "github.com/go-sql-driver/mysql"
)

// stmtCacheEntry Prepared Statement 缓存条目（LRU）
type stmtCacheEntry struct {
	query      string
	stmt       *sql.Stmt
	element    *list.Element // 在LRU链表中的位置
	lastAccess int64         // 最后访问时间（Unix时间戳）
}

// stmtCacheShard Prepared Statement缓存分片
type stmtCacheShard struct {
	cache   map[string]*stmtCacheEntry
	lruList *list.List
	mu      sync.RWMutex
}

const numShards = 16 // Prepared Statement缓存分片数量

// Database 数据库服务
type Database struct {
	DB                  *sql.DB
	config              *config.DatabaseConfig
	timeouts            *config.DatabaseTimeoutsConfig
	queryConfig         *config.DatabaseQueryConfig
	queryAdvanced       *config.DatabaseQueryAdvancedConfig
	repositoryTimeouts  *config.RepositoryTimeoutsConfig
	asyncTasksTimeouts  *config.AsyncTasksConfig
	logger              utils.Logger
	stopMonitor         chan struct{} // 用于停止监控goroutine
	stmtShards          [numShards]*stmtCacheShard
	stmtMaxSizePerShard int
	ctx                 context.Context
	cancel              context.CancelFunc
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg *config.Config) (*Database, error) {
	logger := utils.GetLogger()

	// 构建数据库连接字符串（使用配置的超时参数，时区统一为UTC）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=UTC&timeout=%ds&readTimeout=%ds&writeTimeout=%ds&interpolateParams=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
		cfg.Database.Charset,
		cfg.DatabaseTimeouts.ConnectionTimeout,
		cfg.DatabaseTimeouts.ReadTimeout,
		cfg.DatabaseTimeouts.WriteTimeout,
	)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Error("数据库连接失败", "error", err.Error())
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 设置空闲连接超时（优化：更快释放空闲连接）
	idleTimeout := time.Duration(cfg.DatabaseQuery.IdleTimeoutMinutes) * time.Minute
	if cfg.Database.ConnMaxIdleTime > 0 {
		idleTimeout = cfg.Database.ConnMaxIdleTime
	}
	db.SetConnMaxIdleTime(idleTimeout)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 从配置读取 Prepared Statement 缓存大小限制（默认1000）
	stmtMaxSize := 1000
	if cfg.DatabaseQueryAdvanced.PreparedStmtCacheSize > 0 {
		stmtMaxSize = cfg.DatabaseQueryAdvanced.PreparedStmtCacheSize
	}

	// 创建数据库实例
	dbInstance := &Database{
		DB:                  db,
		config:              &cfg.Database,
		timeouts:            &cfg.DatabaseTimeouts,
		queryConfig:         &cfg.DatabaseQuery,
		queryAdvanced:       &cfg.DatabaseQueryAdvanced,
		repositoryTimeouts:  &cfg.RepositoryTimeouts,
		asyncTasksTimeouts:  &cfg.AsyncTasks,
		logger:              logger,
		stopMonitor:         make(chan struct{}),
		stmtMaxSizePerShard: stmtMaxSize / numShards, // 每个分片的容量
		ctx:                 ctx,
		cancel:              cancel,
	}

	// 初始化所有分片
	for i := 0; i < numShards; i++ {
		dbInstance.stmtShards[i] = &stmtCacheShard{
			cache:   make(map[string]*stmtCacheEntry),
			lruList: list.New(),
		}
	}

	// 启动连接池监控（使用配置的监控间隔）
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.DatabaseTimeouts.PoolMonitorInterval) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stats := db.Stats()
				if stats.OpenConnections > int(float64(cfg.Database.MaxOpenConns)*0.8) {
					logger.Warn("数据库连接池使用率过高",
						"openConnections", stats.OpenConnections,
						"maxOpenConns", cfg.Database.MaxOpenConns,
						"inUse", stats.InUse,
						"idle", stats.Idle)
				}
			case <-dbInstance.stopMonitor:
				logger.Info("数据库连接池监控已停止")
				return
			}
		}
	}()

	// 测试连接（使用配置的超时）
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(cfg.DatabaseTimeouts.TestConnectionTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("数据库连接测试失败", "error", err.Error())
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	logger.Info("数据库连接成功",
		"host", cfg.Database.Host,
		"port", cfg.Database.Port,
		"database", cfg.Database.Database,
		"maxOpenConns", cfg.Database.MaxOpenConns,
		"maxIdleConns", cfg.Database.MaxIdleConns,
		"connMaxLifetime", cfg.Database.ConnMaxLifetime,
		"connMaxIdleTime", idleTimeout)

	// 预热连接池（创建初始连接，避免首次请求慢）
	go dbInstance.warmupConnectionPool(cfg.Database.MaxIdleConns)

	return dbInstance, nil
}

// warmupConnectionPool 预热连接池
func (d *Database) warmupConnectionPool(targetConns int) {
	if targetConns <= 0 {
		targetConns = 5
	}

	d.logger.Info("开始预热数据库连接池", "targetConns", targetConns)
	start := time.Now()

	// 并发创建连接
	errCount := 0
	for i := 0; i < targetConns; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.timeouts.WarmupConnectionTimeout)*time.Second)
		if err := d.DB.PingContext(ctx); err != nil {
			errCount++
		}
		cancel()
	}

	stats := d.DB.Stats()
	d.logger.Info("连接池预热完成",
		"duration", time.Since(start),
		"openConnections", stats.OpenConnections,
		"idle", stats.Idle,
		"errors", errCount)
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.DB != nil {
		d.logger.Info("正在关闭数据库连接")

		// 取消上下文
		if d.cancel != nil {
			d.cancel()
		}

		// 关闭所有分片中缓存的prepared statements
		totalCount := 0
		for i := 0; i < numShards; i++ {
			shard := d.stmtShards[i]
			shard.mu.Lock()
			shardCount := len(shard.cache)
			for key, entry := range shard.cache {
				if err := entry.stmt.Close(); err != nil {
					d.logger.Warn("关闭prepared statement失败", "shard", i, "key", key, "error", err.Error())
				}
			}
			shard.cache = make(map[string]*stmtCacheEntry)
			shard.lruList = list.New()
			shard.mu.Unlock()
			totalCount += shardCount
		}
		d.logger.Info("已清理prepared statements", "count", totalCount, "shards", numShards)

		// 停止监控goroutine
		close(d.stopMonitor)
		// 等待一小段时间让goroutine退出
		time.Sleep(time.Duration(d.queryConfig.RetryWaitMS) * time.Millisecond)

		// 关闭数据库连接
		if err := d.DB.Close(); err != nil {
			d.logger.Error("关闭数据库连接失败", "error", err.Error())
			return err
		}

		d.logger.Info("数据库连接已安全关闭")
		return nil
	}
	return nil
}

// Ping 测试数据库连接（使用配置的超时）
func (d *Database) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.timeouts.PingTimeout)*time.Second)
	defer cancel()
	return d.DB.PingContext(ctx)
}

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	return d.Ping()
}

// GetStats 获取数据库连接池统计信息
func (d *Database) GetStats() sql.DBStats {
	return d.DB.Stats()
}

// WithTransaction 事务辅助方法
func (d *Database) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		d.logger.Error("开始事务失败", "error", err.Error())
		return fmt.Errorf("开始事务失败: %w", err)
	}

	// 确保事务被提交或回滚
	defer func() {
		if p := recover(); p != nil {
			// 发生panic，回滚事务
			_ = tx.Rollback()
			d.logger.Error("事务执行panic，已回滚", "panic", p)
			panic(p) // 重新抛出panic
		}
	}()

	// 执行业务逻辑
	if err := fn(tx); err != nil {
		// 业务逻辑失败，回滚事务
		if rbErr := tx.Rollback(); rbErr != nil {
			d.logger.Error("回滚事务失败", "error", rbErr.Error(), "originalError", err.Error())
			return fmt.Errorf("回滚事务失败: %v (原始错误: %w)", rbErr, err)
		}
		d.logger.Info("事务已回滚", "reason", err.Error())
		return err
	}

	// 业务逻辑成功，提交事务
	if err := tx.Commit(); err != nil {
		d.logger.Error("提交事务失败", "error", err.Error())
		return fmt.Errorf("提交事务失败: %w", err)
	}

	d.logger.Debug("事务提交成功")
	return nil
}

// RetryQuery 带重试的查询执行
func (d *Database) RetryQuery(ctx context.Context, maxRetries int, fn func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// 判断是否为可重试错误
		if !isRetriableError(err) {
			return err
		}

		// 指数退避
		if i < maxRetries-1 {
			backoff := time.Duration(1<<uint(i)) * time.Duration(d.queryConfig.RetryBackoffBaseMS) * time.Millisecond
			d.logger.Warn("查询失败，准备重试",
				"attempt", i+1,
				"maxRetries", maxRetries,
				"backoff", backoff,
				"error", err.Error())

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	d.logger.Error("查询重试次数已用尽", "maxRetries", maxRetries, "error", err.Error())
	return fmt.Errorf("查询重试%d次后仍然失败: %w", maxRetries, err)
}

// isRetriableError 判断是否为可重试的错误
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	// MySQL可重试错误
	retriableErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"too many connections",
		"lock wait timeout",
		"deadlock",
	}

	for _, retryErr := range retriableErrors {
		if contains(errMsg, retryErr) {
			return true
		}
	}
	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	// 使用简单的字符串包含检查
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getShardForQuery 获取查询对应的分片
func (d *Database) getShardForQuery(query string) *stmtCacheShard {
	hash := fnv.New32a()
	hash.Write([]byte(query))
	return d.stmtShards[hash.Sum32()%numShards]
}

// PrepareStmt 获取或创建prepared statement（使用分片锁，提升并发性能）
func (d *Database) PrepareStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	shard := d.getShardForQuery(query)

	// 先尝试从分片缓存获取（只需读锁）
	shard.mu.RLock()
	entry, exists := shard.cache[query]
	shard.mu.RUnlock()

	if exists {
		// 使用原子操作更新访问时间，无需获取锁
		atomic.StoreInt64(&entry.lastAccess, time.Now().UTC().Unix())
		return entry.stmt, nil
	}

	// 缓存中没有，创建新的prepared statement
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 双重检查，防止并发创建
	if entry, exists := shard.cache[query]; exists {
		atomic.StoreInt64(&entry.lastAccess, time.Now().UTC().Unix())
		return entry.stmt, nil
	}

	// 检查是否达到最大容量，淘汰最久未使用的
	if len(shard.cache) >= d.stmtMaxSizePerShard {
		d.evictOldestInShard(shard)
	}

	stmt, err := d.DB.PrepareContext(ctx, query)
	if err != nil {
		d.logger.Error("创建prepared statement失败", "error", err.Error())
		return nil, err
	}

	// 添加到缓存
	nowUnix := time.Now().UTC().Unix()
	entry = &stmtCacheEntry{
		query:      query,
		stmt:       stmt,
		lastAccess: nowUnix,
	}
	entry.element = shard.lruList.PushFront(entry)
	shard.cache[query] = entry

	d.logger.Debug("Prepared statement已缓存",
		"query", utils.TruncateString(query, 50),
		"shardSize", len(shard.cache),
		"maxPerShard", d.stmtMaxSizePerShard)

	return stmt, nil
}

// evictOldestInShard 在分片中淘汰最久未访问的prepared statement
func (d *Database) evictOldestInShard(shard *stmtCacheShard) {
	if len(shard.cache) == 0 {
		return
	}

	var oldestEntry *stmtCacheEntry
	var oldestTime int64 = (1 << 63) - 1 // MaxInt64

	// 找到最久未访问的entry
	for _, entry := range shard.cache {
		accessTime := atomic.LoadInt64(&entry.lastAccess)
		if accessTime < oldestTime {
			oldestTime = accessTime
			oldestEntry = entry
		}
	}

	if oldestEntry != nil {
		// 关闭statement
		if err := oldestEntry.stmt.Close(); err != nil {
			d.logger.Warn("关闭淘汰的prepared statement失败",
				"query", utils.TruncateString(oldestEntry.query, 50),
				"error", err.Error())
		}

		// 从缓存中删除
		delete(shard.cache, oldestEntry.query)
		shard.lruList.Remove(oldestEntry.element)

		d.logger.Debug("淘汰prepared statement",
			"query", utils.TruncateString(oldestEntry.query, 50),
			"lastAccess", time.Unix(oldestTime, 0),
			"shardSize", len(shard.cache))
	}
}

// ExecWithCache 使用缓存的prepared statement执行查询
func (d *Database) ExecWithCache(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()

	stmt, err := d.PrepareStmt(ctx, query)
	if err != nil {
		d.logger.Error("SQL执行失败: prepare失败",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, err
	}

	result, err := stmt.ExecContext(ctx, args...)
	duration := time.Since(start)

	if err != nil {
		d.logger.Error("SQL执行失败",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"error", err.Error(),
			"duration", duration)
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	d.logger.Info("SQL执行成功[ExecWithCache]",
		"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
		"rowsAffected", rowsAffected,
		"lastInsertID", lastInsertID,
		"duration", duration,
		"durationMs", duration.Milliseconds())

	// 慢查询警告（从配置读取阈值）
	slowQueryThreshold := time.Duration(d.queryConfig.SlowQueryThresholdMS) * time.Millisecond
	if duration > slowQueryThreshold {
		d.logger.Warn("检测到慢查询[ExecWithCache]",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"duration", duration,
			"durationMs", duration.Milliseconds(),
			"threshold", slowQueryThreshold.String(),
			"params", utils.FormatSQLParams(args))
	}

	return result, nil
}

// QueryRowWithCache 使用缓存的prepared statement执行单行查询
func (d *Database) QueryRowWithCache(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()

	stmt, err := d.PrepareStmt(ctx, query)
	if err != nil {
		d.logger.Warn("SQL查询: prepare失败，回退到普通查询",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"error", err.Error())
		// 如果prepare失败，回退到普通查询
		return d.DB.QueryRowContext(ctx, query, args...)
	}

	row := stmt.QueryRowContext(ctx, args...)
	duration := time.Since(start)

	// 慢查询警告（从配置读取阈值）
	slowQueryThreshold := time.Duration(d.queryConfig.SlowQueryThresholdMS) * time.Millisecond
	if duration > slowQueryThreshold {
		d.logger.Warn("检测到慢查询[QueryRowWithCache]",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"duration", duration,
			"durationMs", duration.Milliseconds(),
			"threshold", slowQueryThreshold.String(),
			"params", utils.FormatSQLParams(args))
	}

	return row
}

// QueryWithCache 使用缓存的prepared statement执行多行查询
func (d *Database) QueryWithCache(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	stmt, err := d.PrepareStmt(ctx, query)
	if err != nil {
		d.logger.Error("SQL查询失败: prepare失败",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, args...)
	duration := time.Since(start)

	if err != nil {
		d.logger.Error("SQL查询失败",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"error", err.Error(),
			"duration", duration)
		return nil, err
	}

	d.logger.Info("SQL查询成功[QueryWithCache]",
		"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
		"duration", duration,
		"durationMs", duration.Milliseconds())

	// 慢查询警告（从配置读取阈值）
	slowQueryThreshold := time.Duration(d.queryConfig.SlowQueryThresholdMS) * time.Millisecond
	if duration > slowQueryThreshold {
		d.logger.Warn("检测到慢查询[QueryWithCache]",
			"query", utils.TruncateString(query, d.queryAdvanced.QueryLogTruncateLength),
			"duration", duration,
			"durationMs", duration.Milliseconds(),
			"threshold", slowQueryThreshold.String(),
			"params", utils.FormatSQLParams(args))
	}

	return rows, nil
}

// GetQueryTimeout 获取查询操作超时时长（用于SELECT等读操作）
func (d *Database) GetQueryTimeout() time.Duration {
	if d.repositoryTimeouts != nil && d.repositoryTimeouts.DefaultQueryTimeout > 0 {
		return time.Duration(d.repositoryTimeouts.DefaultQueryTimeout) * time.Second
	}
	return 5 * time.Second // 默认5秒
}

// GetUpdateTimeout 获取更新操作超时时长（用于INSERT/UPDATE/DELETE等写操作）
func (d *Database) GetUpdateTimeout() time.Duration {
	if d.repositoryTimeouts != nil && d.repositoryTimeouts.DefaultUpdateTimeout > 0 {
		return time.Duration(d.repositoryTimeouts.DefaultUpdateTimeout) * time.Second
	}
	return 10 * time.Second // 默认10秒
}

// GetAsyncTaskTimeout 获取异步任务超时时长（用于快速异步操作）
func (d *Database) GetAsyncTaskTimeout() time.Duration {
	if d.asyncTasksTimeouts != nil && d.asyncTasksTimeouts.ArticleViewCountTimeout > 0 {
		// 使用ArticleViewCountTimeout作为通用异步任务超时（3秒）
		return time.Duration(d.asyncTasksTimeouts.ArticleViewCountTimeout) * time.Second
	}
	return 3 * time.Second // 默认3秒
}
