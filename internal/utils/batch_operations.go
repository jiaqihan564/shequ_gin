package utils

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// BatchProcessor 批量处理器
// 用于优化批量数据库操作，减少网络往返
type BatchProcessor struct {
	db        *sql.DB
	batchSize int
	logger    Logger
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(db *sql.DB, batchSize int) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &BatchProcessor{
		db:        db,
		batchSize: batchSize,
		logger:    GetLogger(),
	}
}

// BatchInsert 批量插入
func (bp *BatchProcessor) BatchInsert(
	ctx context.Context,
	table string,
	columns []string,
	values [][]interface{},
) error {
	if len(values) == 0 {
		return nil
	}

	start := time.Now()
	totalInserted := 0

	// 分批插入
	for i := 0; i < len(values); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(values) {
			end = len(values)
		}

		batch := values[i:end]
		if err := bp.executeBatchInsert(ctx, table, columns, batch); err != nil {
			bp.logger.Error("批量插入失败",
				"table", table,
				"batch", fmt.Sprintf("%d-%d", i, end),
				"error", err.Error())
			return err
		}

		totalInserted += len(batch)
	}

	bp.logger.Info("批量插入完成",
		"table", table,
		"total", totalInserted,
		"batches", (len(values)+bp.batchSize-1)/bp.batchSize,
		"duration", time.Since(start))

	return nil
}

// executeBatchInsert 执行单批插入
func (bp *BatchProcessor) executeBatchInsert(
	ctx context.Context,
	table string,
	columns []string,
	values [][]interface{},
) error {
	if len(values) == 0 {
		return nil
	}

	// 构建INSERT语句
	columnList := strings.Join(columns, ", ")
	placeholderGroup := "(" + strings.Repeat("?,", len(columns)-1) + "?)"

	placeholders := make([]string, len(values))
	args := make([]interface{}, 0, len(values)*len(columns))

	for i, row := range values {
		placeholders[i] = placeholderGroup
		args = append(args, row...)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		table,
		columnList,
		strings.Join(placeholders, ","),
	)

	_, err := bp.db.ExecContext(ctx, query, args...)
	return err
}

// BatchUpdate 批量更新
func (bp *BatchProcessor) BatchUpdate(
	ctx context.Context,
	table string,
	updates map[string]interface{},
	whereColumn string,
	whereValues []interface{},
) error {
	if len(whereValues) == 0 {
		return nil
	}

	start := time.Now()

	// 构建UPDATE语句
	setClauses := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+len(whereValues))

	for col, val := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	// 添加WHERE IN条件
	placeholders := strings.Repeat("?,", len(whereValues)-1) + "?"
	args = append(args, whereValues...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s IN (%s)",
		table,
		strings.Join(setClauses, ", "),
		whereColumn,
		placeholders,
	)

	result, err := bp.db.ExecContext(ctx, query, args...)
	if err != nil {
		bp.logger.Error("批量更新失败",
			"table", table,
			"error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	bp.logger.Info("批量更新完成",
		"table", table,
		"affected", affected,
		"duration", time.Since(start))

	return nil
}

// BatchDelete 批量删除
func (bp *BatchProcessor) BatchDelete(
	ctx context.Context,
	table string,
	whereColumn string,
	whereValues []interface{},
) error {
	if len(whereValues) == 0 {
		return nil
	}

	start := time.Now()

	placeholders := strings.Repeat("?,", len(whereValues)-1) + "?"
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s IN (%s)",
		table,
		whereColumn,
		placeholders,
	)

	result, err := bp.db.ExecContext(ctx, query, whereValues...)
	if err != nil {
		bp.logger.Error("批量删除失败",
			"table", table,
			"error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	bp.logger.Info("批量删除完成",
		"table", table,
		"affected", affected,
		"duration", time.Since(start))

	return nil
}

