# 企业级日志库设计与集成

## 概述

基于 uber-go/zap 封装的企业级日志库，作为脚手架的一部分自动生成到微服务项目中。支持多场景日志、多可靠性级别、MQ 推送、调用链集成、BFF/SRV 层差异化集成。

## 核心设计理念

```
┌─────────────────────────────────────────────────────────────────────┐
│                    可靠性分级设计                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Vital（审计、订单、支付）                                         │
│   ├─ 同步刷盘，每次 write 都 fsync                                 │
│   ├─ RingBuffer 缓冲，buffer 满或超时触发同步                      │
│   ├─ MQ 异步备份，不影响主路径                                      │
│   └─ 丢失代价：合规风险 + 资损                                     │
│                                                                     │
│   Important（业务、访问、错误）                                      │
│   ├─ 异步批量，定期 flush                                          │
│   └─ 丢失代价：影响排查、可补录                                    │
│                                                                     │
│   Normal（调试、追踪）                                              │
│   ├─ 采样或不落盘                                                   │
│   └─ 丢失代价：几乎为零                                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 功能列表

| 功能 | 描述 |
|------|------|
| 多环境支持 | dev/staging/prod 配置 |
| 日志轮转 | 按日期分割，自动清理 7 天前日志 |
| 结构化日志 | 支持业务字段 |
| 错误堆栈追踪 | 完整堆栈信息 |
| 日志采样 | Error/Warn 全量，Info 采样，Debug 不记录 |
| 多输出 | 同时写文件 + stdout |
| Prometheus 指标 | 日志级别计数 + 写入延迟 |
| **多可靠性级别** | Vital / Important / Normal 三级 |
| **MQ 推送** | Kafka 异步批量推送（可选） |
| **调用链集成** | Jaeger trace_id / span_id 自动注入 |
| **多场景日志** | Business / Access / Audit / Error 四类 |
| **BFF/SRV 集成** | 各层目录集成日志调用，差异化配置 |

## 日志场景

| 场景 | 用途 | 可靠性级别 | 使用层级 |
|------|------|-----------|---------|
| Business | 业务逻辑日志 | Important | BFF/SRV handler、main、repo |
| Access | 访问记录日志 | Important | BFF/SRV middleware、interceptor |
| Audit | 审计追踪日志 | **Vital** | BFF/SRV handler |
| Error | 错误追踪日志 | Important | 所有层级 |

## 目录结构

```
pkg/logger/
├── logger.go           # 主 Logger 封装
├── config.go          # 配置定义 + 加载
├── rotation.go        # 日志轮转
├── cleaner.go         # 过期清理
├── sampler.go         # 采样配置
├── metrics.go         # Prometheus 指标
├── core/
│   ├── vital.go       # Vital 级别 Core（RingBuffer + Sync）
│   ├── important.go    # Important 级别 Core（Async Batch）
│   └── normal.go      # Normal 级别 Core（Sampler）
├── scene/
│   ├── business.go    # 业务日志
│   ├── access.go      # 访问日志
│   ├── audit.go       # 审计日志
│   └── error.go       # 错误日志
├── mq/
│   ├── producer.go    # MQ 生产者接口
│   └── kafka.go       # Kafka 实现
├── tracing/
│   └── jaeger.go      # Jaeger 集成
└── context.go         # Context 传递
```

## 配置结构 (config/log.yaml)

```yaml
env: dev
level: info
service_name: bff-order    # 生成时注入，各服务唯一

sampling:
  initial: 100
  thereafter: 200
  tick: 1s

rotation:
  enabled: true
  max_age_days: 7

vital:
  buffer_size: 1000        # RingBuffer 大小
  sync_interval: 1s        # 同步间隔
  sync_timeout: 5s          # 同步超时

output:
  file: ./logs/app.log
  stdout: true

prometheus:
  enabled: true
  namespace: app
  subsystem: log

mq:
  enabled: true
  type: kafka
  brokers:
    - localhost:9092
  topic_prefix: app-logs
  partition_key: trace_id   # 一致性哈希 key
  async: true
  batch_size: 100
  flush_interval: 3s

tracing:
  enabled: true
  service_name: bff-order    # 生成时注入
  agent_host: localhost      # 边车模式
  agent_port: 6831          # UDP
  sampler:
    type: rate_limiting     # const / probabilistic / rate_limiting / adaptive
    param: 100               # 每秒最大 traces 数（基于配置）
