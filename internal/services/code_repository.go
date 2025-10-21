package services

import (
	"context"
	"database/sql"
	"fmt"
	"gin/internal/models"
	"gin/internal/utils"
	"time"

	"github.com/google/uuid"
)

// CodeRepository 代码仓库接口
type CodeRepository interface {
	// 代码片段相关
	CreateSnippet(snippet *models.CodeSnippet) error
	GetSnippetByID(id uint) (*models.CodeSnippet, error)
	GetSnippetsByUserID(userID uint, limit, offset int) ([]models.CodeSnippetListItem, int, error)
	GetPublicSnippets(language string, limit, offset int) ([]models.CodeSnippetWithUser, int, error)
	UpdateSnippet(snippet *models.CodeSnippet) error
	DeleteSnippet(id uint, userID uint) error
	GetSnippetByShareToken(token string) (*models.CodeSnippet, error)
	GenerateShareToken(snippetID uint, userID uint) (string, error)

	// 执行记录相关
	CreateExecution(execution *models.CodeExecution) error
	GetExecutionsByUserID(userID uint, limit, offset int) ([]models.CodeExecution, int, error)
	GetExecutionsBySnippetID(snippetID uint, limit, offset int) ([]models.CodeExecution, error)

	// 协作会话相关
	CreateCollaboration(collab *models.CodeCollaboration) error
	GetCollaborationByToken(token string) (*models.CodeCollaboration, error)
	UpdateCollaborationUsers(token string, activeUsers string) error
}

// CodeRepositoryImpl 代码仓库实现
type CodeRepositoryImpl struct {
	db *Database
}

// NewCodeRepository 创建新的代码仓库
func NewCodeRepository(db *Database) CodeRepository {
	return &CodeRepositoryImpl{db: db}
}

// CreateSnippet 创建代码片段
func (r *CodeRepositoryImpl) CreateSnippet(snippet *models.CodeSnippet) error {
	query := `
		INSERT INTO code_snippets (user_id, title, language, code, description, is_public)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query,
		snippet.UserID,
		snippet.Title,
		snippet.Language,
		snippet.Code,
		snippet.Description,
		snippet.IsPublic,
	)
	if err != nil {
		return fmt.Errorf("创建代码片段失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	snippet.ID = uint(id)

	utils.GetLogger().Info("创建代码片段成功",
		"snippet_id", snippet.ID,
		"user_id", snippet.UserID,
		"language", snippet.Language)

	return nil
}

// GetSnippetByID 根据ID获取代码片段
func (r *CodeRepositoryImpl) GetSnippetByID(id uint) (*models.CodeSnippet, error) {
	var snippet models.CodeSnippet
	query := `SELECT id, user_id, title, language, code, description, is_public, share_token, created_at, updated_at 
			  FROM code_snippets WHERE id = ?`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := r.db.QueryRowWithCache(ctx, query, id)
	err := row.Scan(&snippet.ID, &snippet.UserID, &snippet.Title, &snippet.Language,
		&snippet.Code, &snippet.Description, &snippet.IsPublic, &snippet.ShareToken,
		&snippet.CreatedAt, &snippet.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("代码片段不存在")
		}
		return nil, fmt.Errorf("查询代码片段失败: %w", err)
	}
	return &snippet, nil
}

// GetSnippetsByUserID 获取用户的代码片段列表
func (r *CodeRepositoryImpl) GetSnippetsByUserID(userID uint, limit, offset int) ([]models.CodeSnippetListItem, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 查询总数
	var total int
	countQuery := `SELECT COUNT(*) FROM code_snippets WHERE user_id = ?`
	row := r.db.QueryRowWithCache(ctx, countQuery, userID)
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询代码片段总数失败: %w", err)
	}

	// 查询列表
	query := `
		SELECT id, title, language, is_public, created_at, updated_at
		FROM code_snippets
		WHERE user_id = ?
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryWithCache(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询代码片段列表失败: %w", err)
	}
	defer rows.Close()

	var snippets []models.CodeSnippetListItem
	for rows.Next() {
		var snippet models.CodeSnippetListItem
		if err := rows.Scan(&snippet.ID, &snippet.Title, &snippet.Language, &snippet.IsPublic,
			&snippet.CreatedAt, &snippet.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描代码片段失败: %w", err)
		}
		snippets = append(snippets, snippet)
	}

	return snippets, total, nil
}