// ParallelBatchProcess 并行批量处理
// 使用WorkerPool并行处理多个批次
func (bp *BatchProcessor) ParallelBatchProcess(
	ctx context.Context,
	items []interface{},
	processor func(context.Context, interface{}) error,
	concurrency int,
) error {
	if len(items) == 0 {
		return nil
	}

	if concurrency <= 0 {
		concurrency = 5
	}

	start := time.Now()

	// 创建错误通道
	errChan := make(chan error, len(items))

	// 创建信号量控制并发
	sem := make(chan struct{}, concurrency)

	// 使用WaitGroup等待所有任务完成
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(index int, data interface{}) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 创建带超时的context
			taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// 处理数据
			if err := processor(taskCtx, data); err != nil {
				bp.logger.Error("并行处理失败",
					"index", index,
					"error", err.Error())
				errChan <- err
			}
		}(i, item)
	}

	// 等待所有任务完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	bp.logger.Info("并行批量处理完成",
		"total", len(items),
		"concurrency", concurrency,
		"errors", len(errors),
		"duration", time.Since(start))

	if len(errors) > 0 {
		return fmt.Errorf("处理失败：%d个错误", len(errors))
	}

	return nil
}

// BulkUpsert 批量插入或更新（MySQL UPSERT）
func (bp *BatchProcessor) BulkUpsert(
	ctx context.Context,
	table string,
	columns []string,
	values [][]interface{},
	updateColumns []string,
) error {
	if len(values) == 0 {
		return nil
	}

	start := time.Now()

	// 构建INSERT ... ON DUPLICATE KEY UPDATE语句
	columnList := strings.Join(columns, ", ")
	placeholderGroup := "(" + strings.Repeat("?,", len(columns)-1) + "?)"

	placeholders := make([]string, len(values))
	args := make([]interface{}, 0, len(values)*len(columns))

	for i, row := range values {
		placeholders[i] = placeholderGroup
		args = append(args, row...)
	}

	// 构建UPDATE部分
	updateClauses := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		updateClauses[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s ON DUPLICATE KEY UPDATE %s",
		table,
		columnList,
		strings.Join(placeholders, ","),
		strings.Join(updateClauses, ", "),
	)

	result, err := bp.db.ExecContext(ctx, query, args...)
	if err != nil {
		bp.logger.Error("批量UPSERT失败",
			"table", table,
			"error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	bp.logger.Info("批量UPSERT完成",
		"table", table,
		"affected", affected,
		"duration", time.Since(start))

	return nil
}

// ChunkedQuery 分块查询（处理大结果集）
func (bp *BatchProcessor) ChunkedQuery(
	ctx context.Context,
	baseQuery string,
	orderBy string,
	chunkSize int,
	processor func([]*sql.Rows) error,
) error {
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	offset := 0
	start := time.Now()
	totalRows := 0

	for {
		// 构建分页查询
		query := fmt.Sprintf(
			"%s ORDER BY %s LIMIT %d OFFSET %d",
			baseQuery,
			orderBy,
			chunkSize,
			offset,
		)

		rows, err := bp.db.QueryContext(ctx, query)
		if err != nil {
			return err
		}

		// 检查是否有数据
		hasData := false
		rowCount := 0

		for rows.Next() {
			hasData = true
			rowCount++
			// 这里简化处理，实际应该根据需求处理
		}
		rows.Close()

		if !hasData {
			break
		}

		totalRows += rowCount
		offset += chunkSize

		bp.logger.Debug("处理查询分块",
			"offset", offset,
			"chunkSize", chunkSize,
			"rowCount", rowCount)

		// 如果返回的行数少于chunkSize，说明已经是最后一批
		if rowCount < chunkSize {
			break
		}
	}

	bp.logger.Info("分块查询完成",
		"totalRows", totalRows,
		"chunks", (totalRows+chunkSize-1)/chunkSize,
		"duration", time.Since(start))

	return nil
}

// TransactionBatch 事务批处理
func (bp *BatchProcessor) TransactionBatch(
	ctx context.Context,
	operations []func(context.Context, *sql.Tx) error,
) error {
	start := time.Now()

	// 开启事务
	tx, err := bp.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 执行所有操作
	for i, op := range operations {
		if err := op(ctx, tx); err != nil {
			bp.logger.Error("事务批处理失败",
				"operation", i,
				"error", err.Error())
			return err
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		bp.logger.Error("事务提交失败", "error", err.Error())
		return err
	}

	bp.logger.Info("事务批处理完成",
		"operations", len(operations),
		"duration", time.Since(start))

	return nil
}

// OptimizedBatchGet 优化的批量查询（使用预处理语句）
func (bp *BatchProcessor) OptimizedBatchGet(
	ctx context.Context,
	query string,
	ids []uint,
	scanFunc func(*sql.Row) error,
) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	// 使用预处理语句
	stmt, err := bp.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 并行查询（带并发控制）
	// 注意：并发数可通过配置调整，这里使用默认值10
	maxConcurrency := 10
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(ids))

	for _, id := range ids {
		wg.Add(1)
		go func(itemID uint) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			row := stmt.QueryRowContext(ctx, itemID)
			if err := scanFunc(row); err != nil && err != sql.ErrNoRows {
				errChan <- err
			}
		}(id)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	bp.logger.Info("批量查询完成",
		"count", len(ids),
		"errors", len(errors),
		"duration", time.Since(start))

	if len(errors) > 0 {
		return fmt.Errorf("批量查询失败：%d个错误", len(errors))
	}

	return nil
}

