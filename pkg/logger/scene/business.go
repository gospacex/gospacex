package scene

import (
	"context"
	"io"

	"gpx/pkg/logger/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type BusinessLogger struct {
	*zap.Logger
	Scene    Scene
	LevelMgr LevelManagerInterface
	Core     zapcore.Core
	Ab       *core.AsyncBatch
}

func NewBusinessLogger(writer io.Writer, cfg *Config, levelMgr LevelManagerInterface) (*BusinessLogger, error) {
	encoderConfig := defaultEncoderConfig()

	ab := core.NewAsyncBatch(writer, cfg.BatchSize, cfg.FlushInterval)
	syncer := &asyncBatchSyncer{ab: ab}

	lvl := levelMgr.GetLevel(string(SceneBusiness))
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		syncer,
		lvl,
	)

	zapLogger := zap.New(core)
	return &BusinessLogger{
		Logger:   zapLogger,
		Scene:    SceneBusiness,
		LevelMgr: levelMgr,
		Core:     core,
		Ab:       ab,
	}, nil
}

func (l *BusinessLogger) Name() Scene {
	return l.Scene
}

func (l *BusinessLogger) SetLevel(level zapcore.Level) {
	l.LevelMgr.SetLevel(string(l.Scene), level)
}

func (l *BusinessLogger) GetLevel() zapcore.Level {
	return l.LevelMgr.GetLevel(string(l.Scene))
}

func (l *BusinessLogger) Sync() error {
	return nil
}

func (l *BusinessLogger) WithContext(ctx context.Context) *BusinessLogger {
	fields := traceFieldsFromContext(ctx)
	return &BusinessLogger{
		Logger:   l.Logger.With(fields...),
		Scene:    l.Scene,
		LevelMgr: l.LevelMgr,
		Core:     l.Core,
		Ab:       l.Ab,
	}
}

func (l *BusinessLogger) InfowContext(ctx context.Context, msg string, keysAndValues ...any) {
	l.WithContext(ctx).Info(msg, keysAndValuesToFields(keysAndValues)...)
}
