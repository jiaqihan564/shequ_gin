package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	_ "github.com/go-sql-driver/mysql"
)

// Database 数据库服务
type Database struct {
	DB     *sql.DB
	config *config.DatabaseConfig
	logger utils.Logger
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg *config.Config) (*Database, error) {
	logger := utils.GetLogger()

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
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

	return &Database{
		DB:     db,
		config: &cfg.Database,
		logger: logger,
	}, nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.DB != nil {
		d.logger.Info("正在关闭数据库连接")
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

// GetStats 获取数据库连接池统计信息
func (d *Database) GetStats() sql.DBStats {
	return d.DB.Stats()
}

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	return d.Ping()
}
