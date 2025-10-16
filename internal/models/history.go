package models

import "time"

// UserLoginHistory 用户登录历史
type UserLoginHistory struct {
	ID          uint      `json:"id" db:"id"`
	UserID      uint      `json:"user_id" db:"user_id"`
	Username    string    `json:"username" db:"username"`
	LoginTime   time.Time `json:"login_time" db:"login_time"`
	LoginIP     string    `json:"login_ip" db:"login_ip"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	LoginStatus int       `json:"login_status" db:"login_status"` // 0-失败，1-成功
	Province    string    `json:"province" db:"province"`         // 登录省份
	City        string    `json:"city" db:"city"`                 // 登录城市
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// LocationStats 地区统计数据
type LocationStats struct {
	Province   string `json:"province"`
	City       string `json:"city"`
	UserCount  int    `json:"user_count"`  // 不同用户数
	LoginCount int    `json:"login_count"` // 总登录次数
}

// LocationDistribution 地区分布响应
type LocationDistribution struct {
	ProvinceStats  []LocationStats `json:"province_stats"`  // 省份统计
	CityStats      []LocationStats `json:"city_stats"`      // 城市统计Top 20
	TotalProvinces int             `json:"total_provinces"` // 覆盖省份数
	TotalCities    int             `json:"total_cities"`    // 覆盖城市数
}

// UserOperationHistory 用户操作历史
type UserOperationHistory struct {
	ID            uint      `json:"id" db:"id"`
	UserID        uint      `json:"user_id" db:"user_id"`
	Username      string    `json:"username" db:"username"`
	OperationType string    `json:"operation_type" db:"operation_type"`
	OperationDesc string    `json:"operation_desc" db:"operation_desc"`
	OperationTime time.Time `json:"operation_time" db:"operation_time"`
	IPAddress     string    `json:"ip_address" db:"ip_address"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ProfileChangeHistory 个人资料修改历史
type ProfileChangeHistory struct {
	ID         uint      `json:"id" db:"id"`
	UserID     uint      `json:"user_id" db:"user_id"`
	FieldName  string    `json:"field_name" db:"field_name"`
	OldValue   string    `json:"old_value" db:"old_value"`
	NewValue   string    `json:"new_value" db:"new_value"`
	ChangeTime time.Time `json:"change_time" db:"change_time"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
