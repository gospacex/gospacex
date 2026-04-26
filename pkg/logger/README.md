# Enterprise Logger

基于 uber-go/zap 的企业级日志库，支持多场景、日志分级、调用链追踪、MQ备份。

## Features

- Multi-scene logging (Business/Access/Audit/Error)
- Reliability levels (Vital/Important/Normal)
- Distributed tracing via OpenTelemetry
- Kafka async backup
- Prometheus metrics
- Dynamic level adjustment

## Features Detail

### 多场景日志

| Scene | Reliability | Description |
|-------|-------------|-------------|
| Business | Important | 业务日志，异步批量写入Kafka |
| Access | Important | 访问日志，异步批量写入Kafka |
| Audit | Vital | 审计日志，双buffer保证不丢日志 |
| Error | Important | 错误日志，带令牌桶限流 |

### 可靠性等级

- **Vital**: 双buffer写盘，Fsync同步落盘，SSD推荐
- **Important**: 异步批量写入Kafka，P99 < 1ms
- **Normal**: 异步批量，采样降级

## Quick Start

```go
package main

import (
    "gpx/pkg/logger"
)

func main() {
    l, err := logger.NewFromYAML("config.yaml")
    if err != nil {
        panic(err)
    }
    defer l.Close()

    l.Business.Infow("order created", "order_id", "ORD-001")
    l.Access.Infow("request received", "path", "/api/orders")
    l.Audit.Logf("create", "order", "order_id=%s", "ORD-001")
    l.Error.Errorw("connection failed", "error", "timeout")
}
```

### YAML Configuration

```yaml
env: "prod"
level: "info"
service_name: "order-service"
topic_prefix: "app-logs"

sampling:
  warn: 1.0
  info:
    initial: 100
    thereafter: 200
    tick: "1s"
  debug: 0.0

rate_limit:
  error:
    rate: 100
    burst: 200
    overflow_action: "aggregate"

rotation:
  enabled: true
  max_age_days: 7

vital:
  buffer_size: 10000
  sync_interval: "1s"
  fsync_timeout: "50ms"
  fallback_on_full: true

storage:
  log_level: "debug"
  log_body: false
  log_slow_threshold: "100ms"

mq:
  brokers:
    - "localhost:9092"
  partition_count: 64
  batch_size: 100
  flush_interval: "1s"

tracing:
  enabled: true
  endpoint: "localhost:4318"
  sample_rate: 1.0

metrics:
  enabled: true
```

## API Reference

### Logger

```go
l, _ := logger.NewFromYAML("config.yaml")

// Business logger
l.Business.Infow("message", "key", "value")
l.Business.Debugw("debug message", "key", "value")

// Access logger
l.Access.Infow("GET /api/users", "status", 200, "latency_ms", 45)

// Audit logger
l.Audit.Log(&AuditRecord{
    Action: "create",
    Resource: "order",
    UserID: "user-123",
    ResourceID: "ORD-001",
})

// Error logger with rate limiting
l.Error.Errorw("failed to connect", "error", err.Error())
```

### Dynamic Level Adjustment

```go
handler := logger.NewLevelHandler(levelManager)

// GET /debug/loglevel - 获取当前日志级别
// POST /debug/loglevel?level=debug - 设置所有场景日志级别
// POST /debug/loglevel?scene=business&level=warn - 设置指定场景日志级别
```

### Health Check

```go
// GET /debug/log/health
// Response: { "status": "healthy", "buffer_usage": {...}, "mq_status": "healthy" }
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Logger                                  │
├──────────────┬──────────────┬──────────────┬───────────────┤
│   Business   │   Access     │    Audit     │    Error      │
│  (Important) │  (Important) │    (Vital)   │  (Important)  │
├──────────────┴──────────────┴──────────────┴───────────────┤
│                   AsyncBatch (Important)                     │
├─────────────────────────────────────────────────────────────┤
│               DoubleBuffer (Vital, Fsync)                   │
├─────────────────────────────────────────────────────────────┤
│                      Kafka Producer                          │
└─────────────────────────────────────────────────────────────┘
```

## License

MIT
