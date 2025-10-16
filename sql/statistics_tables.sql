-- 数据统计表
USE hub;

-- 表1：用户注册登录统计（按天）
CREATE TABLE IF NOT EXISTS `user_statistics` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `date` date NOT NULL COMMENT '统计日期',
  `login_count` int(11) NOT NULL DEFAULT 0 COMMENT '当天登录次数',
  `register_count` int(11) NOT NULL DEFAULT 0 COMMENT '当天注册次数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date` (`date`) COMMENT '确保每天只有一条记录'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户注册登录统计表（按天）';

-- 表2：API接口访问统计（按天+接口路径）
CREATE TABLE IF NOT EXISTS `api_statistics` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `date` date NOT NULL COMMENT '统计日期',
  `endpoint` varchar(255) NOT NULL COMMENT 'API接口路径',
  `method` varchar(10) NOT NULL COMMENT 'HTTP请求方法',
  `success_count` int(11) NOT NULL DEFAULT 0 COMMENT '成功请求数(2xx)',
  `error_count` int(11) NOT NULL DEFAULT 0 COMMENT '失败请求数(4xx,5xx)',
  `total_count` int(11) NOT NULL DEFAULT 0 COMMENT '总请求数',
  `avg_latency_ms` decimal(10,2) NOT NULL DEFAULT 0.00 COMMENT '平均响应时间(毫秒)',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date_endpoint_method` (`date`, `endpoint`, `method`) COMMENT '确保每天每个接口只有一条记录',
  KEY `idx_date` (`date`) COMMENT '按日期查询优化',
  KEY `idx_endpoint` (`endpoint`) COMMENT '按接口查询优化'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API接口访问统计表（按天）';

