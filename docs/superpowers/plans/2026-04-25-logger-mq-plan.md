# 日志系统重构与 MQ 推送功能实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 重构日志系统为 `pkg/log`，支持多场景日志和 Kafka 异步批量推送

**架构：** 统一使用 `pkg/log` 目录，实现 Business/Access/Audit/Error 四种场景日志，通过异步队列将日志批量推送到 Kafka

**技术栈：** Go, zap, sarama (Kafka client)

---

## 文件结构

### 模板文件（创建/修改）

- 修改：`templates/pkg/log/logger.go.tmpl`
- 修改：`templates/pkg/log/config.go.tmpl`
- 修改：`templates/pkg/log/business.go.tmpl`
- 修改：`templates/pkg/log/access.go.tmpl`
- 创建：`templates/pkg/log/audit.go.tmpl`
- 创建：`templates/pkg/log/error.go.tmpl`
- 创建：`templates/pkg/log/mq.go.tmpl`
- 创建：`templates/pkg/log/mq_kafka.go.tmpl`
- 修改：`templates/config/log.yaml.tmpl`

### 代码生成逻辑（修改）

- 修改：`internal/cli/microapp_new.go:1306` - 将 `templates/pkg/log` 改为 `templates/pkg/log`
- 修改：`internal/cli/microapp_new.go:1374-1383` - 移除 pkg/logger 兼容层
- 修改：`internal/generator/template_engine.go:68` - 将 `pkg/logger` 改为 `pkg/log`
- 修改：`internal/generator/script_center.go:34,58,1845` - 移除 pkg/logger 引用
- 修改：`internal/generator/scriptcenter/generator.go:47` - 将 `pkg/logger` 改为 `pkg/log`
- 修改：`internal/generator/microservice.go:162` - 将 `pkg/logger` 改为 `pkg/log`

---

## 任务列表

### 任务 1：创建 AuditLogger 和 ErrorLogger 模板

**文件：**
- 创建：`templates/pkg/log/audit.go.tmpl`
- 创建：`templates/pkg/log/error.go.tmpl`

- [ ] **步骤 1：创建 audit.go.tmpl**

```go
package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/sugar"
)

type AuditLogger struct {
	logger *zap.Logger
	sugar  *sugar.SugaredLogger
}

func NewAuditLogger(logger *zap.Logger) *AuditLogger {
	named := logger.Named("audit")
	return &AuditLogger{
		logger: named,
		sugar:  named.Sugar(),
	}
}

func (l *AuditLogger) Log(action, userID, resource string, details map[string]interface{}) {
	l.sugar.Infow(action,
		"user_id", userID,
		"resource", resource,
		"details", details,
	)
}

func (l *AuditLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *AuditLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}
```

- [ ] **步骤 2：创建 error.go.tmpl**

```go
package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/sugar"
)

type ErrorLogger struct {
	logger *zap.Logger
	sugar  *sugar.SugaredLogger
}

func NewErrorLogger(logger *zap.Logger) *ErrorLogger {
	named := logger.Named("error")
	return &ErrorLogger{
		logger: named,
		sugar:  named.Sugar(),
	}
}

func (l *ErrorLogger) Log(err error, msg string, keysAndValues ...interface{}) {
	fields := []interface{}{"error", err.Error()}
	fields = append(fields, keysAndValues...)
	l.sugar.Errorw(msg, fields...)
}

func (l *ErrorLogger) WithStack(err error) *zap.SugaredLogger {
	return l.logger.With(zap.Stack("stack")).Sugar()
}

func (l *ErrorLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *ErrorLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *ErrorLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}
```

---

### 任务 2：创建 MQ 推送功能模板

**文件：**
- 创建：`templates/pkg/log/mq.go.tmpl`
- 创建：`templates/pkg/log/mq_kafka.go.tmpl`

- [ ] **步骤 1：创建 mq.go.tmpl**

```go
package log

import (
	"encoding/json"
	"sync"
	"time"
)

type MQPusher interface {
	Push(topic string, data []byte) error
	Close() error
}

type MQConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Type          string        `yaml:"type"`
	Brokers       []string      `yaml:"brokers"`
	Topic         string        `yaml:"topic"`
	Async         bool          `yaml:"async"`
	BatchSize     int           `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
}

