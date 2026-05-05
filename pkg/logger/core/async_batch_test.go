package core

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

type mockWriter struct {
	mu    sync.Mutex
	data  []byte
	count int
}

func (m *mockWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append(m.data, p...)
	m.count++
	return len(p), nil
}

func (m *mockWriter) GetData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data
}

func (m *mockWriter) GetCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.count
}

func TestAsyncBatch_Write(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 3, time.Second)
	defer ab.Close()

	if !ab.Write([]byte("test1")) {
		t.Error("Write should succeed")
	}
	if ab.BufferLen() != 1 {
		t.Errorf("BufferLen = %d, want 1", ab.BufferLen())
	}
}

func TestAsyncBatch_BatchSizeFlush(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 3, time.Hour)
	defer ab.Close()

	for i := 0; i < 3; i++ {
		ab.Write([]byte("test"))
	}
	time.Sleep(50 * time.Millisecond)

	if mock.GetCount() != 1 {
		t.Errorf("Write count = %d, want 1 (batch should have flushed)", mock.GetCount())
	}
}

func TestAsyncBatch_IntervalFlush(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 100, 50*time.Millisecond)
	defer ab.Close()

	ab.Write([]byte("test"))
	time.Sleep(100 * time.Millisecond)

	if mock.GetCount() != 1 {
		t.Errorf("Write count = %d, want 1 (timer should have flushed)", mock.GetCount())
	}
}

func TestAsyncBatch_Close(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 100, time.Hour)

	ab.Write([]byte("test1"))
	ab.Write([]byte("test2"))

	ab.Close()

	if ab.closed.Load() != true {
		t.Error("closed should be true after Close")
	}
	if mock.GetCount() != 1 {
		t.Errorf("Write count = %d, want 1 (buffer should be flushed on close)", mock.GetCount())
	}
}

func TestAsyncBatch_WriteAfterClose(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 10, time.Hour)
	ab.Close()

	if ab.Write([]byte("test")) {
		t.Error("Write should return false after close")
	}
}

func TestAsyncBatch_ConcurrentWrite(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 10, time.Hour)
	defer ab.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ab.Write([]byte("test"))
		}()
	}
	wg.Wait()

	time.Sleep(50 * time.Millisecond)
	count := mock.GetCount()
	if count == 0 {
		t.Error("Mock writer should have received data after batch flush")
	}
}

func TestAsyncBatch_GracefulShutdown(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 5, time.Hour)

	for i := 0; i < 4; i++ {
		ab.Write([]byte("test"))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		ab.Close()
	}()

	done := make(chan struct{})
	go func() {
		ab.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Close should not deadlock")
	}
}

func TestAsyncBatch_BufferOverflow(t *testing.T) {
	mock := &mockWriter{}
	ab := NewAsyncBatch(mock, 3, time.Hour)
	ab.Close()

	for i := 0; i < 10; i++ {
		ab.Write([]byte("test"))
	}

	if ab.BufferLen() != 0 {
		t.Errorf("BufferLen = %d after close, want 0", ab.BufferLen())
	}
}

func TestAsyncBatch_EmptyBatchNoWrite(t *testing.T) {
	var buf bytes.Buffer
	ab := NewAsyncBatch(&buf, 10, time.Millisecond)

	ab.Close()

	initialLen := buf.Len()
	time.Sleep(10 * time.Millisecond)

	if buf.Len() != initialLen {
		t.Error("Empty buffer should not trigger write")
	}
}
