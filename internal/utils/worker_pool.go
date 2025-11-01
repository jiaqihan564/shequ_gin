// Package utils 提供Worker Pool实现用于控制并发
package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gin/internal/config"
)

// Task 表示一个异步任务
type Task struct {
	ID      string
	Execute func(context.Context) error
	Timeout time.Duration
}

// WorkerPool Goroutine 池
type WorkerPool struct {
	workers        int
	taskQueue      chan Task
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	logger         Logger
	metrics        *PoolMetrics
	metricsMux     sync.RWMutex
	defaultTimeout time.Duration // 默认任务超时
}

// PoolMetrics 池指标
type PoolMetrics struct {
	TasksSubmitted  uint64
	TasksCompleted  uint64
	TasksFailed     uint64
	TasksTimeout    uint64
	ActiveWorkers   int
	QueuedTasks     int
	TotalExecutions uint64
}

// NewWorkerPool 创建新的 Worker Pool
func NewWorkerPool(workers int, queueSize int, defaultTimeout time.Duration) *WorkerPool {
	if defaultTimeout == 0 {
		defaultTimeout = 30 * time.Second // 回退默认值
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workers:        workers,
		taskQueue:      make(chan Task, queueSize),
		ctx:            ctx,
		cancel:         cancel,
		logger:         GetLogger(),
		defaultTimeout: defaultTimeout,
		metrics: &PoolMetrics{
			ActiveWorkers: 0,
		},
	}

	pool.start()
	return pool
}

// start 启动 worker 池
func (p *WorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("Worker Pool启动成功",
		"workers", p.workers,
		"queueSize", cap(p.taskQueue))
}

// worker 工作协程
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	p.incrementActiveWorkers()
	defer p.decrementActiveWorkers()

	p.logger.Debug("Worker启动", "workerID", id)

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Debug("Worker退出", "workerID", id, "reason", "context cancelled")
			return

		case task, ok := <-p.taskQueue:
			if !ok {
				p.logger.Debug("Worker退出", "workerID", id, "reason", "queue closed")
				return
			}

			p.executeTask(id, task)
		}
	}
}

