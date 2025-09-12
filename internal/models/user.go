package models

import "time"

// User 用户结构体
type User struct {
	ID               uint       `json:"id" db:"id"`
	Username         string     `json:"username" binding:"required" db:"username"`
	PasswordHash     string     `json:"-" db:"password_hash"` // 不序列化密码字段
	Email            string     `json:"email" db:"email"`
	AuthStatus       int        `json:"auth_status" db:"auth_status"`
	AccountStatus    int        `json:"account_status" db:"account_status"`
	LastLoginTime    *time.Time `json:"last_login_time" db:"last_login_time"`
	LastLoginIP      *string    `json:"last_login_ip" db:"last_login_ip"`
	FailedLoginCount int        `json:"failed_login_count" db:"failed_login_count"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// UserProfile 用户基本信息（用于登录注册响应）
type UserProfile struct {
	ID            uint   `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	AuthStatus    int    `json:"auth_status"`
	AccountStatus int    `json:"account_status"`
	AvatarURL     string `json:"avatar_url"`
	Nickname      string `json:"nickname"`
	Bio           string `json:"bio"`
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
	Email    string `json:"email" binding:"required"`
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string      `json:"token"`
		User  UserProfile `json:"user"`
	} `json:"data"`
}

// CommonResponse 通用响应结构体
type CommonResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"error_code,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// UserExtraProfile 对应表 user_profile（扩展资料）
type UserExtraProfile struct {
	UserID    uint      `json:"user_id" db:"user_id"`
	Nickname  string    `json:"nickname" db:"nickname"`
	Bio       string    `json:"bio" db:"bio"`
	AvatarURL string    `json:"avatar_url" db:"avatar_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
