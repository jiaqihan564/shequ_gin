package utils

import (
	"fmt"
	"strings"
)

// BitSet 简单的位图实现（用于分片上传进度跟踪）
// 比JSON序列化/反序列化更高效
type BitSet struct {
	bits []byte
	size int
}

// NewBitSet 创建新的位图
func NewBitSet(size int) *BitSet {
	byteSize := (size + 7) / 8 // 向上取整
	return &BitSet{
		bits: make([]byte, byteSize),
		size: size,
	}
}

// Set 设置某一位为1
func (bs *BitSet) Set(index int) error {
	if index < 0 || index >= bs.size {
		return fmt.Errorf("索引超出范围: %d (size: %d)", index, bs.size)
	}

	byteIndex := index / 8
	bitIndex := uint(index % 8)
	bs.bits[byteIndex] |= (1 << bitIndex)
	return nil
}

// Clear 设置某一位为0
func (bs *BitSet) Clear(index int) error {
	if index < 0 || index >= bs.size {
		return fmt.Errorf("索引超出范围: %d", index)
	}

	byteIndex := index / 8
	bitIndex := uint(index % 8)
	bs.bits[byteIndex] &^= (1 << bitIndex)
	return nil
}

// Test 测试某一位是否为1
func (bs *BitSet) Test(index int) bool {
	if index < 0 || index >= bs.size {
		return false
	}

	byteIndex := index / 8
	bitIndex := uint(index % 8)
	return (bs.bits[byteIndex] & (1 << bitIndex)) != 0
}

// Count 统计设置为1的位数
func (bs *BitSet) Count() int {
	count := 0
	for i := 0; i < bs.size; i++ {
		if bs.Test(i) {
			count++
		}
	}
	return count
}

// ToSlice 转换为索引切片（返回所有设置为1的索引）
func (bs *BitSet) ToSlice() []int {
	result := make([]int, 0, bs.Count())
	for i := 0; i < bs.size; i++ {
		if bs.Test(i) {
			result = append(result, i)
		}
	}
	return result
}

// FromSlice 从索引切片构建位图
func (bs *BitSet) FromSlice(indices []int) {
	for _, index := range indices {
		if index >= 0 && index < bs.size {
			bs.Set(index)
		}
	}
}

// String 转换为字符串表示（用于调试）
func (bs *BitSet) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	first := true
	for i := 0; i < bs.size; i++ {
		if bs.Test(i) {
			if !first {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("%d", i))
			first = false
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// ToCompactString 转换为紧凑字符串（用于数据库存储）
// 格式: "0,1,2,5,6,9"
func (bs *BitSet) ToCompactString() string {
	indices := bs.ToSlice()
	if len(indices) == 0 {
		return ""
	}

	strs := make([]string, len(indices))
	for i, idx := range indices {
		strs[i] = fmt.Sprintf("%d", idx)
	}
	return strings.Join(strs, ",")
}

// FromCompactString 从紧凑字符串恢复
func (bs *BitSet) FromCompactString(s string) error {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	for _, part := range parts {
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
			continue
		}
		bs.Set(idx)
	}
	return nil
}

// IsComplete 检查是否所有位都已设置
func (bs *BitSet) IsComplete() bool {
	return bs.Count() == bs.size
}

// Percentage 返回完成百分比
func (bs *BitSet) Percentage() float64 {
	if bs.size == 0 {
		return 100.0
	}
	return float64(bs.Count()) / float64(bs.size) * 100.0
}
