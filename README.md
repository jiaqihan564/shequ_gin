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
- **密码加密**: bcrypt
- **Go版本**: 1.24.4

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 运行项目

```bash
go run main.go
```

服务器将在 `http://localhost:8080` 启动。

### 3. 测试接口

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

- 使用内存存储（重启后数据丢失）
- JWT密钥为默认值（生产环境需要修改）
- 包含完整的用户认证流程

### 后续开发建议

1. **数据库集成**: 集成MySQL/PostgreSQL等数据库
2. **配置管理**: 使用环境变量或配置文件
3. **日志系统**: 添加结构化日志
4. **单元测试**: 添加测试用例
5. **API文档**: 集成Swagger文档
6. **数据验证**: 增强输入验证
7. **错误处理**: 完善错误处理机制

## 环境变量

可以通过以下环境变量配置：

- `PORT`: 服务端口（默认: 8080）

## 许可证

MIT License
