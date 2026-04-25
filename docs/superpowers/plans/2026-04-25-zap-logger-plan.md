# 企业级日志库实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 基于 uber-go/zap 封装企业级日志库，集成到脚手架自动生成

**架构：** 三层结构：配置层（config.go）+ 核心层（logger.go/rotation.go）+ 业务层（business.go/access.go）

**技术栈：** Go, uber-go/zap, prometheus

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `templates/pkg/log/config.go.tmpl` | 配置定义 + YAML 加载 |
| `templates/pkg/log/logger.go.tmpl` | 主 Logger 封装，多输出 |
| `templates/pkg/log/rotation.go.tmpl` | 按日期轮转日志 |
| `templates/pkg/log/cleaner.go.tmpl` | 过期文件清理 |
| `templates/pkg/log/sampler.go.tmpl` | 采样配置 |
| `templates/pkg/log/metrics.go.tmpl` | Prometheus 指标 |
| `templates/pkg/log/business.go.tmpl` | 业务日志 |
| `templates/pkg/log/access.go.tmpl` | 访问日志 |
| `templates/pkg/log/context.go.tmpl` | Context 传递 |

---

## 任务 1：配置结构 (config.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/config.go.tmpl`

- [ ] **步骤 1：创建配置模板**

```go
package log

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Env        string       `yaml:"env" default:"dev"`
	Level      string       `yaml:"level" default:"info"`
	Sampling   SamplingCfg  `yaml:"sampling"`
	Rotation   RotationCfg  `yaml:"rotation"`
	Output     OutputCfg    `yaml:"output"`
	Prometheus PrometheusCfg `yaml:"prometheus"`
}

type SamplingCfg struct {
	Initial    int           `yaml:"initial" default:"100"`
	Thereafter int           `yaml:"thereafter" default:"200"`
	Tick       time.Duration `yaml:"tick" default:"1s"`
}

type RotationCfg struct {
	Enabled     bool  `yaml:"enabled" default:"true"`
	MaxAgeDays  int   `yaml:"max_age_days" default:"7"`
}

type OutputCfg struct {
	File   string `yaml:"file" default:"./logs/app.log"`
	Stdout bool   `yaml:"stdout" default:"true"`
}

type PrometheusCfg struct {
	Enabled   bool   `yaml:"enabled" default:"true"`
	Namespace string `yaml:"namespace" default:"app"`
	Subsystem string `yaml:"subsystem" default:"log"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if _, err := zapcore.ParseLevel(c.Level); err != nil {
		c.Level = "info"
	}
	return nil
}
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/config.go.tmpl
git commit -m "feat(log): add config structure"
```

---

## 任务 2：日志轮转 (rotation.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/rotation.go.tmpl`

- [ ] **步骤 1：创建日志轮转模板**

```go
package log

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
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/rotation.go.tmpl
git commit -m "feat(log): add rotation writer"
```

---

## 任务 3：过期清理 (cleaner.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/cleaner.go.tmpl`

- [ ] **步骤 1：创建清理模板**

```go
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Cleaner struct {
	dir       string
	baseName  string
	maxAge    time.Duration
	interval  time.Duration
	stopChan  chan struct{}
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
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/cleaner.go.tmpl
git commit -m "feat(log): add log cleaner"
```

---

## 任务 4：采样器 (sampler.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/sampler.go.tmpl`

- [ ] **步骤 1：创建采样配置模板**

```go
package log

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

func NewSamplerCore(core zapcore.Core, cfg SamplingCfg) zapcore.Core {
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
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/sampler.go.tmpl
git commit -m "feat(log): add sampler config"
```

---

## 任务 5：Prometheus 指标 (metrics.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/metrics.go.tmpl`

- [ ] **步骤 1：创建指标模板**

```go
package log

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
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/metrics.go.tmpl
git commit -m "feat(log): add prometheus metrics"
```

---

## 任务 6：Context 传递 (context.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/context.go.tmpl`

- [ ] **步骤 1：创建 Context 模板**

```go
package log

import (
	"context"

	"go.uber.org/zap"
)

type ContextKey string

const loggerKey ContextKey = "logger"

type LoggerWithContext struct {
	*zap.Logger
}

func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

func FromContextWithKey(ctx context.Context, key interface{}) *zap.Logger {
	if logger, ok := ctx.Value(key).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/context.go.tmpl
git commit -m "feat(log): add context support"
```

---

## 任务 7：主 Logger (logger.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/logger.go.tmpl`

- [ ] **步骤 1：创建主 Logger 模板**

