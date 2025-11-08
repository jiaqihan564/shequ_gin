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
	App              AppConfig              `yaml:"app" json:"app"`
	Server           ServerConfig           `yaml:"server" json:"server"`
	JWT              JWTConfig              `yaml:"jwt" json:"jwt"`
	Database         DatabaseConfig         `yaml:"database" json:"database"`
	Log              LogConfig              `yaml:"log" json:"log"`
	Security         SecurityConfig         `yaml:"security" json:"security"`
	Admin            AdminConfig            `yaml:"admin" json:"admin"`
	CORS             CORSConfig             `yaml:"cors" json:"cors"`
	MinIO            MinIOConfig            `yaml:"minio" json:"minio"`
	// 7桶架构配置
	BucketUserAvatars       BucketConfig                  `yaml:"bucket_user_avatars" json:"bucket_user_avatars"`
	BucketResourceChunks    BucketConfig                  `yaml:"bucket_resource_chunks" json:"bucket_resource_chunks"`
	BucketResourcePreviews  BucketConfig                  `yaml:"bucket_resource_previews" json:"bucket_resource_previews"`
	BucketDocumentImages    BucketConfig                  `yaml:"bucket_document_images" json:"bucket_document_images"`
	BucketArticleImages     BucketConfig                  `yaml:"bucket_article_images" json:"bucket_article_images"`
	BucketTempFiles         BucketConfig                  `yaml:"bucket_temp_files" json:"bucket_temp_files"`
	BucketSystemAssets      BucketConfig                  `yaml:"bucket_system_assets" json:"bucket_system_assets"`
	CodeExecutor            CodeExecutorConfig            `yaml:"code_executor" json:"code_executor"`
	WebSocket               WebSocketConfig               `yaml:"websocket" json:"websocket"`
	RateLimiter             RateLimiterConfig             `yaml:"rate_limiter" json:"rate_limiter"`
	Cache                   CacheConfig                   `yaml:"cache" json:"cache"`
	Validation              ValidationConfig              `yaml:"validation" json:"validation"`
	DatabaseTimeouts        DatabaseTimeoutsConfig        `yaml:"database_timeouts" json:"database_timeouts"`
	HTTPClient              HTTPClientConfig              `yaml:"http_client" json:"http_client"`
	AuthPolicy              AuthPolicyConfig              `yaml:"auth_policy" json:"auth_policy"`
	Metrics                 MetricsConfig                 `yaml:"metrics" json:"metrics"`
	AsyncTasks              AsyncTasksConfig              `yaml:"async_tasks" json:"async_tasks"`
	WorkerPool              WorkerPoolConfig              `yaml:"worker_pool" json:"worker_pool"`
	LRUCacheDefaults        LRUCacheDefaultsConfig        `yaml:"lru_cache_defaults" json:"lru_cache_defaults"`
	BatchOperations         BatchOperationsConfig         `yaml:"batch_operations" json:"batch_operations"`
	ObjectPool              ObjectPoolConfig              `yaml:"object_pool" json:"object_pool"`
	PerformanceMonitoring   PerformanceMonitoringConfig   `yaml:"performance_monitoring" json:"performance_monitoring"`
	RepositoryTimeouts      RepositoryTimeoutsConfig      `yaml:"repository_timeouts" json:"repository_timeouts"`
	FileUpload              FileUploadConfig              `yaml:"file_upload" json:"file_upload"`
	Compression             CompressionConfig             `yaml:"compression" json:"compression"`
	Pagination              PaginationConfig              `yaml:"pagination" json:"pagination"`
	ImageUpload             ImageUploadConfig             `yaml:"image_upload" json:"image_upload"`
	AvatarUpload            AvatarUploadConfig            `yaml:"avatar_upload" json:"avatar_upload"`
	DatabaseQuery           DatabaseQueryConfig           `yaml:"database_query" json:"database_query"`
	RepositoryDefaults      RepositoryDefaultsConfig      `yaml:"repository_defaults" json:"repository_defaults"`
	StatisticsQuery         StatisticsQueryConfig         `yaml:"statistics_query" json:"statistics_query"`
	LogAdvanced             LogAdvancedConfig             `yaml:"log_advanced" json:"log_advanced"`
	MetricsCapacity         MetricsCapacityConfig         `yaml:"metrics_capacity" json:"metrics_capacity"`
	Profiler                ProfilerConfig                `yaml:"profiler" json:"profiler"`
	LogExtended             LogExtendedConfig             `yaml:"log_extended" json:"log_extended"`
	SecurityHeaders         SecurityHeadersConfig         `yaml:"security_headers" json:"security_headers"`
	SecurityPassword        SecurityPasswordConfig        `yaml:"security_password" json:"security_password"`
	SecuritySQL             SecuritySQLConfig             `yaml:"security_sql" json:"security_sql"`
	ValidationExtended      ValidationExtendedConfig      `yaml:"validation_extended" json:"validation_extended"`
	JWTExtended             JWTExtendedConfig             `yaml:"jwt_extended" json:"jwt_extended"`
	DateTimeFormats         DateTimeFormatsConfig         `yaml:"date_time_formats" json:"date_time_formats"`
	MinioAdvanced           MinioAdvancedConfig           `yaml:"minio_advanced" json:"minio_advanced"`
	DatabaseQueryAdvanced   DatabaseQueryAdvancedConfig   `yaml:"database_query_advanced" json:"database_query_advanced"`
	StatisticsQueryExtended StatisticsQueryExtendedConfig `yaml:"statistics_query_extended" json:"statistics_query_extended"`
}

