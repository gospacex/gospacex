# 日志目录合并与 BFF/SRV 层集成实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 合并日志目录到 pkg/logger，在 BFF/SRV 层集成日志，生成使用文档

**架构：** 将 pkg/log 的功能移动到 pkg/logger，更新代码生成逻辑，在各层添加日志调用

**技术栈：** Go, zap, sarama

---

## 文件结构

### 模板文件（移动/创建）

- 移动：`templates/pkg/log/*.tmpl` → `templates/pkg/logger/`
- 创建：`templates/pkg/logger/audit.go.tmpl`
- 创建：`templates/pkg/logger/error.go.tmpl`
- 创建：`templates/pkg/logger/mq.go.tmpl`
- 创建：`templates/pkg/logger/mq_kafka.go.tmpl`

### 代码生成逻辑（修改）

- 修改：`internal/cli/microapp_new.go:1306`
- 修改：`internal/generator/template_engine.go:68`
- 修改：`internal/generator/script_center.go:34`
- 修改：`internal/generator/scriptcenter/generator.go:47`
- 修改：`internal/generator/microservice.go:162`

### BFF 层日志集成（修改模板）

- 修改：`templates/micro-app/bff/handler/handler.go.tmpl`
- 修改：`templates/micro-app/bff/middleware/gin_middleware.go.tmpl`
- 修改：`templates/micro-app/bff/main/gin_main.go.tmpl`

### SRV 层日志集成（修改模板）

- 修改：`templates/micro-app/srv/handler/handler.go.tmpl`
- 修改：`templates/micro-app/srv/interceptor/kitex_interceptor.go.tmpl`
- 修改：`templates/micro-app/srv/main/main_*.go.tmpl`
- 修改：`templates/micro-app/srv/repo/repository.go.tmpl`

### 文档（创建）

- 创建：`docs/logging-usage.md`

---

## 任务列表

### 任务 1：移动日志模板文件并重建丢失文件

**文件：**
- 移动：`templates/pkg/log/*.tmpl` → `templates/pkg/logger/`
- 创建：`templates/pkg/logger/audit.go.tmpl`
- 创建：`templates/pkg/logger/error.go.tmpl`
- 创建：`templates/pkg/logger/mq.go.tmpl`
- 创建：`templates/pkg/logger/mq_kafka.go.tmpl`

- [ ] **步骤 1：创建 audit.go.tmpl**

```go
package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/sugar"
)

type AuditLogger struct {
	*zap.SugaredLogger
}

func NewAuditLogger(logger *zap.Logger) *AuditLogger {
	return &AuditLogger{
		logger.Named("audit").Sugar(),
	}
}

func (l *AuditLogger) Log(action, userID, resource string, details map[string]interface{}) {
	l.Infow(action,
		"user_id", userID,
		"resource", resource,
		"details", details,
	)
}

func (l *AuditLogger) With(fields ...zap.Field) *AuditLogger {
	return &AuditLogger{l.SugaredLogger.With(fields...)}
}
```

- [ ] **步骤 2：创建 error.go.tmpl**

```go
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/sugar"
)

type ErrorLogger struct {
	*zap.SugaredLogger
}

func NewErrorLogger(logger *zap.Logger) *ErrorLogger {
	return &ErrorLogger{
		logger.Named("error").Sugar(),
	}
}

func (l *ErrorLogger) Log(err error, msg string, keysAndValues ...interface{}) {
	fields := []interface{}{"error", err.Error()}
	fields = append(fields, keysAndValues...)
	l.Errorw(msg, fields...)
}

func (l *ErrorLogger) WithStack(err error) *zap.SugaredLogger {
	return l.Desugar().With(zap.Stack("stack")).Sugar()
}

func (l *ErrorLogger) With(fields ...zap.Field) *ErrorLogger {
	return &ErrorLogger{l.SugaredLogger.With(fields...)}
}
```

- [ ] **步骤 3：创建 mq.go.tmpl**

```go
package logger

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
	mu        sync.Mutex
	client    MQPusher
	topic     string
	buffer    []struct{topic string; data []byte}
	batchSize int
	interval  time.Duration
	flushCh   chan struct{}
	doneCh    chan struct{}
	closed    bool
	wg        sync.WaitGroup
}

func NewAsyncMQPusher(client MQPusher, topic string, batchSize int, interval time.Duration) *AsyncMQPusher {
	p := &AsyncMQPusher{
		client:    client,
		topic:     topic,
		batchSize: batchSize,
		interval:  interval,
		flushCh:   make(chan struct{}, 1),
		doneCh:   make(chan struct{}, 1),
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
	p.buffer = append(p.buffer, struct{topic string; data []byte}{topic, data})
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
		case <-p.doneCh:
			p.mu.Lock()
			p.doFlush()
			p.mu.Unlock()
			return
		}
	}
}

func (p *AsyncMQPusher) doFlush() {
	if len(p.buffer) == 0 {
		return
	}
	for _, item := range p.buffer {
		_ = p.client.Push(item.topic, item.data)
	}
	p.buffer = p.buffer[:0]
}

func (p *AsyncMQPusher) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.flushCh <- struct{}{}
	p.doneCh <- struct{}{}
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

- [ ] **步骤 4：创建 mq_kafka.go.tmpl**

```go
package logger

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Shopify/sarama"
)

