package main

import (
	"fmt"
	"log"

	"gin/internal/config"
	"gin/internal/routes"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 设置路由
	r := routes.SetupRoutes(cfg)

	// 启动服务
	fmt.Printf("服务器启动在端口 %s\n", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
