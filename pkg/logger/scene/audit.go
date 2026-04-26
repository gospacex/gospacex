package scene

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gpx/pkg/logger/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AuditLogger struct {
	*zap.Logger
	Scene    Scene
	LevelMgr LevelManagerInterface
	Core     zapcore.Core
	Db       *core.DoubleBuffer
}

func NewAuditLogger(writer io.Writer, cfg *Config, levelMgr LevelManagerInterface) (*AuditLogger, error) {
	encoderConfig := defaultEncoderConfig()

	db := core.NewDoubleBuffer(cfg.VitalSyncTimeout, cfg.VitalFallbackOnFull, writer, cfg.VitalBufferSize)

	lvl := levelMgr.GetLevel(string(SceneAudit))
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		db,
		lvl,
	)

	zapLogger := zap.New(core)
	return &AuditLogger{
		Logger:   zapLogger,
		Scene:    SceneAudit,
		LevelMgr: levelMgr,
		Core:     core,
		Db:       db,
	}, nil
}

func (l *AuditLogger) Name() Scene {
	return l.Scene
}

func (l *AuditLogger) SetLevel(level zapcore.Level) {
	l.LevelMgr.SetLevel(string(l.Scene), level)
}

func (l *AuditLogger) GetLevel() zapcore.Level {
	return l.LevelMgr.GetLevel(string(l.Scene))
}

func (l *AuditLogger) Sync() error {
	return l.Db.Sync()
}

type AuditRecord struct {
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	UserID     interface{}    `json:"user_id"`
	ResourceID string         `json:"resource_id"`
	TraceID    string         `json:"trace_id"`
	SpanID     string         `json:"span_id"`
	Details    map[string]any `json:"details,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

func (a *AuditLogger) Log(record *AuditRecord) {
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	data, err := json.Marshal(record)
	if err != nil {
		a.Error("failed to marshal audit record", zap.String("error", err.Error()))
		return
	}
	a.Info(string(data))
}

func (a *AuditLogger) LogContext(ctx context.Context, record *AuditRecord) {
	record.TraceID, record.SpanID = ExtractTraceFields(ctx)
	a.Log(record)
}

func (a *AuditLogger) WithContext(ctx context.Context) *AuditLogger {
	fields := traceFieldsFromContext(ctx)
	return &AuditLogger{
		Logger:   a.Logger.With(fields...),
		Scene:    a.Scene,
		LevelMgr: a.LevelMgr,
		Core:     a.Core,
		Db:       a.Db,
	}
}

func (a *AuditLogger) InfowContext(ctx context.Context, msg string, keysAndValues ...any) {
	a.WithContext(ctx).Info(msg, keysAndValuesToFields(keysAndValues)...)
}

func (a *AuditLogger) Logf(action, resource, format string, args ...any) {
	record := &AuditRecord{
		Action:    action,
		Resource:  resource,
		Timestamp: time.Now(),
	}
	if len(args) > 0 {
		record.Details = map[string]any{
			"message": fmt.Sprintf(format, args...),
		}
	}
	a.Log(record)
}
