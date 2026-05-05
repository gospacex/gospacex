package logger

import (
	"fmt"
	"os"

	"gpx/pkg/logger/scene"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Business *scene.BusinessLogger
	Access   *scene.AccessLogger
	Audit    *scene.AuditLogger
	Error    *scene.ErrorLogger
	storage  *scene.StorageLogger
	config   *Config
}

func New(cfg *Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	levelMgr := NewLevelManager()
	if cfg.Level != "" {
		if lvl, err := zapcore.ParseLevel(cfg.Level); err == nil {
			levelMgr.SetLevels(map[string]zapcore.Level{
				"business": lvl,
				"access":   lvl,
				"audit":    lvl,
				"error":    lvl,
			})
		}
	}

	sceneCfg := &scene.Config{
		BatchSize:            cfg.MQ.BatchSize,
		FlushInterval:        cfg.MQ.FlushInterval,
		Rate:                 cfg.RateLimit.Error.Rate,
		Burst:                cfg.RateLimit.Error.Burst,
		OverflowAction:       cfg.RateLimit.Error.OverflowAction,
		LogLevel:             cfg.Storage.LogLevel,
		LogBody:              cfg.Storage.LogBody,
		SlowThreshold:        cfg.Storage.SlowThreshold,
		VitalBufferSize:      cfg.Vital.BufferSize,
		VitalSyncTimeout:     cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull:  cfg.Vital.FallbackOnFull,
	}

	writer := os.Stdout

	businessLogger, err := scene.NewBusinessLogger(writer, sceneCfg, levelMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create business logger: %w", err)
	}

	accessLogger, err := scene.NewAccessLogger(writer, sceneCfg, levelMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create access logger: %w", err)
	}

	auditLogger, err := scene.NewAuditLogger(writer, sceneCfg, levelMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	errorLogger, err := scene.NewErrorLogger(writer, sceneCfg, levelMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create error logger: %w", err)
	}

	storageLogger, err := scene.NewStorageLogger(writer, sceneCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage logger: %w", err)
	}

	return &Logger{
		Business: businessLogger,
		Access:   accessLogger,
		Audit:    auditLogger,
		Error:    errorLogger,
		storage:  storageLogger,
		config:   cfg,
	}, nil
}

func MustNew(cfg *Config) *Logger {
	l, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return l
}

func NewFromYAML(path string) (*Logger, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return New(cfg)
}

func (l *Logger) Storage() *scene.StorageLogger {
	return l.storage
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		Business: &scene.BusinessLogger{
			Logger:   l.Business.Logger.With(fields...),
			Scene:    l.Business.Scene,
			LevelMgr: l.Business.LevelMgr,
			Core:     l.Business.Core,
			Ab:       l.Business.Ab,
		},
		Access: &scene.AccessLogger{
			Logger:   l.Access.Logger.With(fields...),
			Scene:    l.Access.Scene,
			LevelMgr: l.Access.LevelMgr,
			Core:     l.Access.Core,
			Ab:       l.Access.Ab,
		},
		Audit: &scene.AuditLogger{
			Logger:   l.Audit.Logger.With(fields...),
			Scene:    l.Audit.Scene,
			LevelMgr: l.Audit.LevelMgr,
			Core:     l.Audit.Core,
			Db:       l.Audit.Db,
		},
		Error: &scene.ErrorLogger{
			Logger:      l.Error.Logger.With(fields...),
			Scene:       l.Error.Scene,
			LevelMgr:    l.Error.LevelMgr,
			Core:        l.Error.Core,
			Ab:          l.Error.Ab,
			RateLimiter: l.Error.RateLimiter,
		},
		storage: l.storage,
		config:  l.config,
	}
}