type AsyncMQPusher struct {
	mu       sync.Mutex
	client   MQPusher
	topic    string
	buffer   [][]byte
	batchSize int
	interval time.Duration
	flushCh  chan struct{}
	closed   bool
	wg       sync.WaitGroup
}

func NewAsyncMQPusher(client MQPusher, topic string, batchSize int, interval time.Duration) *AsyncMQPusher {
	p := &AsyncMQPusher{
		client:    client,
		topic:     topic,
		batchSize: batchSize,
		interval:  interval,
		flushCh:   make(chan struct{}, 1),
	}
	p.wg.Add(1)
	go p.flushLoop()
	return p
}

func (p *AsyncMQPusher) Push(topic string, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.buffer = append(p.buffer, data)

	if len(p.buffer) >= p.batchSize {
		p.doFlush()
	}

	return nil
}

func (p *AsyncMQPusher) flushLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			if len(p.buffer) > 0 {
				p.doFlush()
			}
			p.mu.Unlock()
		case <-p.flushCh:
			p.mu.Lock()
			p.doFlush()
			p.mu.Unlock()
		}
	}
}

func (p *AsyncMQPusher) doFlush() {
	if len(p.buffer) == 0 {
		return
	}

	for _, data := range p.buffer {
		_ = p.client.Push(p.topic, data)
	}
	p.buffer = p.buffer[:0]
}

func (p *AsyncMQPusher) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	p.flushCh <- struct{}{}
	p.wg.Wait()

	return p.client.Close()
}

type LogEntry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Scene   string                 `json:"scene"`
	Fields  map[string]interface{} `json:"fields"`
}

func MarshalLogEntry(scene, level, message string, fields map[string]interface{}) ([]byte, error) {
	entry := LogEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Message: message,
		Scene:   scene,
		Fields:  fields,
	}
	return json.Marshal(entry)
}
```

- [ ] **步骤 2：创建 mq_kafka.go.tmpl**

```go
package log

import (
	"context"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
)

type KafkaPusher struct {
	producer sarama.SyncProducer
	topic    string
	mu       sync.Mutex
	closed   bool
}

func NewKafkaPusher(brokers []string, topic string) (*KafkaPusher, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &KafkaPusher{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *KafkaPusher) Push(topic string, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(""),
		Value: sarama.ByteEncoder(data),
	}

	_, _, err := p.producer.SendMessage(msg)
	return err
}

func (p *KafkaPusher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	return p.producer.Close()
}

func NewMQPusherFromConfig(cfg *MQConfig) (MQPusher, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	switch cfg.Type {
	case "kafka":
		pusher, err := NewKafkaPusher(cfg.Brokers, cfg.Topic)
		if err != nil {
			return nil, err
		}
		if cfg.Async {
			return NewAsyncMQPusher(pusher, cfg.Topic, cfg.BatchSize, cfg.FlushInterval), nil
		}
		return pusher, nil
	default:
		return nil, fmt.Errorf("unsupported mq type: %s", cfg.Type)
	}
}

type ContextKey string

const MQPusherKey ContextKey = "mq_pusher"

func WithMQPusher(ctx context.Context, pusher MQPusher) context.Context {
	return context.WithValue(ctx, MQPusherKey, pusher)
}

func FromMQPusher(ctx context.Context) (MQPusher, bool) {
	pusher, ok := ctx.Value(MQPusherKey).(MQPusher)
	return pusher, ok
}
```

---

### 任务 3：更新主 Logger 和配置模板

**文件：**
- 修改：`templates/pkg/log/logger.go.tmpl`
- 修改：`templates/pkg/log/config.go.tmpl`
- 修改：`templates/config/log.yaml.tmpl`

- [ ] **步骤 1：更新 logger.go.tmpl**

在现有 Logger 结构体中添加 Audit 和 Error 字段，并添加 MQ 推送相关代码。

```go
type Logger struct {
	Business *BusinessLogger  // 业务日志
	Access   *AccessLogger    // 访问日志
	Audit    *AuditLogger     // 审计日志
	Error    *ErrorLogger     // 错误日志
	mqPusher MQPusher         // MQ 推送器
	cleaner  *Cleaner
}

