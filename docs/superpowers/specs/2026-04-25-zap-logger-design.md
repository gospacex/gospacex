# 企业级日志库设计

## 概述

基于 uber-go/zap 封装的企业级日志库，作为脚手架的一部分自动生成到微服务项目中。

## 功能列表

| 功能 | 描述 |
|------|------|
| 多环境支持 | dev/staging/prod 配置 |
| 日志轮转 | 按日期分割，自动清理 7 天前日志 |
| 结构化日志 | 支持业务字段 |
| 错误堆栈追踪 | 完整堆栈信息 |
| 日志采样 | Error/Warn 全量，Info 采样 1s/100/200，Debug 不记录 |
| 多输出 | 同时写文件 + stdout |
| Prometheus 指标 | 日志级别计数 + 写入延迟 |
| 日志分级 | Business / Access / Error 三类 |

## 目录结构

```
templates/pkg/log/
├── config.go.tmpl      # 配置定义 + 加载
├── logger.go.tmpl      # 主 Logger 封装
├── rotation.go.tmpl    # 日志轮转
├── cleaner.go.tmpl     # 过期清理
├── sampler.go.tmpl     # 采样配置
├── metrics.go.tmpl     # Prometheus 指标
├── business.go.tmpl    # 业务日志
├── access.go.tmpl      # 访问日志
└── context.go.tmpl     # Context 传递
```

## 配置结构 (config/log.yaml)

```yaml
env: dev

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

## 实现要点

### 1. 日志轮转

- 实现自定义 `zapcore.WriteSyncer`
- 每天凌晨检查或创建新文件（按日期命名）
- 文件名格式：`app-2024-04-25.log`

### 2. 过期清理

- 启动时清理过期文件
- 每 6 小时定时清理
- 按文件修改时间判断

### 3. 采样策略

- Error/Warn: 全量记录
- Info: `zap.NewSampler(core, time.Second, 100, 200)`
- Debug: 不记录

### 4. Prometheus 指标

- Counter: `log_messages_total{level,type}` - 各级别日志计数
- Histogram: `log_write_duration_seconds` - 写入延迟

### 5. 三类日志

- **Business**: 业务操作日志，带业务字段
- **Access**: HTTP/gRPC 访问日志，带请求信息
- **Error**: 错误日志，自动带堆栈

## 使用示例

```go
// 初始化
logger := log.NewLogger(cfg)

// 业务日志
logger.Business.Info("order created", zap.String("order_id", "123"))

// 访问日志
logger.Access.Info("request",
    zap.String("method", "GET"),
    zap.Int("status", 200),
)

// 错误日志
logger.Error.With(zap.Stack("stacktrace")).Error("failed")
```