```

## 调用链集成

### 设计原则

```
┌─────────────────────────────────────────────────────────────────────┐
│                    BFF/SRV 调用链差异                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   BFF（入口层）                                                     │
│   ├─ 依赖 OpenTelemetry + otelgin                                  │
│   ├─ 通过 otelgin.Middleware() 生成/接收 trace                     │
│   ├─ 通过 HTTP Header 透传: traceparent                           │
│   ├─ 接口级别开关：通过 otelgin.Middleware() 按路由开启             │
│   └─ Sidecar 模式连接 Jaeger Agent                                │
│                                                                     │
│   SRV（微服务层）                                                   │
│   ├─ 依赖 OpenTelemetry + otelgrpc                                │
│   ├─ 从 gRPC Metadata 接收 trace context                          │
│   ├─ otelgrpc.UnaryServerInterceptor() 自动创建 child span        │
│   └─ 通过 gRPC Metadata 透传给下游 SRV                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### OpenTelemetry 初始化

```go
// BFF/pkg/tracing/otel.go

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
)

func InitTracer(serviceName string, agentHost string, agentPort int) (func(), error) {
    exp, err := jaeger.New(
        jaeger.WithAgentEndpoint(agentHost, agentPort, jaeger.WithUDP()),
    )
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exp),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )

    otel.SetTracerProvider(tp)

    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return func() { tp.Shutdown(context.Background()) }, nil
}
```

### BFF 中间件

```go
// BFF/main.go

func main() {
    closer, _ := InitTracer("bff-order", "localhost", 6831)
    defer closer()

    r := gin.New()

    // 全局中间件（无调用链）
    r.Use(RecoveryMiddleware(logger))
    r.Use(LoggingMiddleware(logger))

    // 健康检查、监控等不需要调用链
    r.GET("/health", HealthHandler.Check)
    r.GET("/metrics", MetricsHandler.Report)
    r.GET("/ready", ReadyHandler.Check)

    // 需要调用链的接口
    traced := r.Group("")
    traced.Use(otelgin.Middleware("bff-order"))
    {
        traced.POST("/api/orders", OrderHandler.CreateOrder)
        traced.GET("/api/orders/:id", OrderHandler.GetOrder)
        traced.POST("/api/payments", PaymentHandler.Pay)
        traced.POST("/api/refunds", RefundHandler.Create)
        traced.POST("/api/users", UserHandler.Create)
        traced.GET("/api/users/:id", UserHandler.Get)
    }

    r.Run(cfg.Addr())
}
```

### SRV Interceptor

```go
// SRV/main.go

func main() {
    closer, _ := InitTracer("srv-order", "localhost", 6831)
    defer closer()

    s := grpc.NewServer(
        grpc.UnaryInterceptor(ChainInterceptors(
            otelgrpc.UnaryServerInterceptor(),
            LoggingInterceptor(logger),
            AuthInterceptor(),
            RateLimitInterceptor(),
        )),
    )

    pb.RegisterOrderServiceServer(s, &OrderHandler{log: log})
    s.Serve(lis)
}

func ChainInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{},
               info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) {
        return handler(ctx, req)
    }
}
```

### 接口级别调用链开关

BFF 层通过路由分组实现接口级别调用链开关：

