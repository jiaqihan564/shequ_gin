package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gin/internal/models"
	"gin/internal/utils"
	"net/http"
	"time"
)

// CodeExecutor 代码执行器接口
type CodeExecutor interface {
	Execute(ctx context.Context, language, code, stdin string) (*models.ExecuteCodeResponse, error)
	GetSupportedLanguages() []models.LanguageInfo
}

// PistonCodeExecutor Piston API 代码执行器实现
type PistonCodeExecutor struct {
	apiURL  string
	timeout time.Duration
	client  *http.Client
}

// 支持的语言配置
var supportedLanguages = map[string]models.LanguageInfo{
	"python": {
		ID:         "python",
		Name:       "Python",
		Version:    "3.10.0",
		PistonName: "python",
		DefaultCode: `# Python 示例代码
print("Hello, World!")

# 获取用户输入
# name = input("请输入你的名字: ")
# print(f"你好, {name}!")`,
	},
	"javascript": {
		ID:         "javascript",
		Name:       "JavaScript (Node.js)",
		Version:    "18.15.0",
		PistonName: "javascript",
		DefaultCode: `// JavaScript 示例代码
console.log("Hello, World!");

// 获取用户输入
// const readline = require('readline');
// const rl = readline.createInterface({
//   input: process.stdin,
//   output: process.stdout
// });`,
	},
	"java": {
		ID:         "java",
		Name:       "Java",
		Version:    "15.0.2",
		PistonName: "java",
		DefaultCode: `// Java 示例代码
public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
        
        // 获取用户输入
        // Scanner scanner = new Scanner(System.in);
        // String name = scanner.nextLine();
        // System.out.println("你好, " + name + "!");
    }
}`,
	},
	"cpp": {
		ID:         "cpp",
		Name:       "C++",
		Version:    "10.2.0",
		PistonName: "cpp",
		DefaultCode: `// C++ 示例代码
#include <iostream>
using namespace std;

int main() {
    cout << "Hello, World!" << endl;
    
    // 获取用户输入
    // string name;
    // cout << "请输入你的名字: ";
    // cin >> name;
    // cout << "你好, " << name << "!" << endl;
    
    return 0;
}`,
	},
	"c": {
		ID:         "c",
		Name:       "C",
		Version:    "10.2.0",
		PistonName: "c",
		DefaultCode: `// C 示例代码
#include <stdio.h>

int main() {
    printf("Hello, World!\n");
    
    // 获取用户输入
    // char name[100];
    // printf("请输入你的名字: ");
    // scanf("%s", name);
    // printf("你好, %s!\n", name);
    
    return 0;
}`,
	},
	"go": {
		ID:         "go",
		Name:       "Go",
		Version:    "1.16.2",
		PistonName: "go",
		DefaultCode: `// Go 示例代码
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
    
    // 获取用户输入
    // var name string
    // fmt.Print("请输入你的名字: ")
    // fmt.Scanln(&name)
    // fmt.Printf("你好, %s!\n", name)
}`,
	},
	"rust": {
		ID:         "rust",
		Name:       "Rust",
		Version:    "1.68.2",
		PistonName: "rust",
		DefaultCode: `// Rust 示例代码
fn main() {
    println!("Hello, World!");
    
    // 获取用户输入
    // use std::io;
    // let mut name = String::new();
    // io::stdin().read_line(&mut name).expect("Failed to read line");
    // println!("你好, {}!", name.trim());
}`,
	},
	"php": {
		ID:         "php",
		Name:       "PHP",
		Version:    "8.2.3",
		PistonName: "php",
		DefaultCode: `<?php
// PHP 示例代码
echo "Hello, World!\\n";
?>`,
	},
	"typescript": {
		ID:         "typescript",
		Name:       "TypeScript",
		Version:    "5.0.3",
		PistonName: "typescript",
		DefaultCode: `// TypeScript 示例代码
const greeting: string = "Hello, World!";
console.log(greeting);`,
	},
	"ruby": {
		ID:         "ruby",
		Name:       "Ruby",
		Version:    "3.0.1",
		PistonName: "ruby",
		DefaultCode: `# Ruby 示例代码
puts "Hello, World!"`,
	},
	"swift": {
		ID:         "swift",
		Name:       "Swift",
		Version:    "5.3.3",
		PistonName: "swift",
		DefaultCode: `// Swift 示例代码
print("Hello, World!")`,
	},
	"kotlin": {
		ID:         "kotlin",
		Name:       "Kotlin",
		Version:    "1.8.20",
		PistonName: "kotlin",
		DefaultCode: `// Kotlin 示例代码
fun main() {
    println("Hello, World!")
}`,
	},
	"bash": {
		ID:         "bash",
		Name:       "Bash",
		Version:    "5.2.0",
		PistonName: "bash",
		DefaultCode: `#!/bin/bash
# Bash 示例代码
echo "Hello, World!"`,
	},
	"lua": {
		ID:         "lua",
		Name:       "Lua",
		Version:    "5.4.4",
		PistonName: "lua",
		DefaultCode: `-- Lua 示例代码
print("Hello, World!")`,
	},
	"scala": {
		ID:         "scala",
		Name:       "Scala",
		Version:    "3.2.2",
		PistonName: "scala",
		DefaultCode: `// Scala 示例代码
object Main extends App {
  println("Hello, World!")
}`,
	},
	"haskell": {
		ID:         "haskell",
		Name:       "Haskell",
		Version:    "9.0.1",
		PistonName: "haskell",
		DefaultCode: `-- Haskell 示例代码
main :: IO ()
main = putStrLn "Hello, World!"`,
	},
	"perl": {
		ID:         "perl",
		Name:       "Perl",
		Version:    "5.36.0",
		PistonName: "perl",
		DefaultCode: `# Perl 示例代码
print "Hello, World!\\n";`,
	},
}