// AppConfig 应用信息配置
type AppConfig struct {
	Name    string `yaml:"name" json:"name"`       // 应用名称
	Version string `yaml:"version" json:"version"` // 应用版本
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host                     string `yaml:"host" json:"host"`
	Port                     string `yaml:"port" json:"port"`
	Mode                     string `yaml:"mode" json:"mode"`
	MaxHeaderBytes           int    `yaml:"max_header_bytes" json:"max_header_bytes"`                       // 最大请求头大小（字节）
	ReadTimeout              int    `yaml:"read_timeout" json:"read_timeout"`                               // 读取超时（秒）
	WriteTimeout             int    `yaml:"write_timeout" json:"write_timeout"`                             // 写入超时（秒）
	IdleTimeout              int    `yaml:"idle_timeout" json:"idle_timeout"`                               // 空闲超时（秒）
	ReadHeaderTimeout        int    `yaml:"read_header_timeout" json:"read_header_timeout"`                 // 读取请求头超时（秒）
	ShutdownTimeout          int    `yaml:"shutdown_timeout" json:"shutdown_timeout"`                       // 优雅关闭超时（秒）
	StartupHealthCheckDelay  int    `yaml:"startup_health_check_delay" json:"startup_health_check_delay"`   // 启动后健康检查延迟（毫秒）
	HealthCheckClientTimeout int    `yaml:"health_check_client_timeout" json:"health_check_client_timeout"` // 健康检查客户端超时（秒）
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
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"` // 空闲连接最大存活时间
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
	MaxRequestSizeMB int `yaml:"max_request_size_mb" json:"max_request_size_mb"` // 最大请求体大小（MB）
}

// AdminConfig 管理员配置
type AdminConfig struct {
	Usernames       []string `yaml:"usernames" json:"usernames"`
	DefaultPassword string   `yaml:"default_password" json:"default_password"` // 管理员默认密码（首次创建时使用）
	EmailSuffix     string   `yaml:"email_suffix" json:"email_suffix"`         // 管理员邮箱后缀
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
}

// MinIOConfig MinIO 对象存储连接配置
type MinIOConfig struct {
	Endpoint         string `yaml:"endpoint" json:"endpoint"`
	AccessKeyID      string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key" json:"secret_access_key"`
	UseSSL           bool   `yaml:"use_ssl" json:"use_ssl"`
	OperationTimeout int    `yaml:"operation_timeout" json:"operation_timeout"` // 操作超时（秒）
}

// BucketConfig 通用桶配置（7桶架构）
type BucketConfig struct {
	Name                 string  `yaml:"name" json:"name"`                                       // 桶名称
	PublicBaseURL        string  `yaml:"public_base_url" json:"public_base_url"`                 // 公共访问基础URL
	MaxAvatarSizeMB      float64 `yaml:"max_avatar_size_mb" json:"max_avatar_size_mb"`           // 头像最大大小（仅user-avatars）
	MaxHistory           int     `yaml:"max_history" json:"max_history"`                         // 历史版本数（仅user-avatars）
	AutoCleanup          bool    `yaml:"auto_cleanup" json:"auto_cleanup"`                       // 是否自动清理
	ChunkSizeMB          int     `yaml:"chunk_size_mb" json:"chunk_size_mb"`                     // 分片大小（仅resource-chunks）
	MaxImageSizeKB       int     `yaml:"max_image_size_kb" json:"max_image_size_kb"`             // 图片最大大小
	MaxImagesPerResource int     `yaml:"max_images_per_resource" json:"max_images_per_resource"` // 每个资源最大图片数
	ArchiveAfterDays     int     `yaml:"archive_after_days" json:"archive_after_days"`           // 多少天后归档
	AutoExpireHours      int     `yaml:"auto_expire_hours" json:"auto_expire_hours"`             // 自动过期时间（仅temp-files）
	PublicRead           *bool   `yaml:"public_read" json:"public_read"`                         // 是否公开读取（nil=默认true）
	CacheControl         string  `yaml:"cache_control" json:"cache_control"`                     // 缓存控制头
}

// CodeExecutorConfig 代码执行器配置
type CodeExecutorConfig struct {
	PistonAPIURL string `yaml:"piston_api_url" json:"piston_api_url"`
	Timeout      int    `yaml:"timeout" json:"timeout"`             // 超时时间（秒）
	MaxMemoryMB  int    `yaml:"max_memory_mb" json:"max_memory_mb"` // 最大内存（MB）
	RateLimit    int    `yaml:"rate_limit" json:"rate_limit"`       // 限流：每分钟执行次数
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	WriteWait            int `yaml:"write_wait" json:"write_wait"`                           // 写操作超时（秒）
	PongWait             int `yaml:"pong_wait" json:"pong_wait"`                             // Pong等待超时（秒）
	PingPeriod           int `yaml:"ping_period" json:"ping_period"`                         // Ping间隔（秒）
	MaxMessageSize       int `yaml:"max_message_size" json:"max_message_size"`               // 最大消息大小（字节）
	MaxMessageLength     int `yaml:"max_message_length" json:"max_message_length"`           // 最大消息长度（字符数）
	MaxMessagesPerSecond int `yaml:"max_messages_per_second" json:"max_messages_per_second"` // 每秒最大消息数
	ReadBufferSize       int `yaml:"read_buffer_size" json:"read_buffer_size"`               // 读缓冲区大小（字节）
	WriteBufferSize      int `yaml:"write_buffer_size" json:"write_buffer_size"`             // 写缓冲区大小（字节）
	BroadcastBufferSize  int `yaml:"broadcast_buffer_size" json:"broadcast_buffer_size"`     // 广播channel缓冲区大小
	ClientSendBufferSize int `yaml:"client_send_buffer_size" json:"client_send_buffer_size"` // 客户端发送channel缓冲区大小
}

