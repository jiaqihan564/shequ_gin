package middleware

import (
	"context"
	"strconv"
	"time"

	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// StatisticsMiddleware 统计中间件（自动收集数据）
func StatisticsMiddleware(statsRepo *services.StatisticsRepository, cumulativeRepo *services.CumulativeStatsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 并发计数+1（请求开始）
		dailyMgr := services.GetDailyMetricsManager()
		dailyMgr.IncrementConcurrent()

		// 使用 defer 确保并发计数一定会减少（即使发生 panic）
		defer dailyMgr.DecrementConcurrent()

		// 实时指标：记录请求（QPS统计）
		realtimeMgr := services.GetRealtimeMetricsManager()
		realtimeMgr.RecordRequest()

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)
		status := c.Writer.Status()

		// 在请求处理完成后，尝试获取用户ID（用于活跃用户统计）
		userIDForActive := uint(0)
		if uid, exists := c.Get("userID"); exists {
			switch v := uid.(type) {
			case uint:
				userIDForActive = v
			case string:
				// 如果是字符串，尝试转换
				if id, err := strconv.ParseUint(v, 10, 32); err == nil {
					userIDForActive = uint(id)
				}
			}
		}

		// 使用Worker Pool记录统计数据（避免goroutine泄漏）
		taskID := "stats_" + path + "_" + strconv.FormatInt(time.Now().UnixNano(), 36)
		_ = utils.SubmitTask(taskID, func(ctx context.Context) error {
			date := time.Now().Format("2006-01-02")

			// 判断请求状态
			isSuccess := status >= 200 && status < 300
			isError := status >= 400

			// 获取每日指标管理器
			dailyMgr := services.GetDailyMetricsManager()

			// 记录活跃用户（所有已认证的请求）
			if userIDForActive > 0 {
				dailyMgr.RecordLogin(userIDForActive)
				realtimeMgr.RecordUserActivity(userIDForActive) // 实时在线用户
			}

			// 1. 记录登录统计
			if path == "/api/auth/login" && method == "POST" && status == 200 {
				// 按天统计
				if err := statsRepo.IncrementLoginCount(date); err != nil {
					utils.GetLogger().Error("记录登录统计失败", "date", date, "error", err.Error())
				}

				// 累计统计
				if cumulativeRepo != nil {
					if err := cumulativeRepo.IncrementCumulativeStat("total_logins", 1); err != nil {
						utils.GetLogger().Error("更新累计登录统计失败", "error", err.Error())
					}
				}
			}

			// 2. 记录注册统计
			if path == "/api/auth/register" && method == "POST" && (status == 200 || status == 201) {
				// 每日指标：新用户
				dailyMgr.RecordRegister()

				// 按天统计
				if err := statsRepo.IncrementRegisterCount(date); err != nil {
					utils.GetLogger().Error("记录注册统计失败", "date", date, "error", err.Error())
				}
				// 累计统计
				if err := cumulativeRepo.IncrementCumulativeStat("total_users", 1); err != nil {
					utils.GetLogger().Error("更新累计用户统计失败", "error", err.Error())
				}
			}

			// 3. 记录API访问统计（所有接口）
			// 每日指标：记录请求
			dailyMgr.RecordRequest(path, latency.Milliseconds(), isSuccess, isError)

			// 按天按接口统计
			if err := statsRepo.RecordApiCall(date, path, method, isSuccess, isError, latency.Milliseconds()); err != nil {
				utils.GetLogger().Error("记录API统计失败",
					"date", date,
					"endpoint", path,
					"method", method,
					"error", err.Error())
			}

			// 累计统计更新
			if err := cumulativeRepo.IncrementCumulativeStat("total_api_calls", 1); err != nil {
				utils.GetLogger().Error("更新累计API统计失败", "error", err.Error())
			}

			if isError {
				if err := cumulativeRepo.IncrementCumulativeStat("total_errors", 1); err != nil {
					utils.GetLogger().Error("更新累计错误统计失败", "error", err.Error())
				}
			}

			// 4. 记录特定操作的累计统计
			// 文件上传（两个可能的路径）
			if (path == "/api/upload" || path == "/api/files/upload") && method == "POST" && status == 200 {
				if err := cumulativeRepo.IncrementCumulativeStat("total_uploads", 1); err != nil {
					utils.GetLogger().Error("更新累计上传统计失败", "error", err.Error())
				}
			}

			// 修改密码
			if path == "/api/auth/change-password" && method == "POST" && status == 200 {
				if err := cumulativeRepo.IncrementCumulativeStat("total_password_changes", 1); err != nil {
					utils.GetLogger().Error("更新累计修改密码统计失败", "error", err.Error())
				}
			}

			// 重置密码
			if path == "/api/auth/reset-password" && method == "POST" && status == 200 {
				if err := cumulativeRepo.IncrementCumulativeStat("total_password_resets", 1); err != nil {
					utils.GetLogger().Error("更新累计重置密码统计失败", "error", err.Error())
				}
			}

			// 5. 最后：更新每日指标到数据库（在同一个任务中，确保所有内存操作都已完成）
			activeUsers, newUsers, totalReqs, peakConcurrent, avgLatency, successRate, mostPopular := dailyMgr.GetTodayMetrics()
			if err := cumulativeRepo.UpsertDailyMetric(
				date,
				activeUsers,
				newUsers,
				totalReqs,
				avgLatency,
				successRate,
				peakConcurrent,
				mostPopular,
			); err != nil {
				utils.GetLogger().Error("更新每日指标失败", "date", date, "error", err.Error())
			}

			return nil
		}, 10*time.Second)
	}
}
