package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构体
type Config struct {
	Server           ServerConfig           `yaml:"server" json:"server"`
	JWT              JWTConfig              `yaml:"jwt" json:"jwt"`
	Database         DatabaseConfig         `yaml:"database" json:"database"`
	Log              LogConfig              `yaml:"log" json:"log"`
	Security         SecurityConfig         `yaml:"security" json:"security"`
	Admin            AdminConfig            `yaml:"admin" json:"admin"`
	CORS             CORSConfig             `yaml:"cors" json:"cors"`
	Assets           AssetsConfig           `yaml:"assets" json:"assets"`
	MinIO            MinIOConfig            `yaml:"minio" json:"minio"`
	ResourcesStorage ResourcesStorageConfig `yaml:"resources_storage" json:"resources_storage"`
	CodeExecutor     CodeExecutorConfig     `yaml:"code_executor" json:"code_executor"`
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
	Async      bool   `yaml:"async" json:"async"`
	Buffer     int    `yaml:"buffer" json:"buffer"`
	DropPolicy string `yaml:"drop_policy" json:"drop_policy"` // block | drop_new | drop_oldest
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	MaxLoginAttempts int `yaml:"max_login_attempts" json:"max_login_attempts"`
}

// AdminConfig 管理员配置
type AdminConfig struct {
	Usernames       []string `yaml:"usernames" json:"usernames"`
	DefaultPassword string   `yaml:"default_password" json:"default_password"` // 管理员默认密码（首次创建时使用）
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
}

// AssetsConfig 静态资源/对象存储配置
type AssetsConfig struct {
	// PublicBaseURL 是指向桶根目录的可公开访问的基础 URL，例如: http://192.168.200.131:9000/community-assets
	PublicBaseURL string `yaml:"public_base_url" json:"public_base_url"`
	// MaxAvatarSizeMB 头像上传大小上限（MB）
	MaxAvatarSizeMB int `yaml:"max_avatar_size_mb" json:"max_avatar_size_mb"`
	// MaxAvatarHistory 历史头像最大保留数量
	MaxAvatarHistory int `yaml:"max_avatar_history" json:"max_avatar_history"`
}

// MinIOConfig MinIO 对象存储连接配置
type MinIOConfig struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
	UseSSL          bool   `yaml:"use_ssl" json:"use_ssl"`
	Bucket          string `yaml:"bucket" json:"bucket"`
}

// ResourcesStorageConfig 资源存储配置
type ResourcesStorageConfig struct {
	Bucket        string `yaml:"bucket" json:"bucket"`
	PublicBaseURL string `yaml:"public_base_url" json:"public_base_url"`
}