```
┌─────────────────────────────────────────────────────────────────────┐
│                    路由分组实现调用链开关                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   // 不开启调用链的接口（健康检查、监控等）                         │
│   r.GET("/health", HealthHandler.Check)                            │
│   r.GET("/metrics", MetricsHandler.Report)                         │
│                                                                     │
│   // 需要调用链的接口（通过中间件组）                               │
│   traced := r.Group("")                                            │
│   traced.Use(otelgin.Middleware("bff-order"))                      │
│   {                                                                │
│       traced.POST("/api/orders", OrderHandler.CreateOrder)         │
│       traced.GET("/api/orders/:id", OrderHandler.GetOrder)         │
│       traced.POST("/api/payments", PaymentHandler.Pay)             │
│   }                                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### SRV Trace 接收与传播

```go
// SRV/interceptor/trace.go
func TraceInterceptor(tracer *jaeger.Tracer) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{},
               info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) {

        // 从 gRPC Metadata 提取 Jaeger context
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            md = metadata.New(nil)
        }

        // 提取 span context
        spanCtx, err := tracer.Extract(jaeger.TracingHeaderCarrier(md))
        if err != nil {
            spanCtx = nil
        }

        // 创建 child span
        span := tracer.StartSpan(info.FullMethod, jaeger.ChildOf(spanCtx))
        defer span.Finish()

        // 提取关键字段用于日志
        sc := span.Context().(jaeger.SpanContext)
        traceID := fmt.Sprintf("%x", sc.TraceID())
        spanID = fmt.Sprintf("%x", sc.SpanID())
        parentID := fmt.Sprintf("%x", sc.ParentID())

        // 透传给下游（注入到新的 Metadata）
        newMd := md.Copy()
        tracer.Inject(span.Context(), jaeger.TracingHeaderCarrier(newMd))
        ctx = metadata.NewOutgoingContext(ctx, newMd)

        // 放入 context
        ctx = context.WithValue(ctx, "trace_id", traceID)
        ctx = context.WithValue(ctx, "span_id", spanID)
        ctx = context.WithValue(ctx, "parent_id", parentID)
        ctx = context.WithValue(ctx, "span", span)

        return handler(ctx, req)
    }
}
```

### 中间件/拦截器顺序

```
┌─────────────────────────────────────────────────────────────────────┐
│                      执行顺序（洋葱模型）                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   请求进入                                                          │
│       │                                                             │
│       ▼                                                             │
│   1. TraceMiddleware/TraceInterceptor                              │
│      - BFF: 生成 trace_id                                          │
│      - SRV: 接收/创建 child span                                   │
│       │                                                             │
│       ▼                                                             │
│   2. LoggingMiddleware/LoggingInterceptor                         │
│      - 记录 Access Log（开始）                                     │
│       │                                                             │
│       ▼                                                             │
│   3. AuthMiddleware/AuthInterceptor                                │
│      - 认证检查                                                     │
│       │                                                             │
│       ▼                                                             │
│   4. RateLimitMiddleware/RateLimitInterceptor                      │
│      - 限流检查                                                     │
│       │                                                             │
│       ▼                                                             │
│   5. Handler                                                       │
│      - 业务逻辑                                                     │
│      - Business / Audit / Error Log                               │
│       │                                                             │
│       ▼                                                             │
│   响应返回（逆序）                                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## BFF 层日志集成

| 目录 | 日志场景 | 可靠性 | 用途 |
|------|---------|--------|------|
| handler | Business + Audit + Error | Vital/Imp | 请求处理、业务逻辑、审计追踪 |
| middleware | Access + Error | Imp | 请求日志、错误处理 |
| main | Business + Error | Imp | 启动日志、运行状态 |

### handler 示例

```go
// BFF/handler/order.go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    req := &CreateOrderReq{}
    if err := c.BindJSON(req); err != nil {
        h.logger.Error.WithStack(err).Errorw("bind failed",
            "error", err.Error())
        return
    }

    // Business 日志 - Important 级别
    h.logger.Business.Infow("create order start",
        "user_id", req.UserID,
        "amount", req.Amount)

    // Audit 日志 - Vital 级别（订单必须可靠记录）
    h.logger.Audit.Log("create", req.UserID, "order", map[string]any{
        "order_id": generateOrderID(),
        "amount":   req.Amount,
        "items":    req.Items,
    })

    order, err := h.srv.Create(c.Request.Context(), req)
    if err != nil {
        h.logger.Error.Errorw("create order failed",
            "error", err.Error(),
            "user_id", req.UserID)
        return
    }

    h.logger.Business.Infow("create order success", "order_id", order.ID)
}
```

### middleware 示例

```go
// BFF/middleware/logging.go
func LoggingMiddleware(logger *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        traceID := c.Get("trace_id")
        spanID := c.Get("span_id")

        // Access 日志 - Important 级别
        logger.Access.Infow("request start",
            "trace_id", traceID,
            "span_id", spanID,
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "client_ip", c.ClientIP())

        c.Next()

        // Access 日志 - 结束
        logger.Access.Infow("request end",
            "trace_id", traceID,
            "span_id", spanID,
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "status", c.Writer.Status(),
            "latency", time.Since(start).String())
    }
}
```

## SRV 层日志集成

| 目录 | 日志场景 | 可靠性 | 用途 |
|------|---------|--------|------|
| handler | Business + Error | Imp | 业务处理、错误记录 |
| interceptor | Access + Error | Imp | 拦截日志、错误追踪 |
| main | Business + Error | Imp | 启动日志、运行状态 |
| repo | Business + Error | Imp | 数据操作日志 |

### handler 示例

