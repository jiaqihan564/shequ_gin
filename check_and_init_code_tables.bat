@echo off
chcp 65001 >nul
echo ======================================
echo 代码表诊断和初始化工具
echo ======================================
echo.

REM 设置数据库连接信息
set MYSQL_HOST=127.0.0.1
set MYSQL_PORT=3306
set MYSQL_USER=root
set MYSQL_PASS=root
set MYSQL_DB=hub

echo [步骤 1] 检查数据库连接...
mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% -e "SELECT 1;" >nul 2>&1
if %errorlevel% neq 0 (
    echo ✗ 无法连接到数据库！
    echo   请检查 MySQL 是否运行，以及用户名密码是否正确
    echo   当前配置: %MYSQL_USER%@%MYSQL_HOST%:%MYSQL_PORT%
    pause
    exit /b 1
)
echo ✓ 数据库连接成功
echo.

echo [步骤 2] 检查数据库 %MYSQL_DB% 是否存在...
mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% -e "USE %MYSQL_DB%;" >nul 2>&1
if %errorlevel% neq 0 (
    echo ✗ 数据库 %MYSQL_DB% 不存在！
    echo   请先创建数据库或修改配置
    pause
    exit /b 1
)
echo ✓ 数据库 %MYSQL_DB% 存在
echo.

echo [步骤 3] 检查代码相关表...
echo.

REM 检查 code_snippets
mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% %MYSQL_DB% -e "DESC code_snippets;" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✓ code_snippets 表已存在
) else (
    echo ✗ code_snippets 表不存在
)

REM 检查 code_executions
mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% %MYSQL_DB% -e "DESC code_executions;" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✓ code_executions 表已存在
) else (
    echo ✗ code_executions 表不存在 ^(执行历史功能需要此表^)
)

REM 检查 code_collaborations
mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% %MYSQL_DB% -e "DESC code_collaborations;" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✓ code_collaborations 表已存在
) else (
    echo ✗ code_collaborations 表不存在
)

echo.
echo [步骤 4] 是否需要创建/更新表？
echo.
set /p CREATE="输入 Y 创建表，输入 N 退出: "

if /i "%CREATE%"=="Y" (
    echo.
    echo 正在创建表...
    mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% %MYSQL_DB% < sql\code_tables.sql
    
    if %errorlevel% equ 0 (
        echo.
        echo ✓ 表创建/更新成功！
        echo.
        echo 已创建以下表：
        echo   - code_snippets          代码片段表
        echo   - code_executions        代码执行记录表
        echo   - code_collaborations    代码协作会话表
        echo.
        echo [步骤 5] 查看表记录数...
        echo.
        mysql -h%MYSQL_HOST% -P%MYSQL_PORT% -u%MYSQL_USER% -p%MYSQL_PASS% %MYSQL_DB% -e "SELECT 'code_snippets' AS 表名, COUNT(*) AS 记录数 FROM code_snippets UNION ALL SELECT 'code_executions', COUNT(*) FROM code_executions UNION ALL SELECT 'code_collaborations', COUNT(*) FROM code_collaborations;"
        echo.
        echo 提示：
        echo   - 如果 code_executions 记录数为 0，请在代码编辑器中运行一次代码
        echo   - 执行后刷新"代码历史"页面即可看到执行记录
        echo.
    ) else (
        echo.
        echo ✗ 表创建失败！
        echo   请检查 SQL 文件和数据库权限
        echo.
    )
) else (
    echo.
    echo 已取消操作
    echo.
)

pause

