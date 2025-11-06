-- =====================================================
-- Hub 数据库创建脚本
-- =====================================================
-- 用途: 创建 hub 数据库及数据库用户
-- 执行方式: mysql -u root -p < create_database.sql
-- =====================================================

-- 1. 删除已存在的数据库（谨慎使用！）
-- DROP DATABASE IF EXISTS hub;

-- 2. 创建数据库
CREATE DATABASE IF NOT EXISTS `hub` 
  DEFAULT CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;

-- 显示创建结果
SELECT '✓ 数据库 hub 创建成功' AS Result;
SHOW DATABASES LIKE 'hub';

-- 3. 创建专用数据库用户（推荐）
-- 注意: 请修改密码为强密码！
CREATE USER IF NOT EXISTS 'hub_user'@'localhost' IDENTIFIED BY 'Hub@2024!Strong';

-- 4. 授予权限
GRANT ALL PRIVILEGES ON `hub`.* TO 'hub_user'@'localhost';

-- 5. 如果需要远程访问（根据实际需求开启）
-- CREATE USER IF NOT EXISTS 'hub_user'@'%' IDENTIFIED BY 'Hub@2024!Strong';
-- GRANT ALL PRIVILEGES ON `hub`.* TO 'hub_user'@'%';

-- 6. 刷新权限
FLUSH PRIVILEGES;

-- 7. 显示用户信息
SELECT '✓ 数据库用户创建成功' AS Result;
SELECT 
    User AS '用户名', 
    Host AS '访问主机'
FROM mysql.user 
WHERE User = 'hub_user';

-- 8. 显示数据库字符集
SELECT 
    SCHEMA_NAME AS '数据库名',
    DEFAULT_CHARACTER_SET_NAME AS '字符集',
    DEFAULT_COLLATION_NAME AS '排序规则'
FROM information_schema.SCHEMATA 
WHERE SCHEMA_NAME = 'hub';

-- =====================================================
-- 使用说明
-- =====================================================
-- 
-- 1. 直接执行此脚本:
--    mysql -u root -p < create_database.sql
--
-- 2. 或者在 MySQL 命令行中:
--    mysql -u root -p
--    source create_database.sql;
--
-- 3. 创建后的配置信息:
--    数据库名: hub
--    用户名: hub_user
--    密码: Hub@2024!Strong (请修改)
--    字符集: utf8mb4
--
-- 4. 在 config.yaml 中配置:
--    database:
--      host: "127.0.0.1"
--      port: "3306"
--      username: "hub_user"
--      password: "Hub@2024!Strong"
--      database: "hub"
--      charset: "utf8mb4"
--
-- =====================================================

SELECT '========================================' AS '';
SELECT '✓ Hub 数据库初始化完成！' AS Success;
SELECT '========================================' AS '';
SELECT '下一步: 执行 init_database.sh 创建数据表' AS Tip;

