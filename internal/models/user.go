package models

import "time"

// User 用户结构体
type User struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username" binding:"required"`
	Password  string    `json:"-"` // 不序列化密码字段
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
		User  User   `json:"user"`
	} `json:"data"`
}

// CommonResponse 通用响应结构体
type CommonResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// UpdateUserRequest 更新用户信息请求结构体
type UpdateUserRequest struct {
	Email string `json:"email" binding:"omitempty,email"`
}
