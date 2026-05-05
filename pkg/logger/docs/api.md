# API Examples

## Initialization

### From YAML File

```go
l, err := logger.NewFromYAML("config.yaml")
if err != nil {
    panic(err)
}
defer l.Close()
```

### From Config Struct

```go
cfg := logger.DefaultConfig()
cfg.ServiceName = "order-service"
cfg.Level = "info"
cfg.MQ.Brokers = []string{"localhost:9092"}

l, err := logger.New(cfg)
if err != nil {
    panic(err)
}
```

## Business Logger

业务日志，使用 Important 可靠性级别，异步批量写入 Kafka。

```go
// 基础用法
l.Business.Infow("order created",
    "order_id", "ORD-001",
    "user_id", "USR-123",
    "amount", 99.99,
)

// Debug 级别
l.Business.Debugw("cache miss", "key", "product:123")

// 带字段的 Logger
enrichedLog := l.Business.With(zap.String("tenant_id", "tenant-456"))
enrichedLog.Infow("tenant operation", "operation", "create")
```

## Access Logger

访问日志，记录 HTTP 请求。

```go
// 记录请求
l.Access.Infow("GET /api/orders",
    "method", "GET",
    "path", "/api/orders",
    "status", 200,
    "latency_ms", 45,
    "client_ip", "192.168.1.1",
)

// POST 请求带 body 大小
l.Access.Infow("POST /api/orders",
    "method", "POST",
    "path", "/api/orders",
    "status", 201,
    "body_size", 1024,
)
```

## Audit Logger

审计日志，使用 Vital 可靠性级别，双 buffer 保证不丢日志。

### Basic Usage

```go
record := &logger.AuditRecord{
    Action: "create",
    Resource: "order",
    UserID: "USR-123",
    ResourceID: "ORD-001",
    Details: map[string]any{
        "sku": "SKU-001",
        "quantity": 2,
    },
}
l.Audit.Log(record)
```

### With Context (Auto Trace Injection)

```go
ctx := context.WithValue(context.Background(), "trace_id", "abc123")
ctx = context.WithValue(ctx, "span_id", "span-456")

record := &logger.AuditRecord{
    Action: "update",
    Resource: "order",
    UserID: "USR-123",
    ResourceID: "ORD-001",
}
l.Audit.LogContext(ctx, record)
// record.TraceID = "abc123"
// record.SpanID = "span-456"
```

### Convenience Method

```go
l.Audit.Logf("delete", "order", "order_id=%s, reason=%s", "ORD-001", "timeout")
```

### JSON Structure

```json
{
  "action": "create",
  "resource": "order",
  "user_id": "USR-123",
  "resource_id": "ORD-001",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "details": {
    "message": "order created successfully"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Error Logger

错误日志，带令牌桶限流，防止错误风暴。

```go
// 基础错误日志
l.Error.Errorw("connection failed",
    "error", "timeout",
    "host", "db-master:5432",
)

// 带 panic 级别
l.Error.Panicw("unrecoverable error", "reason", "corrupted data")
```

### Rate Limiting Behavior

当 error 日志超过限流阈值时:

```go
// overflow_action: "aggregate" - 聚合为单条日志
// overflow_action: "drop" - 直接丢弃
// overflow_action: "warn" - 记录警告后丢弃
```

## Storage Logger

独立日志级别配置的存储层日志。

```go
// 获取 Storage logger
storageLog := l.Storage()

// 独立设置 debug 级别
storageLog.Debugw("slow query detected",
    "sql", "SELECT * FROM orders WHERE...",
    "duration_ms", 150,
)

// 记录查询结果
storageLog.Infow("query executed",
    "table", "orders",
    "rows_affected", 5,
)
```

## Dynamic Level Adjustment

### HTTP Handler

```go
levelMgr := logger.NewLevelManager()
handler := logger.NewLevelHandler(levelMgr)

// 挂载到 Gin
r := gin.Default()
r.GET("/debug/loglevel", handler.HandleGetLevel)
r.POST("/debug/loglevel", handler.HandlePostLevel)
```

### GET /debug/loglevel

```bash
curl http://localhost:8080/debug/loglevel
```

Response:
```json
{
  "business": "info",
  "access": "info",
  "audit": "info",
  "error": "info"
}
```

### POST /debug/loglevel - Set All Scenes

```bash
curl -X POST "http://localhost:8080/debug/loglevel?level=debug"
```

Response:
```json
{
  "status": "ok",
  "level": "debug"
}
```

### POST /debug/loglevel - Set Specific Scene

```bash
curl -X POST "http://localhost:8080/debug/loglevel?scene=business&level=warn"
```

Response:
```json
{
  "status": "ok",
  "scene": "business",
  "level": "warn"
}
```

### Programmatic Usage

```go
levelMgr := logger.NewLevelManager()

