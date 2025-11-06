-- 在线代码运行平台相关表
-- 创建时间: 2025-10-21

-- 1. 代码片段表
CREATE TABLE IF NOT EXISTS code_snippets (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNSIGNED NOT NULL COMMENT '创建者ID',
    title VARCHAR(255) NOT NULL DEFAULT 'Untitled' COMMENT '代码标题',
    language VARCHAR(50) NOT NULL COMMENT '编程语言: python, javascript, java, cpp等',
    code TEXT NOT NULL COMMENT '代码内容',
    description TEXT COMMENT '代码描述',
    is_public TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否公开: 0-私有, 1-公开',
    share_token VARCHAR(64) UNIQUE COMMENT '分享令牌（唯一）',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_user_id (user_id),
    INDEX idx_language (language),
    INDEX idx_share_token (share_token),
    INDEX idx_created_at (created_at),
    INDEX idx_is_public (is_public)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码片段表';

-- 2. 代码执行记录表
CREATE TABLE IF NOT EXISTS code_executions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    snippet_id BIGINT UNSIGNED COMMENT '代码片段ID（可空，临时执行无snippet_id）',
    user_id INT UNSIGNED NOT NULL COMMENT '执行者ID',
    language VARCHAR(50) NOT NULL COMMENT '编程语言',
    code TEXT NOT NULL COMMENT '执行的代码',
    stdin TEXT COMMENT '标准输入',
    output TEXT COMMENT '执行输出',
    error TEXT COMMENT '错误信息',
    execution_time INT COMMENT '执行耗时（毫秒）',
    memory_usage BIGINT COMMENT '内存使用（字节）',
    status VARCHAR(20) NOT NULL COMMENT '执行状态: success, error, timeout',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_user_id (user_id),
    INDEX idx_snippet_id (snippet_id),
    INDEX idx_language (language),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码执行记录表';

-- 3. 协作会话表
CREATE TABLE IF NOT EXISTS code_collaborations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    snippet_id BIGINT UNSIGNED NOT NULL COMMENT '代码片段ID',
    session_token VARCHAR(64) NOT NULL UNIQUE COMMENT '会话令牌',
    active_users JSON COMMENT '当前在线用户列表',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    expires_at TIMESTAMP NOT NULL COMMENT '过期时间',
    INDEX idx_snippet_id (snippet_id),
    INDEX idx_session_token (session_token),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代码协作会话表';

-- 插入初始测试数据（可选）
-- INSERT INTO code_snippets (user_id, title, language, code, description, is_public)
-- VALUES (1, 'Hello World Python', 'python', 'print("Hello, World!")', '经典的 Hello World 示例', 1);

