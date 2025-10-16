-- 密码重置token表
CREATE TABLE IF NOT EXISTS `password_reset_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `email` varchar(255) NOT NULL COMMENT '邮箱地址',
  `token` varchar(64) NOT NULL COMMENT '重置token',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `used` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否已使用(0:未使用, 1:已使用)',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_email` (`email`),
  KEY `idx_token` (`token`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='密码重置token表';

