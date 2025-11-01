#!/bin/bash
# =================================================================
# 数据库初始化脚本 - Linux/Mac
# =================================================================
# 用途: 一键初始化hub数据库的所有表和数据
# 使用: ./init_database.sh
# =================================================================

set -e  # 遇到错误立即退出

# 配置
DB_NAME="hub"
DB_USER="root"

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  数据库初始化脚本${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 检查MySQL是否安装
if ! command -v mysql &> /dev/null; then
    echo -e "${RED}错误: 未找到mysql命令，请先安装MySQL${NC}"
    exit 1
fi

# 提示输入密码
echo -e "${YELLOW}请输入MySQL root用户密码:${NC}"
read -s DB_PASSWORD

# 测试数据库连接
echo ""
echo -e "${BLUE}正在测试数据库连接...${NC}"
if ! mysql -u "$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1;" &> /dev/null; then
    echo -e "${RED}错误: 数据库连接失败，请检查用户名和密码${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 数据库连接成功${NC}"

# 创建数据库（如果不存在）
echo ""
echo -e "${BLUE}正在创建数据库 '$DB_NAME'...${NC}"
mysql -u "$DB_USER" -p"$DB_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS $DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
echo -e "${GREEN}✓ 数据库创建成功${NC}"

# SQL脚本执行顺序
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
    "create_indexes_simple.sql"
)

# 执行SQL脚本
echo ""
echo -e "${BLUE}开始执行SQL脚本...${NC}"
echo ""

TOTAL=${#SQL_FILES[@]}
CURRENT=0

for file in "${SQL_FILES[@]}"; do
    CURRENT=$((CURRENT + 1))
    
    if [ ! -f "$file" ]; then
        echo -e "${YELLOW}⚠ [$CURRENT/$TOTAL] 跳过: $file (文件不存在)${NC}"
        continue
    fi
    
    echo -e "${BLUE}[$CURRENT/$TOTAL] 执行: $file${NC}"
    
    if mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" < "$file" 2>&1 | grep -v "Unknown table"; then
        echo -e "${GREEN}  ✓ 完成${NC}"
    else
        echo -e "${RED}  ✗ 执行失败${NC}"
        exit 1
    fi
    echo ""
done

# 验证结果
echo ""
echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  验证数据库创建结果${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 统计表数量
TABLE_COUNT=$(mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$DB_NAME';")
echo -e "${GREEN}✓ 已创建表数量: $TABLE_COUNT${NC}"

# 检查关键表
CRITICAL_TABLES=(
    "user_auth"
    "user_profile"
    "articles"
    "code_snippets"
    "resources"
    "chat_messages"
    "private_conversations"
    "cumulative_statistics"
)

echo ""
echo -e "${BLUE}关键表检查:${NC}"
for table in "${CRITICAL_TABLES[@]}"; do
    if mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -e "DESC $table;" &> /dev/null; then
        echo -e "${GREEN}  ✓ $table${NC}"
    else
        echo -e "${RED}  ✗ $table (不存在)${NC}"
    fi
done

# 检查初始数据
echo ""
echo -e "${BLUE}初始数据检查:${NC}"

STATS_COUNT=$(mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -N -e "SELECT COUNT(*) FROM cumulative_statistics;")
echo -e "${GREEN}  ✓ 累计统计数据: $STATS_COUNT 条${NC}"

ARTICLE_CAT_COUNT=$(mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -N -e "SELECT COUNT(*) FROM article_categories;")
echo -e "${GREEN}  ✓ 文章分类: $ARTICLE_CAT_COUNT 条${NC}"

ARTICLE_TAG_COUNT=$(mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -N -e "SELECT COUNT(*) FROM article_tags;")
echo -e "${GREEN}  ✓ 文章标签: $ARTICLE_TAG_COUNT 条${NC}"

RESOURCE_CAT_COUNT=$(mysql -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -N -e "SELECT COUNT(*) FROM resource_categories;")
echo -e "${GREEN}  ✓ 资源分类: $RESOURCE_CAT_COUNT 条${NC}"

# 完成
echo ""
echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN}  ✓ 数据库初始化完成！${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo -e "  1. 管理员账号将在应用首次启动时自动创建"
echo -e "  2. 请在 config.yaml 中配置数据库连接信息"
echo -e "  3. 请在 config.yaml 中设置管理员用户名和密码"
echo ""
echo -e "${GREEN}可以开始运行应用程序了！${NC}"
echo ""

