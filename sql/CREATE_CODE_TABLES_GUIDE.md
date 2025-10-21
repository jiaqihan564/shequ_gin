# 创建代码表指南

## 问题

如果您在访问"代码历史"中的"执行记录"时看不到任何数据，可能是因为数据库中缺少相关表。

## 解决方案

### 方法 1: 使用批处理文件（Windows 推荐）

1. 确保 MySQL 已安装并运行
2. 双击运行 `创建代码表.bat` 文件
3. 等待表创建完成

### 方法 2: 手动执行 SQL（跨平台）

使用 MySQL 命令行或图形化工具（如 Navicat、phpMyAdmin）执行以下步骤：

```bash
# 1. 登录 MySQL
mysql -u root -p

# 2. 选择数据库
use hub;

# 3. 执行 SQL 文件
source C:/Users/A2322/Desktop/bishe/shequ_gin/sql/code_tables.sql;
```

或者直接在 MySQL 客户端中复制粘贴 `code_tables.sql` 的内容执行。

### 方法 3: 使用 PowerShell（Windows）

```powershell
cd C:\Users\A2322\Desktop\bishe\shequ_gin
mysql -h127.0.0.1 -P3306 -uroot -proot hub < sql\code_tables.sql
```

## 创建的表

执行后会创建以下三个表：

1. **code_snippets** - 代码片段表
2. **code_executions** - 代码执行记录表（修复执行历史显示问题）
3. **code_collaborations** - 代码协作会话表

## 验证表是否创建成功

在 MySQL 中执行：

```sql
USE hub;
SHOW TABLES LIKE 'code_%';
```

应该看到三个表：
- code_collaborations
- code_executions
- code_snippets

## 检查表结构

```sql
DESC code_executions;
```

## 常见问题

### Q: 运行 .bat 文件时报错 "Access denied"
**A**: 检查 `创建代码表.bat` 文件中的数据库密码是否正确（默认是 root）

### Q: 表已存在怎么办？
**A**: SQL 文件使用了 `CREATE TABLE IF NOT EXISTS`，重复执行不会报错

### Q: 执行后还是没有数据
**A**: 
1. 检查是否成功创建了表
2. 在代码编辑器中运行一次代码
3. 刷新"执行记录"页面
4. 查看后端日志确认记录是否保存

## 数据库配置

默认配置（在 `config.yaml` 中）：
- 主机: 127.0.0.1
- 端口: 3306
- 用户名: root
- 密码: root
- 数据库: hub

如果您的配置不同，请修改相应的 `.bat` 文件或手动执行 SQL。

