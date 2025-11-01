package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"gin/internal/bootstrap"
	"gin/internal/config"
	"gin/internal/routes"
	"gin/internal/services"
	"gin/internal/utils"
)

func main() {
	// 全局 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("应用崩溃恢复: %v\n", r)
			os.Exit(1)
		}
	}()

	// 加载配置
	cfg := config.Load()

	// 初始化每日指标管理器
	services.InitDailyMetricsManager(cfg)

	// 初始化日志系统
	if err := utils.InitLogger(&cfg.Log); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化性能分析器
	utils.InitGlobalProfiler(&cfg.Profiler)
	utils.InitGlobalSlowQueryDetector(&cfg.Profiler)

	logger := utils.GetLogger()
	logger.Info("应用启动",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"mode", cfg.Server.Mode,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"goVersion", runtime.Version(),
		"numCPU", runtime.NumCPU(),
		"GOMAXPROCS", runtime.GOMAXPROCS(0))

	// 初始化数据库连接
	db, err := services.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("数据库连接失败", "error", err.Error())
	}
	defer func() {
		logger.Info("正在关闭数据库连接...")
		if err := db.Close(); err != nil {
			logger.Error("关闭数据库连接时出错", "error", err.Error())
		} else {
			logger.Info("数据库连接已关闭")
		}
	}()

	logger.Info("数据库连接成功")

	// 数据库健康检查
	if err := db.Ping(); err != nil {
		logger.Fatal("数据库健康检查失败", "error", err.Error())
	}
	logger.Info("数据库健康检查通过")

	// 组装容器
	container, err := bootstrap.New(cfg, db)
	if err != nil {
		logger.Fatal("装配容器失败", "error", err.Error())
	}
	logger.Info("依赖容器装配成功")

	// 初始化管理员账号（如果不存在则自动创建）
	if err := bootstrap.InitAdminAccounts(cfg, container.UserRepo); err != nil {
		logger.Warn("初始化管理员账号失败", "error", err.Error())
	} else {
		logger.Info("管理员账号检查完成")
	}

	// 设置路由
	r := routes.SetupRoutes(cfg, container)

	// 创建HTTP服务器（使用配置的超时设置）
	server := &http.Server{
		Addr:              cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:           r,
		ReadTimeout:       time.Duration(cfg.Server.ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(cfg.Server.ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(cfg.Server.IdleTimeout) * time.Second,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}

	// 启动服务器
	go func() {
		logger.Info("HTTP服务器正在启动...",
			"address", server.Addr,
			"readTimeout", server.ReadTimeout,
			"writeTimeout", server.WriteTimeout,
			"idleTimeout", server.IdleTimeout)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", "error", err.Error())
		}
	}()

	// 启动后延迟健康检查（使用配置的延迟时间）
	time.Sleep(time.Duration(cfg.Server.StartupHealthCheckDelay) * time.Millisecond)
	if err := checkServerHealth(cfg); err != nil {
		logger.Warn("服务器健康检查警告", "error", err.Error())
	} else {
		logger.Info("服务器健康检查通过", "url", fmt.Sprintf("http://%s:%s/health", cfg.Server.Host, cfg.Server.Port))
	}

	// 在线用户管理已迁移到 WebSocket ConnectionHub（内存管理）
	// 不再需要定时清理数据库中的在线用户记录

	logger.Info("✅ 应用启动完成，等待请求...")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("收到关闭信号，正在优雅关闭服务器...",
		"signal", sig.String())

	// 优雅关闭服务器（使用配置的超时时间）
	shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// 停止接受新请求
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败", "error", err.Error(), "timeout", shutdownTimeout)
		// 强制关闭
		if err := server.Close(); err != nil {
			logger.Error("强制关闭服务器失败", "error", err.Error())
		}
	} else {
		logger.Info("服务器已优雅关闭")
	}

	// 关闭日志（flush 异步队列）
	logger.Info("正在关闭日志系统...")
	if err := utils.CloseLogger(); err != nil {
		fmt.Printf("关闭日志失败: %v\n", err)
	} else {
		fmt.Println("应用已完全关闭")
	}
}

// checkServerHealth 检查服务器健康状态
func checkServerHealth(cfg *config.Config) error {
	url := fmt.Sprintf("http://%s:%s/health", cfg.Server.Host, cfg.Server.Port)
	client := &http.Client{
		Timeout: time.Duration(cfg.Server.HealthCheckClientTimeout) * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("健康检查请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("健康检查返回异常状态码: %d", resp.StatusCode)
	}

	return nil
}