```go
// SRV/handler/order.go
func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
    span := ctx.Value("span").(jaeger.Span)
    traceID := ctx.Value("trace_id").(string)

    h.logger.Business.Infow("handle create order",
        "trace_id", traceID,
        "user_id", req.UserID,
        "amount", req.Amount)

    order, err := h.repo.Create(ctx, &Order{
        UserID: req.UserID,
        Amount: req.Amount,
    })
    if err != nil {
        h.logger.Error.Errorw("create order failed",
            "trace_id", traceID,
            "error", err.Error())
        return nil, err
    }

    h.logger.Business.Infow("create order success",
        "trace_id", traceID,
        "order_id", order.ID)

    return &pb.CreateOrderResp{OrderID: order.ID}, nil
}
```

### interceptor 示例

```go
// SRV/interceptor/logging.go
func LoggingInterceptor(logger *logger.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{},
               info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

        start := time.Now()
        traceID := ctx.Value("trace_id").(string)
        spanID := ctx.Value("span_id").(string)
        parentID := ctx.Value("parent_id").(string)

        // Access 日志 - Important 级别
        logger.Access.Infow("gRPC request",
            "trace_id", traceID,
            "span_id", spanID,
            "parent_id", parentID,
            "method", info.FullMethod)

        resp, err = handler(ctx, req)

        if err != nil {
            logger.Error.Errorw("gRPC call failed",
                "trace_id", traceID,
                "method", info.FullMethod,
                "error", err.Error())
        }

        logger.Access.Infow("gRPC response",
            "trace_id", traceID,
            "span_id", spanID,
            "method", info.FullMethod,
            "latency", time.Since(start).String(),
            "status", status.Code(err).String())

        return
    }
}
```

### repo 示例

```go
// SRV/repo/order.go
type OrderRepo struct {
    db     *gorm.DB
    logger *logger.Logger
}

func (r *OrderRepo) Create(ctx context.Context, order *Order) error {
    start := time.Now()
    traceID := ctx.Value("trace_id").(string)

    r.logger.Business.Infow("DB create order",
        "trace_id", traceID,
        "table", "orders",
        "order_id", order.ID)

    if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
        r.logger.Error.Errorw("DB create order failed",
            "trace_id", traceID,
            "error", err.Error(),
            "order_id", order.ID)
        return err
    }

    r.logger.Business.Infow("DB create order success",
        "trace_id", traceID,
        "table", "orders",
        "order_id", order.ID,
        "latency", time.Since(start).String())

    return nil
}
```

## MQ 顺序保证

### 设计方案：一致性哈希

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MQ 消息顺序保证                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   生产端：                                                          │
│   ├─ 按 trace_id 做一致性哈希                                       │
│   ├─ 相同 trace_id → 固定 partition                                │
│   └─ 保证同一 trace 的消息有序                                       │
│                                                                     │
│   消费端：                                                          │
│   ├─ 按 partition 消费                                              │
│   └─ 按 timestamp 或序列号排序                                      │
│                                                                     │
│   部署架构：                                                        │
│                                                                     │
│   ┌──────────┐     ┌──────────────────────────────────────────┐    │
│   │ Producer │────▶│            Kafka Cluster                  │    │
│   │          │     │  ┌─────────┐ ┌─────────┐ ┌─────────┐      │    │
│   │ hash(key)│     │  │ Parit-0 │ │ Parit-1 │ │ Parit-2 │      │    │
│   └──────────┘     │  └────┬────┘ └────┬────┘ └────┬────┘      │    │
│                    └───────┼──────────┼──────────┼────────────┘    │
│   ┌──────────┐              │          │          │               │
│   │Consumer-0│◀─────────────┘          │          │               │
│   │(Parit-0) │                         │          │               │
│   └──────────┘     ┌──────────────────┘          │               │
│                    │                             │               │
│   ┌──────────┐     │     ┌──────────────────────┘               │
│   │Consumer-1│◀────┘     │                                   │
│   │(Parit-1) │            │                                   │
│   └──────────┘            │                                   │
│                           │                                   │
│   ┌──────────┐            │                                   │
│   │Consumer-2│◀───────────┘                                   │
│   │(Parit-2) │                                                │
│   └──────────┘                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Kafka Producer 配置

```go
type KafkaProducer struct {
    producer sarama.AsyncProducer
    topic    string
    hashFunc func(key string) int32  // 一致性哈希
}

func (p *KafkaProducer) Send(scene string, key string, value []byte) error {
    partition := p.hashFunc(key) % int32(p.partitionCount)

    msg := &sarama.ProducerMessage{
        Topic:     fmt.Sprintf("%s-%s", p.topicPrefix, scene),
        Key:       sarama.StringEncoder(key),
        Value:     sarama.ByteEncoder(value),
        Partition: partition,
        Timestamp: time.Now(),
    }

    p.producer.Input() <- msg
    return nil
}
```

