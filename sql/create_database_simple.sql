-- =====================================================
-- 快速创建 Hub 数据库（仅创建数据库）
-- =====================================================
-- 执行: mysql -u root -p < create_database_simple.sql
-- =====================================================

-- 创建数据库
CREATE DATABASE IF NOT EXISTS `hub` 
  DEFAULT CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE hub;

-- 显示结果
SELECT 'Hub 数据库创建成功！' AS message;
SHOW CREATE DATABASE hub;

