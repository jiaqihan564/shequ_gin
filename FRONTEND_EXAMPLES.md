# 前端交互示例

## 快速开始

### 基础配置

```javascript
// API基础配置
const API_BASE_URL = 'http://localhost:8080/api/v1';

// 通用请求函数
async function apiRequest(endpoint, options = {}) {
  const url = `${API_BASE_URL}${endpoint}`;
  const token = localStorage.getItem('token');
  
  const defaultOptions = {
    headers: {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` })
    }
  };
  
  const response = await fetch(url, { ...defaultOptions, ...options });
  const data = await response.json();
  
  return { response, data };
}
```

## 用户注册

### 简单示例

```javascript
async function registerUser(username, password, email) {
  try {
    const { data } = await apiRequest('/register', {
      method: 'POST',
      body: JSON.stringify({ username, password, email })
    });
    
    if (data.code === 201) {
      // 注册成功，保存token
      localStorage.setItem('token', data.data.token);
      localStorage.setItem('user', JSON.stringify(data.data.user));
      return { success: true, user: data.data.user };
    } else {
      return { success: false, message: data.message };
    }
  } catch (error) {
    return { success: false, message: '网络错误' };
  }
}
```

### 表单验证示例

```javascript
function validateRegistration(username, password, email) {
  const errors = [];
  
  // 用户名验证
  if (!username || username.length < 3) {
    errors.push('用户名至少3个字符');
  }
  
  // 密码验证
  if (!password || password.length < 6) {
    errors.push('密码至少6个字符');
  }
  
  // 邮箱验证
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!email || !emailRegex.test(email)) {
    errors.push('请输入有效的邮箱地址');
  }
  
  return errors;
}

// 使用示例
async function handleRegister(event) {
  event.preventDefault();
  
  const username = document.getElementById('username').value;
  const password = document.getElementById('password').value;
  const email = document.getElementById('email').value;
  
  // 前端验证
  const errors = validateRegistration(username, password, email);
  if (errors.length > 0) {
    alert(errors.join('\n'));
    return;
  }
  
  // 调用注册接口
  const result = await registerUser(username, password, email);
  
  if (result.success) {
    alert('注册成功！');
    window.location.href = '/dashboard';
  } else {
    alert(result.message);
  }
}
```

## 用户登录

### 简单示例

```javascript
async function loginUser(username, password) {
  try {
    const { data } = await apiRequest('/login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
    
    if (data.code === 200) {
      // 登录成功，保存token
      localStorage.setItem('token', data.data.token);
      localStorage.setItem('user', JSON.stringify(data.data.user));
      return { success: true, user: data.data.user };
    } else {
      return { success: false, message: data.message };
    }
  } catch (error) {
    return { success: false, message: '网络错误' };
  }
}
```

### 登录表单示例

```html
<form id="loginForm">
  <div>
    <label>用户名:</label>
    <input type="text" id="username" required>
  </div>
  <div>
    <label>密码:</label>
    <input type="password" id="password" required>
  </div>
  <button type="submit">登录</button>
</form>

<script>
document.getElementById('loginForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  
  const username = document.getElementById('username').value;
  const password = document.getElementById('password').value;
  
  const result = await loginUser(username, password);
  
  if (result.success) {
    alert('登录成功！');
    window.location.href = '/dashboard';
  } else {
    alert(result.message);
  }
});
</script>
```

## 获取用户信息

```javascript
async function getUserProfile() {
  try {
    const { data } = await apiRequest('/user/profile');
    
    if (data.code === 200) {
      return { success: true, user: data.data };
    } else {
      return { success: false, message: data.message };
    }
  } catch (error) {
    return { success: false, message: '网络错误' };
  }
}

// 使用示例
async function loadUserProfile() {
  const result = await getUserProfile();
  
  if (result.success) {
    document.getElementById('username').textContent = result.user.username;
    document.getElementById('email').textContent = result.user.email;
    document.getElementById('lastLogin').textContent = result.user.last_login_time;
  } else {
    console.error('获取用户信息失败:', result.message);
  }
}
```

## 更新用户信息

```javascript
async function updateUserProfile(email) {
  try {
    const { data } = await apiRequest('/user/profile', {
      method: 'PUT',
      body: JSON.stringify({ email })
    });
    
    if (data.code === 200) {
      // 更新本地存储的用户信息
      localStorage.setItem('user', JSON.stringify(data.data));
      return { success: true, user: data.data };
    } else {
      return { success: false, message: data.message };
    }
  } catch (error) {
    return { success: false, message: '网络错误' };
  }
}
```

## 用户状态管理

### 检查登录状态

```javascript
function isLoggedIn() {
  const token = localStorage.getItem('token');
  const user = localStorage.getItem('user');
  
  if (!token || !user) {
    return false;
  }
  
  // 检查token是否过期
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    const currentTime = Date.now() / 1000;
    return payload.exp > currentTime;
  } catch (error) {
    return false;
  }
}

