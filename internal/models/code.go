package models

import "time"

// CodeSnippet 代码片段结构体
type CodeSnippet struct {
	ID          uint      `json:"id" db:"id"`
	UserID      uint      `json:"user_id" db:"user_id"`
	Title       string    `json:"title" db:"title"`
	Language    string    `json:"language" db:"language"`
	Code        string    `json:"code" db:"code"`
	Description string    `json:"description" db:"description"`
	IsPublic    bool      `json:"is_public" db:"is_public"`
	ShareToken  *string   `json:"share_token,omitempty" db:"share_token"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CodeExecution 代码执行记录结构体
type CodeExecution struct {
	ID            uint      `json:"id" db:"id"`
	SnippetID     *uint     `json:"snippet_id,omitempty" db:"snippet_id"`
	UserID        uint      `json:"user_id" db:"user_id"`
	Language      string    `json:"language" db:"language"`
	Code          string    `json:"code" db:"code"`
	Stdin         string    `json:"stdin" db:"stdin"`
	Output        string    `json:"output" db:"output"`
	Error         string    `json:"error" db:"error"`
	ExecutionTime *int      `json:"execution_time,omitempty" db:"execution_time"` // 毫秒
	MemoryUsage   *int64    `json:"memory_usage,omitempty" db:"memory_usage"`     // 字节
	Status        string    `json:"status" db:"status"`                           // success, error, timeout
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// CodeCollaboration 代码协作会话结构体
type CodeCollaboration struct {
	ID           uint      `json:"id" db:"id"`
	SnippetID    uint      `json:"snippet_id" db:"snippet_id"`
	SessionToken string    `json:"session_token" db:"session_token"`
	ActiveUsers  string    `json:"active_users" db:"active_users"` // JSON 字符串
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
}

// ExecuteCodeRequest 执行代码请求
type ExecuteCodeRequest struct {
	Language string `json:"language" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Stdin    string `json:"stdin"`
	SaveAs   string `json:"save_as"` // 可选：保存代码片段的标题
}

// ExecuteCodeResponse 执行代码响应
type ExecuteCodeResponse struct {
	Output        string `json:"output"`
	Error         string `json:"error,omitempty"`
	ExecutionTime int    `json:"execution_time"` // 毫秒
	MemoryUsage   int64  `json:"memory_usage"`   // 字节
	Status        string `json:"status"`         // success, error, timeout
	SnippetID     *uint  `json:"snippet_id,omitempty"`
}

// SaveSnippetRequest 保存代码片段请求
type SaveSnippetRequest struct {
	Title       string `json:"title" binding:"required"`
	Language    string `json:"language" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// UpdateSnippetRequest 更新代码片段请求
type UpdateSnippetRequest struct {
	Title       string `json:"title"`
	Code        string `json:"code"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"is_public"`
}

// ShareSnippetResponse 分享代码片段响应
type ShareSnippetResponse struct {
	ShareToken string `json:"share_token"`
	ShareURL   string `json:"share_url"`
}

// LanguageInfo 支持的语言信息
type LanguageInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	PistonName  string `json:"piston_name"`  // Piston API 中的语言名称
	DefaultCode string `json:"default_code"` // 默认代码模板
}

// PistonExecuteRequest Piston API 执行请求
type PistonExecuteRequest struct {
	Language string `json:"language"`
	Version  string `json:"version"`
	Files    []struct {
		Content string `json:"content"`
	} `json:"files"`
	Stdin       string   `json:"stdin,omitempty"`
	CompileArgs []string `json:"compile_args,omitempty"` // 编译参数（可选）
	RunArgs     []string `json:"run_args,omitempty"`     // 运行参数（可选）
}

// PistonExecuteResponse Piston API 执行响应
type PistonExecuteResponse struct {
	Run struct {
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
		Code   int    `json:"code"`
		Signal string `json:"signal"`
		Output string `json:"output"`
	} `json:"run"`
	Language string `json:"language"`
	Version  string `json:"version"`
}

// CodeSnippetListItem 代码片段列表项（简化版）
type CodeSnippetListItem struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Language  string    `json:"language"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CodeSnippetWithUser 代码片段及用户信息
type CodeSnippetWithUser struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	Username    string    `json:"username"`
	Title       string    `json:"title"`
	Language    string    `json:"language"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	ShareToken  *string   `json:"share_token,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
