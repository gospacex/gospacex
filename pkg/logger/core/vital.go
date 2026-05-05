package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type DoubleBuffer struct {
	active        atomic.Pointer[Buffer]
	standby       atomic.Pointer[Buffer]
	swapping      atomic.Bool
	fsyncTimeout  time.Duration
	fallbackOnFull bool
	isDegraded    atomic.Bool
	mu            sync.Mutex
	writer        io.Writer
	stopCh        chan struct{}
}

type syncer interface {
	Sync() error
}

type Buffer struct {
	data      bytes.Buffer
	mutex     sync.Mutex
	capacity  int
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.data.Write(p)
}

func (b *Buffer) Available() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.capacity - b.data.Len()
}

func (b *Buffer) Len() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.data.Len()
}

func (b *Buffer) Data() []byte {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.data.Bytes()
}

func (b *Buffer) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.data.Reset()
}

func NewDoubleBuffer(fsyncTimeout time.Duration, fallbackOnFull bool, w io.Writer, capacity int) *DoubleBuffer {
	db := &DoubleBuffer{
		fsyncTimeout:   fsyncTimeout,
		fallbackOnFull: fallbackOnFull,
		writer:         w,
		swapping:       atomic.Bool{},
		isDegraded:     atomic.Bool{},
		stopCh:         make(chan struct{}),
	}
	db.active.Store(&Buffer{data: bytes.Buffer{}, capacity: capacity})
	db.standby.Store(&Buffer{data: bytes.Buffer{}, capacity: capacity})
	return db
}

func (db *DoubleBuffer) Write(p []byte) (int, error) {
	active := db.active.Load()

	if active.Available() < len(p) {
		if !db.swapping.Load() {
			swapCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			db.swapAndFsync(swapCtx)
			cancel()
		}
		active = db.active.Load()
		if active.Available() < len(p) {
			if db.fallbackOnFull {
				return db.writer.Write(p)
			}
			return 0, fmt.Errorf("buffer full, fallback disabled")
		}
	}

	active.mutex.Lock()
	_, err := active.data.Write(p)
	active.mutex.Unlock()

	if err != nil {
		return 0, err
	}

	if active.Len() > 0 && !db.swapping.Load() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			db.swapAndFsync(ctx)
		}()
	}

	return len(p), nil
}

func (db *DoubleBuffer) swapAndFsync(_ context.Context) error {
	if !db.swapping.CompareAndSwap(false, true) {
		return nil
	}
	defer db.swapping.Store(false)

	standby := db.standby.Load()
	active := db.active.Swap(standby)

	db.standby.Store(active)

	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), db.fsyncTimeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- db.fsyncBuffer(active)
		}()

		select {
		case err := <-done:
			if err != nil {
				fmt.Fprintf(os.Stderr, "vital fsync failed: %v\n", err)
				db.isDegraded.Store(true)
			} else if db.isDegraded.Load() {
				db.isDegraded.Store(false)
			}
		case <-syncCtx.Done():
			fmt.Fprintf(os.Stderr, "vital fsync timeout exceeded: %v\n", db.fsyncTimeout)
			db.isDegraded.Store(true)
		}
	}()

	return nil
}

func (db *DoubleBuffer) fsyncBuffer(buf *Buffer) error {
	data := buf.Data()
	if len(data) == 0 {
		return nil
	}

	if _, err := db.writer.Write(data); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	if s, ok := db.writer.(syncer); ok {
		if err := s.Sync(); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
	}

	buf.Reset()
	return nil
}

func (db *DoubleBuffer) backgroundSync(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			db.swapAndFsync(syncCtx)
			cancel()
		case <-db.stopCh:
			db.flushAll()
			return
		}
	}
}

func (db *DoubleBuffer) flushAll() {
	db.swapping.Store(false)

	active := db.active.Load()
	db.fsyncBuffer(active)

	standby := db.standby.Load()
	db.fsyncBuffer(standby)
}

func (db *DoubleBuffer) Close() error {
	close(db.stopCh)
	return nil
}

func (db *DoubleBuffer) Sync() error {
	active := db.active.Load()
	return db.fsyncBuffer(active)
}
