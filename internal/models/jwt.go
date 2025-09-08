package models

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims JWT声明结构体
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// CreateClaims 创建JWT声明
func CreateClaims(userID uint, username string, issuer string, expireHours int) *Claims {
	now := time.Now()
	expirationTime := now.Add(time.Duration(expireHours) * time.Hour)

	return &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(userID), 10), // 用户ID作为Subject
			Issuer:    issuer,                                 // 使用配置的Issuer
			Audience:  []string{"community-api"},              // 受众
			ExpiresAt: jwt.NewNumericDate(expirationTime),     // 过期时间
			NotBefore: jwt.NewNumericDate(now),                // 生效时间
			IssuedAt:  jwt.NewNumericDate(now),                // 签发时间
			ID:        strconv.FormatUint(uint64(userID), 10), // JWT ID
		},
	}
}
