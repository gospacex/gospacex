package scene

import (
	"io"
	"sync/atomic"
	"time"

	"gpx/pkg/logger/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type StorageLogger struct {
	*zap.Logger
	Scene  Scene
	Level  atomic.Value
	Cfg    *Config
	Core   zapcore.Core
	Ab     *core.AsyncBatch
}

func NewStorageLogger(writer io.Writer, cfg *Config) (*StorageLogger, error) {
	encoderConfig := defaultEncoderConfig()

	ab := core.NewAsyncBatch(writer, cfg.BatchSize, cfg.FlushInterval)
	syncer := &asyncBatchSyncer{ab: ab}

	storageLevel := zapcore.DebugLevel
	if cfg.LogLevel != "" {
		if lvl, err := zapcore.ParseLevel(cfg.LogLevel); err == nil {
			storageLevel = lvl
		}
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		syncer,
		storageLevel,
	)

	zapLogger := zap.New(core)

	sl := &StorageLogger{
		Logger: zapLogger,
		Scene:  SceneStorage,
		Cfg:    cfg,
		Core:   core,
		Ab:     ab,
	}
	sl.Level.Store(storageLevel)

	return sl, nil
}

func (l *StorageLogger) Name() Scene {
	return l.Scene
}

func (l *StorageLogger) SetLevel(level zapcore.Level) {
	l.Level.Store(level)
}

func (l *StorageLogger) GetLevel() zapcore.Level {
	return l.Level.Load().(zapcore.Level)
}

func (l *StorageLogger) Sync() error {
	return nil
}

func (l *StorageLogger) LogBody() bool {
	return l.Cfg.LogBody
}

func (l *StorageLogger) SlowThreshold() time.Duration {
	return l.Cfg.SlowThreshold
}
