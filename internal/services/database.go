package services

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	_ "github.com/go-sql-driver/mysql"
)

// Database 数据库服务
type Database struct {
	DB          *sql.DB
	config      *config.DatabaseConfig
	logger      utils.Logger
	stopMonitor chan struct{} // 用于停止监控goroutine
	stmtCache   map[string]*sql.Stmt
	stmtMutex   sync.RWMutex
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg *config.Config) (*Database, error) {
	logger := utils.GetLogger()

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s&interpolateParams=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
		cfg.Database.Charset,
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

	// 设置连接超时
	db.SetConnMaxIdleTime(5 * time.Minute)

	// 创建数据库实例
	dbInstance := &Database{
		DB:          db,
		config:      &cfg.Database,
		logger:      logger,
		stopMonitor: make(chan struct{}),
		stmtCache:   make(map[string]*sql.Stmt),
	}

	// 启动连接池监控
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
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

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		"maxIdleConns", cfg.Database.MaxIdleConns)

	return dbInstance, nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.DB != nil {
		d.logger.Info("正在关闭数据库连接")

		// 关闭所有缓存的prepared statements
		d.stmtMutex.Lock()
		for key, stmt := range d.stmtCache {
			if err := stmt.Close(); err != nil {
				d.logger.Warn("关闭prepared statement失败", "key", key, "error", err.Error())
			}
		}
		d.stmtCache = make(map[string]*sql.Stmt)
		d.stmtMutex.Unlock()

		// 停止监控goroutine
		close(d.stopMonitor)
		// 等待一小段时间让goroutine退出
		time.Sleep(100 * time.Millisecond)
		return d.DB.Close()
	}
	return nil
}

// Ping 测试数据库连接
func (d *Database) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.DB.PingContext(ctx)
}

//

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	return d.Ping()
}

// PrepareStmt 获取或创建prepared statement（带缓存）
func (d *Database) PrepareStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	// 先尝试从缓存获取
	d.stmtMutex.RLock()
	if stmt, exists := d.stmtCache[query]; exists {
		d.stmtMutex.RUnlock()
		return stmt, nil
	}
	d.stmtMutex.RUnlock()

	// 缓存中没有，创建新的prepared statement
	d.stmtMutex.Lock()
	defer d.stmtMutex.Unlock()

	// 双重检查，防止并发创建
	if stmt, exists := d.stmtCache[query]; exists {
		return stmt, nil
	}

	stmt, err := d.DB.PrepareContext(ctx, query)
	if err != nil {
		d.logger.Error("创建prepared statement失败", "error", err.Error())
		return nil, err
	}

	d.stmtCache[query] = stmt
	return stmt, nil
}

// ExecWithCache 使用缓存的prepared statement执行查询
func (d *Database) ExecWithCache(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	d.logger.Debug("SQL执行开始[ExecWithCache]",
		"query", utils.TruncateString(query, 200),
		"params", utils.FormatSQLParams(args),
		"paramCount", len(args))

	stmt, err := d.PrepareStmt(ctx, query)
	if err != nil {
		d.logger.Error("SQL执行失败: prepare失败",
			"query", utils.TruncateString(query, 200),
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, err
	}

	result, err := stmt.ExecContext(ctx, args...)
	duration := time.Since(start)

	if err != nil {
		d.logger.Error("SQL执行失败",
			"query", utils.TruncateString(query, 200),
			"error", err.Error(),
			"duration", duration)
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	d.logger.Info("SQL执行成功[ExecWithCache]",
		"query", utils.TruncateString(query, 200),
		"rowsAffected", rowsAffected,
		"lastInsertID", lastInsertID,
		"duration", duration,
		"durationMs", duration.Milliseconds())

	// 慢查询警告
	if duration > 100*time.Millisecond {
		d.logger.Warn("检测到慢查询[ExecWithCache]",
			"query", utils.TruncateString(query, 200),
			"duration", duration,
			"durationMs", duration.Milliseconds(),
			"threshold", "100ms")
	}

	return result, nil
}

// QueryRowWithCache 使用缓存的prepared statement执行单行查询
func (d *Database) QueryRowWithCache(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	d.logger.Debug("SQL查询开始[QueryRowWithCache]",
		"query", utils.TruncateString(query, 200),
		"params", utils.FormatSQLParams(args),
		"paramCount", len(args))

	stmt, err := d.PrepareStmt(ctx, query)
	if err != nil {
		d.logger.Warn("SQL查询: prepare失败，回退到普通查询",
			"query", utils.TruncateString(query, 200),
			"error", err.Error())
		// 如果prepare失败，回退到普通查询
		return d.DB.QueryRowContext(ctx, query, args...)
	}

	row := stmt.QueryRowContext(ctx, args...)
	duration := time.Since(start)

	d.logger.Debug("SQL查询完成[QueryRowWithCache]",
		"query", utils.TruncateString(query, 200),
		"duration", duration,
		"durationMs", duration.Milliseconds())

	// 慢查询警告
	if duration > 100*time.Millisecond {
		d.logger.Warn("检测到慢查询[QueryRowWithCache]",
			"query", utils.TruncateString(query, 200),
			"duration", duration,
			"durationMs", duration.Milliseconds(),
			"threshold", "100ms")
	}

	return row
}

// QueryWithCache 使用缓存的prepared statement执行多行查询
func (d *Database) QueryWithCache(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	d.logger.Debug("SQL查询开始[QueryWithCache]",
		"query", utils.TruncateString(query, 200),
		"params", utils.FormatSQLParams(args),
		"paramCount", len(args))

	stmt, err := d.PrepareStmt(ctx, query)
	if err != nil {
		d.logger.Error("SQL查询失败: prepare失败",
			"query", utils.TruncateString(query, 200),
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, args...)
	duration := time.Since(start)

	if err != nil {
		d.logger.Error("SQL查询失败",
			"query", utils.TruncateString(query, 200),
			"error", err.Error(),
			"duration", duration)
		return nil, err
	}

	d.logger.Info("SQL查询成功[QueryWithCache]",
		"query", utils.TruncateString(query, 200),
		"duration", duration,
		"durationMs", duration.Milliseconds())

	// 慢查询警告
	if duration > 100*time.Millisecond {
		d.logger.Warn("检测到慢查询[QueryWithCache]",
			"query", utils.TruncateString(query, 200),
			"duration", duration,
			"durationMs", duration.Milliseconds(),
			"threshold", "100ms")
	}

	return rows, nil
}
