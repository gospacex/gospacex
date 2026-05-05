package scene

import (
	"context"
	"io"

	"gpx/pkg/logger/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ErrorLogger struct {
	*zap.Logger
	Scene       Scene
	LevelMgr    LevelManagerInterface
	Core        zapcore.Core
	Ab          *core.AsyncBatch
	RateLimiter *core.RateLimiter
}

func NewErrorLogger(writer io.Writer, cfg *Config, levelMgr LevelManagerInterface) (*ErrorLogger, error) {
	encoderConfig := defaultEncoderConfig()

	ab := core.NewAsyncBatch(writer, cfg.BatchSize, cfg.FlushInterval)
	rl := core.NewRateLimiter(
		cfg.Rate,
		cfg.Burst,
		cfg.OverflowAction,
	)
	syncer := &rateLimitedSyncer{
		ab: ab,
		rl: rl,
	}

	lvl := levelMgr.GetLevel(string(SceneError))
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		syncer,
		lvl,
	)

	zapLogger := zap.New(core)
	return &ErrorLogger{
		Logger:      zapLogger,
		Scene:       SceneError,
		LevelMgr:    levelMgr,
		Core:        core,
		Ab:          ab,
		RateLimiter: rl,
	}, nil
}

func (l *ErrorLogger) Name() Scene {
	return l.Scene
}

func (l *ErrorLogger) SetLevel(level zapcore.Level) {
	l.LevelMgr.SetLevel(string(l.Scene), level)
}

func (l *ErrorLogger) GetLevel() zapcore.Level {
	return l.LevelMgr.GetLevel(string(l.Scene))
}

func (l *ErrorLogger) Sync() error {
	return nil
}

func (l *ErrorLogger) WithContext(ctx context.Context) *ErrorLogger {
	fields := traceFieldsFromContext(ctx)
	return &ErrorLogger{
		Logger:      l.Logger.With(fields...),
		Scene:       l.Scene,
		LevelMgr:    l.LevelMgr,
		Core:        l.Core,
		Ab:          l.Ab,
		RateLimiter: l.RateLimiter,
	}
}

func (l *ErrorLogger) ErrorwContext(ctx context.Context, msg string, keysAndValues ...any) {
	l.WithContext(ctx).Error(msg, keysAndValuesToFields(keysAndValues)...)
}
