package core

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type AsyncBatch struct {
	ch            chan []byte
	batchSize     int
	flushInterval time.Duration
	writer        io.Writer
	wg            sync.WaitGroup
	closed        atomic.Bool
}

func NewAsyncBatch(writer io.Writer, batchSize int, flushInterval time.Duration) *AsyncBatch {
	ab := &AsyncBatch{
		ch:            make(chan []byte, batchSize*2),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		writer:        writer,
	}
	ab.wg.Add(1)
	go ab.processLoop()
	return ab
}

func (ab *AsyncBatch) processLoop() {
	defer ab.wg.Done()
	var batch []byte
	var count int
	ticker := time.NewTicker(ab.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-ab.ch:
			if !ok {
				if count > 0 {
					ab.writeBatch(batch[:count])
				}
				return
			}
			batch = append(batch, data...)
			count++
			if count >= ab.batchSize {
				ab.writeBatch(batch[:count])
				batch = batch[:0]
				count = 0
			}
		case <-ticker.C:
			if count > 0 {
				ab.writeBatch(batch[:count])
				batch = batch[:0]
				count = 0
			}
		}
	}
}

func (ab *AsyncBatch) writeBatch(data []byte) {
	if len(data) > 0 && ab.writer != nil {
		ab.writer.Write(data)
	}
}

func (ab *AsyncBatch) Write(data []byte) bool {
	if ab.closed.Load() {
		return false
	}
	select {
	case ab.ch <- data:
		return true
	default:
		return false
	}
}

func (ab *AsyncBatch) Close() error {
	if ab.closed.Swap(true) {
		return nil
	}
	close(ab.ch)
	ab.wg.Wait()
	return nil
}

func (ab *AsyncBatch) BufferLen() int {
	return len(ab.ch)
}
