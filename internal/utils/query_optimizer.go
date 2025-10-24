package utils

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// QueryOptimizer 查询优化器
// 提供常用的查询优化工具
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

// EstimateCount 估算总数（大表优化）
// 对于大表，使用估算值代替精确COUNT，提升性能
func (qo *QueryOptimizer) EstimateCount(ctx context.Context, table string, whereClause string, args []interface{}) (int, bool, error) {
	// 先尝试使用EXPLAIN获取估算行数
	explainQuery := fmt.Sprintf("EXPLAIN SELECT COUNT(*) FROM %s %s", table, whereClause)

	var (
		id           sql.NullInt64
		selectType   sql.NullString
		tableName    sql.NullString
		partitions   sql.NullString
		typeStr      sql.NullString
		possibleKeys sql.NullString
		key          sql.NullString
		keyLen       sql.NullString
		ref          sql.NullString
		rows         sql.NullInt64
		filtered     sql.NullFloat64
		extra        sql.NullString
	)

	err := qo.db.QueryRowContext(ctx, explainQuery, args...).Scan(
		&id, &selectType, &tableName, &partitions, &typeStr,
		&possibleKeys, &key, &keyLen, &ref, &rows, &filtered, &extra,
	)

	if err == nil && rows.Valid && rows.Int64 < 10000 {
		// 小表（<10000行），直接返回估算值
		return int(rows.Int64), true, nil
	}

	// 大表或估算失败，返回需要精确查询的信号
	return 0, false, nil
}

// CachedCount 带缓存的COUNT查询
// 对于变化不频繁的数据，使用缓存减少COUNT查询
func (qo *QueryOptimizer) CachedCount(ctx context.Context, cacheKey string, countQuery string, args []interface{}, cacheTTL time.Duration) (int, error) {
	cache := GetCache()

	// 尝试从缓存获取
	if cached, ok := cache.Get(cacheKey); ok {
		if count, ok := cached.(int); ok {
			qo.logger.Debug("COUNT查询缓存命中", "cacheKey", cacheKey, "count", count)
			return count, nil
		}
	}

	// 缓存未命中，执行查询
	var count int
	err := qo.db.QueryRowContext(ctx, countQuery, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	// 写入缓存
	cache.SetWithTTL(cacheKey, count, cacheTTL)
	qo.logger.Debug("COUNT查询结果已缓存", "cacheKey", cacheKey, "count", count, "ttl", cacheTTL)

	return count, nil
}

// OptimizedPagination 优化的分页查询
// 使用游标分页代替OFFSET（更适合深分页）
func (qo *QueryOptimizer) OptimizedPagination(
	ctx context.Context,
	table string,
	columns []string,
	whereClause string,
	orderBy string,
	limit int,
	lastID uint,
	args []interface{},
) (*sql.Rows, error) {
	// 构建查询
	selectColumns := strings.Join(columns, ", ")

	// 如果有lastID，使用游标分页
	if lastID > 0 {
		if whereClause != "" {
			whereClause += " AND id > ?"
		} else {
			whereClause = "WHERE id > ?"
		}
		args = append(args, lastID)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s %s ORDER BY %s LIMIT ?",
		selectColumns, table, whereClause, orderBy,
	)
	args = append(args, limit)

	return qo.db.QueryContext(ctx, query, args...)
}

// BatchQuery 批量查询优化（避免大结果集一次性加载）
// 使用流式处理，边查询边处理
type RowProcessor func(*sql.Rows) error

func (qo *QueryOptimizer) BatchQuery(
	ctx context.Context,
	query string,
	args []interface{},
	batchSize int,
	processor RowProcessor,
) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	offset := 0
	start := time.Now()
	totalProcessed := 0

	for {
		// 分批查询
		batchQuery := fmt.Sprintf("%s LIMIT %d OFFSET %d", query, batchSize, offset)
		rows, err := qo.db.QueryContext(ctx, batchQuery, args...)
		if err != nil {
			return err
		}

		hasData := false
		for rows.Next() {
			hasData = true
			if err := processor(rows); err != nil {
				rows.Close()
				return err
			}
			totalProcessed++
		}
		rows.Close()

		if !hasData {
			break
		}

		offset += batchSize
	}

	qo.logger.Info("批量查询处理完成",
		"totalProcessed", totalProcessed,
		"batchSize", batchSize,
		"duration", time.Since(start))

	return nil
}

