@echo off
REM =================================================================
REM 数据库初始化脚本 - Windows
REM =================================================================
REM 用途: 一键初始化hub数据库的所有表和数据
REM 使用: init_database.bat
REM =================================================================

chcp 65001 >nul
setlocal enabledelayedexpansion

REM 配置
set DB_NAME=hub
set DB_USER=root

echo ======================================
echo   数据库初始化脚本
echo ======================================
echo.

REM 检查MySQL是否安装
where mysql >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [错误] 未找到mysql命令，请先安装MySQL
    pause
    exit /b 1
)

REM 提示输入密码
set /p DB_PASSWORD="请输入MySQL root用户密码: "
echo.

REM 测试数据库连接
echo 正在测试数据库连接...
mysql -u %DB_USER% -p%DB_PASSWORD% -e "SELECT 1;" >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [错误] 数据库连接失败，请检查用户名和密码
    pause
    exit /b 1
)
echo [成功] 数据库连接成功
echo.

REM 创建数据库
echo 正在创建数据库 '%DB_NAME%'...
mysql -u %DB_USER% -p%DB_PASSWORD% -e "CREATE DATABASE IF NOT EXISTS %DB_NAME% CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
if %ERRORLEVEL% neq 0 (
    echo [错误] 数据库创建失败
    pause
    exit /b 1
)
echo [成功] 数据库创建成功
echo.

REM SQL脚本列表
set SQL_FILES=user_tables.sql article_tables.sql code_tables.sql resource_tables.sql resource_comment_tables.sql chat_tables.sql private_messages.sql history_tables.sql statistics_tables.sql cumulative_stats_tables.sql password_reset_tokens.sql add_location_fields_and_test_data.sql create_indexes_simple.sql

echo 开始执行SQL脚本...
echo.

set COUNT=0
set TOTAL=13

for %%f in (%SQL_FILES%) do (
    set /a COUNT+=1
    
    if not exist "%%f" (
        echo [!COUNT!/%TOTAL%] [跳过] %%f ^(文件不存在^)
        echo.
    ) else (
        echo [!COUNT!/%TOTAL%] 执行: %%f
        mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% < "%%f" 2>nul
        if !ERRORLEVEL! equ 0 (
            echo   [成功] 完成
        ) else (
            echo   [警告] 可能有部分错误，但继续执行
        )
        echo.
    )
)

REM 验证结果
echo ======================================
echo   验证数据库创建结果
echo ======================================
echo.

REM 统计表数量
for /f %%i in ('mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%DB_NAME%';"') do set TABLE_COUNT=%%i
echo [成功] 已创建表数量: %TABLE_COUNT%
echo.

REM 检查关键表
echo 关键表检查:
set TABLES=user_auth user_profile articles code_snippets resources chat_messages private_conversations cumulative_statistics

for %%t in (%TABLES%) do (
    mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% -e "DESC %%t;" >nul 2>nul
    if !ERRORLEVEL! equ 0 (
        echo   [成功] %%t
    ) else (
        echo   [错误] %%t ^(不存在^)
    )
)
echo.

REM 检查初始数据
echo 初始数据检查:

for /f %%i in ('mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% -N -e "SELECT COUNT(*) FROM cumulative_statistics;"') do (
    echo   [成功] 累计统计数据: %%i 条
)

for /f %%i in ('mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% -N -e "SELECT COUNT(*) FROM article_categories;"') do (
    echo   [成功] 文章分类: %%i 条
)

for /f %%i in ('mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% -N -e "SELECT COUNT(*) FROM article_tags;"') do (
    echo   [成功] 文章标签: %%i 条
)

for /f %%i in ('mysql -u %DB_USER% -p%DB_PASSWORD% %DB_NAME% -N -e "SELECT COUNT(*) FROM resource_categories;"') do (
    echo   [成功] 资源分类: %%i 条
)

echo.
echo ======================================
echo   [成功] 数据库初始化完成！
echo ======================================
echo.
echo 提示:
echo   1. 管理员账号将在应用首次启动时自动创建
echo   2. 请在 config.yaml 中配置数据库连接信息
echo   3. 请在 config.yaml 中设置管理员用户名和密码
echo.
echo 可以开始运行应用程序了！
echo.
pause

