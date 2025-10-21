// Package utils provides object pooling utilities to reduce memory allocations.
// Object pools (sync.Pool) reuse objects instead of allocating new ones,
// which significantly reduces GC pressure and improves performance.
//
// Available pools:
//   - BufferPool: For bytes.Buffer reuse
//   - StringBuilderPool: For string building operations
//   - MapPool: For temporary map operations
//   - SlicePool: For temporary slice operations
//   - ResponsePool: For HTTP response objects
//
// Example usage:
//
//	buf := GetBuffer()
//	defer PutBuffer(buf)
//	buf.WriteString("data")
//	result := buf.String()
package utils

import (
	"bytes"
	"sync"
)

// BufferPool 字节缓冲池（用于减少内存分配）
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer 从池中获取Buffer
func GetBuffer() *bytes.Buffer {
	return BufferPool.Get().(*bytes.Buffer)
}

// PutBuffer 归还Buffer到池（会先重置）
func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	BufferPool.Put(buf)
}

// StringBuilderPool 字符串构建器池
var StringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// MapPool 通用map池（用于临时map操作）
type MapPool struct {
	pool sync.Pool
}

// NewMapPool 创建map池
func NewMapPool() *MapPool {
	return &MapPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{}, 16)
			},
		},
	}
}

// Get 获取map
func (p *MapPool) Get() map[string]interface{} {
	return p.pool.Get().(map[string]interface{})
}

// Put 归还map（会先清空）
func (p *MapPool) Put(m map[string]interface{}) {
	// 清空map
	for k := range m {
		delete(m, k)
	}
	p.pool.Put(m)
}

// SlicePool 切片池
type SlicePool struct {
	pool sync.Pool
}

// NewSlicePool 创建切片池
func NewSlicePool(initialCap int) *SlicePool {
	return &SlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]interface{}, 0, initialCap)
			},
		},
	}
}

// Get 获取切片
func (p *SlicePool) Get() []interface{} {
	return p.pool.Get().([]interface{})
}

// Put 归还切片（会先清空）
func (p *SlicePool) Put(s []interface{}) {
	// 清空切片
	s = s[:0]
	p.pool.Put(s)
}

// ResponsePool 响应对象池（用于handler响应）
type ResponseObject struct {
	Code    int
	Message string
	Data    interface{}
}

var ResponsePool = sync.Pool{
	New: func() interface{} {
		return &ResponseObject{}
	},
}

// GetResponse 获取响应对象
func GetResponse() *ResponseObject {
	return ResponsePool.Get().(*ResponseObject)
}

// PutResponse 归还响应对象
func PutResponse(resp *ResponseObject) {
	resp.Code = 0
	resp.Message = ""
	resp.Data = nil
	ResponsePool.Put(resp)
}

// 全局对象池实例
var (
	GlobalMapPool   = NewMapPool()
	GlobalSlicePool = NewSlicePool(32)
)
