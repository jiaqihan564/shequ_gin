package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gin/internal/config"
	"gin/internal/routes"
	"gin/internal/services"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库连接
	db, err := services.NewDatabase(cfg)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("关闭数据库连接时出错: %v", err)
		}
	}()

	// 设置路由
	r := routes.SetupRoutes(cfg, db)

	// 设置优雅关闭
	go func() {
		// 启动服务
		fmt.Printf("服务器启动在端口 %s\n", cfg.Server.Port)
		if err := r.Run(":" + cfg.Server.Port); err != nil {
			log.Fatal("服务器启动失败:", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")
}
