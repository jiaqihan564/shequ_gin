package models

import "github.com/golang-jwt/jwt/v4"

// Claims JWT声明结构体
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}
