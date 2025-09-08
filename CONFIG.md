# 配置文件说明

## 概述

项目支持YAML格式的配置文件，提供灵活的配置管理方式。配置文件支持环境变量覆盖，便于不同环境的部署。

## 配置文件类型

### 1. 默认配置文件 (`config.yaml`)
- 包含所有配置项的默认值
- 适用于大多数场景

### 2. 环境特定配置文件
- `config.dev.yaml` - 开发环境配置
- `config.prod.yaml` - 生产环境配置
- 环境特定配置会覆盖默认配置

## 配置项说明

### 服务器配置 (server)

```yaml
server:
  host: "localhost"        # 服务器监听地址
  port: "8080"            # 服务器端口
  mode: "release"         # 运行模式: debug, release, test
```

**环境变量覆盖:**
- `SERVER_HOST`
- `SERVER_PORT`
- `SERVER_MODE`

### 数据库配置 (database)

```yaml
database:
  host: "192.168.200.131"  # 数据库主机
  port: "3306"            # 数据库端口
  username: "root"        # 数据库用户名
  password: "mysql_F7KJNF" # 数据库密码
  database: "hub"         # 数据库名称
  charset: "utf8mb4"      # 字符集
  max_open_conns: 100     # 最大打开连接数
  max_idle_conns: 10      # 最大空闲连接数
  conn_max_lifetime: "1h" # 连接最大生存时间
```

**环境变量覆盖:**
- `DB_HOST`
- `DB_PORT`
- `DB_USERNAME`
- `DB_PASSWORD`
- `DB_DATABASE`

### JWT配置 (jwt)

```yaml
jwt:
  secret_key: "your_secret_key_change_this_in_production"  # JWT密钥
  expire_hours: 24        # Token过期时间（小时）
  issuer: "community-api" # JWT发行者
```

**环境变量覆盖:**
- `JWT_SECRET`
- `JWT_EXPIRE_HOURS`

### 日志配置 (log)

```yaml
log:
  level: "info"           # 日志级别: debug, info, warn, error
  format: "json"          # 日志格式: json, text
  output: "stdout"        # 输出方式: stdout, file
  file_path: "logs/app.log" # 日志文件路径
  max_size: 100           # 日志文件最大大小（MB）
  max_backups: 3          # 最大备份文件数
  max_age: 28             # 日志文件最大保存天数
```

**环境变量覆盖:**
- `LOG_LEVEL`
- `LOG_FORMAT`
- `LOG_OUTPUT`

### 安全配置 (security)

```yaml
security:
  password_min_length: 6   # 密码最小长度
  password_max_length: 50  # 密码最大长度
  username_min_length: 3   # 用户名最小长度
  username_max_length: 20  # 用户名最大长度
  max_login_attempts: 5    # 最大登录尝试次数
  lockout_duration: "30m"  # 账户锁定时间
```

### CORS配置 (cors)

```yaml
cors:
  allow_origins: ["*"]     # 允许的源
  allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"] # 允许的HTTP方法
  allow_headers: ["Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"] # 允许的请求头
  allow_credentials: true  # 是否允许凭证
```

## 配置优先级

配置的加载优先级从高到低：

1. **环境变量** - 最高优先级
2. **环境特定配置文件** - `config.{env}.yaml`
3. **默认配置文件** - `config.yaml`
4. **代码默认值** - 最低优先级

## 使用示例

### 开发环境

```bash
# 设置环境变量
export APP_ENV=dev

# 启动应用
go run main.go
```

应用会按以下顺序查找配置文件：
1. `config.dev.yaml`
2. `config.yaml`
3. 使用默认配置

### 生产环境

```bash
# 设置环境变量
export APP_ENV=prod
export JWT_SECRET=your_production_secret_key
export DB_PASSWORD=your_production_db_password

# 启动应用
go run main.go
```

### 使用环境变量覆盖

```bash
# 临时修改端口
export SERVER_PORT=9090

# 启动应用
go run main.go
```

## 最佳实践

### 1. 配置文件管理

- 将 `config.prod.yaml` 添加到 `.gitignore`，避免敏感信息泄露
- 使用环境变量传递敏感信息（密码、密钥等）
- 为不同环境创建对应的配置文件

### 2. 安全考虑

- 生产环境使用强密码和密钥
- 限制CORS允许的源
- 设置合适的日志级别

### 3. 性能优化

- 根据负载调整数据库连接池大小
- 设置合适的日志轮转策略
- 生产环境使用 `release` 模式

## 故障排除

### 配置文件未生效

1. 检查配置文件路径是否正确
2. 确认配置文件格式是否正确（YAML语法）
3. 查看启动日志中的配置加载信息

### 环境变量未生效

1. 确认环境变量名称是否正确
2. 检查环境变量是否在应用启动前设置
3. 查看配置覆盖逻辑

### 配置冲突

1. 了解配置优先级规则
2. 检查是否有多个配置文件
3. 确认环境变量设置