```go
package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Business *zap.SugaredLogger
	Access   *zap.SugaredLogger
	Error    *zap.SugaredLogger
	cleaner  *Cleaner
}

func NewLogger(cfg *Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	level, _ := zapcore.ParseLevel(cfg.Level)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Env == "dev" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var ws []zapcore.WriteSyncer

	dir := "./logs"
	baseName := "app"
	if cfg.Output.File != "" {
		dir = filepath.Dir(cfg.Output.File)
		baseName = strings.TrimSuffix(filepath.Base(cfg.Output.File), ".log")
	}

	if cfg.Rotation.Enabled {
		rotation, err := NewRotationWriter(dir, baseName)
		if err != nil {
			return nil, err
		}
		ws = append(ws, zapcore.AddSync(rotation))
	} else {
		file, err := os.OpenFile(cfg.Output.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		ws = append(ws, zapcore.AddSync(file))
	}

	if cfg.Output.Stdout {
		ws = append(ws, zapcore.AddSync(os.Stdout))
	}

	wsCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(ws...), level)

	sampledCore := NewSamplerCore(wsCore, cfg.Sampling)

	var finalCore zapcore.Core = sampledCore

	var metrics *Metrics
	if cfg.Prometheus.Enabled {
		metrics = NewMetrics(cfg.Prometheus.Namespace, cfg.Prometheus.Subsystem)
		finalCore = WrapWithMetrics(sampledCore, metrics)
	}

	logger := zap.New(finalCore, zap.AddCaller(), zap.AddCallerSkip(1))

	var cleaner *Cleaner
	if cfg.Rotation.Enabled && cfg.Rotation.MaxAgeDays > 0 {
		cleaner = NewCleaner(dir, baseName, cfg.Rotation.MaxAgeDays, 6*time.Hour)
		go cleaner.Run()
	}

	return &Logger{
		Business: logger.Sugar().Named("business"),
		Access:   logger.Sugar().Named("access"),
		Error:    logger.Sugar().Named("error"),
		cleaner:  cleaner,
	}, nil
}

func (l *Logger) Sync() error {
	return nil
}

func (l *Logger) Stop() {
	if l.cleaner != nil {
		l.cleaner.Stop()
	}
}
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/logger.go.tmpl
git commit -m "feat(log): add main logger"
```

---

## 任务 8：业务日志封装 (business.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/business.go.tmpl`

- [ ] **步骤 1：创建业务日志模板**

```go
package log

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
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/business.go.tmpl
git commit -m "feat(log): add business logger"
```

---

## 任务 9：访问日志 (access.go.tmpl)

**文件：**
- 创建：`templates/pkg/log/access.go.tmpl`

- [ ] **步骤 1：创建访问日志模板**

```go
package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	Latency    int64  `json:"latency_ms"`
	ClientIP   string `json:"client_ip"`
	UserAgent  string `json:"user_agent"`
	RequestID  string `json:"request_id"`
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
```

- [ ] **步骤 2：Commit**

```bash
git add templates/pkg/log/access.go.tmpl
git commit -m "feat(log): add access logger"
```

---

## 任务 10：更新 LogGenerator

**文件：**
- 修改：`internal/generator/log.go`

- [ ] **步骤 1：更新生成器添加新文件**

```go
func (g *LogGenerator) Generate() error {
	files := map[string]string{
		"config.go.tmpl":   logConfigTemplate,
		"logger.go.tmpl":   logLoggerTemplate,
		"context.go.tmpl":  logContextTemplate,
		"rotation.go.tmpl": logRotationTemplate,
		"cleaner.go.tmpl":  logCleanerTemplate,
		"sampler.go.tmpl":  logSamplerTemplate,
		"metrics.go.tmpl":  logMetricsTemplate,
		"business.go.tmpl": logBusinessTemplate,
		"access.go.tmpl":   logAccessTemplate,
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
```

- [ ] **步骤 2：Commit**

```bash
git add internal/generator/log.go
git commit -m "feat(log): update generator for new templates"
```

---

## 任务 11：生成配置模板

**文件：**
- 创建：`templates/config/log.yaml.tmpl`

- [ ] **步骤 1：创建配置模板**

```yaml
env: {{.Env}}
level: info
sampling:
  initial: 100
  thereafter: 200
  tick: 1s
rotation:
  enabled: true
  max_age_days: 7
output:
  file: ./logs/app.log
  stdout: true
prometheus:
  enabled: true
  namespace: app
  subsystem: log
```

- [ ] **步骤 2：Commit**

```bash
git add templates/config/log.yaml.tmpl
git commit -m "feat(log): add config template"
```

---

## 执行交接

计划已完成并保存到 `docs/superpowers/plans/2026-04-25-zap-logger-plan.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

选哪种方式？
