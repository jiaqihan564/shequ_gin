@echo off
chcp 65001
echo 正在创建资源评论表...
mysql -h127.0.0.1 -P3306 -uroot -proot hub < sql\resource_comment_tables.sql
if %errorlevel% == 0 (
    echo 资源评论表创建成功！
) else (
    echo 资源评论表创建失败，请检查数据库连接和SQL文件！
)
pause

