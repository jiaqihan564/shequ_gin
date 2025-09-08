# 项目架构说明

## 整体架构

本项目采用分层架构设计，遵循关注点分离原则，将代码按功能模块化组织。

## 架构层次

### 1. 表现层 (Presentation Layer)
- **位置**: `internal/handlers/`
- **职责**: 处理HTTP请求和响应
- **组件**:
  - `auth.go`: 认证相关请求处理
  - `user.go`: 用户相关请求处理
  - `health.go`: 健康检查处理

### 2. 业务逻辑层 (Business Logic Layer)
- **位置**: `internal/services/`
- **职责**: 实现核心业务逻辑
- **组件**:
  - `auth.go`: 认证业务逻辑
  - `user.go`: 用户管理业务逻辑

### 3. 数据访问层 (Data Access Layer)
- **位置**: `internal/models/`
- **职责**: 定义数据结构和模型
- **组件**:
  - `user.go`: 用户相关数据结构
  - `jwt.go`: JWT相关数据结构

### 4. 中间件层 (Middleware Layer)
- **位置**: `internal/middleware/`
- **职责**: 处理横切关注点
- **组件**:
  - `auth.go`: JWT认证中间件
  - `cors.go`: 跨域处理中间件
  - `logger.go`: 日志记录中间件

### 5. 工具层 (Utility Layer)
- **位置**: `internal/utils/`
- **职责**: 提供通用工具函数
- **组件**:
  - `password.go`: 密码加密/验证工具
  - `response.go`: 统一响应格式工具
  - `validator.go`: 数据验证工具

### 6. 配置层 (Configuration Layer)
- **位置**: `internal/config/`
- **职责**: 管理应用配置
- **组件**:
  - `config.go`: 配置加载和管理

### 7. 路由层 (Routing Layer)
- **位置**: `internal/routes/`
- **职责**: 定义API路由和中间件绑定
- **组件**:
  - `routes.go`: 路由配置

## 数据流向

```
HTTP请求 → 路由层 → 中间件层 → 处理器层 → 服务层 → 数据层
                ↓
HTTP响应 ← 路由层 ← 中间件层 ← 处理器层 ← 服务层 ← 数据层
```

## 设计原则

### 1. 单一职责原则 (SRP)
每个模块只负责一个特定的功能领域。

### 2. 依赖倒置原则 (DIP)
高层模块不依赖低层模块，都依赖于抽象。

### 3. 开闭原则 (OCP)
对扩展开放，对修改关闭。

### 4. 接口隔离原则 (ISP)
使用多个专门的接口，而不是使用单一的总接口。

## 模块依赖关系

```
main.go
├── config (配置管理)
├── routes (路由配置)
    ├── handlers (请求处理)
    │   ├── services (业务逻辑)
    │   │   ├── models (数据模型)
    │   │   └── utils (工具函数)
    │   └── middleware (中间件)
    │       └── config (配置管理)
    └── middleware (中间件)
        └── config (配置管理)
```

## 扩展指南

### 添加新的API端点
1. 在 `internal/models/` 中定义请求/响应结构
2. 在 `internal/services/` 中实现业务逻辑
3. 在 `internal/handlers/` 中创建处理器
4. 在 `internal/routes/routes.go` 中注册路由

### 添加新的中间件
1. 在 `internal/middleware/` 中创建中间件函数
2. 在 `internal/routes/routes.go` 中应用中间件

### 添加新的工具函数
1. 在 `internal/utils/` 中创建工具函数
2. 在需要的地方导入并使用

## 配置管理

配置通过环境变量进行管理：
- `PORT`: 服务端口 (默认: 8080)
- `HOST`: 服务主机 (默认: localhost)
- `JWT_SECRET`: JWT密钥 (默认: your_secret_key_change_this_in_production)

## 错误处理

采用统一的错误响应格式：
```json
{
  "code": 400,
  "message": "错误描述"
}
```

## 日志记录

使用Gin内置的日志中间件，可以自定义日志格式。

## 安全考虑

1. **密码安全**: 使用bcrypt进行密码哈希
2. **JWT安全**: 使用HS256算法签名
3. **CORS配置**: 支持跨域请求
4. **输入验证**: 使用Gin的binding进行参数验证

## 性能优化

1. **中间件优化**: 合理使用中间件，避免不必要的处理
2. **响应优化**: 统一响应格式，减少重复代码
3. **配置优化**: 使用环境变量，支持不同环境配置

## 测试策略

建议的测试结构：
```
internal/
├── handlers/
│   └── *_test.go
├── services/
│   └── *_test.go
└── utils/
    └── *_test.go
```

## 部署考虑

1. **容器化**: 支持Docker部署
2. **环境变量**: 通过环境变量配置
3. **健康检查**: 提供健康检查端点
4. **日志输出**: 支持结构化日志输出