function getCurrentUser() {
  const userStr = localStorage.getItem('user');
  return userStr ? JSON.parse(userStr) : null;
}
```

### 登出功能

```javascript
function logout() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  window.location.href = '/login';
}
```

## React Hook 示例

```javascript
import { useState, useEffect } from 'react';

function useAuth() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(false);

  // 检查登录状态
  useEffect(() => {
    const token = localStorage.getItem('token');
    const userStr = localStorage.getItem('user');
    
    if (token && userStr) {
      try {
        const userData = JSON.parse(userStr);
        setUser(userData);
      } catch (error) {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
      }
    }
  }, []);

  const login = async (username, password) => {
    setLoading(true);
    const result = await loginUser(username, password);
    if (result.success) {
      setUser(result.user);
    }
    setLoading(false);
    return result;
  };

  const register = async (username, password, email) => {
    setLoading(true);
    const result = await registerUser(username, password, email);
    if (result.success) {
      setUser(result.user);
    }
    setLoading(false);
    return result;
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    setUser(null);
  };

  return { user, loading, login, register, logout };
}

// 使用示例
function LoginPage() {
  const { login, loading } = useAuth();
  const [formData, setFormData] = useState({ username: '', password: '' });

  const handleSubmit = async (e) => {
    e.preventDefault();
    const result = await login(formData.username, formData.password);
    
    if (result.success) {
      // 登录成功，跳转
    } else {
      alert(result.message);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="text"
        placeholder="用户名"
        value={formData.username}
        onChange={(e) => setFormData({...formData, username: e.target.value})}
      />
      <input
        type="password"
        placeholder="密码"
        value={formData.password}
        onChange={(e) => setFormData({...formData, password: e.target.value})}
      />
      <button type="submit" disabled={loading}>
        {loading ? '登录中...' : '登录'}
      </button>
    </form>
  );
}
```

## Vue.js 示例

```javascript
// Vue 3 Composition API
import { ref, reactive } from 'vue';

export function useAuth() {
  const user = ref(null);
  const loading = ref(false);

  const login = async (username, password) => {
    loading.value = true;
    const result = await loginUser(username, password);
    if (result.success) {
      user.value = result.user;
    }
    loading.value = false;
    return result;
  };

  const register = async (username, password, email) => {
    loading.value = true;
    const result = await registerUser(username, password, email);
    if (result.success) {
      user.value = result.user;
    }
    loading.value = false;
    return result;
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    user.value = null;
  };

  return { user, loading, login, register, logout };
}

// 组件使用
export default {
  setup() {
    const { login, loading } = useAuth();
    const form = reactive({
      username: '',
      password: ''
    });

    const handleLogin = async () => {
      const result = await login(form.username, form.password);
      if (result.success) {
        // 登录成功
      } else {
        alert(result.message);
      }
    };

    return { form, handleLogin, loading };
  }
};
```

## 错误处理

```javascript
function handleApiError(error) {
  if (error.response) {
    // 服务器返回错误
    const { status, data } = error.response;
    
    switch (status) {
      case 400:
        return '请求参数错误';
      case 401:
        // 清除本地存储并跳转到登录页
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        window.location.href = '/login';
        return '请重新登录';
      case 404:
        return '资源不存在';
      case 409:
        return '数据冲突';
      case 500:
        return '服务器错误';
      default:
        return data.message || '未知错误';
    }
  } else {
    // 网络错误
    return '网络连接失败';
  }
}
```

## 测试数据

### 测试用户

```javascript
// 测试用户信息
const testUsers = {
  admin: {
    username: 'admin',
    password: 'password',
    email: 'admin@example.com'
  },
  testuser: {
    username: 'testuser',
    password: 'password123',
    email: 'test@example.com'
  }
};

// 测试注册
async function testRegister() {
  const result = await registerUser(
    'newuser',
    'password123',
    'newuser@example.com'
  );
  console.log('注册结果:', result);
}

// 测试登录
async function testLogin() {
  const result = await loginUser('admin', 'password');
  console.log('登录结果:', result);
}
```
