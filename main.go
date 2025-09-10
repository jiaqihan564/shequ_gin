package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin/internal/bootstrap"
	"gin/internal/config"
	"gin/internal/routes"
	"gin/internal/services"
	"gin/internal/utils"
)

const (
	AppVersion = "1.0.0"
	AppName    = "Community API"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志系统
	if err := utils.InitLogger(&cfg.Log); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}

	logger := utils.GetLogger()
	logger.Info("应用启动",
		"app", AppName,
		"version", AppVersion,
		"mode", cfg.Server.Mode,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port)

	// 初始化数据库连接
	db, err := services.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("数据库连接失败", "error", err.Error())
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("关闭数据库连接时出错", "error", err.Error())
		}
	}()

	logger.Info("数据库连接成功")

	// 组装容器
	container, err := bootstrap.New(cfg, db)
	if err != nil {
		logger.Error("装配容器失败", "error", err.Error())
	}

	// 设置路由
	r := routes.SetupRoutes(cfg, container)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	go func() {
		logger.Info("服务器启动", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", "error", err.Error())
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到关闭信号，正在关闭服务器...")

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败", "error", err.Error())
	} else {
		logger.Info("服务器已优雅关闭")
	}
}