// RateLimiterItemConfig 限流器单项配置
type RateLimiterItemConfig struct {
	Capacity          int `yaml:"capacity" json:"capacity"`                       // 令牌桶容量
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"` // 每分钟请求数
	MaxCacheSize      int `yaml:"max_cache_size" json:"max_cache_size"`           // LRU缓存最大IP数
}

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	Global          RateLimiterItemConfig `yaml:"global" json:"global"`                       // 全局API限流
	Login           RateLimiterItemConfig `yaml:"login" json:"login"`                         // 登录限流
	Register        RateLimiterItemConfig `yaml:"register" json:"register"`                   // 注册限流
	CleanupInterval int                   `yaml:"cleanup_interval" json:"cleanup_interval"`   // 清理间隔（分钟）
	EntryExpireTime int                   `yaml:"entry_expire_time" json:"entry_expire_time"` // 条目过期时间（分钟）
}

// CacheItemConfig 缓存单项配置
type CacheItemConfig struct {
	Capacity    int `yaml:"capacity" json:"capacity"`           // 容量
	MaxMemoryMB int `yaml:"max_memory_mb" json:"max_memory_mb"` // 最大内存（MB）
	TTLMinutes  int `yaml:"ttl_minutes" json:"ttl_minutes"`     // 缓存有效期（分钟）
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Article                 CacheItemConfig `yaml:"article" json:"article"`                                       // 文章缓存
	User                    CacheItemConfig `yaml:"user" json:"user"`                                             // 用户缓存
	List                    CacheItemConfig `yaml:"list" json:"list"`                                             // 列表缓存
	CategoriesTTLMinutes    int             `yaml:"categories_ttl_minutes" json:"categories_ttl_minutes"`         // 分类缓存有效期（分钟）
	TagsTTLMinutes          int             `yaml:"tags_ttl_minutes" json:"tags_ttl_minutes"`                     // 标签缓存有效期（分钟）
	ArticleDetailTTLMinutes int             `yaml:"article_detail_ttl_minutes" json:"article_detail_ttl_minutes"` // 文章详情缓存有效期（分钟）
	OnlineCountTTLSeconds   int             `yaml:"online_count_ttl_seconds" json:"online_count_ttl_seconds"`     // 在线人数缓存有效期（秒）
	WarmupTimeout           int             `yaml:"warmup_timeout" json:"warmup_timeout"`                         // 缓存预热超时（秒）
}

// ValidationUsernameConfig 用户名验证配置
type ValidationUsernameConfig struct {
	MinLength int `yaml:"min_length" json:"min_length"` // 最小长度
	MaxLength int `yaml:"max_length" json:"max_length"` // 最大长度
}

// ValidationPasswordConfig 密码验证配置
type ValidationPasswordConfig struct {
	MinLength      int `yaml:"min_length" json:"min_length"`             // 最小长度
	MaxLength      int `yaml:"max_length" json:"max_length"`             // 最大长度（注册）
	MaxLengthLogin int `yaml:"max_length_login" json:"max_length_login"` // 最大长度（登录）
}

// ValidationNicknameConfig 昵称验证配置
type ValidationNicknameConfig struct {
	MinLength int `yaml:"min_length" json:"min_length"` // 最小长度
	MaxLength int `yaml:"max_length" json:"max_length"` // 最大长度
}

// ValidationBioConfig 简介验证配置
type ValidationBioConfig struct {
	MaxLength int `yaml:"max_length" json:"max_length"` // 最大长度
}

// ValidationPhoneConfig 手机号验证配置
type ValidationPhoneConfig struct {
	Length int `yaml:"length" json:"length"` // 手机号长度
}

// ValidationConfig 验证规则配置
type ValidationConfig struct {
	Username ValidationUsernameConfig `yaml:"username" json:"username"` // 用户名验证
	Password ValidationPasswordConfig `yaml:"password" json:"password"` // 密码验证
	Nickname ValidationNicknameConfig `yaml:"nickname" json:"nickname"` // 昵称验证
	Bio      ValidationBioConfig      `yaml:"bio" json:"bio"`           // 简介验证
	Phone    ValidationPhoneConfig    `yaml:"phone" json:"phone"`       // 手机号验证
}

// DatabaseTimeoutsConfig 数据库超时配置
type DatabaseTimeoutsConfig struct {
	ConnectionTimeout       int `yaml:"connection_timeout" json:"connection_timeout"`               // 连接超时（秒）
	ReadTimeout             int `yaml:"read_timeout" json:"read_timeout"`                           // 读取超时（秒）
	WriteTimeout            int `yaml:"write_timeout" json:"write_timeout"`                         // 写入超时（秒）
	PoolMonitorInterval     int `yaml:"pool_monitor_interval" json:"pool_monitor_interval"`         // 连接池监控间隔（分钟）
	PingTimeout             int `yaml:"ping_timeout" json:"ping_timeout"`                           // Ping超时（秒）
	TestConnectionTimeout   int `yaml:"test_connection_timeout" json:"test_connection_timeout"`     // 连接测试超时（秒）
	WarmupConnectionTimeout int `yaml:"warmup_connection_timeout" json:"warmup_connection_timeout"` // 连接预热超时（秒）
}

// HTTPClientConfig HTTP客户端配置
type HTTPClientConfig struct {
	MaxIdleConns        int `yaml:"max_idle_conns" json:"max_idle_conns"`                   // 最大空闲连接数
	MaxIdleConnsPerHost int `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"` // 每个host最大空闲连接数
	IdleConnTimeout     int `yaml:"idle_conn_timeout" json:"idle_conn_timeout"`             // 空闲连接超时（秒）
}

// AuthPolicyConfig 认证策略配置
type AuthPolicyConfig struct {
	PasswordResetTokenExpireMinutes int `yaml:"password_reset_token_expire_minutes" json:"password_reset_token_expire_minutes"` // 密码重置token有效期（分钟）
	ResetTokenBytes                 int `yaml:"reset_token_bytes" json:"reset_token_bytes"`                                     // 重置token字节数
	AsyncTaskTimeout                int `yaml:"async_task_timeout" json:"async_task_timeout"`                                   // 异步任务超时（秒）
}