// executeTask 执行任务
func (p *WorkerPool) executeTask(workerID int, task Task) {
	startTime := time.Now()
	p.metricsMux.Lock()
	p.metrics.TotalExecutions++
	execNum := p.metrics.TotalExecutions
	p.metricsMux.Unlock()

	p.logger.Debug("Worker开始执行任务",
		"workerID", workerID,
		"taskID", task.ID,
		"execNum", execNum)

	// 设置任务超时（使用默认值或任务指定的超时）
	timeout := task.Timeout
	if timeout == 0 {
		timeout = p.defaultTimeout // 使用pool配置的默认超时
	}

	taskCtx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	// 使用 channel 接收任务结果
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.logger.Error("任务执行发生panic",
					"workerID", workerID,
					"taskID", task.ID,
					"panic", r)
				done <- fmt.Errorf("panic: %v", r)
			}
		}()

		done <- task.Execute(taskCtx)
	}()

	// 等待任务完成或超时
	select {
	case err := <-done:
		duration := time.Since(startTime)
		if err != nil {
			p.incrementFailedTasks()
			p.logger.Error("任务执行失败",
				"workerID", workerID,
				"taskID", task.ID,
				"error", err.Error(),
				"duration", duration)
		} else {
			p.incrementCompletedTasks()
			p.logger.Debug("任务执行成功",
				"workerID", workerID,
				"taskID", task.ID,
				"duration", duration)
		}

	case <-taskCtx.Done():
		p.incrementTimeoutTasks()
		p.logger.Warn("任务执行超时",
			"workerID", workerID,
			"taskID", task.ID,
			"timeout", timeout,
			"duration", time.Since(startTime))
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(task Task) error {
	p.incrementSubmittedTasks()

	select {
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool已关闭")
	case p.taskQueue <- task:
		p.logger.Debug("任务已提交", "taskID", task.ID, "queueLength", len(p.taskQueue))
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// SubmitWithTimeout 提交任务（带超时）
func (p *WorkerPool) SubmitWithTimeout(task Task, submitTimeout time.Duration) error {
	p.incrementSubmittedTasks()

	ctx, cancel := context.WithTimeout(context.Background(), submitTimeout)
	defer cancel()

	select {
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool已关闭")
	case <-ctx.Done():
		return fmt.Errorf("提交任务超时")
	case p.taskQueue <- task:
		p.logger.Debug("任务已提交", "taskID", task.ID)
		return nil
	}
}

// Shutdown 优雅关闭池
func (p *WorkerPool) Shutdown(timeout time.Duration) error {
	p.logger.Info("开始关闭Worker Pool", "timeout", timeout)

	// 停止接收新任务
	close(p.taskQueue)

	// 等待所有任务完成或超时
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker Pool已优雅关闭",
			"completedTasks", p.metrics.TasksCompleted,
			"failedTasks", p.metrics.TasksFailed,
			"timeoutTasks", p.metrics.TasksTimeout)
		return nil

	case <-time.After(timeout):
		p.cancel() // 强制取消所有任务
		p.logger.Warn("Worker Pool关闭超时，强制终止",
			"timeout", timeout)
		return fmt.Errorf("关闭超时")
	}
}

// GetMetrics 获取池指标
func (p *WorkerPool) GetMetrics() PoolMetrics {
	p.metricsMux.RLock()
	defer p.metricsMux.RUnlock()

	metrics := *p.metrics
	metrics.QueuedTasks = len(p.taskQueue)
	return metrics
}

// 指标更新方法
func (p *WorkerPool) incrementSubmittedTasks() {
	p.metricsMux.Lock()
	p.metrics.TasksSubmitted++
	p.metricsMux.Unlock()
}

func (p *WorkerPool) incrementCompletedTasks() {
	p.metricsMux.Lock()
	p.metrics.TasksCompleted++
	p.metricsMux.Unlock()
}

func (p *WorkerPool) incrementFailedTasks() {
	p.metricsMux.Lock()
	p.metrics.TasksFailed++
	p.metricsMux.Unlock()
}

func (p *WorkerPool) incrementTimeoutTasks() {
	p.metricsMux.Lock()
	p.metrics.TasksTimeout++
	p.metricsMux.Unlock()
}

func (p *WorkerPool) incrementActiveWorkers() {
	p.metricsMux.Lock()
	p.metrics.ActiveWorkers++
	p.metricsMux.Unlock()
}

func (p *WorkerPool) decrementActiveWorkers() {
	p.metricsMux.Lock()
	p.metrics.ActiveWorkers--
	p.metricsMux.Unlock()
}

// 全局 Worker Pool 实例
var (
	globalPool *WorkerPool
	poolOnce   sync.Once
)

// GetGlobalPool 获取全局 Worker Pool
func GetGlobalPool() *WorkerPool {
	// Note: 在配置加载前可能被调用，使用默认值
	return GetGlobalPoolWithConfig(nil)
}

// GetGlobalPoolWithConfig 使用配置获取全局 Worker Pool
func GetGlobalPoolWithConfig(cfg *config.WorkerPoolConfig) *WorkerPool {
	poolOnce.Do(func() {
		// 使用默认配置
		workers := 10
		queueSize := 1000
		defaultTimeout := 30 * time.Second

		// 如果提供了配置，使用配置值
		if cfg != nil {
			workers = cfg.Workers
			queueSize = cfg.QueueSize
			defaultTimeout = time.Duration(cfg.DefaultTaskTimeout) * time.Second
		}

		globalPool = NewWorkerPool(workers, queueSize, defaultTimeout)
	})
	return globalPool
}

// InitGlobalPool 初始化全局 Worker Pool（带配置）
func InitGlobalPool(cfg *config.Config) {
	GetGlobalPoolWithConfig(&cfg.WorkerPool)
}

// SubmitTask 提交任务到全局池
func SubmitTask(taskID string, fn func(context.Context) error, timeout time.Duration) error {
	task := Task{
		ID:      taskID,
		Execute: fn,
		Timeout: timeout,
	}
	return GetGlobalPool().Submit(task)
}

// SubmitSimpleTask 提交简单任务（无context参数）
// 注意：使用pool配置的默认超时，如需自定义请使用 SubmitTask
func SubmitSimpleTask(taskID string, fn func() error) error {
	task := Task{
		ID: taskID,
		Execute: func(ctx context.Context) error {
			return fn()
		},
		Timeout: 0, // 0表示使用pool的默认超时
	}
	return GetGlobalPool().Submit(task)
}
