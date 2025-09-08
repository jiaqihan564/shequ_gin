# 社区后端API文档

## 基础信息

- 基础URL: `http://localhost:8080`
- API版本: v1
- 认证方式: JWT Bearer Token

## 接口列表

### 1. 健康检查

**GET** `/health`

检查服务是否正常运行。

**响应示例:**
```json
{
  "code": 200,
  "message": "服务运行正常"
}
```

### 2. 用户注册

**POST** `/api/v1/register`

注册新用户。

**请求体:**
```json
{
  "username": "testuser",
  "password": "password123",
  "email": "test@example.com"
}
```

**响应示例:**
```json
{
  "code": 201,
  "message": "注册成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1234567890,
      "username": "testuser",
      "email": "test@example.com",
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  }
}
```

### 3. 用户登录

**POST** `/api/v1/login`

用户登录。

**请求体:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**响应示例:**
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  }
}
```

### 4. 获取用户信息

**GET** `/api/v1/user/profile`

获取当前登录用户的详细信息。

**请求头:**
```
Authorization: Bearer <your_jwt_token>
```

**响应示例:**
```json
{
  "code": 200,
  "message": "获取用户信息成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

### 5. 更新用户信息

**PUT** `/api/v1/user/profile`

更新当前登录用户的信息。

**请求头:**
```
Authorization: Bearer <your_jwt_token>
```

**请求体:**
```json
{
  "email": "newemail@example.com"
}
```

**响应示例:**
```json
{
  "code": 200,
  "message": "用户信息更新成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "newemail@example.com",
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T13:00:00Z"
  }
}
```

## 错误响应格式

所有错误响应都遵循以下格式：

```json
{
  "code": 400,
  "message": "错误描述"
}
```

## 状态码说明

- `200`: 成功
- `201`: 创建成功
- `400`: 请求参数错误
- `401`: 未授权/认证失败
- `404`: 资源不存在
- `409`: 冲突（如用户名已存在）
- `500`: 服务器内部错误

## 测试用户

系统预设了一个测试用户（需要先在数据库中创建）：

- 用户名: `admin`
- 密码: `password`
- 邮箱: `admin@example.com`

## 数据库配置

系统使用MySQL数据库进行用户认证，数据库配置如下：

- 主机: `192.168.200.131`
- 端口: `3306`
- 用户名: `root`
- 密码: `mysql_F7KJNF`
- 数据库: `hub`
- 表名: `user_auth`

### 数据库表结构

```sql
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

## 注意事项

1. JWT token有效期为24小时
2. 密码使用bcrypt加密存储
3. 所有需要认证的接口都需要在请求头中携带有效的JWT token
4. 当前版本使用内存存储，重启服务后数据会丢失
5. 生产环境请修改JWT密钥

## 使用示例

### 使用curl测试登录接口

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password"
  }'
```

### 使用curl测试获取用户信息

```bash
curl -X GET http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```
