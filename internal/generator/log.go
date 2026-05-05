package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

type LogGenerator struct {
	OutputDir string
}

func NewLogGenerator(outputDir string) *LogGenerator {
	return &LogGenerator{
		OutputDir: outputDir,
	}
}

func (g *LogGenerator) Generate() error {
	files := map[string]string{
		"config.go.tmpl":   logConfigTemplate,
		"logger.go.tmpl":   logLoggerTemplate,
		"context.go.tmpl":  logContextTemplate,
		"rotation.go.tmpl": logRotationTemplate,
		"cleaner.go.tmpl": logCleanerTemplate,
		"sampler.go.tmpl": logSamplerTemplate,
		"metrics.go.tmpl": logMetricsTemplate,
		"business.go.tmpl": logBusinessTemplate,
		"access.go.tmpl":  logAccessTemplate,
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

type Config struct {
	Level      string ` + "`yaml:\"level\" default:\"info\"`" + `
	Format     string ` + "`yaml:\"format\" default:\"json\"`" + `
	OutputPath string ` + "`yaml:\"output_path\" default:\"stdout\"`" + `
}

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

type Logger struct {
	*zap.Logger
}

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

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
`

const logContextTemplate = `package log

import (
	"context"

	"go.uber.org/zap"
)

type ContextKey string

const loggerKey ContextKey = "logger"

func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
`

const logRotationTemplate = `package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RotationWriter struct {
	dir         string
	baseName    string
	currentFile *os.File
	currentDate string
	mu          sync.Mutex
}

func NewRotationWriter(dir, baseName string) (*RotationWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	r := &RotationWriter{
		dir:      dir,
		baseName: baseName,
	}
	if err := r.rotate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RotationWriter) rotate() error {
	now := time.Now()
	dateStr := now.Format("2006-01-02")

	if r.currentDate == dateStr && r.currentFile != nil {
		return nil
	}

	if r.currentFile != nil {
		r.currentFile.Close()
	}

	fileName := fmt.Sprintf("%s-%s.log", r.baseName, dateStr)
	filePath := filepath.Join(r.dir, fileName)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	r.currentFile = f
	r.currentDate = dateStr
	return nil
}

func (r *RotationWriter) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.rotate(); err != nil {
		return 0, err
	}

	return r.currentFile.Write(p)
}

func (r *RotationWriter) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentFile != nil {
		return r.currentFile.Sync()
	}
	return nil
}

func (r *RotationWriter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentFile != nil {
		return r.currentFile.Close()
	}
	return nil
}

func GetLogFileDate(fileName string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	parts := strings.Split(base, "-")
	if len(parts) >= 3 {
		return parts[len(parts)-3] + "-" + parts[len(parts)-2] + "-" + parts[len(parts)-1]
	}
	return ""
}
`

const logCleanerTemplate = `package log

import (
	"os"
	"path/filepath"
	"time"
)

type Cleaner struct {
	dir      string
	baseName string
	maxAge   time.Duration
	interval time.Duration
	stopChan chan struct{}
}

func NewCleaner(dir, baseName string, maxAgeDays int, interval time.Duration) *Cleaner {
	return &Cleaner{
		dir:      dir,
		baseName: baseName,
		maxAge:   time.Duration(maxAgeDays) * 24 * time.Hour,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (c *Cleaner) Run() {
	c.clean()
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.clean()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cleaner) Stop() {
	close(c.stopChan)
}

func (c *Cleaner) clean() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isLogFile(c.baseName, name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		age := now.Sub(info.ModTime())
		if age > c.maxAge {
			path := filepath.Join(c.dir, name)
			os.Remove(path)
		}
	}
}

func isLogFile(baseName, fileName string) bool {
	return len(fileName) > len(baseName)+11 &&
		fileName[:len(baseName)] == baseName &&
		fileName[len(fileName)-4:] == ".log"
}
`

const logSamplerTemplate = `package log

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SamplerConfig struct {
	Initial    int
	Thereafter int
	Tick       time.Duration
}

func NewSamplerCore(core zapcore.Core, cfg SamplerConfig) zapcore.Core {
	if cfg.Initial == 0 {
		cfg.Initial = 100
	}
	if cfg.Thereafter == 0 {
		cfg.Thereafter = 200
	}
	if cfg.Tick == 0 {
		cfg.Tick = time.Second
	}

	return zapcore.NewSampler(core, cfg.Tick, cfg.Initial, cfg.Thereafter)
}

var DefaultSamplerConfig = SamplerConfig{
	Initial:    100,
	Thereafter: 200,
	Tick:       time.Second,
}
`

const logMetricsTemplate = `package log

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap/zapcore"
)

type Metrics struct {
	messagesTotal *prometheus.CounterVec
	writeDuration *prometheus.HistogramVec
}

func NewMetrics(namespace, subsystem string) *Metrics {
	if namespace == "" {
		namespace = "app"
	}
	if subsystem == "" {
		subsystem = "log"
	}

	return &Metrics{
		messagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "messages_total",
				Help:      "Total number of log messages",
			},
			[]string{"level", "type"},
		),
		writeDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "write_duration_seconds",
				Help:      "Log write duration in seconds",
			},
			[]string{"level"},
		),
	}
}

func (m *Metrics) Inc(level, logType string) {
	m.messagesTotal.WithLabelValues(level, logType).Inc()
}

func (m *Metrics) Observe(level string, duration float64) {
	m.writeDuration.WithLabelValues(level).Observe(duration)
}

type metricsCore struct {
	zapcore.Core
	metrics *Metrics
}

func WrapWithMetrics(core zapcore.Core, metrics *Metrics) zapcore.Core {
	return &metricsCore{
		Core:    core,
		metrics: metrics,
	}
}

func (m *metricsCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	level := entry.Level.String()
	m.metrics.Inc(level, "total")

	for _, field := range fields {
		if field.Key == "log_type" {
			m.metrics.Inc(level, field.StringValue())
		}
	}

	return m.Core.Write(entry, fields)
}

func (m *metricsCore) Sync() error {
	return m.Core.Sync()
}
`

const logBusinessTemplate = `package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type BusinessLogger struct {
	*zap.SugaredLogger
}

func NewBusinessLogger(logger *zap.Logger) *BusinessLogger {
	return &BusinessLogger{
		logger.Named("business").Sugar(),
	}
}

func (l *BusinessLogger) With(fields ...zap.Field) *BusinessLogger {
	return &BusinessLogger{l.SugaredLogger.With(fields...)}
}

func (l *BusinessLogger) WithLevel(level zapcore.Level, msg string, args ...interface{}) {
	switch level {
	case zapcore.DebugLevel:
		l.Debugw(msg, args...)
	case zapcore.InfoLevel:
		l.Infow(msg, args...)
	case zapcore.WarnLevel:
		l.Warnw(msg, args...)
	case zapcore.ErrorLevel:
		l.Errorw(msg, args...)
	case zapcore.DPanicLevel:
		l.DPanicw(msg, args...)
	case zapcore.PanicLevel:
		l.Panicw(msg, args...)
	case zapcore.FatalLevel:
		l.Fatalw(msg, args...)
	}
}
`

const logAccessTemplate = `package log

import (
	"go.uber.org/zap"
)

type AccessLogger struct {
	*zap.SugaredLogger
}

func NewAccessLogger(logger *zap.Logger) *AccessLogger {
	return &AccessLogger{
		logger.Named("access").Sugar(),
	}
}

type AccessLog struct {
	Method    string ` + "`json:\"method\"`" + `
	Path      string ` + "`json:\"path\"`" + `
	Status    int    ` + "`json:\"status\"`" + `
	Latency   int64  ` + "`json:\"latency_ms\"`" + `
	ClientIP  string ` + "`json:\"client_ip\"`" + `
	UserAgent string ` + "`json:\"user_agent\"`" + `
	RequestID string ` + "`json:\"request_id\"`" + `
}

func (l *AccessLogger) Log(method, path, clientIP, userAgent, requestID string, status int, latency int64) {
	l.Infow("access",
		"method", method,
		"path", path,
		"status", status,
		"latency_ms", latency,
		"client_ip", clientIP,
		"user_agent", userAgent,
		"request_id", requestID,
	)
}

func (l *AccessLogger) With(fields ...zap.Field) *AccessLogger {
	return &AccessLogger{l.SugaredLogger.With(fields...)}
}
`