// MetricsConfig 实时指标配置
type MetricsConfig struct {
	OnlineUsersInitialCapacity int `yaml:"online_users_initial_capacity" json:"online_users_initial_capacity"` // 在线用户map初始容量
	OnlineUserCleanupInterval  int `yaml:"online_user_cleanup_interval" json:"online_user_cleanup_interval"`   // 清理间隔（分钟）
	OnlineUserExpireTime       int `yaml:"online_user_expire_time" json:"online_user_expire_time"`             // 用户过期时间（分钟）
	CPUGoroutineBaseline       int `yaml:"cpu_goroutine_baseline" json:"cpu_goroutine_baseline"`               // CPU估算基准Goroutine数
}

// AsyncTasksConfig 异步任务超时配置
type AsyncTasksConfig struct {
	ResourceViewCountTimeout     int `yaml:"resource_view_count_timeout" json:"resource_view_count_timeout"`         // 资源浏览计数超时（秒）
	ResourceDownloadCountTimeout int `yaml:"resource_download_count_timeout" json:"resource_download_count_timeout"` // 资源下载计数超时（秒）
	UploadHistoryTimeout         int `yaml:"upload_history_timeout" json:"upload_history_timeout"`                   // 上传历史记录超时（秒）
	AvatarCleanupTimeout         int `yaml:"avatar_cleanup_timeout" json:"avatar_cleanup_timeout"`                   // 头像清理超时（秒）
	AvatarOperationTimeout       int `yaml:"avatar_operation_timeout" json:"avatar_operation_timeout"`               // 头像操作超时（秒）
	UserUpdateHistoryTimeout     int `yaml:"user_update_history_timeout" json:"user_update_history_timeout"`         // 用户更新历史超时（秒）
	MessageMarkReadTimeout       int `yaml:"message_mark_read_timeout" json:"message_mark_read_timeout"`             // 标记消息已读超时（秒）
	ArticleViewCountTimeout      int `yaml:"article_view_count_timeout" json:"article_view_count_timeout"`           // 文章浏览计数超时（秒）
}

// WorkerPoolConfig Worker Pool配置
type WorkerPoolConfig struct {
	Workers            int `yaml:"workers" json:"workers"`                           // worker数量
	QueueSize          int `yaml:"queue_size" json:"queue_size"`                     // 任务队列大小
	DefaultTaskTimeout int `yaml:"default_task_timeout" json:"default_task_timeout"` // 默认任务超时（秒）
}

// LRUCacheDefaultsConfig LRU缓存默认配置
type LRUCacheDefaultsConfig struct {
	Capacity        int `yaml:"capacity" json:"capacity"`                 // 默认最大容量
	MaxMemoryMB     int `yaml:"max_memory_mb" json:"max_memory_mb"`       // 默认最大内存（MB）
	TTLMinutes      int `yaml:"ttl_minutes" json:"ttl_minutes"`           // 默认TTL（分钟）
	CleanupInterval int `yaml:"cleanup_interval" json:"cleanup_interval"` // 过期清理间隔（分钟）
}

// BatchOperationsConfig 批量操作配置
type BatchOperationsConfig struct {
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"` // 批量查询最大并发数
}

// ObjectPoolConfig 对象池配置
type ObjectPoolConfig struct {
	MapInitialCapacity int `yaml:"map_initial_capacity" json:"map_initial_capacity"` // map池初始容量
	MagicBufferSize    int `yaml:"magic_buffer_size" json:"magic_buffer_size"`       // 文件魔数验证buffer大小
}

// PerformanceMonitoringConfig 性能监控配置
type PerformanceMonitoringConfig struct {
	SampleRate             int     `yaml:"sample_rate" json:"sample_rate"`                             // 采样率（%）
	MemoryGrowthWarningMB  int     `yaml:"memory_growth_warning_mb" json:"memory_growth_warning_mb"`   // 内存增长警告阈值（MB）
	GoroutineGrowthWarning int     `yaml:"goroutine_growth_warning" json:"goroutine_growth_warning"`   // Goroutine增长警告阈值
	DBPoolWarningThreshold float64 `yaml:"db_pool_warning_threshold" json:"db_pool_warning_threshold"` // 数据库连接池警告阈值
	VerySlowRequestMS      int     `yaml:"very_slow_request_ms" json:"very_slow_request_ms"`           // 非常慢请求阈值（毫秒）
	SlowRequestMS          int     `yaml:"slow_request_ms" json:"slow_request_ms"`                     // 慢请求阈值（毫秒）
	NormalRequestLogMS     int     `yaml:"normal_request_log_ms" json:"normal_request_log_ms"`         // 正常请求日志阈值（毫秒）
}

// RepositoryTimeoutsConfig Repository操作超时配置
type RepositoryTimeoutsConfig struct {
	DefaultQueryTimeout  int `yaml:"default_query_timeout" json:"default_query_timeout"`   // 默认查询操作超时（秒）
	DefaultUpdateTimeout int `yaml:"default_update_timeout" json:"default_update_timeout"` // 默认更新操作超时（秒）
	ChatOperationTimeout int `yaml:"chat_operation_timeout" json:"chat_operation_timeout"` // 聊天操作超时（秒）
	UserQueryTimeout     int `yaml:"user_query_timeout" json:"user_query_timeout"`         // 用户查询超时（秒）
	UserUpdateTimeout    int `yaml:"user_update_timeout" json:"user_update_timeout"`       // 用户更新超时（秒）
}

// FileUploadConfig 文件上传配置
type FileUploadConfig struct {
	ChunkSizeMB       int `yaml:"chunk_size_mb" json:"chunk_size_mb"`               // 分片大小（MB）
	UploadExpireHours int `yaml:"upload_expire_hours" json:"upload_expire_hours"`   // 上传任务过期时间（小时）
	MaxResourceSizeMB int `yaml:"max_resource_size_mb" json:"max_resource_size_mb"` // 资源文件最大大小（MB）
}

