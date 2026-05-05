# Deployment Requirements

## Overview

Enterprise Logger 组件部署要求，包括硬件规格、Kubernetes 配置和运维注意事项。

## Hardware Requirements

### Vital Scene (Audit Logger) - SSD Required

**Audit Logger 使用双 buffer + Fsync 同步落盘，对磁盘 IO 要求极高。**

| Resource | Minimum | Recommended |
|----------|----------|--------------|
| Disk Type | SSD (NVMe preferred) | NVMe SSD |
| IOPS | 5,000 | 10,000+ |
| Throughput | 100 MB/s | 200 MB/s+ |
| Latency | < 5ms | < 2ms |

**注意:** 使用 HDD 会导致 Fsync 延迟超过 50ms 阈值，触发 degraded 状态。

### Important Scene (Business/Access/Error)

异步批量写入 Kafka，对本地磁盘无特殊要求。

| Resource | Minimum | Recommended |
|----------|----------|--------------|
| Disk Type | Any | SSD |
| Buffer Size | 10,000 | 50,000+ |

## Kafka Configuration

### Partition Count

```yaml
mq:
  partition_count: 64  # 建议 >= 64
```

**Partition 数量建议:**
- 生产环境: 64+
- 测试环境: 8-16
- Partition 数量应与消费者线程数匹配

### Broker Configuration

```yaml
mq:
  brokers:
    - "kafka-1:9092"
    - "kafka-2:9092"
    - "kafka-3:9092"
```

### Retention Settings

Kafka topic 保留策略建议:

```bash
# 创建 topic 时设置
kafka-topics.sh --create \
  --topic app-logs-business \
  --partitions 64 \
  --replication-factor 3 \
  --config retention.bytes=10737418240 \
  --config retention.ms=604800000
```

## Kubernetes Deployment

### Deployment Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  labels:
    app: order-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
    spec:
      containers:
        - name: order-service
          image: order-service:v1.2.3
          ports:
            - containerPort: 8080
          volumeMounts:
            - name: log-volume
              mountPath: /var/log
          env:
            - name: SERVICE_NAME
              value: "order-service"
            - name: CONFIG_PATH
              value: "/config/logger.yaml"
          livenessProbe:
            httpGet:
              path: /debug/log/health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /debug/log/health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
      volumes:
        - name: log-volume
          emptyDir:
            medium: Memory
```

### Health Check Configuration

#### Liveness Probe

检查 Logger 进程是否存活。

```yaml
livenessProbe:
  httpGet:
    path: /debug/log/health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 3
```

#### Readiness Probe

检查 Logger 是否可以接收流量（MQ 是否健康）。

```yaml
readinessProbe:
  httpGet:
    path: /debug/log/health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3
```

### Resource Limits

```yaml
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "2000m"
    memory: "2Gi"
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| SERVICE_NAME | Yes | - | 服务名称 |
| CONFIG_PATH | No | ./config.yaml | 配置文件路径 |
| ENV | No | dev | 运行环境 |

## Monitoring

### Prometheus Alerts

```yaml
groups:
  - name: logger-alerts
    rules:
      # Buffer 使用率过高
      - alert: LoggerBufferUsageHigh
        expr: buffer_usage_ratio > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          description: "Logger {{ $labels.scene }} buffer usage is {{ $value }}"

      # MQ 不健康
      - alert: LoggerMQUnhealthy
        expr: mq_status == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          description: "Logger MQ producer is unhealthy"

      # Fsync 延迟过高
      - alert: LoggerFsyncLatencyHigh
        expr: histogram_quantile(0.99, rate(fsync_latency_seconds_bucket[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          description: "Logger {{ $labels.scene }} P99 fsync latency is {{ $value }}s"
```

### Grafana Dashboard

建议监控面板:

1. **Buffer Usage**: 各场景 buffer 使用率
2. **MQ Push Rate**: Kafka 推送 QPS
3. **Fsync Latency**: Fsync P50/P95/P99
4. **Log Entries**: 各场景日志产生速率
5. **Error Rate**: error 级别日志占比

## Log Rotation

### Vital Scene (Audit)

Audit 日志使用自定义 DoubleBuffer 实现轮转:

```yaml
rotation:
  enabled: true
  max_age_days: 7
```

**注意:** 不使用 lumberjack，直接通过 DoubleBuffer 控制文件切换。

### File Naming

```
/var/log/audit-2024-01-15.000.json
/var/log/audit-2024-01-15.001.json
/var/log/audit-2024-01-16.000.json
```

## Troubleshooting

### degraded 状态

当 Fsync 延迟超过 50ms 或 Buffer 使用率超过 90% 时进入 degraded 状态。

**排查步骤:**

1. 检查磁盘 IOPS 是否足够
2. 检查 Kafka broker 是否响应正常
3. 检查网络延迟

```bash
# 查看 health endpoint
curl http://localhost:8080/debug/log/health
```

### 日志丢失

**检查项:**

1. Vital scene 是否使用 SSD
2. fallback_on_full 是否启用
3. Kafka 连接是否正常

### MQ 推送失败

**排查步骤:**

1. 检查 Kafka broker 地址
2. 检查网络连通性
3. 检查 partition 数量是否匹配

## Performance Tuning

### High Throughput Scenario

```yaml
vital:
  buffer_size: 50000
  sync_interval: "500ms"

mq:
  batch_size: 500
  flush_interval: "500ms"
```

### Low Latency Scenario

```yaml
vital:
  buffer_size: 10000
  sync_interval: "100ms"

mq:
  batch_size: 50
  flush_interval: "100ms"
```