## 实现要点

### 1. Vital 场景：RingBuffer + 同步刷盘

```go
type VitalWriter struct {
    buffer     *ringbuffer.RingBuffer
    file       *os.File
    bufferSize int
    syncTick   time.Duration
    lastSync   time.Time
    mu         sync.Mutex
}

func (w *VitalWriter) Write(p []byte) (n int, err error) {
    // 1. 先写入 buffer
    w.buffer.Write(p)

    // 2. buffer 满 或 超时 → 触发同步
    if w.buffer.Length() >= w.bufferSize || time.Since(w.lastSync) >= w.syncTick {
        if err := w.Sync(); err != nil {
            return 0, err
        }
    }

    return len(p), nil
}

func (w *VitalWriter) Sync() error {
    w.mu.Lock()
    defer w.mu.Unlock()

    // 强制刷盘
    if err := w.file.Sync(); err != nil {
        return err
    }

    // 清空 buffer
    w.buffer.Reset()
    w.lastSync = time.Now()

    return nil
}
```

### 2. 日志轮转

- 实现自定义 `zapcore.WriteSyncer`
- 每天凌晨检查或创建新文件（按日期命名）
- 文件名格式：`app-2024-04-25.log`

### 3. 过期清理

- 启动时清理过期文件
- 每 6 小时定时清理
- 按文件修改时间判断

### 4. Jaeger 采样策略

```go
func newSampler(cfg TracingSamplerConfig) jaeger.Sampler {
    switch cfg.Type {
    case "const":
        if cfg.Param == 1 {
            return jaeger.NewConstSampler(true)
        }
        return jaeger.NewConstSampler(false)

    case "probabilistic":
        return jaeger.NewProbabilisticSampler(cfg.Param)

    case "rate_limiting":
        return jaeger.NewRateLimitingSampler(float64(cfg.Param))

    case "adaptive":
        return jaeger.NewAdaptiveSampler(nil, cfg.Param)

    default:
        return jaeger.NewConstSampler(true)
    }
}
```

### 5. Prometheus 指标

- Counter: `log_messages_total{level,type,scene}` - 各级别日志计数
- Histogram: `log_write_duration_seconds{scene}` - 写入延迟
- Counter: `log_vital_sync_total` - Vital 场景同步次数
- Counter: `log_mq_push_total{scene,status}` - MQ 推送次数

## 日志输出示例

```json
// BFF Access Log
{
  "level": "info",
  "ts": "2024-04-25T10:30:00.000Z",
  "msg": "request end",
  "service": "bff-order",
  "scene": "access",
  "trace_id": "abc123def4560001",
  "span_id": "0000000000000001",
  "method": "POST",
  "path": "/api/orders",
  "status": 200,
  "latency": "45ms"
}

// SRV Audit Log (Vital)
{
  "level": "info",
  "ts": "2024-04-25T10:30:00.005Z",
  "msg": "order created",
  "service": "srv-order",
  "scene": "audit",
  "reliability": "vital",
  "trace_id": "abc123def4560001",
  "span_id": "0000000000000002",
  "parent_id": "0000000000000001",
  "order_id": "ORD-001",
  "user_id": "U-100",
  "amount": 100.00
}
```

## 依赖

| 依赖 | 用途 |
|------|------|
| go.uber.org/zap | 日志库 |
| gopkg.in/yaml.v3 | YAML 配置解析 |
| gopkg.in/natefinch/lumberjack.v2 | 日志轮转 |
| github.com/prometheus/client_golang | Prometheus 指标 |
| github.com/IBM/sarama | Kafka 客户端（MQ 功能） |
| github.com/uber/jaeger-client-go | Jaeger 客户端（SRV） |

## 设计决策记录

| 日期 | 决策 | 原因 |
|------|------|------|
| 2026-04-25 | Vital 场景采用 RingBuffer + Sync 刷盘 | 兼顾性能和可靠性，1万 QPS 下每条 sync 不可行 |
| 2026-04-25 | MQ 作为异步备份而非主路径 | File 是 Source of Truth，MQ 挂了不影响业务 |
| 2026-04-25 | BFF 生成 trace_id，SRV 接收并传播 | BFF 是入口，唯一可信的 ID 生成点 |
| 2026-04-25 | MQ 顺序通过一致性哈希保证 | 相同 trace_id 到固定 partition，消费端保序 |
| 2026-04-25 | Service name 在 logger 初始化时注入 | 统一、避免遗漏 |
