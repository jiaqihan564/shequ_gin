package utils

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	db     *sql.DB
	logger Logger
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *sql.DB) *QueryOptimizer {
	return &QueryOptimizer{
		db:     db,
		logger: GetLogger(),
	}
}

// BatchQuery 批量查询优化（解决N+1问题）
// 示例: 批量查询多个用户的信息
func (o *QueryOptimizer) BatchQuery(
	ctx context.Context,
	baseQuery string,
	ids []uint,
	scanFunc func(*sql.Rows) error,
) error {
	if len(ids) == 0 {
		return nil
	}

	// 构建IN查询
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := strings.ReplaceAll(baseQuery, "?", strings.Join(placeholders, ","))

	start := time.Now()
	rows, err := o.db.QueryContext(ctx, query, args...)
	if err != nil {
		o.logger.Error("批量查询失败", "error", err.Error(), "idsCount", len(ids))
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := scanFunc(rows); err != nil {
			o.logger.Warn("扫描行失败", "error", err.Error())
			continue
		}
	}

	o.logger.Debug("批量查询完成",
		"idsCount", len(ids),
		"duration", time.Since(start))

	return rows.Err()
}

// ExplainQuery 分析查询计划
func (o *QueryOptimizer) ExplainQuery(ctx context.Context, query string, args ...interface{}) (map[string]interface{}, error) {
	explainQuery := "EXPLAIN " + query

	row := o.db.QueryRowContext(ctx, explainQuery, args...)

	var (
		id           int
		selectType   string
		table        string
		partitions   sql.NullString
		typ          string
		possibleKeys sql.NullString
		key          sql.NullString
		keyLen       sql.NullString
		ref          sql.NullString
		rows         sql.NullInt64
		filtered     sql.NullFloat64
		extra        sql.NullString
	)

	err := row.Scan(
		&id, &selectType, &table, &partitions, &typ,
		&possibleKeys, &key, &keyLen, &ref, &rows,
		&filtered, &extra)

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":            id,
		"select_type":   selectType,
		"table":         table,
		"type":          typ,
		"possible_keys": possibleKeys.String,
		"key":           key.String,
		"key_len":       keyLen.String,
		"ref":           ref.String,
		"rows":          rows.Int64,
		"filtered":      filtered.Float64,
		"extra":         extra.String,
	}

	// 检查是否使用了索引
	if !key.Valid || key.String == "" {
		o.logger.Warn("查询未使用索引",
			"table", table,
			"type", typ,
			"rows", rows.Int64)
	}

	return result, nil
}

// OptimizeBulkInsert 优化批量插入
func (o *QueryOptimizer) OptimizeBulkInsert(
	ctx context.Context,
	baseQuery string,
	valueGroups [][]interface{},
	batchSize int,
) error {
	if len(valueGroups) == 0 {
		return nil
	}

	// 设置默认批量大小
	if batchSize <= 0 {
		batchSize = 100
	}

	start := time.Now()
	totalInserted := 0

	// 分批插入
	for i := 0; i < len(valueGroups); i += batchSize {
		end := i + batchSize
		if end > len(valueGroups) {
			end = len(valueGroups)
		}

		batch := valueGroups[i:end]
		if err := o.executeBulkInsert(ctx, baseQuery, batch); err != nil {
			return err
		}

		totalInserted += len(batch)
	}

	o.logger.Info("批量插入完成",
		"total", totalInserted,
		"batches", (len(valueGroups)+batchSize-1)/batchSize,
		"duration", time.Since(start))

	return nil
}

// executeBulkInsert 执行单批插入
func (o *QueryOptimizer) executeBulkInsert(
	ctx context.Context,
	baseQuery string,
	valueGroups [][]interface{},
) error {
	if len(valueGroups) == 0 {
		return nil
	}

	// 构建VALUES部分
	placeholderCount := len(valueGroups[0])
	placeholder := "(" + strings.Repeat("?,", placeholderCount-1) + "?)"

	placeholders := make([]string, len(valueGroups))
	args := make([]interface{}, 0, len(valueGroups)*placeholderCount)

	for i, values := range valueGroups {
		placeholders[i] = placeholder
		args = append(args, values...)
	}

	query := baseQuery + " VALUES " + strings.Join(placeholders, ",")

	_, err := o.db.ExecContext(ctx, query, args...)
	return err
}

