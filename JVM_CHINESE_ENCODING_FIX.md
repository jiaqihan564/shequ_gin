# JVM 语言中文编码问题修复指南

## 问题描述

在使用在线代码编辑器执行 Java、Scala、Kotlin 代码时，中文字符显示为"？"。

## 根本原因

JVM 语言默认使用平台编码，而 Piston API 运行环境可能使用：
- 默认编码: ISO-8859-1 或 ASCII
- 而不是 UTF-8

这导致中文字符（Unicode范围 U+4E00 到 U+9FA5）无法正确编码和显示。

## 已实施的多层修复方案

### 第一层：JVM 参数设置

**文件**: `internal/services/code_executor.go`

为 JVM 语言添加编译和运行参数：

#### Java
```go
CompileArgs: ["-encoding", "UTF-8", "-J-Dfile.encoding=UTF-8"]
RunArgs: ["-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"]
```
- `-encoding UTF-8`: 告诉 javac 源文件使用 UTF-8 编码
- `-J-Dfile.encoding=UTF-8`: 编译器 JVM 使用 UTF-8
- `-Dfile.encoding=UTF-8`: 运行时 JVM 使用 UTF-8
- `-Duser.language=zh -Duser.country=CN`: 设置区域为中国

#### Scala
```go
CompileArgs: ["-encoding", "UTF-8"]
RunArgs: ["-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"]
```

#### Kotlin
```go
CompileArgs: ["-Dfile.encoding=UTF-8"]
RunArgs: ["-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"]
```

### 第二层：代码智能包装

当检测到 Java 代码包含中文时，自动添加必要的 import：
```java
import java.io.*;
import java.nio.charset.StandardCharsets;
```

这为可能需要的编码转换操作提供支持。

### 第三层：HTTP 请求头设置

设置请求头：
```go
req.Header.Set("Content-Type", "application/json; charset=utf-8")
```

确保请求本身使用 UTF-8 编码。

### 第四层：详细日志记录

记录所有关键信息：
- 发送的编译参数和运行参数
- 完整的请求 JSON
- Piston API 的响应详情
- 输出预览（便于查看编码问题）

## 使用说明

### 1. 重启后端服务

修改代码执行器后，必须重启后端：

```powershell
# 停止当前服务（Ctrl+C）
cd shequ_gin
go run main.go
```

### 2. 测试中文显示

访问代码编辑器，使用以下测试代码：

**Java 测试代码**:
```java
public class Main {
    public static void main(String[] args) {
        System.out.println("你好，世界！");
        System.out.println("中文测试：成功");
        System.out.println("特殊字符：©®™");
        
        // 测试变量
        String name = "张三";
        int age = 25;
        System.out.println("姓名：" + name + "，年龄：" + age);
    }
}
```

**Scala 测试代码**:
```scala
object Main extends App {
  println("你好，世界！")
  println("中文测试：成功")
  
  val name = "李四"
  val age = 30
  println(s"姓名：$name，年龄：$age")
}
```

**Kotlin 测试代码**:
```kotlin
fun main() {
    println("你好，世界！")
    println("中文测试：成功")
    
    val name = "王五"
    val age = 28
    println("姓名：$name，年龄：$age")
}
```

### 3. 查看调试日志

如果中文仍然显示为"？"，检查后端日志：

```powershell
# 查看最新日志
Get-Content shequ_gin\log\2025.10.21.log | Select-Object -Last 50
```

查找以下信息：
- "为 Java 添加 UTF-8 编码参数"
- "Piston API 请求详情"
- "Piston API 响应详情"

## 故障排除

### 情况 1: 参数未生效

**检查日志中的 request_body**，应该包含：
```json
{
  "language": "java",
  "version": "15.0.2",
  "files": [...],
  "compile_args": ["-encoding", "UTF-8", "-J-Dfile.encoding=UTF-8"],
  "run_args": ["-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"]
}
```

### 情况 2: Piston API 不支持这些参数

如果 Piston API 忽略了参数，可能需要：
1. 使用不同的代码执行服务
2. 自建代码执行环境
3. 使用 Unicode 转义（最后手段）

### 情况 3: 前端发送的代码已经损坏

检查前端发送的请求，确保中文字符正确编码。

## 技术细节

### Piston API v2 规范

根据 Piston API 文档，支持的字段：
- `language` (必需): 语言名称
- `version` (必需): 版本号
- `files` (必需): 代码文件数组
- `stdin` (可选): 标准输入
- `args` (可选): 程序参数
- `compile_args` (可选): 编译器参数
- `run_args` (可选): 运行时参数
- `compile_timeout` (可选): 编译超时
- `run_timeout` (可选): 运行超时
- `compile_memory_limit` (可选): 编译内存限制
- `run_memory_limit` (可选): 运行内存限制

### 字符编码流程

1. **前端** → UTF-8 JSON → **后端** 
2. **后端** → UTF-8 JSON → **Piston API**
3. **Piston** → 编译（使用 compile_args）
4. **Piston** → 运行（使用 run_args）
5. **Piston** → UTF-8 响应 → **后端** → **前端**

任何一个环节的编码问题都会导致中文乱码。

## 备选方案

如果所有方案都失败，可以考虑：

### 方案 A: 前端提示
在 Java/Scala/Kotlin 编辑器中添加提示：
"注意：当前环境可能不完全支持中文字符，建议使用英文或拼音"

### 方案 B: 后端转义
将中文转换为 Unicode 转义：
```java
System.out.println("\u4F60\u597D"); // 你好
```

### 方案 C: 切换执行引擎
考虑使用其他代码执行服务，如：
- Judge0
- OneCompiler API
- 自建 Docker 容器执行环境

## 预期结果

修复后，所有 JVM 语言应该能正确显示：
- ✅ 中文字符（你好世界）
- ✅ 中文标点（，。！？）
- ✅ 特殊符号（©®™）
- ✅ 中文变量和注释

## 联系支持

如果问题持续存在，请提供：
1. 测试代码
2. 实际输出（显示"？"的部分）
3. 后端日志中的"Piston API 请求详情"
4. 后端日志中的"Piston API 响应详情"