// UpdateSnippet 更新代码片段
func (r *CodeRepositoryImpl) UpdateSnippet(snippet *models.CodeSnippet) error {
	query := `
		UPDATE code_snippets
		SET title = ?, code = ?, description = ?, is_public = ?
		WHERE id = ? AND user_id = ?
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query,
		snippet.Title,
		snippet.Code,
		snippet.Description,
		snippet.IsPublic,
		snippet.ID,
		snippet.UserID,
	)
	if err != nil {
		return fmt.Errorf("更新代码片段失败: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("代码片段不存在或无权限")
	}

	utils.GetLogger().Info("更新代码片段成功",
		"snippet_id", snippet.ID,
		"user_id", snippet.UserID)

	return nil
}

// DeleteSnippet 删除代码片段
func (r *CodeRepositoryImpl) DeleteSnippet(id uint, userID uint) error {
	query := `DELETE FROM code_snippets WHERE id = ? AND user_id = ?`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("删除代码片段失败: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("代码片段不存在或无权限")
	}

	utils.GetLogger().Info("删除代码片段成功",
		"snippet_id", id,
		"user_id", userID)

	return nil
}

// GetSnippetByShareToken 通过分享令牌获取代码片段
func (r *CodeRepositoryImpl) GetSnippetByShareToken(token string) (*models.CodeSnippet, error) {
	var snippet models.CodeSnippet
	query := `SELECT id, user_id, title, language, code, description, is_public, share_token, created_at, updated_at 
			  FROM code_snippets WHERE share_token = ?`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := r.db.QueryRowWithCache(ctx, query, token)
	err := row.Scan(&snippet.ID, &snippet.UserID, &snippet.Title, &snippet.Language,
		&snippet.Code, &snippet.Description, &snippet.IsPublic, &snippet.ShareToken,
		&snippet.CreatedAt, &snippet.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("分享链接无效或已过期")
		}
		return nil, fmt.Errorf("查询代码片段失败: %w", err)
	}
	return &snippet, nil
}

// GenerateShareToken 生成分享令牌
func (r *CodeRepositoryImpl) GenerateShareToken(snippetID uint, userID uint) (string, error) {
	// 生成 UUID 作为分享令牌
	token := uuid.New().String()

	query := `
		UPDATE code_snippets
		SET share_token = ?
		WHERE id = ? AND user_id = ?
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query, token, snippetID, userID)
	if err != nil {
		return "", fmt.Errorf("生成分享令牌失败: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if rows == 0 {
		return "", fmt.Errorf("代码片段不存在或无权限")
	}

	utils.GetLogger().Info("生成分享令牌成功",
		"snippet_id", snippetID,
		"user_id", userID,
		"token", token)

	return token, nil
}

// CreateExecution 创建执行记录
func (r *CodeRepositoryImpl) CreateExecution(execution *models.CodeExecution) error {
	query := `
		INSERT INTO code_executions (snippet_id, user_id, language, code, stdin, output, error, execution_time, memory_usage, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query,
		execution.SnippetID,
		execution.UserID,
		execution.Language,
		execution.Code,
		execution.Stdin,
		execution.Output,
		execution.Error,
		execution.ExecutionTime,
		execution.MemoryUsage,
		execution.Status,
	)
	if err != nil {
		return fmt.Errorf("创建执行记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	execution.ID = uint(id)

	return nil
}

// GetExecutionsByUserID 获取用户的执行记录列表
func (r *CodeRepositoryImpl) GetExecutionsByUserID(userID uint, limit, offset int) ([]models.CodeExecution, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 查询总数
	var total int
	countQuery := `SELECT COUNT(*) FROM code_executions WHERE user_id = ?`
	row := r.db.QueryRowWithCache(ctx, countQuery, userID)
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询执行记录总数失败: %w", err)
	}

	// 查询列表
	query := `
		SELECT id, snippet_id, user_id, language, code, stdin, output, error, execution_time, memory_usage, status, created_at
		FROM code_executions
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryWithCache(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询执行记录列表失败: %w", err)
	}
	defer rows.Close()

	var executions []models.CodeExecution
	for rows.Next() {
		var execution models.CodeExecution
		if err := rows.Scan(&execution.ID, &execution.SnippetID, &execution.UserID, &execution.Language,
			&execution.Code, &execution.Stdin, &execution.Output, &execution.Error,
			&execution.ExecutionTime, &execution.MemoryUsage, &execution.Status, &execution.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描执行记录失败: %w", err)
		}
		executions = append(executions, execution)
	}

	return executions, total, nil
}

