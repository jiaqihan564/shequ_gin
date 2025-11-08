package models

import "time"

// User 用户结构体
type User struct {
	ID               uint       `json:"id" db:"id"`
	Username         string     `json:"username" binding:"required" db:"username"`
	PasswordHash     string     `json:"-" db:"password_hash"` // 不序列化密码字段
	Email            string     `json:"email" db:"email"`
	Role             string     `json:"role" db:"role"` // 用户角色
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
	AvatarURL     string `json:"avatar"` // 前端期望字段名为 avatar
	Nickname      string `json:"nickname"`
	Bio           string `json:"bio"`
	Role          string `json:"role"` // 用户角色：admin 或 user
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Province string `json:"province"` // 登录省份（可选）
	City     string `json:"city"`     // 登录城市（可选）
}

// RegisterRequest 注册请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required"`
	Province string `json:"province"` // 注册省份（可选）
	City     string `json:"city"`     // 注册城市（可选）
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
	Phone     string    `json:"phone" db:"phone"`
	Gender    *int      `json:"gender" db:"gender"`     // 0-未知，1-男，2-女
	Birthday  *string   `json:"birthday" db:"birthday"` // 日期格式
	Province  string    `json:"province" db:"province"` // 省份
	City      string    `json:"city" db:"city"`         // 城市
	Website   string    `json:"website" db:"website"`   // 个人网站
	Github    string    `json:"github" db:"github"`     // GitHub用户名
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ChangePasswordRequest 修改密码请求结构体
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

// Validate 验证用户数据
func (u *User) Validate() error {
	if u.Username == "" {
		return &ValidationError{Field: "username", Message: "用户名不能为空"}
	}
	if len(u.Username) < 3 || len(u.Username) > 20 {
		return &ValidationError{Field: "username", Message: "用户名长度必须在3-20之间"}
	}
	if u.Email == "" {
		return &ValidationError{Field: "email", Message: "邮箱不能为空"}
	}
	return nil
}

// SanitizeForJSON 清理敏感字段后返回用于JSON序列化
func (u *User) SanitizeForJSON() *UserProfile {
	return &UserProfile{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		AuthStatus:    u.AuthStatus,
		AccountStatus: u.AccountStatus,
		Role:          u.Role,
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
