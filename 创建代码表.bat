@echo off
chcp 65001 >nul
echo ======================================
echo 创建在线代码运行平台相关数据库表
echo ======================================
echo.

REM 设置数据库连接信息
set MYSQL_HOST=127.0.0.1
set MYSQL_PORT=3306
set MYSQL_USER=root
set MYSQL_PASS=root
set MYSQL_DB=hub

echo 连接数据库: %MYSQL_DB%@%MYSQL_HOST%:%MYSQL_PORT%
echo.

REM 执行 SQL 文件
mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% %MYSQL_DB% < sql\code_tables.sql

if %errorlevel% equ 0 (
    echo.
    echo ✓ 数据库表创建成功！
    echo.
    echo 已创建以下表：
    echo   - code_snippets          代码片段表
    echo   - code_executions        代码执行记录表
    echo   - code_collaborations    代码协作会话表
    echo.
) else (
    echo.
    echo ✗ 数据库表创建失败！
    echo 请检查数据库连接信息和 SQL 文件。
    echo.
)

pause


