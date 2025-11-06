-- =====================================================
-- 累计统计系统数据库表
-- =====================================================
-- 说明: 累计统计、每日指标和实时指标表
-- 依赖: 无
-- =====================================================

USE hub;

-- ============================
-- 1. 累计统计表 (cumulative_statistics)
-- ============================
-- 存储系统累计统计数据（总用户数、总请求数等）

CREATE TABLE IF NOT EXISTS `cumulative_statistics` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `stat_key` varchar(100) NOT NULL COMMENT '统计项键名（唯一标识）',
  `stat_value` bigint(20) NOT NULL DEFAULT 0 COMMENT '统计值',
  `stat_desc` varchar(200) DEFAULT NULL COMMENT '统计项描述',
  `category` varchar(50) NOT NULL COMMENT '分类：user-用户，api-接口，security-安全，content-内容',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_stat_key` (`stat_key`) COMMENT '统计键唯一索引',
  KEY `idx_category` (`category`) COMMENT '分类索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='累计统计表';

-- ============================
-- 2. 每日指标表 (daily_metrics)
-- ============================
-- 存储每日汇总的指标数据

CREATE TABLE IF NOT EXISTS `daily_metrics` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `date` date NOT NULL COMMENT '日期',
  `active_users` int(11) NOT NULL DEFAULT 0 COMMENT '活跃用户数',
  `avg_response_time` decimal(10,2) NOT NULL DEFAULT 0.00 COMMENT '平均响应时间（毫秒）',
  `success_rate` decimal(5,2) NOT NULL DEFAULT 0.00 COMMENT '成功率（百分比）',
  `peak_concurrent` int(11) NOT NULL DEFAULT 0 COMMENT '峰值并发数',
  `most_popular_endpoint` varchar(255) DEFAULT NULL COMMENT '最热门的API端点',
  `new_users` int(11) NOT NULL DEFAULT 0 COMMENT '新增用户数',
  `total_requests` int(11) NOT NULL DEFAULT 0 COMMENT '总请求数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date` (`date`) COMMENT '日期唯一索引',
  KEY `idx_active_users` (`active_users`) COMMENT '活跃用户索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每日指标表';

-- ============================
-- 3. 实时指标表 (realtime_metrics)
-- ============================
-- 存储实时更新的系统指标

CREATE TABLE IF NOT EXISTS `realtime_metrics` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `metric_key` varchar(100) NOT NULL COMMENT '指标键名（唯一标识）',
  `metric_value` varchar(500) NOT NULL COMMENT '指标值（JSON或纯文本）',
  `metric_desc` varchar(200) DEFAULT NULL COMMENT '指标描述',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_metric_key` (`metric_key`) COMMENT '指标键唯一索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='实时指标表';

-- ============================
-- 4. 初始化累计统计数据
-- ============================

-- 用户相关统计
INSERT INTO `cumulative_statistics` (`stat_key`, `stat_value`, `stat_desc`, `category`) VALUES
('total_users', 0, '总用户数', 'user'),
('total_logins', 0, '总登录次数', 'user'),
('total_registrations', 0, '总注册用户数', 'user'),
('active_users_today', 0, '今日活跃用户数', 'user')
ON DUPLICATE KEY UPDATE `stat_desc` = VALUES(`stat_desc`);

-- API相关统计
INSERT INTO `cumulative_statistics` (`stat_key`, `stat_value`, `stat_desc`, `category`) VALUES
('total_api_calls', 0, '总API调用次数', 'api'),
('total_api_errors', 0, '总API错误次数', 'api'),
('avg_response_time', 0, '平均响应时间（毫秒）', 'api')
ON DUPLICATE KEY UPDATE `stat_desc` = VALUES(`stat_desc`);

-- 安全相关统计
INSERT INTO `cumulative_statistics` (`stat_key`, `stat_value`, `stat_desc`, `category`) VALUES
('failed_login_attempts', 0, '失败登录尝试次数', 'security'),
('blocked_ips', 0, '被封禁的IP数', 'security'),
('security_alerts', 0, '安全告警次数', 'security')
ON DUPLICATE KEY UPDATE `stat_desc` = VALUES(`stat_desc`);

-- 内容相关统计
INSERT INTO `cumulative_statistics` (`stat_key`, `stat_value`, `stat_desc`, `category`) VALUES
('total_articles', 0, '总文章数', 'content'),
('total_code_snippets', 0, '总代码片段数', 'content'),
('total_resources', 0, '总资源数', 'content'),
('total_comments', 0, '总评论数', 'content'),
('total_chat_messages', 0, '总聊天消息数', 'content')
ON DUPLICATE KEY UPDATE `stat_desc` = VALUES(`stat_desc`);

-- ============================
-- 5. 初始化实时指标数据
-- ============================

INSERT INTO `realtime_metrics` (`metric_key`, `metric_value`, `metric_desc`) VALUES
('online_users', '0', '当前在线用户数'),
('current_qps', '0', '当前QPS（每秒查询数）'),
('system_cpu', '0.0', '系统CPU使用率（百分比）'),
('system_memory', '0.0', '系统内存使用率（百分比）')
ON DUPLICATE KEY UPDATE `metric_value` = VALUES(`metric_value`);

-- ============================
-- 6. 验证表创建
-- ============================

SHOW TABLES LIKE '%statistics%';
SHOW TABLES LIKE '%metrics%';
DESC cumulative_statistics;
DESC daily_metrics;
DESC realtime_metrics;

-- 查看初始化数据
SELECT * FROM cumulative_statistics ORDER BY category, stat_key;
SELECT * FROM realtime_metrics;

-- ============================
-- 7. 使用说明
-- ============================
-- 
-- cumulative_statistics 表：
--   - 存储系统累计统计数据
--   - stat_key 必须唯一，作为统计项的标识
--   - stat_value 可以通过 UPDATE ... SET stat_value = stat_value + 1 递增
--   - category 用于分类查询和展示
--   - 适合存储：总用户数、总文章数、总请求数等累计值
--
-- daily_metrics 表：
--   - 存储每日汇总的指标数据
--   - date 字段作为唯一键，每天一条记录
--   - 使用 ON DUPLICATE KEY UPDATE 进行更新
--   - 适合存储：每日活跃用户、每日新增用户、每日请求数等
--   - 可用于生成趋势图表
--
-- realtime_metrics 表：
--   - 存储实时更新的系统指标
--   - metric_key 必须唯一，作为指标的标识
--   - metric_value 可以是简单值或JSON字符串
--   - updated_at 自动更新，记录最后更新时间
--   - 适合存储：在线用户数、当前QPS、CPU使用率等实时值
--   - 高频更新，建议使用 ON DUPLICATE KEY UPDATE
--
-- 数据维护建议：
--   - daily_metrics 表建议定期清理旧数据（如保留最近90天）
--   - cumulative_statistics 和 realtime_metrics 数据量小，可长期保留
--   - 建议定期备份统计数据
--