// AnalyzeSlowQuery 分析慢查询
func (o *QueryOptimizer) AnalyzeSlowQuery(ctx context.Context, threshold time.Duration) ([]SlowQuery, error) {
	// 查询慢查询日志（需要启用MySQL慢查询日志）
	query := `
		SELECT 
			sql_text,
			start_time,
			query_time,
			lock_time,
			rows_sent,
			rows_examined
		FROM mysql.slow_log
		WHERE query_time > ?
		ORDER BY start_time DESC
		LIMIT 100
	`

	rows, err := o.db.QueryContext(ctx, query, threshold.Seconds())
	if err != nil {
		// 如果慢查询日志未启用，返回空结果
		if strings.Contains(err.Error(), "slow_log") {
			o.logger.Warn("慢查询日志未启用")
			return []SlowQuery{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	slowQueries := []SlowQuery{}
	for rows.Next() {
		var sq SlowQuery
		var queryTime, lockTime float64

		err := rows.Scan(
			&sq.SQLText,
			&sq.StartTime,
			&queryTime,
			&lockTime,
			&sq.RowsSent,
			&sq.RowsExamined,
		)
		if err != nil {
			continue
		}

		sq.QueryTime = time.Duration(queryTime * float64(time.Second))
		sq.LockTime = time.Duration(lockTime * float64(time.Second))
		slowQueries = append(slowQueries, sq)
	}

	return slowQueries, nil
}

// SlowQuery 慢查询信息
type SlowQuery struct {
	SQLText      string
	StartTime    time.Time
	QueryTime    time.Duration
	LockTime     time.Duration
	RowsSent     int64
	RowsExamined int64
}

// String 格式化慢查询信息
func (sq SlowQuery) String() string {
	return fmt.Sprintf(
		"Query: %s\nTime: %v\nRows: sent=%d, examined=%d\nLock: %v\n",
		TruncateString(sq.SQLText, 200),
		sq.QueryTime,
		sq.RowsSent,
		sq.RowsExamined,
		sq.LockTime,
	)
}

// IndexUsageStats 索引使用统计
type IndexUsageStats struct {
	TableName  string
	IndexName  string
	UsageCount int64
	LastUsed   time.Time
}

// GetIndexUsageStats 获取索引使用统计
func (o *QueryOptimizer) GetIndexUsageStats(ctx context.Context) ([]IndexUsageStats, error) {
	query := `
		SELECT 
			object_schema AS table_schema,
			object_name AS table_name,
			index_name,
			count_star AS usage_count
		FROM performance_schema.table_io_waits_summary_by_index_usage
		WHERE object_schema = DATABASE()
		  AND index_name IS NOT NULL
		  AND count_star > 0
		ORDER BY count_star DESC
		LIMIT 100
	`

	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []IndexUsageStats{}
	for rows.Next() {
		var schema, tableName, indexName string
		var usageCount int64

		if err := rows.Scan(&schema, &tableName, &indexName, &usageCount); err != nil {
			continue
		}

		stats = append(stats, IndexUsageStats{
			TableName:  tableName,
			IndexName:  indexName,
			UsageCount: usageCount,
		})
	}

	return stats, nil
}

// SuggestIndexes 建议创建的索引
func (o *QueryOptimizer) SuggestIndexes(ctx context.Context) ([]string, error) {
	// 查询未使用索引的查询
	query := `
		SELECT DISTINCT
			table_name,
			column_name
		FROM information_schema.STATISTICS
		WHERE table_schema = DATABASE()
		  AND index_name = 'PRIMARY'
	`

	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	suggestions := []string{}
	for rows.Next() {
		var tableName, columnName string
		if err := rows.Scan(&tableName, &columnName); err != nil {
			continue
		}

		// 这里可以基于查询模式给出索引建议
		// 简化版本：基于常见查询模式
		suggestions = append(suggestions,
			fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s(%s);",
				tableName, columnName, tableName, columnName))
	}

	return suggestions, nil
}
