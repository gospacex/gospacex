package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// LogGenerator generates log package templates
type LogGenerator struct {
	OutputDir string
}

// NewLogGenerator creates a new LogGenerator
func NewLogGenerator(outputDir string) *LogGenerator {
	return &LogGenerator{
		OutputDir: outputDir,
	}
}

// Generate creates log package template files
func (g *LogGenerator) Generate() error {
	files := map[string]string{
		"config.go.tmpl":  logConfigTemplate,
		"logger.go.tmpl":  logLoggerTemplate,
		"context.go.tmpl": logContextTemplate,
	}

	dir := filepath.Join(g.OutputDir, "templates", "pkg", "log")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write file %s: %w", name, err)
		}
	}

	return nil
}

const logConfigTemplate = `package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config Log configuration
type Config struct {
	Level      string ` + "`yaml:\"level\" default:\"info\"`" + `
	Format     string ` + "`yaml:\"format\" default:\"json\"`" + `
	OutputPath string ` + "`yaml:\"output_path\" default:\"stdout\"`" + `
}

// NewLogger creates a new Logger
func NewLogger(cfg *Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)), nil
}
`

const logLoggerTemplate = `package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger Logger wrapper
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new Logger
func NewLogger(cfg *Config) (*Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{Logger: logger}, nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// With creates a new Logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
`

const logContextTemplate = `package log

import (
	"context"

	"go.uber.org/zap"
)

// ContextKey key for logger in context
type ContextKey string

const loggerKey ContextKey = "logger"

// WithContext adds logger to context
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext gets logger from context
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
`