// CachedBatchGet 带缓存的批量查询
func (bp *BatchProcessor) CachedBatchGet(
	ctx context.Context,
	cache *MemoryCache,
	cacheKeyPrefix string,
	query string,
	ids []uint,
	cacheTTL time.Duration,
	scanFunc func(*sql.Row, uint) (interface{}, error),
) (map[uint]interface{}, error) {
	result := make(map[uint]interface{})
	missingIDs := []uint{}

	// 1. 从缓存获取
	for _, id := range ids {
		cacheKey := fmt.Sprintf("%s%d", cacheKeyPrefix, id)
		if cached, ok := cache.Get(cacheKey); ok {
			result[id] = cached
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	bp.logger.Debug("缓存查询结果",
		"total", len(ids),
		"cached", len(result),
		"missing", len(missingIDs))

	if len(missingIDs) == 0 {
		return result, nil
	}

	// 2. 查询缺失的数据
	start := time.Now()
	stmt, err := bp.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(missingIDs))

	for _, id := range missingIDs {
		wg.Add(1)
		go func(itemID uint) {
			defer wg.Done()

			row := stmt.QueryRowContext(ctx, itemID)
			data, err := scanFunc(row, itemID)
			if err != nil {
				if err != sql.ErrNoRows {
					errChan <- err
				}
				return
			}

			// 写入结果
			mu.Lock()
			result[itemID] = data
			mu.Unlock()

			// 写入缓存
			cacheKey := fmt.Sprintf("%s%d", cacheKeyPrefix, itemID)
			cache.SetWithTTL(cacheKey, data, cacheTTL)
		}(id)
	}

	wg.Wait()
	close(errChan)

	// 检查错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	bp.logger.Info("批量查询完成（带缓存）",
		"queried", len(missingIDs),
		"errors", len(errors),
		"duration", time.Since(start))

	if len(errors) > 0 {
		return result, fmt.Errorf("部分查询失败：%d个错误", len(errors))
	}

	return result, nil
}

// BulkUpdateCounter 批量更新计数器（原子操作）
func (bp *BatchProcessor) BulkUpdateCounter(
	ctx context.Context,
	table string,
	counterColumn string,
	increment int,
	whereColumn string,
	whereValues []uint,
) error {
	if len(whereValues) == 0 {
		return nil
	}

	start := time.Now()

	placeholders := strings.Repeat("?,", len(whereValues)-1) + "?"
	args := make([]interface{}, 0, len(whereValues)+1)
	args = append(args, increment)
	for _, v := range whereValues {
		args = append(args, v)
	}

	var query string
	if increment >= 0 {
		query = fmt.Sprintf(
			"UPDATE %s SET %s = %s + ? WHERE %s IN (%s)",
			table, counterColumn, counterColumn, whereColumn, placeholders,
		)
	} else {
		query = fmt.Sprintf(
			"UPDATE %s SET %s = GREATEST(%s + ?, 0) WHERE %s IN (%s)",
			table, counterColumn, counterColumn, whereColumn, placeholders,
		)
	}

	result, err := bp.db.ExecContext(ctx, query, args...)
	if err != nil {
		bp.logger.Error("批量更新计数器失败",
			"table", table,
			"error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	bp.logger.Info("批量更新计数器完成",
		"table", table,
		"counter", counterColumn,
		"increment", increment,
		"affected", affected,
		"duration", time.Since(start))

	return nil
}