// CodeExecutorConfig 代码执行器配置
type CodeExecutorConfig struct {
	PistonAPIURL string `yaml:"piston_api_url" json:"piston_api_url"`
	Timeout      int    `yaml:"timeout" json:"timeout"`             // 超时时间（秒）
	MaxMemoryMB  int    `yaml:"max_memory_mb" json:"max_memory_mb"` // 最大内存（MB）
	RateLimit    int    `yaml:"rate_limit" json:"rate_limit"`       // 限流：每分钟执行次数
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
			Output:     "file",
			FilePath:   "log/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Async:      true,
			Buffer:     1024,
			DropPolicy: "block",
		},
		Security: SecurityConfig{
			MaxLoginAttempts: 5,
		},
		Admin: AdminConfig{
			Usernames:       []string{"admin"}, // 默认管理员
			DefaultPassword: "admin123",        // 默认密码
		},
		CORS: CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
			AllowCredentials: true,
		},
		Assets: AssetsConfig{
			PublicBaseURL: getEnv("ASSETS_PUBLIC_BASE_URL", "http://localhost:9000/community-assets"),
			MaxAvatarSizeMB: func() int {
				if v := getEnv("ASSETS_MAX_AVATAR_MB", ""); v != "" {
					n := parseInt(v)
					if n > 0 {
						return n
					}
				}
				return 5
			}(),
			MaxAvatarHistory: func() int {
				if v := getEnv("ASSETS_MAX_AVATAR_HISTORY", ""); v != "" {
					n := parseInt(v)
					if n > 0 {
						return n
					}
				}
				return 9
			}(),
		},
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:          strings.ToLower(getEnv("MINIO_USE_SSL", "false")) == "true" || getEnv("MINIO_USE_SSL", "false") == "1",
			Bucket:          getEnv("MINIO_BUCKET", "community-assets"),
		},
		ResourcesStorage: ResourcesStorageConfig{
			Bucket:        getEnv("RESOURCES_BUCKET", "community-resources"),
			PublicBaseURL: getEnv("RESOURCES_PUBLIC_BASE_URL", "http://127.0.0.1:9000/community-resources"),
		},
		CodeExecutor: CodeExecutorConfig{
			PistonAPIURL: getEnv("PISTON_API_URL", "https://emkc.org/api/v2/piston"),
			Timeout: func() int {
				if v := getEnv("CODE_EXECUTOR_TIMEOUT", ""); v != "" {
					n := parseInt(v)
					if n > 0 {
						return n
					}
				}
				return 10
			}(),
			MaxMemoryMB: func() int {
				if v := getEnv("CODE_EXECUTOR_MAX_MEMORY", ""); v != "" {
					n := parseInt(v)
					if n > 0 {
						return n
					}
				}
				return 128
			}(),
			RateLimit: func() int {
				if v := getEnv("CODE_EXECUTOR_RATE_LIMIT", ""); v != "" {
					n := parseInt(v)
					if n > 0 {
						return n
					}
				}
				return 10
			}(),
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
	// 使用辅助函数简化配置覆盖
	setEnvString := func(target *string, envKey string) {
		if val := getEnv(envKey, ""); val != "" {
			*target = val
		}
	}

	setEnvInt := func(target *int, envKey string) {
		if val := getEnv(envKey, ""); val != "" {
			if n := parseInt(val); n > 0 {
				*target = n
			}
		}
	}

	setEnvBool := func(target *bool, envKey string) {
		if val := getEnv(envKey, ""); val != "" {
			*target = strings.ToLower(val) == "true" || val == "1"
		}
	}

	// 服务器配置
	setEnvString(&config.Server.Host, "SERVER_HOST")
	setEnvString(&config.Server.Port, "SERVER_PORT")
	setEnvString(&config.Server.Mode, "SERVER_MODE")

	// 数据库配置
	setEnvString(&config.Database.Host, "DB_HOST")
	setEnvString(&config.Database.Port, "DB_PORT")
	setEnvString(&config.Database.Username, "DB_USERNAME")
	setEnvString(&config.Database.Password, "DB_PASSWORD")
	setEnvString(&config.Database.Database, "DB_DATABASE")

	// JWT配置
	setEnvString(&config.JWT.SecretKey, "JWT_SECRET")
	setEnvInt(&config.JWT.ExpireHours, "JWT_EXPIRE_HOURS")

	// 日志配置
	setEnvString(&config.Log.Level, "LOG_LEVEL")
	setEnvString(&config.Log.Format, "LOG_FORMAT")
	setEnvString(&config.Log.Output, "LOG_OUTPUT")

	// 静态资源配置
	setEnvString(&config.Assets.PublicBaseURL, "ASSETS_PUBLIC_BASE_URL")
	setEnvInt(&config.Assets.MaxAvatarSizeMB, "ASSETS_MAX_AVATAR_MB")

	// MinIO 配置
	setEnvString(&config.MinIO.Endpoint, "MINIO_ENDPOINT")
	setEnvString(&config.MinIO.AccessKeyID, "MINIO_ACCESS_KEY")
	setEnvString(&config.MinIO.SecretAccessKey, "MINIO_SECRET_KEY")
	setEnvBool(&config.MinIO.UseSSL, "MINIO_USE_SSL")
	setEnvString(&config.MinIO.Bucket, "MINIO_BUCKET")

	// 资源存储配置
	setEnvString(&config.ResourcesStorage.Bucket, "RESOURCES_BUCKET")
	setEnvString(&config.ResourcesStorage.PublicBaseURL, "RESOURCES_PUBLIC_BASE_URL")

	// 代码执行器配置
	setEnvString(&config.CodeExecutor.PistonAPIURL, "PISTON_API_URL")
	setEnvInt(&config.CodeExecutor.Timeout, "CODE_EXECUTOR_TIMEOUT")
	setEnvInt(&config.CodeExecutor.MaxMemoryMB, "CODE_EXECUTOR_MAX_MEMORY")
	setEnvInt(&config.CodeExecutor.RateLimit, "CODE_EXECUTOR_RATE_LIMIT")
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