// 设置单个场景
levelMgr.SetLevel("business", zapcore.WarnLevel)

// 获取单个场景
lvl := levelMgr.GetLevel("business")

// 批量设置
levelMgr.SetLevels(map[string]zapcore.Level{
    "business": zapcore.InfoLevel,
    "access":   zapcore.InfoLevel,
    "audit":    zapcore.WarnLevel,
    "error":    zapcore.ErrorLevel,
})
```

## Health Check Endpoint

### Setup

```go
import "gpx/pkg/logger/health"

// 注册 MQ Producer
health.RegisterMQProducer(kafkaProducer)

// 定期更新 buffer 使用率
health.UpdateBufferStatus("business", 0.45)
health.UpdateBufferStatus("audit", 0.12)

// 挂载 health handler
r.GET("/debug/log/health", health.HandleHealth)
```

### GET /debug/log/health

Healthy response (200):
```json
{
  "status": "healthy",
  "buffer_usage": {
    "business": 0.45,
    "access": 0.30,
    "audit": 0.12,
    "error": 0.55
  },
  "mq_status": "healthy"
}
```

Degraded response (503):
```json
{
  "status": "degraded",
  "buffer_usage": {
    "business": 0.95,
    "access": 0.30,
    "audit": 0.12,
    "error": 0.55
  },
  "mq_status": "healthy",
  "degraded_reasons": [
    "buffer_usage for business exceeds 0.9"
  ]
}
```

Unhealthy response (503):
```json
{
  "status": "degraded",
  "buffer_usage": {
    "business": 0.45,
    "access": 0.30,
    "audit": 0.12,
    "error": 0.55
  },
  "mq_status": "unhealthy",
  "degraded_reasons": [
    "mq producer is unhealthy"
  ]
}
```

## With Context (Logger Cloning)

```go
// 创建带固定字段的新 Logger
enriched := l.With(
    zap.String("service", "order-service"),
    zap.String("version", "v1.2.3"),
)

// 使用 enrichd 日志器
enriched.Business.Infow("order processed", "order_id", "ORD-001")
enriched.Access.Infow("request completed", "path", "/api/orders")
```

## Gin Integration

```go
import (
    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin/gin/otelgin"
    "gpx/pkg/logger"
    "gpx/pkg/logger/tracing"
)

r := gin.Default()

// OpenTelemetry tracing middleware
r.Use(otelgin.Middleware("order-service"))

// Logger middleware for access log
r.Use(func(c *gin.Context) {
    start := time.Now()
    c.Next()
    latency := time.Since(start)

    l.Access.Infow("request",
        "method", c.Request.Method,
        "path", c.Request.URL.Path,
        "status", c.Writer.Status(),
        "latency_ms", latency.Milliseconds(),
    )
})
```

## gRPC Integration

```go
import (
    "google.golang.org/grpc"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
    "gpx/pkg/logger/tracing"
)

// Server interceptor
serverOpts := []grpc.ServerOption{
    grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
    grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
}

// Client interceptor
conn, err := grpc.Dial(addr,
    grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
)
```

## Prometheus Metrics

### Endpoint

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "gpx/pkg/logger/metrics"
)

// 挂载 metrics handler
r.GET("/metrics", promhttp.Handler())
```

### Available Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| log_entries_total | Counter | scene, level | 日志条目总数 |
| log_level_total | Counter | scene, level | 各级别日志数 |
| mq_push_total | Counter | status | MQ 推送次数 |
| fsync_latency_seconds | Histogram | scene | Fsync 延迟 |
| buffer_usage_ratio | Gauge | scene | Buffer 使用率 |

### Example Prometheus Query

```promql
# QPS by scene
rate(log_entries_total[5m])

# Error rate
rate(log_level_total{level="error"}[5m])

# P99 fsync latency
histogram_quantile(0.99, rate(fsync_latency_seconds_bucket[5m]))
```