// CompressionConfig 压缩配置
type CompressionConfig struct {
	MinSizeBytes int `yaml:"min_size_bytes" json:"min_size_bytes"` // 最小压缩大小（字节）
	Level        int `yaml:"level" json:"level"`                   // 压缩级别（1-9）
}

// PaginationConfig 分页配置
type PaginationConfig struct {
	DefaultPageSize      int `yaml:"default_page_size" json:"default_page_size"`             // 默认每页大小
	MaxPageSize          int `yaml:"max_page_size" json:"max_page_size"`                     // 最大每页大小
	DefaultLimit         int `yaml:"default_limit" json:"default_limit"`                     // 默认限制数量
	MaxLimit             int `yaml:"max_limit" json:"max_limit"`                             // 最大限制数量
	HistoryDefaultLimit  int `yaml:"history_default_limit" json:"history_default_limit"`     // 历史记录默认限制
	AvatarHistoryMaxList int `yaml:"avatar_history_max_list" json:"avatar_history_max_list"` // 头像历史列表最大数量
}

// ImageUploadConfig 图片上传配置
type ImageUploadConfig struct {
	MaxSizeMB int `yaml:"max_size_mb" json:"max_size_mb"` // 文档和资源图片最大大小（MB）
}

// AvatarUploadConfig 头像上传配置（前端已裁剪和压缩）
type AvatarUploadConfig struct {
	UploadRateLimit int `yaml:"upload_rate_limit" json:"upload_rate_limit"` // 每分钟最大上传次数
}

// DatabaseQueryConfig 数据库查询配置
type DatabaseQueryConfig struct {
	SlowQueryThresholdMS int `yaml:"slow_query_threshold_ms" json:"slow_query_threshold_ms"` // 慢查询阈值（毫秒）
	IdleTimeoutMinutes   int `yaml:"idle_timeout_minutes" json:"idle_timeout_minutes"`       // 空闲连接超时（分钟）
	RetryWaitMS          int `yaml:"retry_wait_ms" json:"retry_wait_ms"`                     // 连接重试等待（毫秒）
	RetryBackoffBaseMS   int `yaml:"retry_backoff_base_ms" json:"retry_backoff_base_ms"`     // 重试退避基数（毫秒）
}

// RepositoryDefaultsConfig Repository操作默认配置
type RepositoryDefaultsConfig struct {
	QuickOperationTimeout  int `yaml:"quick_operation_timeout" json:"quick_operation_timeout"`   // 快速操作(查询)超时（秒）
	NormalOperationTimeout int `yaml:"normal_operation_timeout" json:"normal_operation_timeout"` // 普通操作超时（秒）
}

// StatisticsQueryConfig 统计查询配置
type StatisticsQueryConfig struct {
	ApiRankingDefault  int `yaml:"api_ranking_default" json:"api_ranking_default"`   // API排行榜默认数量
	TagsListLimit      int `yaml:"tags_list_limit" json:"tags_list_limit"`           // 标签列表限制
	ChatMessagesBuffer int `yaml:"chat_messages_buffer" json:"chat_messages_buffer"` // 聊天消息缓冲区大小
}

// LogAdvancedConfig 日志配置扩展
type LogAdvancedConfig struct {
	FlushWaitMS int `yaml:"flush_wait_ms" json:"flush_wait_ms"` // 日志刷新等待时间（毫秒）
}

// MetricsCapacityConfig Metrics初始容量配置
type MetricsCapacityConfig struct {
	ActiveUsersInitial   int `yaml:"active_users_initial" json:"active_users_initial"`     // 活跃用户map初始容量
	EndpointCallsInitial int `yaml:"endpoint_calls_initial" json:"endpoint_calls_initial"` // 端点调用计数map初始容量
}

// ProfilerConfig 性能分析器配置
type ProfilerConfig struct {
	LatencyMaxRecords      int `yaml:"latency_max_records" json:"latency_max_records"`           // 延迟记录最大数量
	LatencyCleanupRatio    int `yaml:"latency_cleanup_ratio" json:"latency_cleanup_ratio"`       // 延迟记录清理百分比
	GoroutineLeakThreshold int `yaml:"goroutine_leak_threshold" json:"goroutine_leak_threshold"` // Goroutine泄露阈值
	SlowQueryMaxRecords    int `yaml:"slow_query_max_records" json:"slow_query_max_records"`     // 慢查询最大记录数
	SlowQueryCleanupRatio  int `yaml:"slow_query_cleanup_ratio" json:"slow_query_cleanup_ratio"` // 慢查询记录清理百分比
	SlowQueryThresholdMS   int `yaml:"slow_query_threshold_ms" json:"slow_query_threshold_ms"`   // 慢查询阈值（毫秒）
}

// LogExtendedConfig 日志扩展配置
type LogExtendedConfig struct {
	SkipPaths               []string `yaml:"skip_paths" json:"skip_paths"`                                 // 跳过详细日志的路径列表
	RequestBodyTruncateSize int      `yaml:"request_body_truncate_size" json:"request_body_truncate_size"` // 请求体截断大小（字节）
	SampleRateProduction    int      `yaml:"sample_rate_production" json:"sample_rate_production"`         // 生产环境采样率(%)
	SampleRateDevelopment   int      `yaml:"sample_rate_development" json:"sample_rate_development"`       // 开发环境采样率(%)
}