type KafkaPusher struct {
	producer sarama.SyncProducer
	topic    string
	closed   atomic.Bool
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
	if p.closed.Load() {
		return nil
	}
	if topic == "" {
		topic = p.topic
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
	p.closed.Store(true)
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

- [ ] **步骤 5：移动文件并提交**

```bash
# 移动文件
mv templates/pkg/log/*.tmpl templates/pkg/logger/

# 删除空目录
rmdir templates/pkg/log

# 提交
git add templates/pkg/logger/ templates/pkg/log/ internal/...
git commit -m "refactor: 合并日志目录到 pkg/logger"
```

---

### 任务 2：更新代码生成逻辑

**文件：**
- 修改：`internal/cli/microapp_new.go:1306`
- 修改：`internal/generator/template_engine.go:68`
- 修改：`internal/generator/script_center.go:34`
- 修改：`internal/generator/scriptcenter/generator.go:47`
- 修改：`internal/generator/microservice.go:162`

- [ ] **步骤 1：修改 microapp_new.go**

将第 1306 行从 `templates/pkg/log` 改为 `templates/pkg/logger`

- [ ] **步骤 2：修改 template_engine.go**

将第 68 行从 `"pkg/log"` 改为 `"pkg/logger"`

- [ ] **步骤 3：修改 script_center.go**

将第 34 行从 `"pkg/log"` 改为 `"pkg/logger"`

- [ ] **步骤 4：修改 scriptcenter/generator.go**

将第 47 行从 `"pkg/log"` 改为 `"pkg/logger"`

- [ ] **步骤 5：修改 microservice.go**

将第 162 行从 `"pkg/log"` 改为 `"pkg/logger"`

- [ ] **步骤 6：提交**

```bash
git commit -m "refactor: 更新代码生成逻辑使用 pkg/logger"
```

---

### 任务 3：BFF 层日志集成

**文件：**
- 修改：`templates/micro-app/bff/handler/handler.go.tmpl`
- 修改：`templates/micro-app/bff/middleware/gin_middleware.go.tmpl`
- 修改：`templates/micro-app/bff/main/gin_main.go.tmpl`

- [ ] **步骤 1：修改 handler.go.tmpl**

在 handler 中添加日志调用：
- 请求入口记录 Access 日志
- 业务处理记录 Business 日志
- 错误记录 Error 日志

示例代码：
```go
// 在 handler 函数中添加
logger := log.NewLogger(cfg)
defer logger.Sync()

// 业务处理
logger.Business.Infow("handle request", "method", c.Request.Method, "path", c.Request.URL.Path)

// 错误处理
logger.Error.Errorw("request failed", "error", err.Error())
```

- [ ] **步骤 2：修改 gin_middleware.go.tmpl**

添加日志中间件，记录请求日志和错误

- [ ] **步骤 3：修改 gin_main.go.tmpl**

在 main 函数中初始化日志，记录启动信息

- [ ] **步骤 4：提交**

---

### 任务 4：SRV 层日志集成

**文件：**
- 修改：`templates/micro-app/srv/handler/handler.go.tmpl`
- 修改：`templates/micro-app/srv/interceptor/kitex_interceptor.go.tmpl`
- 修改：`templates/micro-app/srv/main/main_*.go.tmpl`
- 修改：`templates/micro-app/srv/repo/repository.go.tmpl`

- [ ] **步骤 1：修改 handler.go.tmpl**

添加业务日志和错误日志

- [ ] **步骤 2：修改 kitex_interceptor.go.tmpl**

添加拦截器日志

- [ ] **步骤 3：修改 main_*.go.tmpl**

添加启动日志

- [ ] **步骤 4：修改 repository.go.tmpl**

添加数据操作日志

- [ ] **步骤 5：提交**

---

### 任务 5：生成日志使用文档

**文件：**
- 创建：`docs/logging-usage.md`

- [ ] **步骤 1：创建文档**

```markdown
# 日志使用指南

## 概述

本文档说明如何在生成的 BFF 和 SRV 项目中使用日志功能。

## 日志目录结构

```
pkg/logger/
├── logger.go           # 主日志
├── config.go          # 配置
├── business.go        # 业务日志
├── access.go         # 访问日志
├── audit.go          # 审计日志
├── error.go          # 错误日志
├── mq.go            # MQ 推送
├── mq_kafka.go      # Kafka 实现
└── config.yaml      # 配置文件
```

## 日志场景

| 场景 | 用途 | 方法 |
|------|------|------|
| Business | 业务逻辑 | Infow, Errorw, Warnw, Debugw |
| Access | 访问记录 | Log |
| Audit | 审计追踪 | Log |
| Error | 错误追踪 | Log, Errorw |

## BFF 层日志使用

### Handler

```go
logger.Business.Infow("业务处理", "param", param)
```

### Middleware

```go
logger.Access.Log(method, path, clientIP, ...)
```

## SRV 层日志使用

### Handler

```go
logger.Business.Infow("处理请求", "req", req)
```

### Interceptor

```go
logger.Access.Log(method, service, ...)
```

## MQ 推送配置

在 config.yaml 中配置：

```yaml
mq:
  enabled: true
  type: kafka
  brokers:
    - localhost:9092
  topic: app-logs
  async: true
  batch_size: 100
  flush_interval: 3s
```
```

- [ ] **步骤 2：提交**

```bash
git add docs/logging-usage.md
git commit -m "docs: 添加日志使用指南"
```
