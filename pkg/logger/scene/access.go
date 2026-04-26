package scene

import (
	"context"
	"io"

	"gpx/pkg/logger/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AccessLogger struct {
	*zap.Logger
	Scene    Scene
	LevelMgr LevelManagerInterface
	Core     zapcore.Core
	Ab       *core.AsyncBatch
}

func NewAccessLogger(writer io.Writer, cfg *Config, levelMgr LevelManagerInterface) (*AccessLogger, error) {
	encoderConfig := defaultEncoderConfig()

	ab := core.NewAsyncBatch(writer, cfg.BatchSize, cfg.FlushInterval)
	syncer := &asyncBatchSyncer{ab: ab}

	lvl := levelMgr.GetLevel(string(SceneAccess))
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		syncer,
		lvl,
	)

	zapLogger := zap.New(core)
	return &AccessLogger{
		Logger:   zapLogger,
		Scene:    SceneAccess,
		LevelMgr: levelMgr,
		Core:     core,
		Ab:       ab,
	}, nil
}

func (l *AccessLogger) Name() Scene {
	return l.Scene
}

func (l *AccessLogger) SetLevel(level zapcore.Level) {
	l.LevelMgr.SetLevel(string(l.Scene), level)
}

func (l *AccessLogger) GetLevel() zapcore.Level {
	return l.LevelMgr.GetLevel(string(l.Scene))
}

func (l *AccessLogger) Sync() error {
	return nil
}

func (l *AccessLogger) WithContext(ctx context.Context) *AccessLogger {
	fields := traceFieldsFromContext(ctx)
	return &AccessLogger{
		Logger:   l.Logger.With(fields...),
		Scene:    l.Scene,
		LevelMgr: l.LevelMgr,
		Core:     l.Core,
		Ab:       l.Ab,
	}
}

func (l *AccessLogger) InfowContext(ctx context.Context, msg string, keysAndValues ...any) {
	l.WithContext(ctx).Info(msg, keysAndValuesToFields(keysAndValues)...)
}