// 在 NewLogger 函数中添加：
mqPusher, err := NewMQPusherFromConfig(&cfg.MQ)
if err != nil {
	return nil, fmt.Errorf("failed to create mq pusher: %w", err)
}

return &Logger{
	Business: NewBusinessLogger(logger),
	Access:   NewAccessLogger(logger),
	Audit:    NewAuditLogger(logger),
	Error:    NewErrorLogger(logger),
	mqPusher: mqPusher,
	cleaner:  cleaner,
}, nil

// 添加 PushToMQ 方法：
func (l *Logger) PushToMQ(scene, level, message string, fields map[string]interface{}) error {
	if l.mqPusher == nil {
		return nil
	}
	topic := fmt.Sprintf("%s-%s", l.mqPusher.(*AsyncMQPusher).(*KafkaPusher).topic, scene)
	data, err := MarshalLogEntry(scene, level, message, fields)
	if err != nil {
		return err
	}
	return l.mqPusher.Push(topic, data)
}
```

- [ ] **步骤 2：更新 config.go.tmpl**

在 Config 结构体中添加 MQ 配置字段。

```go
type Config struct {
	Env        string         `yaml:"env"`
	Level      string         `yaml:"level"`
	Sampling   SamplingConfig `yaml:"sampling"`
	Rotation   RotationConfig `yaml:"rotation"`
	Output     OutputConfig   `yaml:"output"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	MQ         MQConfig       `yaml:"mq"`
}
```

- [ ] **步骤 3：更新 log.yaml.tmpl**

添加 MQ 配置项。

```yaml
mq:
  enabled: false
  type: kafka
  brokers:
    - localhost:9092
  topic: app-logs
  async: true
  batch_size: 100
  flush_interval: 3s
```

---

### 任务 4：更新代码生成逻辑

**文件：**
- 修改：`internal/cli/microapp_new.go:1306,1374-1383`
- 修改：`internal/generator/template_engine.go:68`
- 修改：`internal/generator/script_center.go:34,58,1845`
- 修改：`internal/generator/scriptcenter/generator.go:47`
- 修改：`internal/generator/microservice.go:162`

- [ ] **步骤 1：修改 microapp_new.go**

将第 1306 行的 `srcDir := "templates/pkg/log"` 保持不变（已经是正确的）。

将第 1374-1383 行关于生成 pkg/logger 兼容层的代码移除。

- [ ] **步骤 2：修改 template_engine.go**

将第 68 行的 `"pkg/logger",` 改为 `"pkg/log",`

- [ ] **步骤 3：修改 script_center.go**

将第 34 行和第 58 行的 `"pkg/logger"` 改为 `"pkg/log"`，删除或注释掉 `generatePkgLogger` 函数。

- [ ] **步骤 4：修改 scriptcenter/generator.go**

将第 47 行的 `"pkg/logger"` 改为 `"pkg/log"`

- [ ] **步骤 5：修改 microservice.go**

将第 162 行的目录列表中的 `"pkg/logger"` 改为 `"pkg/log"`

---

### 任务 5：更新 BusinessLogger 和 AccessLogger 支持 MQ

**文件：**
- 修改：`templates/pkg/log/business.go.tmpl`
- 修改：`templates/pkg/log/access.go.tmpl`

- [ ] **步骤 1：更新 business.go.tmpl 添加 MQ 推送**

在 BusinessLogger 中添加 PushToMQ 方法，调用 Logger 的 PushToMQ。

```go
func (l *BusinessLogger) WithMQ(pusher MQPusher, topic string) *BusinessLoggerWithMQ {
	return &BusinessLoggerWithMQ{
		logger: l.logger,
		pusher: pusher,
		topic:  topic,
	}
}

type BusinessLoggerWithMQ struct {
	logger *zap.Logger
	pusher MQPusher
	topic  string
}

func (l *BusinessLoggerWithMQ) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Infow(msg, keysAndValues...)
	if l.pusher != nil {
		fields := make(map[string]interface{})
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				fields[fmt.Sprintf("%v", keysAndValues[i])] = keysAndValues[i+1]
			}
		}
		data, _ := MarshalLogEntry("business", "info", msg, fields)
		l.pusher.Push(l.topic, data)
	}
}
```

- [ ] **步骤 2：类似更新 access.go.tmpl**
