#!/bin/bash
# =================================================================
# Hub 数据库一键部署脚本 - 完整版
# =================================================================
# 用途: 一键创建数据库、用户并初始化所有表
# 使用: chmod +x deploy_all.sh && ./deploy_all.sh
# =================================================================

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
DB_NAME="hub"
DB_USER_NEW="hub_user"
DEFAULT_PASS="Hub@2024!Strong"  # 默认密码

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  Hub 数据库一键部署脚本${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 检查MySQL是否安装
if ! command -v mysql &> /dev/null; then
    echo -e "${RED}错误: 未找到 MySQL，请先安装 MySQL${NC}"
    exit 1
fi

# 输入 root 密码
echo -e "${YELLOW}步骤 1/4: 连接数据库${NC}"
echo -e "${YELLOW}请输入 MySQL root 用户密码:${NC}"
read -s MYSQL_ROOT_PASS
echo ""

# 测试连接
echo -e "${BLUE}正在测试数据库连接...${NC}"
if ! mysql -u root -p"$MYSQL_ROOT_PASS" -e "SELECT 1;" &> /dev/null; then
    echo -e "${RED}错误: 数据库连接失败，请检查密码${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 数据库连接成功${NC}"
echo ""

# 创建数据库和用户
echo -e "${YELLOW}步骤 2/4: 创建数据库和用户${NC}"
echo -e "${BLUE}正在创建数据库 '$DB_NAME'...${NC}"

mysql -u root -p"$MYSQL_ROOT_PASS" << EOF
-- 创建数据库
CREATE DATABASE IF NOT EXISTS \`$DB_NAME\` 
  DEFAULT CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;

-- 创建用户
CREATE USER IF NOT EXISTS '$DB_USER_NEW'@'localhost' IDENTIFIED BY '$DEFAULT_PASS';

-- 授予权限
GRANT ALL PRIVILEGES ON \`$DB_NAME\`.* TO '$DB_USER_NEW'@'localhost';
FLUSH PRIVILEGES;

SELECT 'Database and user created successfully!' AS Result;
EOF

echo -e "${GREEN}✓ 数据库和用户创建成功${NC}"
echo -e "${GREEN}  数据库名: $DB_NAME${NC}"
echo -e "${GREEN}  用户名: $DB_USER_NEW${NC}"
echo -e "${GREEN}  密码: $DEFAULT_PASS${NC}"
echo ""

# 执行表结构脚本
echo -e "${YELLOW}步骤 3/4: 创建数据表${NC}"

SQL_FILES=(
    "user_tables.sql"
    "article_tables.sql"
    "code_tables.sql"
    "resource_tables.sql"
    "resource_comment_tables.sql"
    "chat_tables.sql"
    "private_messages.sql"
    "history_tables.sql"
    "statistics_tables.sql"
    "cumulative_stats_tables.sql"
    "password_reset_tokens.sql"
    "add_location_fields_and_test_data.sql"
)

TOTAL=${#SQL_FILES[@]}
CURRENT=0

for file in "${SQL_FILES[@]}"; do
    CURRENT=$((CURRENT + 1))
    
    if [ ! -f "$file" ]; then
        echo -e "${YELLOW}⚠ [$CURRENT/$TOTAL] 跳过: $file (文件不存在)${NC}"
        continue
    fi
    
    echo -e "${BLUE}[$CURRENT/$TOTAL] 执行: $file${NC}"
    
    if mysql -u root -p"$MYSQL_ROOT_PASS" "$DB_NAME" < "$file" 2>&1 | grep -v "Unknown table"; then
        echo -e "${GREEN}  ✓ 完成${NC}"
    else
        echo -e "${RED}  ✗ 执行失败${NC}"
        exit 1
    fi
done

echo ""

# 创建性能索引
echo -e "${YELLOW}步骤 4/4: 创建性能索引${NC}"
if [ -f "create_indexes_simple.sql" ]; then
    echo -e "${BLUE}正在创建索引...${NC}"
    mysql -u root -p"$MYSQL_ROOT_PASS" "$DB_NAME" < "create_indexes_simple.sql" 2>&1 | grep -v "Duplicate key"
    echo -e "${GREEN}✓ 索引创建完成${NC}"
else
    echo -e "${YELLOW}⚠ 索引文件不存在，跳过${NC}"
fi

echo ""

# 验证部署结果
echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  验证部署结果${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 统计表数量
TABLE_COUNT=$(mysql -u root -p"$MYSQL_ROOT_PASS" "$DB_NAME" -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$DB_NAME';")
echo -e "${GREEN}✓ 已创建表数量: $TABLE_COUNT${NC}"

# 检查关键表
CRITICAL_TABLES=(
    "user_auth"
    "user_profile"
    "articles"
    "code_snippets"
    "resources"
    "chat_messages"
    "cumulative_statistics"
)

echo ""
echo -e "${BLUE}关键表检查:${NC}"
for table in "${CRITICAL_TABLES[@]}"; do
    if mysql -u root -p"$MYSQL_ROOT_PASS" "$DB_NAME" -e "DESC $table;" &> /dev/null; then
        echo -e "${GREEN}  ✓ $table${NC}"
    else
        echo -e "${RED}  ✗ $table (不存在)${NC}"
    fi
done

# 统计初始数据
echo ""
echo -e "${BLUE}初始数据检查:${NC}"
STATS_COUNT=$(mysql -u root -p"$MYSQL_ROOT_PASS" "$DB_NAME" -N -e "SELECT COUNT(*) FROM cumulative_statistics;" 2>/dev/null || echo "0")
echo -e "${GREEN}  ✓ 累计统计数据: $STATS_COUNT 条${NC}"

RESOURCE_CAT=$(mysql -u root -p"$MYSQL_ROOT_PASS" "$DB_NAME" -N -e "SELECT COUNT(*) FROM resource_categories;" 2>/dev/null || echo "0")
echo -e "${GREEN}  ✓ 资源分类: $RESOURCE_CAT 条${NC}"

# 完成
echo ""
echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN}  ✓ 部署完成！${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo -e "${YELLOW}数据库配置信息:${NC}"
echo -e "  数据库名: ${GREEN}$DB_NAME${NC}"
echo -e "  用户名: ${GREEN}$DB_USER_NEW${NC}"
echo -e "  密码: ${GREEN}$DEFAULT_PASS${NC}"
echo -e "  字符集: ${GREEN}utf8mb4${NC}"
echo ""
echo -e "${YELLOW}在 config.yaml 中配置:${NC}"
cat << CONFIG
  database:
    host: "127.0.0.1"
    port: "3306"
    username: "$DB_USER_NEW"
    password: "$DEFAULT_PASS"
    database: "$DB_NAME"
    charset: "utf8mb4"
CONFIG
echo ""
echo -e "${RED}重要: 请修改数据库密码为更强的密码！${NC}"
echo -e "${GREEN}现在可以启动应用程序了！${NC}"
echo ""