// SecurityHeadersConfig 安全响应头配置
type SecurityHeadersConfig struct {
	XFrameOptions         string `yaml:"x_frame_options" json:"x_frame_options"`                 // X-Frame-Options
	XContentTypeOptions   string `yaml:"x_content_type_options" json:"x_content_type_options"`   // X-Content-Type-Options
	XXSSProtection        string `yaml:"x_xss_protection" json:"x_xss_protection"`               // X-XSS-Protection
	ContentSecurityPolicy string `yaml:"content_security_policy" json:"content_security_policy"` // Content-Security-Policy
	ReferrerPolicy        string `yaml:"referrer_policy" json:"referrer_policy"`                 // Referrer-Policy
	PermissionsPolicy     string `yaml:"permissions_policy" json:"permissions_policy"`           // Permissions-Policy
	EnableHSTS            bool   `yaml:"enable_hsts" json:"enable_hsts"`                         // 启用HSTS（生产环境推荐）
	HSTSMaxAge            int    `yaml:"hsts_max_age" json:"hsts_max_age"`                       // HSTS有效期（秒）
}

// SecurityPasswordConfig 密码加密配置
type SecurityPasswordConfig struct {
	BcryptCostMin    int `yaml:"bcrypt_cost_min" json:"bcrypt_cost_min"`       // bcrypt最小成本
	BcryptCostMax    int `yaml:"bcrypt_cost_max" json:"bcrypt_cost_max"`       // bcrypt最大成本
	PasswordMaxBytes int `yaml:"password_max_bytes" json:"password_max_bytes"` // 密码最大字节数（bcrypt限制）
}

// SecuritySQLConfig SQL注入防护配置
type SecuritySQLConfig struct {
	KeywordsBlacklist []string `yaml:"keywords_blacklist" json:"keywords_blacklist"` // SQL关键词黑名单
}

// ValidationExtendedConfig 验证规则扩展配置
type ValidationExtendedConfig struct {
	URLMinLength           int    `yaml:"url_min_length" json:"url_min_length"`                     // URL最小长度
	PhoneFirstDigit        string `yaml:"phone_first_digit" json:"phone_first_digit"`               // 手机号首位数字
	PhoneSecondDigitMin    string `yaml:"phone_second_digit_min" json:"phone_second_digit_min"`     // 手机号第二位最小值
	PhoneSecondDigitMax    string `yaml:"phone_second_digit_max" json:"phone_second_digit_max"`     // 手机号第二位最大值
	ResourceTitleMax       int    `yaml:"resource_title_max" json:"resource_title_max"`             // 资源标题最大长度
	ResourceDescriptionMax int    `yaml:"resource_description_max" json:"resource_description_max"` // 资源描述最大长度
	ArticleTitleMax        int    `yaml:"article_title_max" json:"article_title_max"`               // 文章标题最大长度
	ArticleDescriptionMax  int    `yaml:"article_description_max" json:"article_description_max"`   // 文章描述最大长度
	CommentContentMax      int    `yaml:"comment_content_max" json:"comment_content_max"`           // 评论内容最大长度
}

// JWTExtendedConfig JWT扩展配置
type JWTExtendedConfig struct {
	TokenPrefix string `yaml:"token_prefix" json:"token_prefix"` // Token前缀（例如 "Bearer "）
}

// DateTimeFormatsConfig 日期时间格式配置
type DateTimeFormatsConfig struct {
	DateOnly     string `yaml:"date_only" json:"date_only"`         // 仅日期格式
	DatetimeFull string `yaml:"datetime_full" json:"datetime_full"` // 完整日期时间格式
	RFC3339      string `yaml:"rfc3339" json:"rfc3339"`             // RFC3339格式
}

// MinioAdvancedConfig MinIO高级配置
type MinioAdvancedConfig struct {
	PolicyVersion string `yaml:"policy_version" json:"policy_version"` // S3策略版本号
	PolicyEffect  string `yaml:"policy_effect" json:"policy_effect"`   // 策略效果
	PolicyAction  string `yaml:"policy_action" json:"policy_action"`   // 策略允许的操作
}

// DatabaseQueryAdvancedConfig 数据库查询高级配置
type DatabaseQueryAdvancedConfig struct {
	QueryLogTruncateLength int `yaml:"query_log_truncate_length" json:"query_log_truncate_length"` // 查询日志截断长度
	PreparedStmtCacheSize  int `yaml:"prepared_stmt_cache_size" json:"prepared_stmt_cache_size"`   // Prepared Statement 缓存大小限制（默认1000）
}

// StatisticsQueryExtendedConfig 统计查询扩展配置
type StatisticsQueryExtendedConfig struct {
	DefaultDateRangeDays int `yaml:"default_date_range_days" json:"default_date_range_days"` // 默认查询日期范围（天数）
}

