# 日志系统重构与 MQ 推送功能设计

## 概述

重构项目日志系统，统一日志目录为 `pkg/log`，支持多场景日志记录（Business、Access、Audit、Error），并增加 Kafka 异步批量推送功能。

## 目标

1. 统一日志目录，消除 `pkg/log` 和 `pkg/logger` 并存
2. 支持多场景日志记录
3. 支持配置开启 MQ 推送，将日志异步批量推送到 Kafka

## 日志场景

| 场景 | 用途 | 典型内容 |
|------|------|----------|
| Business | 业务逻辑 | 业务操作、状态变化、业务错误 |
| Access | 访问记录 | 请求路径、方法、耗时、状态码 |
| Audit | 审计追踪 | 用户操作、敏感数据变更 |
| Error | 错误追踪 | 堆栈信息、错误上下文 |

## 目录结构

```
pkg/log/
├── logger.go          # 主 Logger，实现不同场景的日志记录
├── config.go          # 配置结构体
├── business.go        # BusinessLogger - 业务日志
├── access.go         # AccessLogger - 访问日志
├── audit.go          # AuditLogger - 审计日志
├── error.go          # ErrorLogger - 错误日志
├── rotation.go       # 日志轮转
├── cleaner.go        # 过期日志清理
├── sampler.go        # 采样
├── mq.go             # MQ 推送功能
├── mq_kafka.go       # Kafka 实现
└── config.yaml       # 配置模板
```

## 配置结构

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

## 核心设计

### Logger 结构

```go
type Logger struct {
    Business *BusinessLogger  // 业务日志
    Access   *AccessLogger    // 访问日志
    Audit    *AuditLogger     // 审计日志
    Error    *ErrorLogger     // 错误日志
    mqPusher MQPusher         // MQ 推送器
}
```

### MQ 推送接口

```go
type MQPusher interface {
    Push(topic string, data []byte) error
    Close() error
}

type MQConfig struct {
    Enabled       bool
    Type          string   // kafka
    Brokers       []string
    Topic         string   // 基础 topic，支持场景区分
    Async         bool     // 异步推送
    BatchSize     int      // 批量大小
    FlushInterval time.Duration // 刷新间隔
}
```

### 推送策略

- 异步队列缓冲：达到 BatchSize 或 FlushInterval 触发推送
- Topic 命名：每个场景独立 Topic，格式为 `{topic}-{scene}`，如 `app-logs-business`
- 失败处理：推送失败时日志写入本地文件

## 实现任务

1. 创建 `pkg/log` 目录结构
2. 实现各场景 Logger
3. 实现 Kafka MQ 推送
4. 更新配置模板
5. 更新代码生成逻辑，统一使用 `pkg/log`
