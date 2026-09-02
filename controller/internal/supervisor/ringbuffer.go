package supervisor

import (
	"sync"
)

// RingBuffer 线程安全的固定容量环形日志缓冲区
type RingBuffer struct {
	mu       sync.RWMutex
	capacity int
	entries  []LogLine
	head     int
	size     int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 2000
	}
	return &RingBuffer{
		capacity: capacity,
		entries:  make([]LogLine, capacity),
	}
}

func (rb *RingBuffer) Push(line LogLine) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.entries[rb.head] = line
	rb.head = (rb.head + 1) % rb.capacity
	if rb.size < rb.capacity {
		rb.size++
	}
}

// Slice 返回当前缓冲区中最新 limit 条日志（按时间升序）
func (rb *RingBuffer) Slice(limit int) []LogLine {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if limit <= 0 || limit > rb.size {
		limit = rb.size
	}

	result := make([]LogLine, limit)
	start := (rb.head - limit + rb.capacity) % rb.capacity

	for i := 0; i < limit; i++ {
		idx := (start + i) % rb.capacity
		result[i] = rb.entries[idx]
	}
	return result
}

func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.head = 0
	rb.size = 0
}

func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}
