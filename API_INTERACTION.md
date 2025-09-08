# 前后端交互文档

## 概述

本文档详细描述了社区后端API与前端应用的交互方式，包括请求格式、响应格式、错误处理等。

## 基础信息

- **基础URL**: `http://localhost:8080`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **数据格式**: JSON
- **字符编码**: UTF-8

## 通用响应格式

### 成功响应格式

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    // 具体数据内容
  }
}
```

### 错误响应格式

```json
{
  "code": 400,
  "message": "错误描述信息"
}
```

### HTTP状态码说明

| 状态码 | 说明 | 使用场景 |
|--------|------|----------|
| 200 | 成功 | 请求成功处理 |
| 201 | 创建成功 | 资源创建成功 |
| 400 | 请求错误 | 参数错误、格式错误 |
| 401 | 未授权 | 认证失败、token无效 |
| 404 | 资源不存在 | 用户不存在、接口不存在 |
| 409 | 冲突 | 用户名已存在、邮箱已注册 |
| 500 | 服务器错误 | 内部服务器错误 |

## 用户注册接口

### 接口信息

- **URL**: `POST /api/v1/register`
- **描述**: 用户注册新账户
- **认证**: 不需要

### 请求格式

#### 请求头

```http
Content-Type: application/json
```

#### 请求体

```json
{
  "username": "testuser",
  "password": "password123",
  "email": "test@example.com"
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 | 验证规则 |
|------|------|------|------|----------|
| username | string | 是 | 用户名 | 3-20位，支持字母、数字、下划线 |
| password | string | 是 | 密码 | 最少6位 |
| email | string | 是 | 邮箱地址 | 有效的邮箱格式 |

### 响应格式

#### 成功响应 (201)

```json
{
  "code": 201,
  "message": "注册成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxNzU3MzIyMzkwLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwiZXhwIjoxNzU3NDA4NzI5LCJuYmYiOjE3NTczMjIzMjksImlhdCI6MTc1NzMyMjMyOX0.example_signature",
    "user": {
      "id": 1757322390,
      "username": "testuser",
      "email": "test@example.com",
      "auth_status": 1,
      "account_status": 1,
      "last_login_time": null,
      "last_login_ip": null,
      "failed_login_count": 0,
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  }
}
```

#### 错误响应示例

**用户名已存在 (409)**
```json
{
  "code": 409,
  "message": "用户名已存在"
}
```

**邮箱已被注册 (409)**
```json
{
  "code": 409,
  "message": "邮箱已被注册"
}
```

**参数错误 (400)**
```json
{
  "code": 400,
  "message": "请求参数错误: Key: 'RegisterRequest.Username' Error:Field validation for 'Username' failed on the 'min' tag"
}
```

### 前端实现示例

#### JavaScript (Fetch API)

```javascript
async function register(username, password, email) {
  try {
    const response = await fetch('http://localhost:8080/api/v1/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        username: username,
        password: password,
        email: email
      })
    });

    const data = await response.json();

    if (data.code === 201) {
      // 注册成功
      console.log('注册成功:', data.message);
      console.log('Token:', data.data.token);
      console.log('用户信息:', data.data.user);
      
      // 保存token到localStorage
      localStorage.setItem('token', data.data.token);
      localStorage.setItem('user', JSON.stringify(data.data.user));
      
      return { success: true, data: data.data };
    } else {
      // 注册失败
      console.error('注册失败:', data.message);
      return { success: false, message: data.message };
    }
  } catch (error) {
    console.error('网络错误:', error);
    return { success: false, message: '网络连接失败' };
  }
}

// 使用示例
register('newuser', 'password123', 'newuser@example.com')
  .then(result => {
    if (result.success) {
      // 跳转到主页面
      window.location.href = '/dashboard';
    } else {
      // 显示错误信息
      alert(result.message);
    }
  });
```

#### JavaScript (Axios)

```javascript
import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  headers: {
    'Content-Type': 'application/json'
  }
});

async function register(username, password, email) {
  try {
    const response = await api.post('/register', {
      username,
      password,
      email
    });

    if (response.data.code === 201) {
      // 注册成功
      localStorage.setItem('token', response.data.data.token);
      localStorage.setItem('user', JSON.stringify(response.data.data.user));
      return { success: true, data: response.data.data };
    }
  } catch (error) {
    if (error.response) {
      // 服务器返回错误
      return { 
        success: false, 
        message: error.response.data.message 
      };
    } else {
      // 网络错误
      return { success: false, message: '网络连接失败' };
    }
  }
}
```

## 用户登录接口

### 接口信息

- **URL**: `POST /api/v1/login`
- **描述**: 用户登录验证
- **认证**: 不需要

### 请求格式

#### 请求头

```http
Content-Type: application/json
```

#### 请求体

```json
{
  "username": "admin",
  "password": "password"
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名或邮箱 |
| password | string | 是 | 密码 |

### 响应格式

#### 成功响应 (200)

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwiZXhwIjoxNzU3NDA5NTI5LCJuYmYiOjE3NTczMjMxMjksImlhdCI6MTc1NzMyMzEyOX0.HOZk8byY4dQ7aXZYLv1mjBFPub1F05e5Ee_jBM3y4E4",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "auth_status": 1,
      "account_status": 1,
      "last_login_time": "2024-01-01T12:30:00Z",
      "last_login_ip": "127.0.0.1",
      "failed_login_count": 0,
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-01T12:30:00Z"
    }
  }
}
```

#### 错误响应示例

**认证失败 (401)**
```json
{
  "code": 401,
  "message": "用户名或密码错误"
}
```

**账户被禁用 (401)**
```json
{
  "code": 401,
  "message": "账户已被禁用"
}
```

**参数错误 (400)**
```json
{
  "code": 400,
  "message": "请求参数错误: Key: 'LoginRequest.Username' Error:Field validation for 'Username' failed on the 'required' tag"
}
```

### 前端实现示例

#### JavaScript (Fetch API)

```javascript
async function login(username, password) {
  try {
    const response = await fetch('http://localhost:8080/api/v1/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        username: username,
        password: password
      })
    });

    const data = await response.json();

    if (data.code === 200) {
      // 登录成功
      console.log('登录成功:', data.message);
      console.log('Token:', data.data.token);
      console.log('用户信息:', data.data.user);
      
      // 保存token和用户信息
      localStorage.setItem('token', data.data.token);
      localStorage.setItem('user', JSON.stringify(data.data.user));
      
      return { success: true, data: data.data };
    } else {
      // 登录失败
      console.error('登录失败:', data.message);
      return { success: false, message: data.message };
    }
  } catch (error) {
    console.error('网络错误:', error);
    return { success: false, message: '网络连接失败' };
  }
}

// 使用示例
login('admin', 'password')
  .then(result => {
    if (result.success) {
      // 跳转到主页面
      window.location.href = '/dashboard';
    } else {
      // 显示错误信息
      alert(result.message);
    }
  });
```

#### React Hook 示例

```javascript
import { useState } from 'react';

function useAuth() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(false);

  const login = async (username, password) => {
    setLoading(true);
    try {
      const response = await fetch('http://localhost:8080/api/v1/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password })
      });

      const data = await response.json();

      if (data.code === 200) {
        localStorage.setItem('token', data.data.token);
        localStorage.setItem('user', JSON.stringify(data.data.user));
        setUser(data.data.user);
        return { success: true };
      } else {
        return { success: false, message: data.message };
      }
    } catch (error) {
      return { success: false, message: '网络连接失败' };
    } finally {
      setLoading(false);
    }
  };

  const register = async (username, password, email) => {
    setLoading(true);
    try {
      const response = await fetch('http://localhost:8080/api/v1/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password, email })
      });

      const data = await response.json();

      if (data.code === 201) {
        localStorage.setItem('token', data.data.token);
        localStorage.setItem('user', JSON.stringify(data.data.user));
        setUser(data.data.user);
        return { success: true };
      } else {
        return { success: false, message: data.message };
      }
    } catch (error) {
      return { success: false, message: '网络连接失败' };
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    setUser(null);
  };

  return { user, loading, login, register, logout };
}
```

## 获取用户信息接口

### 接口信息

- **URL**: `GET /api/v1/user/profile`
- **描述**: 获取当前登录用户的详细信息
- **认证**: 需要JWT Token

### 请求格式

#### 请求头

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

### 响应格式

#### 成功响应 (200)

```json
{
  "code": 200,
  "message": "获取用户信息成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "auth_status": 1,
    "account_status": 1,
    "last_login_time": "2024-01-01T12:30:00Z",
    "last_login_ip": "127.0.0.1",
    "failed_login_count": 0,
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T12:30:00Z"
  }
}
```

#### 错误响应示例

**未授权 (401)**
```json
{
  "code": 401,
  "message": "缺少Authorization头"
}
```

**Token无效 (401)**
```json
{
  "code": 401,
  "message": "无效的token"
}
```

### 前端实现示例

```javascript
async function getUserProfile() {
  const token = localStorage.getItem('token');
  
  if (!token) {
    return { success: false, message: '未登录' };
  }

  try {
    const response = await fetch('http://localhost:8080/api/v1/user/profile', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    });

    const data = await response.json();

    if (data.code === 200) {
      return { success: true, data: data.data };
    } else {
      return { success: false, message: data.message };
    }
  } catch (error) {
    return { success: false, message: '网络连接失败' };
  }
}
```

## 更新用户信息接口

### 接口信息

- **URL**: `PUT /api/v1/user/profile`
- **描述**: 更新当前登录用户的信息
- **认证**: 需要JWT Token

### 请求格式

#### 请求头

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

#### 请求体

```json
{
  "email": "newemail@example.com"
}
```

### 响应格式

#### 成功响应 (200)

```json
{
  "code": 200,
  "message": "用户信息更新成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "newemail@example.com",
    "auth_status": 1,
    "account_status": 1,
    "last_login_time": "2024-01-01T12:30:00Z",
    "last_login_ip": "127.0.0.1",
    "failed_login_count": 0,
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T13:00:00Z"
  }
}
```

## 健康检查接口

### 接口信息

- **URL**: `GET /health`
- **描述**: 检查服务是否正常运行
- **认证**: 不需要

### 响应格式

```json
{
  "code": 200,
  "message": "服务运行正常"
}
```

## 错误处理最佳实践

### 前端错误处理

```javascript
function handleApiError(error, response) {
  if (response) {
    // 服务器返回的错误
    switch (response.status) {
      case 400:
        return '请求参数错误';
      case 401:
        return '认证失败，请重新登录';
      case 404:
        return '资源不存在';
      case 409:
        return '数据冲突';
      case 500:
        return '服务器内部错误';
      default:
        return '未知错误';
    }
  } else {
    // 网络错误
    return '网络连接失败，请检查网络';
  }
}
```

### Token管理

```javascript
// 检查token是否过期
function isTokenExpired(token) {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    const currentTime = Date.now() / 1000;
    return payload.exp < currentTime;
  } catch (error) {
    return true;
  }
}

// 自动刷新token（如果需要）
function refreshToken() {
  // 实现token刷新逻辑
}

// 请求拦截器
axios.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token');
    if (token && !isTokenExpired(token)) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  error => Promise.reject(error)
);

// 响应拦截器
axios.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      // Token过期，清除本地存储并跳转到登录页
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

## 测试用例

### 注册测试用例

```javascript
// 测试用例1: 正常注册
async function testRegisterSuccess() {
  const result = await register('testuser1', 'password123', 'test1@example.com');
  console.assert(result.success === true, '注册应该成功');
}

// 测试用例2: 用户名已存在
async function testRegisterUsernameExists() {
  const result = await register('admin', 'password123', 'test2@example.com');
  console.assert(result.success === false, '用户名已存在应该失败');
  console.assert(result.message === '用户名已存在', '错误信息应该正确');
}

// 测试用例3: 邮箱已存在
async function testRegisterEmailExists() {
  const result = await register('testuser3', 'password123', 'admin@example.com');
  console.assert(result.success === false, '邮箱已存在应该失败');
  console.assert(result.message === '邮箱已被注册', '错误信息应该正确');
}
```

### 登录测试用例

```javascript
// 测试用例1: 正常登录
async function testLoginSuccess() {
  const result = await login('admin', 'password');
  console.assert(result.success === true, '登录应该成功');
}

// 测试用例2: 密码错误
async function testLoginWrongPassword() {
  const result = await login('admin', 'wrongpassword');
  console.assert(result.success === false, '密码错误应该失败');
  console.assert(result.message === '用户名或密码错误', '错误信息应该正确');
}

// 测试用例3: 用户不存在
async function testLoginUserNotExists() {
  const result = await login('nonexistent', 'password');
  console.assert(result.success === false, '用户不存在应该失败');
  console.assert(result.message === '用户名或密码错误', '错误信息应该正确');
}
```

## 安全注意事项

1. **HTTPS**: 生产环境必须使用HTTPS
2. **Token存储**: 避免在localStorage中存储敏感信息
3. **输入验证**: 前端和后端都要进行输入验证
4. **错误信息**: 不要暴露敏感的系统信息
5. **频率限制**: 实现登录尝试频率限制
6. **CORS配置**: 正确配置跨域访问策略
