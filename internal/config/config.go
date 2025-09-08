package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构体
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	JWT      JWTConfig      `yaml:"jwt" json:"jwt"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Log      LogConfig      `yaml:"log" json:"log"`
	Security SecurityConfig `yaml:"security" json:"security"`
	CORS     CORSConfig     `yaml:"cors" json:"cors"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port string `yaml:"port" json:"port"`
	Mode string `yaml:"mode" json:"mode"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey   string `yaml:"secret_key" json:"secret_key"`
	ExpireHours int    `yaml:"expire_hours" json:"expire_hours"`
	Issuer      string `yaml:"issuer" json:"issuer"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `yaml:"host" json:"host"`
	Port            string        `yaml:"port" json:"port"`
	Username        string        `yaml:"username" json:"username"`
	Password        string        `yaml:"password" json:"password"`
	Database        string        `yaml:"database" json:"database"`
	Charset         string        `yaml:"charset" json:"charset"`
	MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"`
	Output     string `yaml:"output" json:"output"`
	FilePath   string `yaml:"file_path" json:"file_path"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	PasswordMinLength     int           `yaml:"password_min_length" json:"password_min_length"`
	PasswordMaxLength     int           `yaml:"password_max_length" json:"password_max_length"`
	UsernameMinLength     int           `yaml:"username_min_length" json:"username_min_length"`
	UsernameMaxLength     int           `yaml:"username_max_length" json:"username_max_length"`
	MaxLoginAttempts      int           `yaml:"max_login_attempts" json:"max_login_attempts"`
	LockoutDuration       time.Duration `yaml:"lockout_duration" json:"lockout_duration"`
	EnablePasswordHistory bool          `yaml:"enable_password_history" json:"enable_password_history"`
	PasswordHistoryCount  int           `yaml:"password_history_count" json:"password_history_count"`
	SessionTimeout        time.Duration `yaml:"session_timeout" json:"session_timeout"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
}

// Load 加载配置
func Load() *Config {
	// 获取环境变量
	env := getEnv("APP_ENV", "dev")
	configFile := getConfigFile(env)

	// 创建默认配置
	config := getDefaultConfig()

	// 从配置文件加载
	if configFile != "" {
		if err := loadFromFile(config, configFile); err != nil {
			fmt.Printf("警告: 加载配置文件失败 %s: %v\n", configFile, err)
		} else {
			fmt.Printf("已加载配置文件: %s\n", configFile)
		}
	}

	// 环境变量覆盖配置文件
	overrideWithEnvVars(config)

	return config
}

// getConfigFile 获取配置文件路径
func getConfigFile(env string) string {
	// 检查当前目录
	configFiles := []string{
		fmt.Sprintf("config.%s.yaml", env),
		"config.yaml",
	}

	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}

	return ""
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: "8080",
			Mode: "release",
		},
		JWT: JWTConfig{
			SecretKey:   getEnv("JWT_SECRET", "default_secret_key_change_in_production"),
			ExpireHours: 24,
			Issuer:      "community-api",
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "3306"),
			Username:        getEnv("DB_USERNAME", "root"),
			Password:        getEnv("DB_PASSWORD", ""),
			Database:        getEnv("DB_DATABASE", "community"),
			Charset:         "utf8mb4",
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: time.Hour,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			FilePath:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
		Security: SecurityConfig{
			PasswordMinLength:     8,
			PasswordMaxLength:     50,
			UsernameMinLength:     3,
			UsernameMaxLength:     20,
			MaxLoginAttempts:      5,
			LockoutDuration:       30 * time.Minute,
			EnablePasswordHistory: false,
			PasswordHistoryCount:  5,
			SessionTimeout:        24 * time.Hour,
		},
		CORS: CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
			AllowCredentials: true,
		},
	}
}

// loadFromFile 从文件加载配置
func loadFromFile(config *Config, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// overrideWithEnvVars 用环境变量覆盖配置
func overrideWithEnvVars(config *Config) {
	// 服务器配置
	if val := getEnv("SERVER_HOST", ""); val != "" {
		config.Server.Host = val
	}
	if val := getEnv("SERVER_PORT", ""); val != "" {
		config.Server.Port = val
	}
	if val := getEnv("SERVER_MODE", ""); val != "" {
		config.Server.Mode = val
	}

	// 数据库配置
	if val := getEnv("DB_HOST", ""); val != "" {
		config.Database.Host = val
	}
	if val := getEnv("DB_PORT", ""); val != "" {
		config.Database.Port = val
	}
	if val := getEnv("DB_USERNAME", ""); val != "" {
		config.Database.Username = val
	}
	if val := getEnv("DB_PASSWORD", ""); val != "" {
		config.Database.Password = val
	}
	if val := getEnv("DB_DATABASE", ""); val != "" {
		config.Database.Database = val
	}

	// JWT配置
	if val := getEnv("JWT_SECRET", ""); val != "" {
		config.JWT.SecretKey = val
	}
	if val := getEnv("JWT_EXPIRE_HOURS", ""); val != "" {
		if hours := parseInt(val); hours > 0 {
			config.JWT.ExpireHours = hours
		}
	}

	// 日志配置
	if val := getEnv("LOG_LEVEL", ""); val != "" {
		config.Log.Level = val
	}
	if val := getEnv("LOG_FORMAT", ""); val != "" {
		config.Log.Format = val
	}
	if val := getEnv("LOG_OUTPUT", ""); val != "" {
		config.Log.Output = val
	}
}

// parseInt 解析整数
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
