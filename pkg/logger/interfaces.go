package logger

import (
	"context"
	"sync/atomic"

	"go.uber.org/zap/zapcore"
)

type Producer interface {
	Push(ctx context.Context, scene, key string, data []byte) error
	Healthy() bool
	Close() error
}

type Sampler interface {
	ShouldSample(ent zapcore.Entry, fields []zapcore.Field) bool
	Close()
}

type InfoSampler struct {
	initial    int
	thereafter int
	tick       int64
	counter    atomic.Int64
}

func NewInfoSampler(initial, thereafter int, tickMs int64) *InfoSampler {
	return &InfoSampler{
		initial:    initial,
		thereafter: thereafter,
		tick:       tickMs,
	}
}

func (s *InfoSampler) ShouldSample(ent zapcore.Entry, fields []zapcore.Field) bool {
	if ent.Level != zapcore.InfoLevel {
		return true
	}
	count := s.counter.Add(1)
	if count <= int64(s.initial) {
		return true
	}
	return (count-int64(s.initial))%int64(s.thereafter) == 0
}

func (s *InfoSampler) Close() {}

type LevelManager struct {
	levels atomic.Value
}

func NewLevelManager() *LevelManager {
	m := &LevelManager{}
	m.levels.Store(map[string]zapcore.Level{
		"business": zapcore.InfoLevel,
		"access":   zapcore.InfoLevel,
		"audit":    zapcore.InfoLevel,
		"error":    zapcore.ErrorLevel,
	})
	return m
}

func (m *LevelManager) SetLevel(scene string, level zapcore.Level) {
	old := m.levels.Load().(map[string]zapcore.Level)
	newMap := make(map[string]zapcore.Level, len(old)+1)
	for k, v := range old {
		newMap[k] = v
	}
	newMap[scene] = level
	m.levels.Store(newMap)
}

func (m *LevelManager) GetLevel(scene string) zapcore.Level {
	levels := m.levels.Load().(map[string]zapcore.Level)
	if level, ok := levels[scene]; ok {
		return level
	}
	return zapcore.InfoLevel
}

func (m *LevelManager) SetLevels(levels map[string]zapcore.Level) {
	old := m.levels.Load().(map[string]zapcore.Level)
	newMap := make(map[string]zapcore.Level, len(old)+len(levels))
	for k, v := range old {
		newMap[k] = v
	}
	for k, v := range levels {
		newMap[k] = v
	}
	m.levels.Store(newMap)
}

func (m *LevelManager) AllLevels() map[string]zapcore.Level {
	return m.levels.Load().(map[string]zapcore.Level)
}
