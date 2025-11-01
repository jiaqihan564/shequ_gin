# =================================================================
# 数据库初始化脚本 - PowerShell
# =================================================================
# 用途: 一键初始化hub数据库的所有表和数据
# 使用: .\init_database.ps1
# =================================================================

$ErrorActionPreference = "Stop"

# 配置
$DB_NAME = "hub"
$DB_USER = "root"

Write-Host "======================================" -ForegroundColor Blue
Write-Host "  数据库初始化脚本" -ForegroundColor Blue
Write-Host "======================================" -ForegroundColor Blue
Write-Host ""

# 检查MySQL是否安装
try {
    $null = Get-Command mysql -ErrorAction Stop
} catch {
    Write-Host "[错误] 未找到mysql命令，请先安装MySQL" -ForegroundColor Red
    exit 1
}

# 提示输入密码
$DB_PASSWORD = Read-Host "请输入MySQL root用户密码" -AsSecureString
$DB_PASSWORD_PLAIN = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($DB_PASSWORD)
)
Write-Host ""

# 测试数据库连接
Write-Host "正在测试数据库连接..." -ForegroundColor Blue
try {
    $null = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN -e "SELECT 1;" 2>&1
    if ($LASTEXITCODE -ne 0) { throw }
    Write-Host "[成功] 数据库连接成功" -ForegroundColor Green
} catch {
    Write-Host "[错误] 数据库连接失败，请检查用户名和密码" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 创建数据库
Write-Host "正在创建数据库 '$DB_NAME'..." -ForegroundColor Blue
try {
    $null = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN -e "CREATE DATABASE IF NOT EXISTS $DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>&1
    Write-Host "[成功] 数据库创建成功" -ForegroundColor Green
} catch {
    Write-Host "[错误] 数据库创建失败" -ForegroundColor Red
    exit 1
}
Write-Host ""

# SQL脚本列表
$SQL_FILES = @(
    "user_tables.sql",
    "article_tables.sql",
    "code_tables.sql",
    "resource_tables.sql",
    "resource_comment_tables.sql",
    "chat_tables.sql",
    "private_messages.sql",
    "history_tables.sql",
    "statistics_tables.sql",
    "cumulative_stats_tables.sql",
    "password_reset_tokens.sql",
    "add_location_fields_and_test_data.sql",
    "create_indexes_simple.sql"
)

Write-Host "开始执行SQL脚本..." -ForegroundColor Blue
Write-Host ""

$TOTAL = $SQL_FILES.Count
$CURRENT = 0

foreach ($file in $SQL_FILES) {
    $CURRENT++
    
    if (-not (Test-Path $file)) {
        Write-Host "[$CURRENT/$TOTAL] [跳过] $file (文件不存在)" -ForegroundColor Yellow
        Write-Host ""
        continue
    }
    
    Write-Host "[$CURRENT/$TOTAL] 执行: $file" -ForegroundColor Blue
    
    try {
        Get-Content $file -Raw -Encoding UTF8 | & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME 2>&1 | Out-Null
        Write-Host "  [成功] 完成" -ForegroundColor Green
    } catch {
        Write-Host "  [警告] 可能有部分错误，但继续执行" -ForegroundColor Yellow
    }
    Write-Host ""
}

# 验证结果
Write-Host ""
Write-Host "======================================" -ForegroundColor Blue
Write-Host "  验证数据库创建结果" -ForegroundColor Blue
Write-Host "======================================" -ForegroundColor Blue
Write-Host ""

# 统计表数量
$TABLE_COUNT = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$DB_NAME';"
Write-Host "[成功] 已创建表数量: $TABLE_COUNT" -ForegroundColor Green
Write-Host ""

# 检查关键表
Write-Host "关键表检查:" -ForegroundColor Blue
$CRITICAL_TABLES = @(
    "user_auth",
    "user_profile",
    "articles",
    "code_snippets",
    "resources",
    "chat_messages",
    "private_conversations",
    "cumulative_statistics"
)

foreach ($table in $CRITICAL_TABLES) {
    try {
        $null = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME -e "DESC $table;" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  [成功] $table" -ForegroundColor Green
        } else {
            Write-Host "  [错误] $table (不存在)" -ForegroundColor Red
        }
    } catch {
        Write-Host "  [错误] $table (不存在)" -ForegroundColor Red
    }
}
Write-Host ""

# 检查初始数据
Write-Host "初始数据检查:" -ForegroundColor Blue

$STATS_COUNT = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME -N -e "SELECT COUNT(*) FROM cumulative_statistics;"
Write-Host "  [成功] 累计统计数据: $STATS_COUNT 条" -ForegroundColor Green

$ARTICLE_CAT_COUNT = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME -N -e "SELECT COUNT(*) FROM article_categories;"
Write-Host "  [成功] 文章分类: $ARTICLE_CAT_COUNT 条" -ForegroundColor Green

$ARTICLE_TAG_COUNT = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME -N -e "SELECT COUNT(*) FROM article_tags;"
Write-Host "  [成功] 文章标签: $ARTICLE_TAG_COUNT 条" -ForegroundColor Green

$RESOURCE_CAT_COUNT = & mysql -u $DB_USER -p$DB_PASSWORD_PLAIN $DB_NAME -N -e "SELECT COUNT(*) FROM resource_categories;"
Write-Host "  [成功] 资源分类: $RESOURCE_CAT_COUNT 条" -ForegroundColor Green

# 完成
Write-Host ""
Write-Host "======================================" -ForegroundColor Blue
Write-Host "  [成功] 数据库初始化完成！" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Blue
Write-Host ""
Write-Host "提示:" -ForegroundColor Yellow
Write-Host "  1. 管理员账号将在应用首次启动时自动创建"
Write-Host "  2. 请在 config.yaml 中配置数据库连接信息"
Write-Host "  3. 请在 config.yaml 中设置管理员用户名和密码"
Write-Host ""
Write-Host "可以开始运行应用程序了！" -ForegroundColor Green
Write-Host ""

