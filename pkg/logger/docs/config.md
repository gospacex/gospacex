# Configuration Guide

## Overview

Enterprise Logger 使用 YAML 配置文件，通过 `logger.NewFromYAML(path)` 或 `logger.LoadConfig(path)` 加载。

## Complete Configuration

```yaml
env: "prod"                          # 环境: dev/staging/prod
level: "info"                        # 全局日志级别: debug/info/warn/error
service_name: "order-service"       # 服务名称 (必填)
topic_prefix: "app-logs"             # Kafka topic 前缀

sampling:                            # 采样配置
  warn: 1.0                          # warn 级别采样率 (0.0-1.0)
  info:                              # info 级别采样
    initial: 100                      # 初始令牌数
    thereafter: 200                   # 之后每 tick 补充令牌数
    tick: "1s"                       # 补充间隔
  debug: 0.0                         # debug 级别采样率

rate_limit:                           # 限流配置
  error:                             # error 场景限流
    rate: 100                        # 每秒令牌数
    burst: 200                       # 桶容量
    overflow_action: "aggregate"     # 溢出动作: aggregate/drop/warn

rotation:                             # 日志轮转配置
  enabled: true                      # 是否启用轮转
  max_age_days: 7                   # 文件最大保留天数

vital:                                # Vital 可靠性配置
  buffer_size: 10000                # 双 buffer 缓冲区大小
  sync_interval: "1s"               # 同步间隔
  fsync_timeout: "50ms"             # Fsync 超时阈值 (>50ms 标记为 degraded)
  fallback_on_full: true             # buffer 满时是否回退到同步写

storage:                              # Storage Logger 配置
  log_level: "debug"                # 独立日志级别
  log_body: false                    # 是否记录请求/响应体
  log_slow_threshold: "100ms"       # 慢查询阈值

mq:                                   # Kafka 配置
  brokers:                           # Broker 地址列表
    - "localhost:9092"
  partition_count: 64               # 分区数 (建议 >= 64)
  batch_size: 100                   # 批量发送大小
  flush_interval: "1s"              # 刷新间隔

tracing:                              # OpenTelemetry 配置
  enabled: true                     # 是否启用追踪
  endpoint: "localhost:4318"        # OTLP gRPC 端点
  sample_rate: 1.0                  # 采样率 (0.0-1.0)

metrics:                              # Prometheus 配置
  enabled: true                     # 是否启用指标
```

## Scene-Specific Configuration

### Business Logger (Important)

使用 AsyncBatch 异步批量写入 Kafka，适合高频业务日志。

```yaml
mq:
  brokers:
    - "kafka-1:9092"
    - "kafka-2:9092"
  partition_count: 64
  batch_size: 100
  flush_interval: "1s"
```

### Access Logger (Important)

记录 HTTP 请求访问日志，与 Business Logger 共用 AsyncBatch 配置。

```yaml
level: "info"  # 建议 access 使用 info 级别
```

### Audit Logger (Vital)

使用 DoubleBuffer + Fsync 保证日志不丢失，**必须使用 SSD**。

```yaml
vital:
  buffer_size: 10000
  sync_interval: "1s"
  fsync_timeout: "50ms"
  fallback_on_full: true
```

### Error Logger (Important + Rate Limiting)

带令牌桶限流的错误日志，防止错误风暴。

```yaml
rate_limit:
  error:
    rate: 100
    burst: 200
    overflow_action: "aggregate"  # aggregate: 聚合后记录 / drop: 丢弃 / warn: 记录警告
```

### Storage Logger (Independent)

独立配置的存储层日志，与主日志级别解耦。

```yaml
storage:
  log_level: "debug"          # 可独立设置 debug 级别
  log_body: false             # 不记录 body，避免敏感数据泄露
  log_slow_threshold: "100ms" # 100ms 以上的 DB 操作记录 Warn
```

## Reliability Level Configurations

### Vital (Audit Logger)

```yaml
vital:
  buffer_size: 10000           # 建议 10000-50000
  fsync_timeout: "50ms"       # 超过此值标记为 degraded
  fallback_on_full: true      # 启用同步写回退
```

### Important (Business/Access/Error)

```yaml
mq:
  brokers:
    - "kafka:9092"
  partition_count: 64
  batch_size: 100
  flush_interval: "1s"
```

## MQ (Kafka) Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| brokers | localhost:9092 | Kafka broker 地址列表 |
| partition_count | 64 | 分区数，建议与消费者数量匹配 |
| batch_size | 100 | 批量发送消息数 |
| flush_interval | 1s | 强制刷新间隔 |

**注意事项:**
- `partition_count` 必须为正整数
- 生产环境建议使用 64+ 分区以支持高并发
- `batch_size` 和 `flush_interval` 任一满足即发送

## Tracing (OpenTelemetry) Configuration

```yaml
tracing:
  enabled: true
  endpoint: "localhost:4318"   # OTLP gRPC receiver 地址
  sample_rate: 1.0             # 1.0 = 100% 采样
```

**Trace Context Propagation:**
- Gin: 使用 `otelgin` middleware 自动注入 trace_id
- gRPC: 使用 `otelgrpc` interceptor 自动传播
- 日志自动携带 trace_id, span_id 字段

## Metrics (Prometheus) Configuration

```yaml
metrics:
  enabled: true
```

**暴露指标:**

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| log_entries_total | Counter | scene, level | 各场景日志条目计数 |
| log_level_total | Counter | scene, level | 各级别日志计数 |
| mq_push_total | Counter | status | MQ 推送计数 (success/error) |
| fsync_latency_seconds | Histogram | scene | Fsync 延迟分布 |
| buffer_usage_ratio | Gauge | scene | Buffer 使用率 |

## Validation Rules

配置加载时会进行校验:

| Field | Rule |
|-------|------|
| service_name | 必填，不能为空 |
| vital.buffer_size | 必须 > 0 |
| vital.fsync_timeout | 必须 > 0 |
| mq.partition_count | 必须 > 0 |

## Environment-Specific Examples

### Development

```yaml
env: "dev"
level: "debug"
service_name: "order-service-dev"

vital:
  buffer_size: 1000
  fsync_timeout: "100ms"
  fallback_on_full: true

mq:
  brokers:
    - "localhost:9092"
  partition_count: 4

tracing:
  enabled: false
```

### Production

```yaml
env: "prod"
level: "info"
service_name: "order-service"

vital:
  buffer_size: 50000
  sync_interval: "500ms"
  fsync_timeout: "50ms"
  fallback_on_full: true

mq:
  brokers:
    - "kafka-1:9092"
    - "kafka-2:9092"
    - "kafka-3:9092"
  partition_count: 64
  batch_size: 500
  flush_interval: "500ms"

tracing:
  enabled: true
  endpoint: "otel-collector:4318"
  sample_rate: 0.1

metrics:
  enabled: true
```