// NewPistonCodeExecutor 创建新的 Piston 代码执行器
func NewPistonCodeExecutor(apiURL string, timeout time.Duration) CodeExecutor {
	return &PistonCodeExecutor{
		apiURL:  apiURL,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Execute 执行代码
func (e *PistonCodeExecutor) Execute(ctx context.Context, language, code, stdin string) (*models.ExecuteCodeResponse, error) {
	logger := utils.GetLogger()

	// 验证语言是否支持
	langInfo, ok := supportedLanguages[language]
	if !ok {
		return nil, fmt.Errorf("不支持的语言: %s", language)
	}

	// 对于JVM语言，如果代码包含中文，尝试在代码层面处理编码
	if language == "java" && containsChinese(code) {
		code = wrapJavaCodeWithUTF8(code)
		logger.Debug("为 Java 代码添加 UTF-8 编码设置")
	}

	// 构建 Piston API 请求
	pistonReq := models.PistonExecuteRequest{
		Language: langInfo.PistonName,
		Version:  langInfo.Version,
		Files: []struct {
			Content string `json:"content"`
		}{
			{Content: code},
		},
		Stdin: stdin,
	}

	// 为 JVM 语言添加 UTF-8 编码参数，解决中文显示问题
	// 尝试多种参数组合方式
	switch language {
	case "java":
		// Java 需要编译和运行时都指定UTF-8
		pistonReq.CompileArgs = []string{"-encoding", "UTF-8", "-J-Dfile.encoding=UTF-8"}
		pistonReq.RunArgs = []string{"-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"}
		logger.Debug("为 Java 添加 UTF-8 编码参数",
			"compile_args", pistonReq.CompileArgs,
			"run_args", pistonReq.RunArgs)
	case "scala":
		// Scala 使用 scalac 编译器参数
		pistonReq.CompileArgs = []string{"-encoding", "UTF-8"}
		pistonReq.RunArgs = []string{"-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"}
		logger.Debug("为 Scala 添加 UTF-8 编码参数",
			"compile_args", pistonReq.CompileArgs,
			"run_args", pistonReq.RunArgs)
	case "kotlin":
		// Kotlin 编译器参数
		pistonReq.CompileArgs = []string{"-Dfile.encoding=UTF-8"}
		pistonReq.RunArgs = []string{"-Dfile.encoding=UTF-8", "-Duser.language=zh", "-Duser.country=CN"}
		logger.Debug("为 Kotlin 添加 UTF-8 编码参数",
			"compile_args", pistonReq.CompileArgs,
			"run_args", pistonReq.RunArgs)
	}

	reqBody, err := json.Marshal(pistonReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 记录完整的请求体（用于调试JVM编码问题）
	logger.Debug("Piston API 请求详情",
		"language", language,
		"piston_name", langInfo.PistonName,
		"version", langInfo.Version,
		"compile_args", pistonReq.CompileArgs,
		"run_args", pistonReq.RunArgs,
		"request_body", string(reqBody))

	// 记录开始时间
	startTime := time.Now()

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", e.apiURL+"/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// 发送请求
	resp, err := e.client.Do(req)
	if err != nil {
		logger.Error("Piston API 请求失败", "error", err)
		return &models.ExecuteCodeResponse{
			Output:        "",
			Error:         "代码执行超时或服务不可用",
			ExecutionTime: int(time.Since(startTime).Milliseconds()),
			Status:        "timeout",
		}, nil
	}
	defer resp.Body.Close()

	// 计算执行时间
	executionTime := int(time.Since(startTime).Milliseconds())

	// 解析响应
	var pistonResp models.PistonExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&pistonResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 记录响应详情（用于调试编码问题）
	logger.Debug("Piston API 响应详情",
		"language", language,
		"stdout_length", len(pistonResp.Run.Stdout),
		"stderr_length", len(pistonResp.Run.Stderr),
		"exit_code", pistonResp.Run.Code,
		"stdout_preview", truncateString(pistonResp.Run.Stdout, 200),
		"stderr_preview", truncateString(pistonResp.Run.Stderr, 200))

	// 构建返回结果（不包含内存数据，因为公共 Piston API 不提供真实内存信息）
	result := &models.ExecuteCodeResponse{
		ExecutionTime: executionTime,
		MemoryUsage:   0, // 不再提供内存数据
	}

	// 判断执行状态
	if pistonResp.Run.Code == 0 && pistonResp.Run.Stderr == "" {
		result.Status = "success"
		result.Output = pistonResp.Run.Stdout
	} else if pistonResp.Run.Signal != "" {
		result.Status = "timeout"
		result.Error = fmt.Sprintf("执行被信号终止: %s", pistonResp.Run.Signal)
		result.Output = pistonResp.Run.Stdout
	} else {
		result.Status = "error"
		result.Error = pistonResp.Run.Stderr
		result.Output = pistonResp.Run.Stdout
	}

	logger.Info("代码执行完成",
		"language", language,
		"status", result.Status,
		"execution_time", executionTime,
		"code_length", len(code))

	return result, nil
}

// GetSupportedLanguages 获取支持的语言列表
func (e *PistonCodeExecutor) GetSupportedLanguages() []models.LanguageInfo {
	languages := make([]models.LanguageInfo, 0, len(supportedLanguages))
	for _, lang := range supportedLanguages {
		languages = append(languages, lang)
	}
	return languages
}

// truncateString 截断字符串用于日志
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// containsChinese 检查字符串是否包含中文字符
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FA5 {
			return true
		}
	}
	return false
}

// wrapJavaCodeWithUTF8 为Java代码包装UTF-8编码设置
func wrapJavaCodeWithUTF8(code string) string {
	// 在 System.out.println 调用前设置编码
	// 注意：这是一个备用方案，主要依赖运行参数
	wrapper := `import java.io.*;
import java.nio.charset.StandardCharsets;

// 原始代码开始
`
	// 检查是否已经有import语句
	if !bytes.Contains([]byte(code), []byte("import")) {
		return wrapper + code
	}
	// 如果已有import，直接返回原代码（避免重复包装）
	return code
}
