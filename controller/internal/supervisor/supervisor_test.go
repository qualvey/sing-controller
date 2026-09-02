package supervisor

import (
	"testing"
	"time"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(5)

	if rb.Len() != 0 {
		t.Fatalf("expected len 0, got %d", rb.Len())
	}

	for i := 1; i <= 3; i++ {
		rb.Push(LogLine{
			Timestamp: time.Now(),
			Source:    "stdout",
			Message:   string(rune('A' + i - 1)),
		})
	}

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}

	slice := rb.Slice(10)
	if len(slice) != 3 {
		t.Fatalf("expected slice len 3, got %d", len(slice))
	}
	if slice[0].Message != "A" || slice[2].Message != "C" {
		t.Fatalf("unexpected slice content: %+v", slice)
	}

	// 压入更多，测试溢出环形覆盖
	for i := 4; i <= 7; i++ {
		rb.Push(LogLine{
			Timestamp: time.Now(),
			Source:    "stdout",
			Message:   string(rune('A' + i - 1)),
		})
	}

	if rb.Len() != 5 {
		t.Fatalf("expected len 5 after overflow, got %d", rb.Len())
	}

	fullSlice := rb.Slice(5)
	if fullSlice[0].Message != "C" || fullSlice[4].Message != "G" {
		t.Fatalf("expected range C..G, got %+v", fullSlice)
	}

	partial := rb.Slice(2)
	if len(partial) != 2 || partial[0].Message != "F" || partial[1].Message != "G" {
		t.Fatalf("expected latest 2 [F, G], got %+v", partial)
	}
}

func TestManagerCreation(t *testing.T) {
	mgr, err := New(Config{
		BinaryPath:  "sing-box",
		ConfigPath:  "config.json",
		LogCapacity: 500,
	})
	if err != nil {
		t.Fatalf("unexpected error creating supervisor: %v", err)
	}
	defer mgr.Close()

	status := mgr.Status()
	if status.State != StateStopped {
		t.Fatalf("expected state stopped, got %s", status.State)
	}
	if status.BinaryPath != "sing-box" {
		t.Fatalf("expected binary path 'sing-box', got %s", status.BinaryPath)
	}
}