// Load 加载配置（优先级：环境变量 > 配置文件 > 默认值）
func Load() *Config {
	// 获取环境变量
	env := getEnv("APP_ENV", "dev")
	configFile := getConfigFile(env)

	// 创建默认配置
	config := getDefaultConfig()

	// 从配置文件加载
	if configFile != "" {
		if err := loadFromFile(config, configFile); err != nil {
			fmt.Printf("Warning: Failed to load config file %s: %v\n", configFile, err)
		} else {
			fmt.Printf("Loaded configuration from: %s\n", configFile)
		}
	}

	// 使用环境变量覆盖
	overrideWithEnvVars(config)

	// 验证配置
	if err := config.Validate(); err != nil {
		fmt.Printf("Warning: Configuration validation failed: %v\n", err)
	}

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
		App: AppConfig{
			Name:    "Community API",
			Version: "1.0.0",
		},
		Server: ServerConfig{
			Host:                     "localhost",
			Port:                     "8080",
			Mode:                     "release",
			MaxHeaderBytes:           1048576, // 1MB
			ReadTimeout:              30,
			WriteTimeout:             30,
			IdleTimeout:              120,
			ReadHeaderTimeout:        10,
			ShutdownTimeout:          30,
			StartupHealthCheckDelay:  500, // 500ms
			HealthCheckClientTimeout: 3,
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
			ConnMaxIdleTime: 5 * time.Minute, // 默认空闲5分钟
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
			MaxRequestSizeMB: 10,
		},
		Admin: AdminConfig{
			Usernames:       []string{"admin"}, // 默认管理员
			DefaultPassword: "admin123",        // 默认密码
			EmailSuffix:     "@admin.local",    // 管理员邮箱后缀
		},
		CORS: CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
			AllowCredentials: true,
		},
		MinIO: MinIOConfig{
			Endpoint:         getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:           strings.ToLower(getEnv("MINIO_USE_SSL", "false")) == "true" || getEnv("MINIO_USE_SSL", "false") == "1",
			OperationTimeout: 10,
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
		WebSocket: WebSocketConfig{
			WriteWait:            10,
			PongWait:             60,
			PingPeriod:           30,
			MaxMessageSize:       4096,
			MaxMessageLength:     500,
			MaxMessagesPerSecond: 3,
			ReadBufferSize:       1024,
			WriteBufferSize:      1024,
			BroadcastBufferSize:  256,
			ClientSendBufferSize: 256,
		},
		RateLimiter: RateLimiterConfig{
			Global: RateLimiterItemConfig{
				Capacity:          100,
				RequestsPerMinute: 100,
				MaxCacheSize:      10000,
			},
			Login: RateLimiterItemConfig{
				Capacity:          5,
				RequestsPerMinute: 5,
				MaxCacheSize:      1000,
			},
			Register: RateLimiterItemConfig{
				Capacity:          10,
				RequestsPerMinute: 10,
				MaxCacheSize:      1000,
			},
			CleanupInterval: 10,
			EntryExpireTime: 30,
		},
		Cache: CacheConfig{
			Article: CacheItemConfig{
				Capacity:    500,
				MaxMemoryMB: 50,
				TTLMinutes:  5,
			},
			User: CacheItemConfig{
				Capacity:    1000,
				MaxMemoryMB: 10,
				TTLMinutes:  10,
			},
			List: CacheItemConfig{
				Capacity:    100,
				MaxMemoryMB: 20,
				TTLMinutes:  2,
			},
			CategoriesTTLMinutes:    60,
			TagsTTLMinutes:          30,
			ArticleDetailTTLMinutes: 5,
			OnlineCountTTLSeconds:   10,
			WarmupTimeout:           30,
		},
		Validation: ValidationConfig{
			Username: ValidationUsernameConfig{
				MinLength: 3,
				MaxLength: 20,
			},
			Password: ValidationPasswordConfig{
				MinLength:      6,
				MaxLength:      50,
				MaxLengthLogin: 100,
			},
			Nickname: ValidationNicknameConfig{
				MinLength: 1,
				MaxLength: 50,
			},
			Bio: ValidationBioConfig{
				MaxLength: 500,
			},
			Phone: ValidationPhoneConfig{
				Length: 11,
			},
		},
		DatabaseTimeouts: DatabaseTimeoutsConfig{
			ConnectionTimeout:       10,
			ReadTimeout:             30,
			WriteTimeout:            30,
			PoolMonitorInterval:     5,
			PingTimeout:             5,
			TestConnectionTimeout:   10,
			WarmupConnectionTimeout: 3,
		},
		HTTPClient: HTTPClientConfig{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90,
		},
		AuthPolicy: AuthPolicyConfig{
			PasswordResetTokenExpireMinutes: 15,
			ResetTokenBytes:                 48,
			AsyncTaskTimeout:                10,
		},
		Metrics: MetricsConfig{
			OnlineUsersInitialCapacity: 1000,
			OnlineUserCleanupInterval:  1,
			OnlineUserExpireTime:       5,
			CPUGoroutineBaseline:       200,
		},
		AsyncTasks: AsyncTasksConfig{
			ResourceViewCountTimeout:     3,
			ResourceDownloadCountTimeout: 3,
			UploadHistoryTimeout:         5,
			AvatarCleanupTimeout:         30,
			AvatarOperationTimeout:       60,
			UserUpdateHistoryTimeout:     5,
			MessageMarkReadTimeout:       3,
			ArticleViewCountTimeout:      3,
		},
		WorkerPool: WorkerPoolConfig{
			Workers:            10,
			QueueSize:          1000,
			DefaultTaskTimeout: 30,
		},
		LRUCacheDefaults: LRUCacheDefaultsConfig{
			Capacity:        10000,
			MaxMemoryMB:     100,
			TTLMinutes:      5,
			CleanupInterval: 1,
		},
		BatchOperations: BatchOperationsConfig{
			MaxConcurrency: 10,
		},
		ObjectPool: ObjectPoolConfig{
			MapInitialCapacity: 16,
			MagicBufferSize:    16,
		},
		PerformanceMonitoring: PerformanceMonitoringConfig{
			SampleRate:             10,
			MemoryGrowthWarningMB:  10,
			GoroutineGrowthWarning: 10,
			DBPoolWarningThreshold: 0.8,
			VerySlowRequestMS:      1000,
			SlowRequestMS:          500,
			NormalRequestLogMS:     200,
		},
		RepositoryTimeouts: RepositoryTimeoutsConfig{
			DefaultQueryTimeout:  5,
			DefaultUpdateTimeout: 10,
			ChatOperationTimeout: 5,
			UserQueryTimeout:     5,
			UserUpdateTimeout:    10,
		},
		FileUpload: FileUploadConfig{
			ChunkSizeMB:       2,
			UploadExpireHours: 24,
		},
		Compression: CompressionConfig{
			MinSizeBytes: 1024,
			Level:        1,
		},
		Pagination: PaginationConfig{
			DefaultPageSize:      20,
			MaxPageSize:          100,
			DefaultLimit:         50,
			MaxLimit:             100,
			HistoryDefaultLimit:  10,
			AvatarHistoryMaxList: 50,
		},
		ImageUpload: ImageUploadConfig{
			MaxSizeMB: 5,
		},
		DatabaseQuery: DatabaseQueryConfig{
			SlowQueryThresholdMS: 50,
			IdleTimeoutMinutes:   5,
			RetryWaitMS:          200,
			RetryBackoffBaseMS:   100,
		},
		RepositoryDefaults: RepositoryDefaultsConfig{
			QuickOperationTimeout:  5,
			NormalOperationTimeout: 10,
		},
		StatisticsQuery: StatisticsQueryConfig{
			ApiRankingDefault:  10,
			TagsListLimit:      100,
			ChatMessagesBuffer: 100,
		},
		LogAdvanced: LogAdvancedConfig{
			FlushWaitMS: 100,
		},
		MetricsCapacity: MetricsCapacityConfig{
			ActiveUsersInitial:   500,
			EndpointCallsInitial: 50,
		},
		Profiler: ProfilerConfig{
			LatencyMaxRecords:      1000,
			LatencyCleanupRatio:    10,
			GoroutineLeakThreshold: 100,
			SlowQueryMaxRecords:    100,
			SlowQueryCleanupRatio:  20,
			SlowQueryThresholdMS:   50,
		},
		LogExtended: LogExtendedConfig{
			SkipPaths:               []string{"/health", "/ready", "/live", "/metrics"},
			RequestBodyTruncateSize: 512,
			SampleRateProduction:    10,
			SampleRateDevelopment:   100,
		},
		SecurityHeaders: SecurityHeadersConfig{
			XFrameOptions:         "DENY",
			XContentTypeOptions:   "nosniff",
			XXSSProtection:        "1; mode=block",
			ContentSecurityPolicy: "default-src 'self'",
			ReferrerPolicy:        "strict-origin-when-cross-origin",
			PermissionsPolicy:     "geolocation=(), microphone=(), camera=()",
			EnableHSTS:            false,
			HSTSMaxAge:            31536000, // 1年
		},
		SecurityPassword: SecurityPasswordConfig{
			BcryptCostMin:    10,
			BcryptCostMax:    14,
			PasswordMaxBytes: 72,
		},
		SecuritySQL: SecuritySQLConfig{
			KeywordsBlacklist: []string{"select", "insert", "update", "delete", "drop", "union", "exec", "script", "javascript"},
		},
		ValidationExtended: ValidationExtendedConfig{
			URLMinLength:           7,
			PhoneFirstDigit:        "1",
			PhoneSecondDigitMin:    "3",
			PhoneSecondDigitMax:    "9",
			ResourceTitleMax:       200,
			ResourceDescriptionMax: 1000,
			ArticleTitleMax:        200,
			ArticleDescriptionMax:  500,
			CommentContentMax:      1000,
		},
		JWTExtended: JWTExtendedConfig{
			TokenPrefix: "Bearer ",
		},
		DateTimeFormats: DateTimeFormatsConfig{
			DateOnly:     "2006-01-02",
			DatetimeFull: "2006-01-02 15:04:05",
			RFC3339:      "RFC3339",
		},
		MinioAdvanced: MinioAdvancedConfig{
			PolicyVersion: "2012-10-17",
			PolicyEffect:  "Allow",
			PolicyAction:  "s3:GetObject",
		},
		DatabaseQueryAdvanced: DatabaseQueryAdvancedConfig{
			QueryLogTruncateLength: 200,
			PreparedStmtCacheSize:  1000, // 默认缓存1000个prepared statements
		},
		StatisticsQueryExtended: StatisticsQueryExtendedConfig{
			DefaultDateRangeDays: 7,
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

	// MinIO 配置
	setEnvString(&config.MinIO.Endpoint, "MINIO_ENDPOINT")
	setEnvString(&config.MinIO.AccessKeyID, "MINIO_ACCESS_KEY")
	setEnvString(&config.MinIO.SecretAccessKey, "MINIO_SECRET_KEY")
	setEnvBool(&config.MinIO.UseSSL, "MINIO_USE_SSL")

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

// getEnv 获取环境变量或返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Validate 验证配置并返回错误（如果无效）
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.Port == "" {
		return fmt.Errorf("server.port is required")
	}
	if c.Server.Mode != "debug" && c.Server.Mode != "release" && c.Server.Mode != "test" {
		return fmt.Errorf("server.mode must be one of: debug, release, test")
	}

	// 验证数据库配置
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if c.Database.Port == "" {
		return fmt.Errorf("database.port is required")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("database.username is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database.max_open_conns must be positive")
	}
	if c.Database.MaxIdleConns <= 0 {
		return fmt.Errorf("database.max_idle_conns must be positive")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("database.max_idle_conns cannot exceed max_open_conns")
	}

	// 验证JWT配置
	if c.JWT.SecretKey == "" || c.JWT.SecretKey == "default_secret_key_change_in_production" {
		fmt.Println("Warning: Using default JWT secret key. Change it in production!")
	}
	if c.JWT.ExpireHours <= 0 {
		return fmt.Errorf("jwt.expire_hours must be positive")
	}

	// 验证MinIO配置
	if c.MinIO.Endpoint == "" {
		return fmt.Errorf("minio.endpoint is required")
	}
	if c.MinIO.AccessKeyID == "" {
		return fmt.Errorf("minio.access_key_id is required")
	}
	if c.MinIO.SecretAccessKey == "" {
		return fmt.Errorf("minio.secret_access_key is required")
	}

	// 验证7桶配置
	if c.BucketUserAvatars.Name == "" {
		return fmt.Errorf("bucket_user_avatars.name is required")
	}
	if c.BucketResourceChunks.Name == "" {
		return fmt.Errorf("bucket_resource_chunks.name is required")
	}

	return nil
}
