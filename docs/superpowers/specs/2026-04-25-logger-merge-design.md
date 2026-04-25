# 日志目录合并与 BFF/SRV 层日志集成设计

## 概述

将 `pkg/log` 目录合并到 `pkg/logger`，保留所有日志功能，并在 BFF 层和 SRV 层的各目录中集成日志调用。

## 目标

1. 合并日志目录，统一使用 `pkg/logger`
2. 保留所有现有日志功能（多场景、MQ 推送等）
3. 在 BFF 层和 SRV 层集成日志记录
4. 生成日志使用文档

## 目录合并方案

### 合并后目录结构

```
templates/pkg/logger/
├── logger.go           # 主日志（来自 pkg/log）
├── config.go          # 配置（来自 pkg/log）
├── business.go        # 业务日志（来自 pkg/log）
├── access.go         # 访问日志（来自 pkg/log）
├── audit.go          # 审计日志（来自 pkg/log）
├── error.go          # 错误日志（来自 pkg/log）
├── mq.go             # MQ 推送（来自 pkg/log）
├── mq_kafka.go       # Kafka 实现（来自 pkg/log）
├── rotation.go       # 日志轮转（来自 pkg/log）
├── cleaner.go        # 清理器（来自 pkg/log）
├── sampler.go        # 采样（来自 pkg/log）
├── metrics.go        # Prometheus 指标（来自 pkg/log）
├── context.go        # 上下文（来自 pkg/log）
└── formatter.go      # 格式化器（原有）
```

### 删除目录

- `templates/pkg/log/` - 合并后删除

## 日志场景

| 场景 | 用途 |
|------|------|
| Business | 业务逻辑日志 |
| Access | 访问记录日志（请求/响应） |
| Audit | 审计追踪日志 |
| Error | 错误追踪日志 |

### MQ 推送配置

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

## BFF 层日志集成

### handler 目录

- **场景**: Business + Access + Audit + Error
- **用途**: 请求处理、业务逻辑、审计追踪、错误记录

### middleware 目录

- **场景**: Access + Error
- **用途**: 请求日志、错误处理

### main 目录

- **场景**: Business + Error
- **用途**: 启动日志、运行状态

## SRV 层日志集成

### handler 目录

- **场景**: Business + Error
- **用途**: 业务处理、错误记录

### interceptor 目录

- **场景**: Access + Error
- **用途**: 拦截日志、错误追踪

### main 目录

- **场景**: Business + Error
- **用途**: 启动日志、运行状态

### repo 目录

- **场景**: Business + Error
- **用途**: 数据操作日志

## 生成的文档

生成 `docs/logging-usage.md`，包含：

1. 日志目录结构说明
2. 各层日志使用场景
3. 代码示例
4. 配置说明

## 实现任务

1. 将 pkg/log 的文件移动到 pkg/logger
2. 更新代码生成逻辑中的目录引用
3. 在 BFF 层各目录添加日志调用
4. 在 SRV 层各目录添加日志调用
5. 生成日志使用文档
6. 更新代码生成逻辑
