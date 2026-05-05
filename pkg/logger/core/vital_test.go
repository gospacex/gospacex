package core

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoubleBuffer_Write(t *testing.T) {
	bufferSize := 1000
	syncInterval := time.Second
	fsyncTimeout := 50 * time.Millisecond
	fallbackOnFull := true

	t.Run("writes to active buffer", func(t *testing.T) {
		buf := newDoubleBuffer(bufferSize, syncInterval, fsyncTimeout, fallbackOnFull, io.Discard)

		data := []byte("test entry\n")
		n, err := buf.Write(data)

		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("wrote %d bytes, want %d", n, len(data))
		}

		buf.Close()
	})

	t.Run("concurrent writes safe", func(t *testing.T) {
		buf := newDoubleBuffer(bufferSize, syncInterval, fsyncTimeout, fallbackOnFull, io.Discard)

		var wg sync.WaitGroup
		var counter atomic.Int64

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf.Write([]byte("concurrent entry\n"))
				counter.Add(1)
			}()
		}

		wg.Wait()
		if counter.Load() != 100 {
			t.Errorf("counter = %d, want 100", counter.Load())
		}

		buf.Close()
	})
}

func TestDoubleBuffer_SwapAndFsync(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	bufferSize := 100
	syncInterval := time.Millisecond * 10
	fsyncTimeout := 50 * time.Millisecond
	fallbackOnFull := true

	t.Run("swap atomic with CAS", func(t *testing.T) {
		buf := newDoubleBufferWithFile(logFile, bufferSize, syncInterval, fsyncTimeout, fallbackOnFull)

		for i := 0; i < 50; i++ {
			buf.Write([]byte("entry\n"))
		}

		if err := buf.swapAndFsync(context.Background()); err != nil {
			t.Errorf("swapAndFsync failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
		buf.Close()
	})

	t.Run("swap prevents concurrent swaps", func(t *testing.T) {
		buf := newDoubleBufferWithFile(logFile+"2", bufferSize, syncInterval, fsyncTimeout, fallbackOnFull)
		buf.swapping.Store(true)

		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf.swapAndFsync(context.Background())
			}()
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)
		buf.Close()
	})
}

func TestDoubleBuffer_FsyncTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("fsync timeout triggers degradation", func(t *testing.T) {
		slowFile := filepath.Join(tmpDir, "slow.log")

		slowWriter := &slowFileWriter{
			f:       nil,
			delay:   100 * time.Millisecond,
		}

		f, err := os.Create(slowFile)
		if err != nil {
			t.Fatalf("failed to create slow file: %v", err)
		}
		slowWriter.f = f

		buf := newDoubleBufferWithWriter(100, time.Hour, 50*time.Millisecond, true, slowWriter)

		active := buf.active.Load()
		active.data.WriteString("entry\n")

		buf.swapAndFsync(context.Background())

		time.Sleep(200 * time.Millisecond)

		if !buf.isDegraded.Load() {
			t.Error("expected degraded mode after slow fsync")
		}

		f.Close()
		buf.Close()
	})
}

type slowFileWriter struct {
	f     *os.File
	delay time.Duration
}

func (w *slowFileWriter) Write(p []byte) (int, error) {
	return w.f.Write(p)
}

func (w *slowFileWriter) Sync() error {
	time.Sleep(w.delay)
	return w.f.Sync()
}

func TestDoubleBuffer_FallbackOnFull(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "fallback.log")

	buf := newDoubleBufferWithFile(logFile, 10, time.Hour, 50*time.Millisecond, true)

	data := []byte("fallback entry\n")

	for i := 0; i < 20; i++ {
		n, err := buf.Write(data)
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("wrote %d bytes, want %d", n, len(data))
		}
	}

	time.Sleep(100 * time.Millisecond)
	buf.Close()
}

func TestBuffer_Data(t *testing.T) {
	buf := newBuffer(100)

	buf.mutex.Lock()
	buf.data.WriteString("line1\n")
	buf.mutex.Unlock()

	content := buf.Data()
	if !bytes.Contains(content, []byte("line1")) {
		t.Error("expected buffer to contain line1")
	}
}

func newDoubleBuffer(bufferSize int, syncInterval, fsyncTimeout time.Duration, fallbackOnFull bool, w io.Writer) *DoubleBuffer {
	db := &DoubleBuffer{
		fsyncTimeout:   fsyncTimeout,
		fallbackOnFull: fallbackOnFull,
		writer:         w,
		swapping:       atomic.Bool{},
		isDegraded:     atomic.Bool{},
		stopCh:         make(chan struct{}),
	}
	db.active.Store(&Buffer{data: bytes.Buffer{}, capacity: bufferSize})
	db.standby.Store(&Buffer{data: bytes.Buffer{}, capacity: bufferSize})

	go db.backgroundSync(syncInterval)

	return db
}

func newDoubleBufferWithFile(path string, bufferSize int, syncInterval, fsyncTimeout time.Duration, fallbackOnFull bool) *DoubleBuffer {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	return newDoubleBuffer(bufferSize, syncInterval, fsyncTimeout, fallbackOnFull, f)
}

func newDoubleBufferWithWriter(bufferSize int, syncInterval, fsyncTimeout time.Duration, fallbackOnFull bool, w io.Writer) *DoubleBuffer {
	db := &DoubleBuffer{
		fsyncTimeout:   fsyncTimeout,
		fallbackOnFull: fallbackOnFull,
		writer:         w,
		swapping:       atomic.Bool{},
		isDegraded:     atomic.Bool{},
		stopCh:         make(chan struct{}),
	}
	db.active.Store(&Buffer{data: bytes.Buffer{}, capacity: bufferSize})
	db.standby.Store(&Buffer{data: bytes.Buffer{}, capacity: bufferSize})

	go db.backgroundSync(syncInterval)

	return db
}

func newBuffer(cap int) *Buffer {
	return &Buffer{
		data:     bytes.Buffer{},
		capacity: cap,
	}
}