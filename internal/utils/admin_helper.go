package utils

import (
	"gin/internal/config"
	"sync"
)

// AdminChecker 管理员检查器（优化：使用map代替循环）
type AdminChecker struct {
	adminMap map[string]bool
	mu       sync.RWMutex
}

var (
	globalAdminChecker *AdminChecker
	adminCheckerOnce   sync.Once
)

// InitAdminChecker 初始化管理员检查器
func InitAdminChecker(cfg *config.Config) {
	adminCheckerOnce.Do(func() {
		adminMap := make(map[string]bool, len(cfg.Admin.Usernames))
		for _, username := range cfg.Admin.Usernames {
			adminMap[username] = true
		}
		globalAdminChecker = &AdminChecker{
			adminMap: adminMap,
		}
	})
}

// GetAdminChecker 获取全局管理员检查器
func GetAdminChecker() *AdminChecker {
	return globalAdminChecker
}

// IsAdmin 检查用户是否是管理员（O(1)查找）
func (ac *AdminChecker) IsAdmin(username string) bool {
	if ac == nil {
		return false
	}

	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return ac.adminMap[username]
}

// GetRole 获取用户角色
func (ac *AdminChecker) GetRole(username string) string {
	if ac.IsAdmin(username) {
		return "admin"
	}
	return "user"
}

// UpdateAdminList 更新管理员列表（动态更新）
func (ac *AdminChecker) UpdateAdminList(usernames []string) {
	if ac == nil {
		return
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	// 重建map
	ac.adminMap = make(map[string]bool, len(usernames))
	for _, username := range usernames {
		ac.adminMap[username] = true
	}
}

// 便捷函数
func IsAdminUser(cfg *config.Config, username string) bool {
	// 如果还没有初始化，先初始化
	if globalAdminChecker == nil {
		InitAdminChecker(cfg)
	}

	checker := GetAdminChecker()
	if checker != nil {
		return checker.IsAdmin(username)
	}

	// 降级处理：使用循环查找
	for _, adminUsername := range cfg.Admin.Usernames {
		if adminUsername == username {
			return true
		}
	}
	return false
}

// GetUserRole 获取用户角色
func GetUserRole(cfg *config.Config, username string) string {
	if IsAdminUser(cfg, username) {
		return "admin"
	}
	return "user"
}
