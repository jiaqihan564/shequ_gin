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

系统预设了一个测试用户：

- 用户名: `admin`
- 密码: `password`
- 邮箱: `admin@example.com`

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
