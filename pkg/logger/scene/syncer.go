package scene

import (
	"gpx/pkg/logger/core"
)

type asyncBatchSyncer struct {
	ab *core.AsyncBatch
}

func (s *asyncBatchSyncer) Write(p []byte) (n int, err error) {
	if s.ab.Write(p) {
		return len(p), nil
	}
	return len(p), nil
}

func (s *asyncBatchSyncer) Sync() error {
	return nil
}

type rateLimitedSyncer struct {
	ab *core.AsyncBatch
	rl *core.RateLimiter
}

func (s *rateLimitedSyncer) Write(p []byte) (n int, err error) {
	if s.rl.Allow() {
		s.ab.Write(p)
	}
	return len(p), nil
}

func (s *rateLimitedSyncer) Sync() error {
	return nil
}

type doubleBufferSyncer struct {
	db *core.DoubleBuffer
}

func (s *doubleBufferSyncer) Write(p []byte) (n int, err error) {
	return s.db.Write(p)
}

func (s *doubleBufferSyncer) Sync() error {
	return s.db.Sync()
}