// GetExecutionsBySnippetID 获取代码片段的执行记录
func (r *CodeRepositoryImpl) GetExecutionsBySnippetID(snippetID uint, limit, offset int) ([]models.CodeExecution, error) {
	query := `
		SELECT id, snippet_id, user_id, language, code, stdin, output, error, execution_time, memory_usage, status, created_at
		FROM code_executions
		WHERE snippet_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.QueryWithCache(ctx, query, snippetID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}
	defer rows.Close()

	var executions []models.CodeExecution
	for rows.Next() {
		var execution models.CodeExecution
		if err := rows.Scan(&execution.ID, &execution.SnippetID, &execution.UserID, &execution.Language,
			&execution.Code, &execution.Stdin, &execution.Output, &execution.Error,
			&execution.ExecutionTime, &execution.MemoryUsage, &execution.Status, &execution.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描执行记录失败: %w", err)
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// CreateCollaboration 创建协作会话
func (r *CodeRepositoryImpl) CreateCollaboration(collab *models.CodeCollaboration) error {
	query := `
		INSERT INTO code_collaborations (snippet_id, session_token, active_users, expires_at)
		VALUES (?, ?, ?, ?)
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query,
		collab.SnippetID,
		collab.SessionToken,
		collab.ActiveUsers,
		collab.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("创建协作会话失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	collab.ID = uint(id)

	return nil
}

// GetCollaborationByToken 通过令牌获取协作会话
func (r *CodeRepositoryImpl) GetCollaborationByToken(token string) (*models.CodeCollaboration, error) {
	var collab models.CodeCollaboration
	query := `SELECT id, snippet_id, session_token, active_users, created_at, expires_at 
			  FROM code_collaborations WHERE session_token = ?`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := r.db.QueryRowWithCache(ctx, query, token)
	err := row.Scan(&collab.ID, &collab.SnippetID, &collab.SessionToken,
		&collab.ActiveUsers, &collab.CreatedAt, &collab.ExpiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("协作会话不存在")
		}
		return nil, fmt.Errorf("查询协作会话失败: %w", err)
	}
	return &collab, nil
}

// UpdateCollaborationUsers 更新协作会话的在线用户列表
func (r *CodeRepositoryImpl) UpdateCollaborationUsers(token string, activeUsers string) error {
	query := `UPDATE code_collaborations SET active_users = ? WHERE session_token = ?`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.db.ExecWithCache(ctx, query, activeUsers, token)
	if err != nil {
		return fmt.Errorf("更新协作会话用户列表失败: %w", err)
	}
	return nil
}

// GetPublicSnippets 获取公开的代码片段列表
func (r *CodeRepositoryImpl) GetPublicSnippets(language string, limit, offset int) ([]models.CodeSnippetWithUser, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建查询条件
	var countQuery, listQuery string
	var args []interface{}

	if language != "" {
		countQuery = `SELECT COUNT(*) FROM code_snippets WHERE is_public = 1 AND language = ?`
		listQuery = `
			SELECT cs.id, cs.user_id, u.username, cs.title, cs.language, cs.code, cs.description, cs.share_token, cs.created_at, cs.updated_at
			FROM code_snippets cs
			LEFT JOIN user_auth u ON cs.user_id = u.id
			WHERE cs.is_public = 1 AND cs.language = ?
			ORDER BY cs.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{language, limit, offset}
	} else {
		countQuery = `SELECT COUNT(*) FROM code_snippets WHERE is_public = 1`
		listQuery = `
			SELECT cs.id, cs.user_id, u.username, cs.title, cs.language, cs.code, cs.description, cs.share_token, cs.created_at, cs.updated_at
			FROM code_snippets cs
			LEFT JOIN user_auth u ON cs.user_id = u.id
			WHERE cs.is_public = 1
			ORDER BY cs.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	// 查询总数
	var total int
	var countArgs []interface{}
	if language != "" {
		countArgs = []interface{}{language}
	}
	row := r.db.QueryRowWithCache(ctx, countQuery, countArgs...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询公开代码片段总数失败: %w", err)
	}

	// 查询列表
	rows, err := r.db.QueryWithCache(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询公开代码片段列表失败: %w", err)
	}
	defer rows.Close()

	var snippets []models.CodeSnippetWithUser
	for rows.Next() {
		var snippet models.CodeSnippetWithUser
		var username sql.NullString
		if err := rows.Scan(&snippet.ID, &snippet.UserID, &username, &snippet.Title, &snippet.Language,
			&snippet.Code, &snippet.Description, &snippet.ShareToken, &snippet.CreatedAt, &snippet.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描公开代码片段失败: %w", err)
		}
		if username.Valid {
			snippet.Username = username.String
		} else {
			snippet.Username = "未知用户"
		}
		snippets = append(snippets, snippet)
	}

	utils.GetLogger().Info("查询公开代码片段成功",
		"total", total,
		"language", language,
		"limit", limit,
		"offset", offset)

	return snippets, total, nil
}
