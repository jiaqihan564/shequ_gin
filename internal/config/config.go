package config

import (
	"os"
)

// Config 应用配置结构体
type Config struct {
	Server ServerConfig `json:"server"`
	JWT    JWTConfig    `json:"jwt"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
	Host string `json:"host"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey   string `json:"secret_key"`
	ExpireHours int    `json:"expire_hours"`
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "localhost"),
		},
		JWT: JWTConfig{
			SecretKey:   getEnv("JWT_SECRET", "your_secret_key_change_this_in_production"),
			ExpireHours: 24,
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
