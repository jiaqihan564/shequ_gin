package services

import (
	"context"
	"database/sql"
	"time"
)

// updateCumulativeStat 将聚合结果写入 cumulative_statistics 表
// 如果记录不存在则插入，存在则更新 stat_value / stat_desc / category
func (r *CumulativeStatsRepository) updateCumulativeStat(statKey string, value int64) {
	definition := resolveStatDefinition(statKey)

	query := `INSERT INTO cumulative_statistics (stat_key, stat_value, stat_desc, category)
			  VALUES (?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE
			    stat_value = VALUES(stat_value),
			    stat_desc = VALUES(stat_desc),
			    category = VALUES(category)`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	if _, err := r.db.DB.ExecContext(ctx, query, statKey, value, definition.Description, definition.Category); err != nil {
		r.logger.Error("更新/插入累计统计失败",
			"statKey", statKey,
			"value", value,
			"error", err.Error())
	}
}

// queryInt64 执行返回单个整数结果的查询，出错时返回 0 并记录日志
func (r *CumulativeStatsRepository) queryInt64(query string, args ...interface{}) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	var result sql.NullInt64
	if err := r.db.DB.QueryRowContext(ctx, query, args...).Scan(&result); err != nil {
		r.logger.Error("查询累计统计聚合结果失败",
			"query", query,
			"error", err.Error())
		return 0
	}
	if !result.Valid {
		return 0
	}
	return result.Int64
}

// refreshCumulativeStatsFromSources 基于实际业务表重新计算需要的累计统计
// 只覆盖可以从权威数据源精确计算的指标，其余仍沿用增量累加结果
func (r *CumulativeStatsRepository) refreshCumulativeStatsFromSources() {
	// 1. 用户相关
	totalUsers := r.queryInt64(`SELECT COUNT(*) FROM user_auth`)
	r.updateCumulativeStat("total_users", totalUsers)

	totalLogins := r.queryInt64(`SELECT COALESCE(SUM(login_count), 0) FROM user_statistics`)
	r.updateCumulativeStat("total_logins", totalLogins)

	totalRegistrations := r.queryInt64(`SELECT COALESCE(SUM(register_count), 0) FROM user_statistics`)
	r.updateCumulativeStat("total_registrations", totalRegistrations)

	// 当天活跃用户数使用 daily_metrics 中的 active_users
	dateFormat := "2006-01-02"
	if r.config != nil {
		dateFormat = r.config.DateTimeFormats.DateOnly
	}
	today := time.Now().UTC().Format(dateFormat)

	activeUsersToday := r.queryInt64(`SELECT COALESCE(active_users, 0) FROM daily_metrics WHERE date = ?`, today)
	r.updateCumulativeStat("active_users_today", activeUsersToday)

	// 2. API 相关（从 api_statistics 聚合）
	totalApiCalls := r.queryInt64(`SELECT COALESCE(SUM(total_count), 0) FROM api_statistics`)
	r.updateCumulativeStat("total_api_calls", totalApiCalls)

	totalApiErrors := r.queryInt64(`SELECT COALESCE(SUM(error_count), 0) FROM api_statistics`)
	// 兼容 total_errors 与 total_api_errors 两个统计键，保持含义一致
	r.updateCumulativeStat("total_errors", totalApiErrors)
	r.updateCumulativeStat("total_api_errors", totalApiErrors)

	// 平均响应时间（毫秒），按请求量加权平均
	var avgResponseMs int64
	{
		ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
		defer cancel()

		var avg sql.NullFloat64
		err := r.db.DB.QueryRowContext(ctx, `
			SELECT 
				COALESCE(
					SUM(avg_latency_ms * total_count) / NULLIF(SUM(total_count), 0),
					0
				) AS weighted_avg_ms
			FROM api_statistics
		`).Scan(&avg)
		if err != nil {
			r.logger.Error("查询平均响应时间失败", "error", err.Error())
		} else if avg.Valid {
			avgResponseMs = int64(avg.Float64 + 0.5) // 四舍五入
		}
	}
	r.updateCumulativeStat("avg_response_time", avgResponseMs)

	// 上传总次数：资源创建 + 头像上传（通过 profile_change_history 中的 avatar 变更估算）
	totalUploads := r.queryInt64(`
		SELECT 
			COALESCE((SELECT COUNT(*) FROM resources WHERE status != 0), 0) +
			COALESCE((SELECT COUNT(*) FROM profile_change_history WHERE field_name = 'avatar'), 0)
	`)
	r.updateCumulativeStat("total_uploads", totalUploads)

	// 3. 安全相关
	// 失败登录尝试：使用 user_auth 中的 failed_login_count 之和（当前连续失败次数总和）
	failedLoginAttempts := r.queryInt64(`SELECT COALESCE(SUM(failed_login_count), 0) FROM user_auth`)
	r.updateCumulativeStat("failed_login_attempts", failedLoginAttempts)

	// blocked_ips 与 security_alerts 当前没有持久化来源，这里保持为 0（仅确保记录存在）
	r.updateCumulativeStat("blocked_ips", 0)
	r.updateCumulativeStat("security_alerts", 0)

	// total_password_changes / total_password_resets 继续沿用增量统计，避免与未来逻辑冲突

	// 4. 内容相关
	totalArticles := r.queryInt64(`SELECT COUNT(*) FROM articles WHERE status != 2`)
	r.updateCumulativeStat("total_articles", totalArticles)

	totalCodeSnippets := r.queryInt64(`SELECT COUNT(*) FROM code_snippets`)
	r.updateCumulativeStat("total_code_snippets", totalCodeSnippets)

	totalResources := r.queryInt64(`SELECT COUNT(*) FROM resources WHERE status != 0`)
	r.updateCumulativeStat("total_resources", totalResources)

	totalComments := r.queryInt64(`
		SELECT 
			COALESCE((SELECT COUNT(*) FROM article_comments WHERE status = 1), 0) +
			COALESCE((SELECT COUNT(*) FROM resource_comments WHERE status = 1), 0)
	`)
	r.updateCumulativeStat("total_comments", totalComments)

	totalChatMessages := r.queryInt64(`SELECT COUNT(*) FROM chat_messages WHERE status = 1`)
	r.updateCumulativeStat("total_chat_messages", totalChatMessages)
}

// RefreshFromSources 对外导出的方法，便于在 handler 中显式刷新一次累计统计
func (r *CumulativeStatsRepository) RefreshFromSources() {
	r.refreshCumulativeStatsFromSources()
}

