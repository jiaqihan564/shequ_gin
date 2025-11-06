-- 为登录历史表添加地区字段并插入测试数据
USE hub;

-- 检查字段是否已存在，如果不存在则添加
SET @dbname = DATABASE();
SET @tablename = 'user_login_history';
SET @columnname_province = 'province';
SET @columnname_city = 'city';
SET @preparedStatement_province = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname_province)
  ) > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname_province, ' VARCHAR(50) DEFAULT NULL COMMENT ''登录省份''')
));
PREPARE alterIfNotExists_province FROM @preparedStatement_province;
EXECUTE alterIfNotExists_province;
DEALLOCATE PREPARE alterIfNotExists_province;

SET @preparedStatement_city = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname_city)
  ) > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname_city, ' VARCHAR(50) DEFAULT NULL COMMENT ''登录城市''')
));
PREPARE alterIfNotExists_city FROM @preparedStatement_city;
EXECUTE alterIfNotExists_city;
DEALLOCATE PREPARE alterIfNotExists_city;

-- 添加索引（如果不存在）
-- 检查并添加 province 索引
SET @index_exists_province = (
  SELECT COUNT(*) 
  FROM information_schema.statistics 
  WHERE table_schema = DATABASE() 
    AND table_name = 'user_login_history' 
    AND index_name = 'idx_province'
);

SET @add_index_province = IF(@index_exists_province = 0,
  'ALTER TABLE user_login_history ADD INDEX idx_province (province);',
  'SELECT ''索引 idx_province 已存在'' AS message;'
);

PREPARE stmt_province FROM @add_index_province;
EXECUTE stmt_province;
DEALLOCATE PREPARE stmt_province;

-- 检查并添加 city 索引
SET @index_exists_city = (
  SELECT COUNT(*) 
  FROM information_schema.statistics 
  WHERE table_schema = DATABASE() 
    AND table_name = 'user_login_history' 
    AND index_name = 'idx_city'
);

SET @add_index_city = IF(@index_exists_city = 0,
  'ALTER TABLE user_login_history ADD INDEX idx_city (city);',
  'SELECT ''索引 idx_city 已存在'' AS message;'
);

PREPARE stmt_city FROM @add_index_city;
EXECUTE stmt_city;
DEALLOCATE PREPARE stmt_city;

-- 插入测试地区数据（更新现有记录）
-- 为现有的登录记录随机分配地区信息
UPDATE user_login_history
SET 
  province = CASE (id % 10)
    WHEN 0 THEN '北京'
    WHEN 1 THEN '上海'
    WHEN 2 THEN '广东'
    WHEN 3 THEN '浙江'
    WHEN 4 THEN '江苏'
    WHEN 5 THEN '四川'
    WHEN 6 THEN '湖北'
    WHEN 7 THEN '湖南'
    WHEN 8 THEN '河南'
    WHEN 9 THEN '山东'
  END,
  city = CASE (id % 20)
    WHEN 0 THEN '北京'
    WHEN 1 THEN '上海'
    WHEN 2 THEN '广州'
    WHEN 3 THEN '深圳'
    WHEN 4 THEN '杭州'
    WHEN 5 THEN '宁波'
    WHEN 6 THEN '南京'
    WHEN 7 THEN '苏州'
    WHEN 8 THEN '成都'
    WHEN 9 THEN '绵阳'
    WHEN 10 THEN '武汉'
    WHEN 11 THEN '襄阳'
    WHEN 12 THEN '长沙'
    WHEN 13 THEN '岳阳'
    WHEN 14 THEN '郑州'
    WHEN 15 THEN '洛阳'
    WHEN 16 THEN '济南'
    WHEN 17 THEN '青岛'
    WHEN 18 THEN '天津'
    WHEN 19 THEN '重庆'
  END
WHERE province IS NULL OR province = '';

-- 验证数据
SELECT 
  COUNT(*) as total_records,
  COUNT(DISTINCT province) as province_count,
  COUNT(DISTINCT city) as city_count
FROM user_login_history
WHERE province IS NOT NULL AND province != '';

-- 查看地区分布
SELECT 
  province,
  COUNT(DISTINCT user_id) as user_count,
  COUNT(*) as login_count
FROM user_login_history
WHERE province IS NOT NULL AND province != ''
GROUP BY province
ORDER BY user_count DESC
LIMIT 10;

SELECT 
  province,
  city,
  COUNT(DISTINCT user_id) as user_count,
  COUNT(*) as login_count
FROM user_login_history
WHERE city IS NOT NULL AND city != ''
GROUP BY province, city
ORDER BY user_count DESC
LIMIT 10;