// OptimizeLIKEQuery 优化LIKE查询
// 返回优化建议和改进的查询
func (qo *QueryOptimizer) OptimizeLIKEQuery(keyword string) (optimized string, useFullText bool) {
	// 去除首尾的%
	trimmed := strings.Trim(keyword, "%")

	// 如果是前缀匹配（keyword%），可以使用索引
	if strings.HasSuffix(keyword, "%") && !strings.HasPrefix(keyword, "%") {
		return trimmed + "%", false
	}

	// 如果是双边模糊匹配（%keyword%），建议使用全文索引
	if strings.HasPrefix(keyword, "%") && strings.HasSuffix(keyword, "%") {
		return trimmed, true
	}

	return keyword, false
}

// GetQueryPlan 获取查询执行计划（用于调试）
func (qo *QueryOptimizer) GetQueryPlan(ctx context.Context, query string, args []interface{}) (string, error) {
	explainQuery := "EXPLAIN " + query
	rows, err := qo.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var plan strings.Builder
	plan.WriteString("查询执行计划:\n")

	// 获取列名
	cols, _ := rows.Columns()
	plan.WriteString(fmt.Sprintf("Columns: %v\n", cols))

	// 读取行
	rowNum := 0
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowNum++
		plan.WriteString(fmt.Sprintf("Row %d: ", rowNum))
		for i, col := range cols {
			plan.WriteString(fmt.Sprintf("%s=%v ", col, values[i]))
		}
		plan.WriteString("\n")
	}

	return plan.String(), nil
}

// UseIndexHint 添加索引提示（强制使用某个索引）
func (qo *QueryOptimizer) UseIndexHint(table, indexName string) string {
	return fmt.Sprintf("%s USE INDEX (%s)", table, indexName)
}

// ForceIndexHint 强制使用索引
func (qo *QueryOptimizer) ForceIndexHint(table, indexName string) string {
	return fmt.Sprintf("%s FORCE INDEX (%s)", table, indexName)
}

// IgnoreIndexHint 忽略某个索引
func (qo *QueryOptimizer) IgnoreIndexHint(table, indexName string) string {
	return fmt.Sprintf("%s IGNORE INDEX (%s)", table, indexName)
}

// ParallelCountAndQuery 并行执行COUNT和列表查询（优化分页性能）
func (qo *QueryOptimizer) ParallelCountAndQuery(
	ctx context.Context,
	countQuery string,
	listQuery string,
	countArgs []interface{},
	listArgs []interface{},
) (int, *sql.Rows, error) {
	// 使用channel并行执行
	type countResult struct {
		count int
		err   error
	}
	type queryResult struct {
		rows *sql.Rows
		err  error
	}

	countChan := make(chan countResult, 1)
	queryChan := make(chan queryResult, 1)

	// 并行执行COUNT查询
	go func() {
		var count int
		err := qo.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&count)
		countChan <- countResult{count: count, err: err}
	}()

	// 并行执行列表查询
	go func() {
		rows, err := qo.db.QueryContext(ctx, listQuery, listArgs...)
		queryChan <- queryResult{rows: rows, err: err}
	}()

	// 收集结果
	countRes := <-countChan
	queryRes := <-queryChan

	if countRes.err != nil {
		if queryRes.rows != nil {
			queryRes.rows.Close()
		}
		return 0, nil, countRes.err
	}

	if queryRes.err != nil {
		return 0, nil, queryRes.err
	}

	return countRes.count, queryRes.rows, nil
}

// OptimizeIN 优化IN查询（大量ID时拆分为多个查询）
func (qo *QueryOptimizer) OptimizeIN(ids []uint, maxBatchSize int) [][]uint {
	if maxBatchSize <= 0 {
		maxBatchSize = 1000 // MySQL默认推荐
	}

	if len(ids) <= maxBatchSize {
		return [][]uint{ids}
	}

	// 拆分为多个批次
	batches := make([][]uint, 0)
	for i := 0; i < len(ids); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batches = append(batches, ids[i:end])
	}

	qo.logger.Info("IN查询已拆分为多个批次",
		"totalIDs", len(ids),
		"batchSize", maxBatchSize,
		"batches", len(batches))

	return batches
}

// GetSlowQueryLog 获取慢查询日志（需要开启慢查询日志）
func (qo *QueryOptimizer) GetSlowQueryLog(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT sql_text, start_time, query_time, lock_time, rows_sent, rows_examined
		FROM mysql.slow_log
		ORDER BY query_time DESC
		LIMIT ?
	`

	rows, err := qo.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			sqlText      string
			startTime    time.Time
			queryTime    time.Duration
			lockTime     time.Duration
			rowsSent     int
			rowsExamined int
		)

		if err := rows.Scan(&sqlText, &startTime, &queryTime, &lockTime, &rowsSent, &rowsExamined); err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"sql_text":      sqlText,
			"start_time":    startTime,
			"query_time":    queryTime,
			"lock_time":     lockTime,
			"rows_sent":     rowsSent,
			"rows_examined": rowsExamined,
		})
	}

	return results, nil
}
