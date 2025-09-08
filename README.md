# 社区后端项目

这是一个基于Gin框架的社区后端项目，提供用户认证和管理功能。

## 功能特性

- ✅ 用户注册
- ✅ 用户登录
- ✅ JWT认证
- ✅ 用户信息管理
- ✅ 密码加密存储
- ✅ CORS支持
- ✅ 统一的响应格式

## 技术栈

- **框架**: Gin (Go Web Framework)
- **认证**: JWT (JSON Web Token)
- **数据库**: MySQL
- **密码加密**: bcrypt
- **Go版本**: 1.24.4

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置数据库

确保MySQL数据库已启动，并创建相应的数据库和表：

```sql
-- 创建数据库
CREATE DATABASE hub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE hub;

-- 创建用户表
CREATE TABLE `user_auth` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '用户唯一标识，自增主键',
  `username` varchar(50) NOT NULL COMMENT '登录用户名（唯一，支持字母/数字/下划线）',
  `password_hash` varchar(255) NOT NULL COMMENT '密码哈希值（采用bcrypt加密，不存明文）',
  `email` varchar(100) NOT NULL COMMENT '用户邮箱（唯一，用于登录验证、密码找回）',
  `auth_status` tinyint(1) NOT NULL DEFAULT 1 COMMENT '认证状态：0-未验证（需邮箱激活），1-已验证',
  `account_status` tinyint(1) NOT NULL DEFAULT 1 COMMENT '账户状态：0-禁用，1-正常，2-临时锁定',
  `last_login_time` datetime DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(50) DEFAULT NULL COMMENT '最后登录IP',
  `failed_login_count` int(11) NOT NULL DEFAULT 0 COMMENT '连续登录失败次数（用于防暴力破解）',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '账户创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '信息更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`) COMMENT '确保用户名唯一',
  UNIQUE KEY `uk_email` (`email`) COMMENT '确保邮箱唯一',
  KEY `idx_account_status` (`account_status`) COMMENT '优化按状态查询效率'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT '用户登录验证核心表';
```

### 3. 配置应用

#### 使用配置文件（推荐）

项目支持多种配置文件格式：

- `config.yaml` - 默认配置文件
- `config.dev.yaml` - 开发环境配置
- `config.prod.yaml` - 生产环境配置

配置文件优先级：`config.{env}.yaml` > `config.yaml` > 默认配置

#### 使用环境变量（可选）

```bash
# 应用环境
export APP_ENV=dev  # dev, prod

# 服务器配置
export SERVER_HOST=localhost
export SERVER_PORT=8080
export SERVER_MODE=debug  # debug, release, test

# 数据库配置
export DB_HOST=192.168.200.131
export DB_PORT=3306
export DB_USERNAME=root
export DB_PASSWORD=mysql_F7KJNF
export DB_DATABASE=hub

# JWT配置
export JWT_SECRET=your_secret_key_change_this_in_production
export JWT_EXPIRE_HOURS=24

# 日志配置
export LOG_LEVEL=info  # debug, info, warn, error
export LOG_FORMAT=json  # json, text
export LOG_OUTPUT=stdout  # stdout, file
```

环境变量会覆盖配置文件中的设置。

### 4. 运行项目

#### 使用默认配置
```bash
go run main.go
```

#### 使用开发环境配置
```bash
# Windows PowerShell
$env:APP_ENV="dev"; go run main.go

# Linux/macOS
APP_ENV=dev go run main.go
```

#### 使用生产环境配置
```bash
# Windows PowerShell
$env:APP_ENV="prod"; go run main.go

# Linux/macOS
APP_ENV=prod go run main.go
```

服务器将在配置的地址启动，默认 `http://localhost:8080`。

### 5. 测试接口

#### 健康检查
```bash
curl http://localhost:8080/health
```

#### 用户登录
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'
```

#### 用户注册
```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username": "newuser", "password": "password123", "email": "newuser@example.com"}'
```

## 预设测试用户

- **用户名**: admin
- **密码**: password
- **邮箱**: admin@example.com

> 注意：需要先在数据库中创建admin用户，可以通过注册接口或直接插入数据库记录。

## API文档

详细的API文档请查看 [API_DOCS.md](./API_DOCS.md)

## 项目结构

```
.
├── main.go                    # 主程序入口
├── go.mod                     # Go模块文件
├── go.sum                     # 依赖校验文件
├── API_DOCS.md                # API文档
├── README.md                  # 项目说明
└── internal/                  # 内部包
    ├── config/                # 配置管理
    │   └── config.go
    ├── handlers/              # 请求处理器
    │   ├── auth.go           # 认证相关处理器
    │   ├── user.go           # 用户相关处理器
    │   └── health.go         # 健康检查处理器
    ├── middleware/            # 中间件
    │   ├── auth.go           # JWT认证中间件
    │   ├── cors.go           # CORS中间件
    │   └── logger.go         # 日志中间件
    ├── models/               # 数据模型
    │   ├── user.go          # 用户模型
    │   └── jwt.go           # JWT模型
    ├── routes/               # 路由配置
    │   └── routes.go
    ├── services/             # 业务逻辑层
    │   ├── auth.go          # 认证服务
    │   └── user.go          # 用户服务
    └── utils/                # 工具函数
        ├── password.go      # 密码相关工具
        ├── response.go      # 响应工具
        └── validator.go     # 验证工具
```

## 开发说明

### 当前状态

- ✅ 集成MySQL数据库进行用户认证
- ✅ 支持用户注册、登录、信息管理
- ✅ 密码使用bcrypt加密存储
- ✅ JWT token认证
- ✅ 登录失败次数统计
- ✅ 最后登录时间和IP记录
- ✅ 账户状态管理

### 后续开发建议

1. **日志系统**: 添加结构化日志
2. **单元测试**: 添加测试用例
3. **API文档**: 集成Swagger文档
4. **数据验证**: 增强输入验证
5. **错误处理**: 完善错误处理机制
6. **缓存系统**: 添加Redis缓存
7. **邮件服务**: 添加邮箱验证功能
8. **权限管理**: 添加角色和权限系统

## 配置文件说明

### 配置文件结构

项目支持YAML格式的配置文件，包含以下主要配置项：

- **服务器配置**: 主机、端口、运行模式
- **数据库配置**: 连接信息、连接池设置
- **JWT配置**: 密钥、过期时间、发行者
- **日志配置**: 级别、格式、输出方式
- **安全配置**: 密码策略、登录限制
- **CORS配置**: 跨域设置

### 配置优先级

1. 环境变量（最高优先级）
2. 环境特定配置文件（`config.{env}.yaml`）
3. 默认配置文件（`config.yaml`）
4. 代码中的默认值（最低优先级）

### 环境变量

支持的环境变量包括：

- `APP_ENV`: 应用环境（dev, prod）
- `SERVER_HOST`: 服务主机
- `SERVER_PORT`: 服务端口
- `SERVER_MODE`: 运行模式（debug, release, test）
- `DB_HOST`: 数据库主机
- `DB_PORT`: 数据库端口
- `DB_USERNAME`: 数据库用户名
- `DB_PASSWORD`: 数据库密码
- `DB_DATABASE`: 数据库名称
- `JWT_SECRET`: JWT密钥
- `JWT_EXPIRE_HOURS`: JWT过期时间（小时）
- `LOG_LEVEL`: 日志级别
- `LOG_FORMAT`: 日志格式
- `LOG_OUTPUT`: 日志输出方式

## 许可证

MIT License
