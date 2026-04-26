# 日志库需求规格

## 概述

基于 uber-go/zap 封装企业级日志库，作为脚手架的一部分自动生成到微服务项目中。

## 目标

- 支持多场景日志（Business / Access / Audit / Error）
- 支持多可靠性级别（Vital / Important / Normal）
- 支持调用链集成（OpenTelemetry + Jaeger）
- 支持 MQ 异步备份
- 支持 BFF/SRV 差异化集成

## 术语定义

### 日志场景（Business / Access / Audit / Error）

| 场景 | 用途 | 示例 | 使用层级 |
|------|------|------|---------|
| **Business** | 业务逻辑，记录业务操作执行过程 | 订单创建、支付成功、用户登录 | handler、repo、main |
| **Access** | 访问记录，记录请求的元信息 | POST /api/orders 200 45ms | middleware、interceptor |
| **Audit** | 审计追踪，记录关键操作用于合规追溯 | 用户A修改了订单B的收货地址 | handler |
| **Error** | 错误追踪，记录异常和故障 | create order failed: timeout | 所有层级 |

#### 场景分离的原因

```
┌─────────────────────────────────────────────────────────────────────┐
│                        分离的原因                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   1. 用途不同                                                       │
│   ├─ Business: 开发调试、业务分析                                   │
│   ├─ Access: 运维监控、流量分析                                    │
│   ├─ Audit: 合规审计、数据追溯                                    │
│   └─ Error: 故障排查、根因分析                                    │
│                                                                     │
│   2. 可靠性要求不同                                                 │
│   ├─ Audit: Vital（必须不丢）                                     │
│   ├─ Error: Important（建议不丢）                                 │
│   └─ Access/Business: Important（可丢可补）                        │
│                                                                     │
│   3. 存储策略不同                                                   │
│   ├─ Audit: 长期保留（3-5年）                                     │
│   ├─ Access: 中期保留（30天）                                     │
│   └─ Business/Error: 短期保留（7天）                               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

#### 场景必选字段清单

```
┌─────────────────────────────────────────────────────────────────────┐
│                        场景必选字段                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Access（访问日志）- 必须字段：                                       │
│   ├─ trace_id: string      # 调用链 ID（无则自动生成）              │
│   ├─ span_id: string       # 当前 Span ID                         │
│   ├─ method: string       # HTTP 方法：GET/POST/PUT/DELETE        │
│   ├─ path: string         # 请求路径（不含 query）                │
│   ├─ status: int          # HTTP 状态码                          │
│   ├─ latency_ms: int64    # 请求耗时（毫秒）                      │
│   ├─ client_ip: string    # 客户端 IP                            │
│   ├─ user_agent: string    # User-Agent 头                       │
│   └─ service_name: string # 服务名（自动注入）                   │
│                                                                     │
│   Business（业务日志）- 必须字段：                                    │
│   ├─ trace_id: string      # 调用链 ID                            │
│   ├─ span_id: string       # 当前 Span ID                         │
│   ├─ scene: string         # 场景名（自动填充）                   │
│   ├─ action: string        # 操作名称（调用方传入）               │
│   ├─ success: bool        # 是否成功                             │
│   └─ service_name: string  # 服务名（自动注入）                   │
│                                                                     │
│   Audit（审计日志）- 必须字段：                                       │
│   ├─ trace_id: string      # 调用链 ID                            │
│   ├─ span_id: string       # 当前 Span ID                         │
│   ├─ action: string        # 操作类型：create/update/delete        │
│   ├─ resource: string      # 资源类型：order/user/payment         │
│   ├─ user_id: interface{}  # 操作用户 ID（支持 string/int64）     │
│   ├─ resource_id: string   # 资源 ID                             │
│   ├─ timestamp: int64     # Unix 时间戳（毫秒）                  │
│   └─ details: map         # 扩展详情（敏感字段需脱敏）           │
│                                                                     │
│   Error（错误日志）- 必须字段：                                       │
│   ├─ trace_id: string      # 调用链 ID                            │
│   ├─ span_id: string       # 当前 Span ID                         │
│   ├─ error_msg: string     # 错误信息                             │
│   ├─ error_stack: string   # 错误堆栈（可选，Error 级必填）      │
│   ├─ scene: string         # 场景名（自动填充）                   │
│   ├─ service_name: string  # 服务名（自动注入）                   │
│   └─ timestamp: int64     # Unix 时间戳（毫秒）                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**JSON 序列化格式**：所有字段使用 **snake_case** 命名（如 `trace_id`, `span_id`, `user_id`），与 OpenTelemetry 的 baggage/metrics 规范保持一致。

#### 代码示例

```go
// 请求：POST /api/orders

// Access Log（访问日志）
// - 谁、何时、何地、做了什么请求
logger.Access.Infow("request",
    "method", "POST",
    "path", "/api/orders",
    "status", 200,
    "client_ip", "10.0.0.1",
    "latency", "45ms")

// Business Log（业务日志）
// - 业务逻辑的执行过程
logger.Business.Infow("order created",
    "order_id", "ORD-001",
    "user_id", 100,
    "amount", 100)

// Audit Log（审计日志）
// - 合规要求的操作记录，可追溯
// - 使用结构化 AuditRecord，确保字段一致性
auditRecord := &AuditRecord{
    Action:     "create",
    Resource:   "order",
    UserID:     100,
    ResourceID: "ORD-001",
    Details: map[string]any{
        "amount": 100,
        "items":  []string{"item1", "item2"},
    },
}
logger.Audit.Log(auditRecord)

// Error Log（错误日志）
// - 系统异常和故障
logger.Error.Errorw("create failed",
    "error", "timeout",
    "order_id", "ORD-001")
```

#### AuditRecord 结构定义

```go
// AuditRecord 审计日志结构（JSON 使用 snake_case）
type AuditRecord struct {
    Action     string         `json:"action"`                 // 操作类型：create/update/delete
    Resource   string         `json:"resource"`              // 资源类型：order/user/payment
    UserID     interface{}    `json:"user_id"`               // 操作用户 ID（支持 string/int64）
    ResourceID string         `json:"resource_id"`           // 资源 ID
    TraceID    string         `json:"trace_id"`              // 调用链 ID（自动注入）
    SpanID     string         `json:"span_id"`               // 当前 Span ID（自动注入）
    Details    map[string]any `json:"details,omitempty"`     // 扩展详情（敏感字段需脱敏）
    Timestamp  time.Time      `json:"timestamp"`              // 时间戳（自动生成）
}

// 脱敏要求：手机号、身份证等敏感字段在 Details 中脱敏处理
// 可通过配置启用自动脱敏规则

// AuditRecord JSON 输出示例：
// {
//   "action": "update",
//   "resource": "order",
//   "user_id": 100,
//   "resource_id": "ORD-001",
//   "trace_id": "abc123",
//   "span_id": "def456",
//   "details": {"amount": 100, "items": ["item1"]},
//   "timestamp": "2026-04-26T10:00:00Z"
// }
```

### 可靠性级别（Vital / Important / Normal）

| 级别 | 丢失代价 | 写入策略 | 适用场景 |
|------|---------|---------|---------|
| **Vital** | 合规风险、资损 | RingBuffer + 同步刷盘（fsync）+ MQ 备份 | Audit（审计）、订单、支付 |
| **Important** | 可补录 | 异步批量，定期 flush | Business、Access、Error |
| **Normal** | 几乎为零 | 采样或不落盘 | Debug、调试信息 |

#### Vital 级别详解

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Vital 级别                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   特点：                                                            │
│   ├─ 双缓冲（Double Buffering）减少 fsync 阻塞                     │
│   ├─ buffer 满 或 超时 → 触发同步刷盘（fsync）                    │
│   ├─ MQ 异步备份，不影响主路径                                      │
│   └─ 丢失代价：合规风险、资损                                      │
│                                                                     │
│   双缓冲设计：                                                       │
│   ┌────────────────────────────────────────────────────────────┐   │
│   │                                                            │   │
│   │   Buffer A ← 接收写入 ←→ Buffer B ←→ fsync 写入            │   │
│   │       ↑                      │                              │   │
│   │       └───── swap ──────────┘                              │   │
│   │                                                            │   │
│   │   写入不阻塞，fsync 在独立 buffer 进行                       │   │
│   │   参考 Linux stdio 的 setbuffer 机制                        │   │
│   │                                                            │   │
│   └────────────────────────────────────────────────────────────┘   │
│                                                                     │
│   fsync 超时降级策略：                                               │
│   ┌────────────────────────────────────────────────────────────┐   │
│   │  单次 fsync > 50ms → 降级为异步批量 + 触发告警              │   │
│   │                                                            │   │
│   │  降级期间：                                                  │   │
│   │  1. 切换为异步批量写入（不完全阻塞业务）                   │   │
│   │  2. 记录 critical 日志：fsync latency > 50ms                │   │
│   │  3. 监控告警：log_vital_sync_slow_total                   │   │
│   └────────────────────────────────────────────────────────────┘   │
│                                                                     │
│   配置参数：                                                        │
│   ├─ buffer_size: 10000     # 每个 Buffer 大小                   │
│   ├─ sync_interval: 1s       # 定时同步间隔                        │
│   ├─ fsync_timeout: 50ms     # 单次 fsync 超时阈值                 │
│   ├─ buffer_swap_timeout: 10ms # 双缓冲交换超时                   │
│   └─ fallback_on_full: true   # buffer 满 时 fallback 同步写       │
│                                                                     │
│   双缓冲实现伪代码：                                               │
│   ┌────────────────────────────────────────────────────────────┐   │
│   │                                                            │   │
│   │ type DoubleBuffer struct {                                │   │
│   │     active   atomic.Pointer[Buffer]  // 当前写入 buffer   │   │
│   │     standby  atomic.Pointer[Buffer]  // 备用 buffer      │   │
│   │     swapping atomic.Bool              // 是否正在交换        │   │
│   │ }                                                          │   │
│   │                                                            │   │
│   │ func (db *DoubleBuffer) Write(data []byte) error {      │   │
│   │     buf := db.active.Load()                              │   │
│   │     if buf.Write(data) {                                │   │
│   │         return nil  // 写入成功                           │   │
│   │     }                                                     │   │
│   │                                                            │   │
│   │     // active 满，尝试 swap                               │   │
│   │     if db.swapping.CompareAndSwap(false, true) {        │   │
│   │         go db.swapAndFsync()  // 后台 fsync 旧 buffer   │   │
│   │         db.swapping.Store(false)                        │   │
│   │     }                                                     │   │
│   │     return nil                                            │   │
│   │ }                                                          │   │
│   │                                                            │   │
│   └────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**磁盘要求**：Vital 场景**必须使用 SSD**。机械硬盘或云盘（EBS）无法保证 P99 < 20ms。

#### Important 级别详解

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Important 级别                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   特点：                                                            │
│   ├─ 异步批量写入                                                   │
│   ├─ 定期 flush                                                    │
│   └─ 丢失代价：可补录                                              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Normal 级别详解

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Normal 级别                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   特点：                                                            │
│   ├─ 采样或不落盘                                                   │
│   ├─ Error/Warn 全量记录                                           │
│   └─ 丢失代价：几乎为零                                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 调用链术语

| 术语 | 定义 |
|------|------|
| trace_id | 调用链唯一标识，由 BFF 层 OpenTelemetry 生成 |
| span_id | 调用链中当前操作的标识 |
| parent_id | 上游 span 的标识 |

## 功能需求

### FR-1: 多场景日志

详见「术语定义 - 日志场景」章节。

| 场景 | 可靠性级别 | 使用层级 |
|------|-----------|---------|
| Business | Important | BFF/SRV handler、main、repo |
| Access | Important | BFF/SRV middleware、interceptor |
| Audit | **Vital** | BFF/SRV handler |
| Error | Important | 所有层级 |

### FR-2: 多可靠性级别

详见「术语定义 - 可靠性级别」章节。

### FR-3: 日志采样

- **Error: 100% 全量记录**，不采样（故障排查必需）
- Warn: 可配置采样率（默认 100%）
- Info: 可配置采样率
- Debug: 不记录

**注**：Error 日志使用**限流**而非采样控制写入速率，避免关键错误丢失。

```
┌─────────────────────────────────────────────────────────────────────┐
│                        采样与限流策略                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   级别        │ 默认采样率 │ 说明                                  │
│ ─────────────────────────────────────────────────────────────────  │
│   Error       │ 100%       │ 不采样，使用限流控制速率             │
│   Warn        │ 100%       │ 可配置采样率                         │
│   Info        │ 可配置     │ 1s/100/200                           │
│   Debug       │ 0%         │ 不记录                                │
│                                                                     │
│   限流策略（针对 Error 高频场景）：                                 │
│   ┌────────────────────────────────────────────────────────────┐   │
│   │  令牌桶限流配置：                                         │   │
│   │  rate: 100        # 100 条/秒/进程                       │   │
│   │  burst: 200      # 突发容量                             │   │
│   │  overflow_action: aggregate  # aggregate / drop / warn   │   │
│   │                                                            │   │
│   │  超限行为：                                               │   │
│   │  - aggregate: 聚合统计后批量记录，不丢弃                │   │
│   │  - drop: 丢弃（不推荐）                                │   │
│   │  - warn: 降级为 Warning + 日志记录                     │   │
│   └────────────────────────────────────────────────────────────┘   │
│                                                                     │
│   特殊采样（Adaptive Sampling，可选 P2）：                         │
│   ├─ 按错误率动态调整：错误率高时提高采样率                      │
│   └─ 按延迟动态调整：延迟高时提高采样率                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

```go
// 限流实现示例
type RateLimiter struct {
    rate     int           // 速率（条/秒）
    burst   int           // 突发容量
    tokens  atomic.Int64  // 当前令牌数
    lastRefill time.Time  // 上次补充时间
}

func (r *RateLimiter) Allow() bool {
    r.refill()
    if r.tokens.Load() > 0 {
        r.tokens.Dec()
        return true
    }
    return false
}
```

```yaml
# 限流配置
rate_limit:
  error:
    rate: 100        # 100 条/秒/进程
    burst: 200       # 突发容量
    overflow_action: aggregate  # aggregate / drop / warn
```

### FR-4: 日志轮转

- 按日期分割，每天凌晨检查或创建新文件
- 文件名格式：`{service}-{date}.log`
- 启动时清理过期文件
- 每 6 小时定时清理
- 按文件名中的日期判断（防止 touch 篡改）

```
┌─────────────────────────────────────────────────────────────────────┐
│                    日志轮转与 Vital 的兼容性                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   问题：lumberjack 重命名文件时可能导致 Vital 写入失败              │
│                                                                     │
│   解决方案：                                                        │
│   1. 轮转时持有文件锁，确保写入 atomic                           │
│   2. 使用 zap 的 file-rotation 替代方案（推荐）                   │
│   3. 新文件创建后，再释放旧文件锁                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### FR-5: 结构化日志

- 支持业务字段
- JSON 格式输出
- 错误堆栈追踪

### FR-6: 多输出

- 同时写文件和 stdout
- 输出路径可配置

### FR-7: Prometheus 指标

- Counter: `log_messages_total{level,type,scene}` - 各级别日志计数
- Histogram: `log_write_duration_seconds{scene}` - 写入延迟
- Counter: `log_vital_sync_total` - Vital 场景同步次数
- Counter: `log_vital_buffer_full_total` - Vital buffer full 次数
- Counter: `log_vital_fallback_total` - Vital fallback 同步写次数
- Counter: `log_vital_sync_slow_total` - Vital fsync 超时（> 50ms）次数
- Counter: `log_mq_push_total{scene,status}` - MQ 推送次数

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Histogram Bucket 配置                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   log_write_duration_seconds_bucket:                                │
│   [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10, 20]             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### FR-8: 存储层日志对接

- MySQL（GORM）、MongoDB、Redis、ES 等操作自动记录日志
- 日志自动携带 trace_id、span_id
- **日志级别**：存储层使用独立 Logger，其级别与全局级别**解耦**
- 避免全局 Debug 关闭时，存储层日志不可见

```
┌─────────────────────────────────────────────────────────────────────┐
│                    存储层日志级别策略                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   存储层 Logger API：                                              │
│   ┌────────────────────────────────────────────────────────────┐ │
│   │                                                            │ │
│   │  // 全局 Logger 提供存储层实例                            │ │
│   │  storageLogger := logger.NewStorageLogger(cfg.Storage)   │ │
│   │                                                            │ │
│   │  // 使用方法1：直接调用                                    │ │
│   │  storageLogger.Debugw("mysql query",                      │ │
│   │      "sql", sql,                                          │ │
│   │      "latency", latency,                                  │ │
│   │  )                                                         │ │
│   │                                                            │ │
│   │  // 使用方法2：通过 Logger.Storage() 获取                 │ │
│   │  logger.Storage().Debugw("mysql query",                  │ │
│   │      "sql", sql,                                          │ │
│   │  )                                                         │ │
│   └────────────────────────────────────────────────────────────┘ │
│                                                                     │
│   存储层 Logger 特点：                                           │
│   ├─ 独立配置 `storage.log_level`，与全局级别解耦             │
│   ├─ 不受 `logger.SetLevel()` 影响                            │
│   └─ 支持 Debug/Info/Warn/Error 级别                         │
│                                                                     │
│   默认配置：                                                      │
│   ├─ storage.log_level: debug                                 │
│   ├─ storage.log_body: false       # 默认不记录 Body        │
│   └─ storage.log_slow_threshold: 100ms  # 慢查询阈值        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```
┌─────────────────────────────────────────────────────────────────────┐
│                    存储层日志级别策略                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   默认配置：                                                        │
│   ├─ storage.log_level: debug                                     │
│   ├─ storage.log_body: false        # 默认不记录请求/响应 Body   │
│   └─ storage.log_slow_threshold: 100ms  # 慢查询阈值              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### FR-9: 动态日志级别调整

- 运行时修改日志级别，无需重启服务
- 支持 HTTP 端点 `/debug/loglevel` 或配置中心监听
- **并发安全**：`atomic.Value` 全量替换 map，保证并发安全

```go
// HTTP 接口
// GET /debug/loglevel              # 查询当前级别
// POST /debug/loglevel?level=debug # 设置全局级别
// POST /debug/loglevel?scene=audit&level=info # 设置单个场景级别

// 并发安全实现示例
type LevelManager struct {
    levels atomic.Value // map[string]zapcore.Level
}

func (m *LevelManager) SetLevel(scene string, level zapcore.Level) {
    old := m.load().(map[string]zapcore.Level)

    // 全量替换，避免 map 内部修改的 data race
    newMap := make(map[string]zapcore.Level, len(old)+1)
    for k, v := range old {
        newMap[k] = v
    }
    newMap[scene] = level
    m.levels.Store(newMap)
}

func (m *LevelManager) GetLevel(scene string) zapcore.Level {
    return m.load().(map[string]zapcore.Level)[scene]
}
```

### FR-10: 敏感信息脱敏

- 审计日志中可能包含身份证、手机号、银行卡等敏感信息
- 支持配置脱敏规则（正则匹配）
- 通过 context 传递脱敏标记

**性能优化**：提供字段级标记 + 字符串替换方案，避免高并发下正则匹配开销。

```go
// 字段级标记方案（推荐，性能更优）
type AuditRecord struct {
    Action     string
    UserPhone  string `sensitive:"phone"`     // 自动脱敏
    IDCard     string `sensitive:"id_card"`   // 自动脱敏
    BankCard   string `sensitive:"bank_card"` // 自动脱敏
    Details    map[string]any
}
```

```yaml
# 脱敏配置
sensitive:
  enabled: true

  # 字段级标记方案（推荐）
  # 结构体 tag: sensitive:"phone"
  use_field_tags: true

  # 正则匹配方案（备选，默认规则集预编译）
  default_rules:
    - pattern: "1[3-9]\\d{9}"
      type: phone           # 手机号：138****1234
      replacement: "$1****$2"
    - pattern: "\\d{17}[\\dXx]"
      type: id_card        # 身份证：110***********1234
      replacement: "${1}***********${2}"
    - pattern: "\\d{16}\\d?"
      type: bank_card       # 银行卡：6225 **** **** 1234
      replacement: "$1 **** **** $2"

  # 自定义规则
  rules:
    - field: "*.mobile"
      type: phone
    - field: "*.id_card"
      type: id_card
    - field: "*.bank_card"
      type: bank_card
```

**性能指标**：
- 字段标记方案：额外延迟 < 0.1μs/条
- 正则匹配方案：额外延迟 < 5μs/条（1万 QPS 下约 5% CPU）

### FR-11: 日志上下文传递

- 支持 `logger.With(fields...)` 创建绑定上下文的 Logger 实例
- 减少每次调用手动传参
- **并发安全**：`With()` 返回新实例，无锁竞争

```go
// 创建带上下文的 Logger
ctxLogger := logger.With(
    zap.String("user_id", "100"),
    zap.String("order_id", "ORD-001"),
)

// 后续调用无需再传这些字段
ctxLogger.Business.Infow("order processing") // 自动带 user_id, order_id

// 从 context 统一提取 trace 信息
func (l *Logger) BusinessFromContext(ctx context.Context) *SceneLogger {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        sc := span.SpanContext()
        return l.sceneLoggers["business"].With(
            zap.String("trace_id", sc.TraceID().String()),
            zap.String("span_id", sc.SpanID().String()),
        )
    }
    return l.sceneLoggers["business"]
}
```

### FR-12: 日志压缩与归档

- 轮转后的文件应压缩存储
- 归档策略：压缩算法（gzip/zstd）、归档路径、生命周期管理

```yaml
archive:
  enabled: true
  compress: true
  algorithm: gzip  # gzip / zstd
  retention_days: 90  # 保留 90 天后删除
  archive_path: /data/logs/archive  # 归档路径
```

### FR-13: 测试辅助工具

- 提供 `TestLogger` 实现，支持内存缓冲区
- 便于单元测试中验证日志输出

```go
// 测试辅助
func TestOrderHandler(t *testing.T) {
    buf := &.Buffer{}
    testLogger := logger.NewLoggerForTest(buf)

    // 执行业务逻辑
    handler.CreateOrder(testLogger, req)

    // 断言日志输出
    assert.Contains(t, buf.String(), "order created")
    assert.Contains(t, buf.String(), "order_id=ORD-001")
}
```
```

## 调用链实现参考

### Jaeger All-in-One 环境

本地开发环境已配置 Jaeger All-in-One，参考项目根目录 `docker-compose.yaml`（`/Users/hyx/work/gowork/src/gpx/docker-compose.yaml`）：

```yaml
jaeger-all-in-one:
  image: jaegertracing/all-in-one:latest
  container_name: jaeger-all-in-one
  environment:
    COLLECTOR_OTLP_ENABLED: "true"
  ports:
    - "16686:16686"    # Jaeger UI
    - "14268:14268"    # Jaeger Collector (HTTP)
    - "14250:14250"    # Jaeger Collector (gRPC)
    - "6831:6831/udp"  # Jaeger Agent (UDP)
    - "4317:4317"      # OTLP gRPC
    - "4318:4318"      # OTLP HTTP
```

**端口说明**：

| 端口 | 协议 | 用途 |
|------|------|------|
| 16686 | HTTP | Jaeger Web UI |
| 4317 | gRPC | OTLP Collector (gRPC) |
| 4318 | HTTP | OTLP Collector (HTTP) |
| 6831 | UDP | Jaeger Agent |

**启动命令**：

```bash
# 启动完整可观测性栈
docker-compose -f /Users/hyx/work/gowork/src/gpx/docker-compose.yaml up -d jaeger-all-in-one

# 查看 Jaeger UI
open http://localhost:16686
```

### 环境准备 Checklist

```
开发环境准备：
  [ ] Docker 运行中
  [ ] docker-compose up -d jaeger-all-in-one
  [ ] 确认 16686、4317、4318 端口可用
  [ ] go mod tidy（拉取 OTel 依赖）
```

### 参考代码

#### 本地实现参考（完整可运行）

| 组件 | 绝对路径 | 说明 |
|------|---------|------|
| Gin 集成 | `/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/ginChain/main.go` | otelgin 中间件 + Gin 接入示例 |
| gRPC 客户端 | `/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/grpcChain/bff` | otelgrpc 客户端拦截器 |
| gRPC 服务端 | `/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/grpcChain/srv` | otelgrpc 服务端拦截器 |
| GORM 集成 | `/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/gormchain` | GORM OTel 插件使用 |
| Redis 集成 | `/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/redisChain` | redisotel 插件使用 |

#### 官方集成参考（GitHub）

| 组件 | 参考链接 | 说明 |
|------|---------|------|
| Gin 集成 | [open-telemetry/opentelemetry-go-contrib/instrumentation/github.com/gin-gonic/gin/otelgin](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gin-gonic/gin) | 官方维护的 Gin 中间件 |
| gRPC 集成 | [open-telemetry/opentelemetry-go-contrib/instrumentation/google.golang.org/grpc/otelgrpc](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/google.golang.org/grpc/otelgrpc) | 官方维护的 gRPC 拦截器 |
| GORM 集成 | [go-gorm/opentelemetry](https://github.com/go-gorm/opentelemetry) | 官方 GORM OTel 插件（含示例） |
| Redis 集成 | [redis/go-redis/extra/redisotel](https://github.com/redis/go-redis/tree/master/extra/redisotel) | go-redis 官方 OTel 集成 |
| 日志集成 | [CSDN 自定义 log 包](https://blog.csdn.net/the_shy_faker/article/details/129420308) | 结构化日志实现参考 |

### Gin 集成参考

**完整示例**：`/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/ginChain/main.go`

```go
import (
    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
    serviceName    = "Gin-Jaeger-Demo"
    jaegerEndpoint = "127.0.0.1:4318"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    exp, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint(jaegerEndpoint),
        otlptracehttp.WithInsecure())
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithResource(resource.New(ctx,
            resource.WithAttributes(semconv.ServiceName(serviceName)),
        )),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
        sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(time.Second)),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(
        propagation.NewCompositeTextMapPropagator(
            propagation.TraceContext{},
            propagation.Baggage{},
        ),
    )
    return tp, nil
}

func main() {
    r := gin.New()
    r.Use(otelgin.Middleware(serviceName))  // 设置 otelgin 中间件
    // ...
}

// 获取 trace_id
r.Use(func(c *gin.Context) {
    traceID := trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()
    c.Header("Trace-Id", traceID)
})
```

### gRPC 集成参考

**完整示例**：
- 客户端：`/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/grpcChain/bff/bff.go`
- 服务端：`/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/grpcChain/srv/srv.go`

```go
// gRPC Client（使用 otelgrpc.NewClientHandler）
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

conn, err := grpc.NewClient(
    addr,
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithStatsHandler(otelgrpc.NewClientHandler()),  // 启用 OTel
)

// gRPC Server（使用 otelgrpc.NewServerHandler）
import (
    "google.golang.org/grpc"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

s := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),  // 启用 OTel
)
```

### GORM 集成参考

**完整示例**：`/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/gormchain/main.go`

```go
// GORM OpenTelemetry 插件
import (
    "gorm.io/gorm"
    "gorm.io/plugin/opentelemetry/tracing"
)

db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
    panic(err)
}

// 启用 GORM OpenTelemetry 插件
if err := db.Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
    panic(err)
}

// 使用时传入 context，自动记录 SQL 调用链
db.WithContext(ctx).Create(&book)
```

### Redis 集成参考

**完整示例**：`/Users/hyx/work/gowork/src/go_tutorial/framework/callChain/redisChain/main.go`

```go
// Redis OpenTelemetry 插件
import (
    "github.com/redis/go-redis/v9"
    "github.com/redis/go-redis/extra/redisotel/v9"
)

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6380",
})

// 启用 tracing
if err := redisotel.InstrumentTracing(rdb); err != nil {
    panic(err)
}

// 启用 metrics
if err := redisotel.InstrumentMetrics(rdb); err != nil {
    panic(err)
}

// 使用时传入 context，自动记录 Redis 调用链
ctx, span := tracer.Start(ctx, "doSomething")
defer span.End()
rdb.Get(ctx, "key")
```

### 官方集成链接

| 组件 | 官方链接 |
|------|---------|
| OpenTelemetry Go | https://opentelemetry.io/ |
| otelgin | https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gin-gonic/gin |
| otelgrpc | https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/google.golang.org/grpc/otelgrpc |
| GORM OTel | https://github.com/go-gorm/opentelemetry |
| redisotel | https://github.com/redis/go-redis/tree/master/extra/redisotel |
| go-redis | https://github.com/redis/go-redis |
| Elasticsearch OTel | https://github.com/elastic/go-elasticsearch/v8 |

### 博客参考

| 文章 | 链接 |
|------|------|
| OpenTelemetry 实现链路追踪 | https://blog.csdn.net/the_shy_faker/article/details/130805514 |

## 调用链需求

### LR-1: 调用链路概述

```
┌─────────────────────────────────────────────────────────────────────┐
│                        完整调用链路                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Client                                                           │
│      │                                                             │
│      ▼                                                             │
│   BFF  ──────────────────────────────────────────────────────▶ Storage │
│   │         │                                                     │
│   │         │  HTTP Header: traceparent                          │
│   │         ▼                                                     │
│   └──────▶ SRV-1  ──────────────────────────────────────────▶ Storage │
│               │                                                   │
│               │  gRPC Metadata: traceparent                       │
│               ▼                                                   │
│            SRV-2  ──────────────────────────────────────────▶ Storage │
│               │                                                   │
│               │  gRPC Metadata: traceparent                       │
│               ▼                                                   │
│            SRV-3  ──────────────────────────────────────────▶ Storage │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### LR-2: BFF 层

- 依赖 OpenTelemetry + otelgin
- 通过 `otelgin.Middleware()` 生成/接收 trace
- 通过 HTTP Header 透传：`traceparent`（标准 W3C 格式）
- Sidecar 模式连接 Jaeger Agent
- **接口级别开关**：通过路由分组 + 中间件按需开启

### LR-3: SRV 层

- 依赖 OpenTelemetry + otelgrpc（Sidecar 模式）
- 从 gRPC Metadata 接收 trace context
- `otelgrpc.UnaryServerInterceptor()` 自动创建 child span
- 通过 gRPC Metadata 透传给下游 SRV
- 使用标准 `propagation.TraceContext{}` 传播
- **无 trace 时兼容**：如果 SRV 收到请求无 trace context，自动生成新的 trace_id 并打 warning 日志

```
┌─────────────────────────────────────────────────────────────────────┐
│                    无 trace 时的处理策略                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   场景：BFF 未开启调用链，但 SRV 需要排查问题                       │
│                                                                     │
│   处理：                                                            │
│   1. SRV 检测到无 trace context                                    │
│   2. 自动生成新的 trace_id                                         │
│   3. 打 warning 日志: "no trace context, generated new trace_id"    │
│   4. 后续日志带上此 trace_id                                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### LR-4: BFF → SRV 透传机制

#### 4.1 gRPC 透传

```go
// BFF 调用 SRV 时，自动通过 otelgrpc 透传
conn, err := grpc.Dial("srv-order:8080",
    grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
)
```

```
┌─────────────────────────────────────────────────────────────────────┐
│                        gRPC 透传流程                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   BFF Client                                                       │
│   ──────────                                                       │
│   spanCtx, _ := tr.Extract(ctx, propagation.HeaderCarrier(req.Header))│
│   span := tr.Start(ctx, "call-srv", otelgrpc.WithSpan(spanCtx))   │
│   // 自动注入 traceparent 到 gRPC Metadata                         │
│                                                                     │
│   SRV Server                                                       │
│   ──────────                                                       │
│   otelgrpc.UnaryServerInterceptor() 自动：                          │
│   1. 从 Metadata 提取 traceparent                                  │
│   2. 创建 child span                                              │
│   3. 放入 context                                                 │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 4.2 Kitex（字节 RPC 框架）透传

```go
// BFF 调用 SRV 时
import "github.com/cloudwego/kitex/client/callopt"

// Kitex 客户端初始化时配置 tracer
client, err := echo.NewClient("echo",
    client.WithTracer(otel.NewTracer()),
)

// 调用时自动透传
resp, err := client.Echo(ctx, req)
```

```go
// SRV 服务端配置
import "github.com/cloudwego/kitex/pkg/tracer"

// Kitex Server 初始化时配置 tracer
svr := echo.NewServer(
    server.WithTracer(otel.NewTracer()),
)
```

#### 4.3 手动透传（扩展用）

```go
// 如果需要手动透传（如自定义 RPC 框架）
func injectTraceContext(ctx context.Context, md *metadata.MD) {
    p := otel.GetTextMapPropagator()
    carrier := propagation.MapCarrier{}
    p.Inject(ctx, carrier)
    for k, v := range carrier {
        md.Set(k, v)
    }
}

func extractTraceContext(ctx context.Context, md metadata.MD) context.Context {
    p := otel.GetTextMapPropagator()
    carrier := propagation.MapCarrier{}
    for k, v := range md {
        carrier[k] = v[0]
    }
    return p.Extract(ctx, carrier)
}
```

### LR-5: 存储层日志对接



存储层（MySQL、MongoDB、Redis、ES 等）操作需要自动记录调用链信息。

```
┌─────────────────────────────────────────────────────────────────────┐
│                      存储层日志对接                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Handler                                                          │
│      │                                                             │
│      ├── logger.Business.Infow("query order", "table", "orders")  │
│      │                                                             │
│      └── repo.Create(ctx, order)                                  │
│                    │                                              │
│                    ├── MySQL Hook/Interceptor                     │
│                    │   └── logger.Business.Infow("mysql", ...)    │
│                    │                                              │
│                    ├── MongoDB Hook/Interceptor                    │
│                    │   └── logger.Business.Infow("mongodb", ...)  │
│                    │                                              │
│                    ├── Redis Hook/Interceptor                      │
│                    │   └── logger.Business.Infow("redis", ...)   │
│                    │                                              │
│                    └── ES Hook/Interceptor                         │
│                        └── logger.Business.Infow("elasticsearch", ...)|
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 5.1 MySQL（GORM）

```go
// GORM OpenTelemetry 插件
import "gorm.io/plugin/opentelemetry"

db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
db.Use(opentelemetry.NewPlugin(opentelemetry.Config{
    SpanRequestBody: true,
    SpanResponseBody: true,
}))
```

#### 5.2 MongoDB

```go
// MongoDB OpenTelemetry 插件
import "go.mongodb.org/mongo-driver/mongo/readconcern"

clientOpts := options.Client().
    ApplyURI("mongodb://localhost:27017").
    SetMonitor(otel.NewMonitor())
```

#### 5.3 Redis

```go
// Redis OpenTelemetry 插件
import "github.com/redis/go-redis/extra/redisotel/v9"

rdb := redis.NewClient(&redis.Options{})
redisotel.InstrumentTracing(rdb)
```

#### 5.4 Elasticsearch

```go
// ES OpenTelemetry 插件
import "github.com/elastic/go-elasticsearch/v8"

client, err := elastic.NewClient(
    elasticsearch.Config{
        Tracer: otel.Tracer("elasticsearch"),
    },
)
```

### LR-6: 中间件/拦截器顺序

```
请求进入
    │
    ▼
1. otelgin.Middleware / otelgrpc.UnaryServerInterceptor
   - BFF: 生成/接收 trace context
   - SRV: 接收 trace + 创建 child span
    │
    ▼
2. LoggingMiddleware / LoggingInterceptor
   - 记录 Access Log
    │
    ▼
3. AuthMiddleware / AuthInterceptor
   - 认证检查
    │
    ▼
4. RateLimitMiddleware / RateLimitInterceptor
   - 限流检查
    │
    ▼
5. Handler
   - Business / Audit / Error Log
    │
    ▼
   Repository / Storage Layer
   - MySQL / MongoDB / Redis / ES Hook
   - 自动记录存储操作 + trace context
    │
    ▼
响应返回（逆序）
```

### LR-7: OpenTelemetry 集成

- 使用 `otelgin.Middleware()` 作为 Gin 中间件
- 使用 `otelgrpc.UnaryServerInterceptor()` 作为 gRPC 拦截器
- 使用 `otelgrpc.UnaryClientInterceptor()` 作为 gRPC 客户端拦截器
- 使用 `propagation.TraceContext{}` 进行 context 传播
- 支持 Jaeger 后端

## MQ 需求

### MQ-0: MQ 接口抽象

```go
// MQ 生产者接口（抽象，支持多种实现）
type Producer interface {
    // scene: 场景名（如 business/audit/access）
    // key: 一致性哈希 key（通常是 trace_id）
    // data: 日志数据
    Push(ctx context.Context, scene string, key string, data []byte) error
    Close() error
}

// Kafka 实现（默认）
type KafkaProducer struct {
    brokers     []string
    topicPrefix string  // 从配置读取，如 "app-logs"
    partitionCount int32  // 固定 partition 数（默认 64）
}

// Push 内部拼接 topic = topicPrefix + "-" + scene
func (p *KafkaProducer) Push(ctx context.Context, scene, key string, data []byte) error {
    topic := p.topicPrefix + "-" + scene
    // 一致性哈希选择 partition
    partition := hash(key) % p.partitionCount
    // 发送到 Kafka
    return p.send(ctx, topic, partition, key, data)
}

// Pulsar 实现（可选）
type PulsarProducer struct {
    // ...
}
```

**好处**：后续可轻松支持 Pulsar/RabbitMQ，无需改动核心代码。

**注意**：Kafka partition 数量固定（默认 64），不支持动态扩缩容。

### MQ-1: 顺序保证

- 按 trace_id 做一致性哈希
- **相同 trace_id 到固定 partition**
- 消费端按 partition 消费

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MQ 顺序保证限制                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   约束：                                                            │
│   1. partition 数量固定（默认 64），不支持动态扩缩容             │
│      - partition 变化时一致性哈希环会 rehash                      │
│      - 短期内同一 trace_id 可能映射到不同 partition               │
│   2. 仅保证同一 trace_id 内有序，不保证全局顺序                  │
│   3. 消费者组多实例时，按 partition 并行消费                       │
│                                                                     │
│   警告：                                                            │
│   - 如果需要全局有序，需使用单 partition（牺牲吞吐）               │
│   - 推荐方案：同一 trace_id 内最终一致即可                        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### MQ-2: 批量推送

- 异步批量推送
- 可配置批量大小和刷新间隔
- MQ 是备份路径，不影响主流程

### MQ-3: 消费端要求

本库只负责生产 MQ 消息，消费端由独立服务实现（如 Flink、Logstash、Loki）。消费端需遵循以下约定：

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MQ 消费端约定                                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Topic 命名规则：                                                    │
│   ├─ {topic_prefix}-{scene}                                        │
│   ├─ topic_prefix: 从配置读取，默认 "app-logs"                      │
│   └─ scene: business/access/audit/error                            │
│                                                                     │
│   示例：app-logs-audit, app-logs-business                          │
│                                                                     │
│   Partition 策略：                                                    │
│   ├─ 按 trace_id 一致性哈希选择 partition                          │
│   ├─ 相同 trace_id 的日志必须在同一 partition 内                   │
│   └─ 消费端按 partition 并行消费（多消费者实例）                     │
│                                                                     │
│   消息格式（JSON）：                                                  │
│   {                                                                 │
│     "trace_id": "abc123",                                          │
│     "span_id": "def456",                                           │
│     "scene": "audit",                                              │
│     "timestamp": 1745721600000,                                    │
│     "data": { ... }  // 场景相关字段                               │
│   }                                                                 │
│                                                                     │
│   消费端职责：                                                       │
│   ├─ 按 scene 路由到不同的存储（ES/Hive/HDFS）                      │
│   ├─ 处理乱序：同 trace_id 内保序，不同 trace_id 可并行            │
│   └─ 消费位点管理：使用 Kafka offset 而非时间戳                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 配置需求

### 配置结构 (config/log.yaml)

```yaml
env: dev
level: info
service_name: bff-order    # 生成时注入，各服务唯一

sampling:
  # error: 1.0  # 固定 100%，不采样，使用限流控制速率
  warn: 1.0            # Warn 采样率（默认 100%）
  info:
    initial: 100
    thereafter: 200
    tick: 1s
  debug: 0             # Debug 不记录

rate_limit:
  error:
    rate: 100        # 100 条/秒/进程
    burst: 200       # 突发容量
    overflow_action: aggregate  # aggregate / drop / warn

rotation:
  enabled: true
  max_age_days: 7

vital:
  buffer_size: 10000       # 每个 Buffer 大小（双缓冲）
                          # 1万 QPS 下，建议 buffer_size >= 10000
                          # 确保 sync_interval (1s) 内 buffer 不会频繁写满
  sync_interval: 1s        # 定时同步间隔
  fsync_timeout: 50ms      # 单次 fsync 超时阈值，超过则降级
  buffer_swap_timeout: 10ms # 双缓冲交换超时
  fallback_on_full: true    # buffer 满时 fallback 同步写

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
  service_name: ${SERVICE_NAME}  # 从环境变量读取，支持容器化部署
  agent_host: localhost      # 边车模式
  agent_port: 6831          # UDP
  sampler:
    type: rate_limiting     # const / probabilistic / rate_limiting / adaptive
    param: 100               # 每秒最大 traces 数（基于配置）

storage:
  log_level: debug          # 存储层日志级别
  log_body: false           # 是否记录请求/响应 Body
  log_slow_threshold: 100ms # 慢查询阈值

sensitive:
  enabled: true
  rules:
    - field: "*.mobile"
      type: phone
    - field: "*.id_card"
      type: id_card
    - field: "*.bank_card"
      type: bank_card

archive:
  enabled: true
  compress: true
  algorithm: gzip
  retention_days: 90
  archive_path: /data/logs/archive
```

## BFF 层需求

### BFF-1: 目录结构

| 目录 | 日志场景 | 可靠性 | 用途 |
|------|---------|--------|------|
| handler | Business + Audit + Error | Vital/Imp | 请求处理、业务逻辑、审计追踪 |
| middleware | Access + Error | Imp | 请求日志、错误处理 |
| main | Business + Error | Imp | 启动日志、运行状态 |

### BFF-2: 路由分组实现调用链开关

```go
// 不需要调用链的接口（健康检查、监控等）
r.GET("/health", HealthHandler.Check)
r.GET("/metrics", MetricsHandler.Report)

// 需要调用链的接口（通过中间件组）
traced := r.Group("")
traced.Use(otelgin.Middleware("bff-order"))
{
    traced.POST("/api/orders", OrderHandler.CreateOrder)
    traced.GET("/api/orders/:id", OrderHandler.GetOrder)
    traced.POST("/api/payments", PaymentHandler.Pay)
}
```

## SRV 层需求

### SRV-1: 目录结构

| 目录 | 日志场景 | 可靠性 | 用途 |
|------|---------|--------|------|
| handler | Business + Error | Imp | 业务处理、错误记录 |
| interceptor | Access + Error | Imp | 拦截日志、错误追踪 |
| main | Business + Error | Imp | 启动日志、运行状态 |
| repo | Business + Error | Imp | 数据操作日志 |

### SRV-2: Interceptor

- `otelgrpc.UnaryServerInterceptor()`: 接收/传播 trace context，自动创建 child span
- LoggingInterceptor: 记录 Access Log
- AuthInterceptor: 认证
- RateLimitInterceptor: 限流

## 非功能需求

### NFR-1: 性能指标

| 指标 | 要求 | 说明 |
|------|------|------|
| Vital 场景 P99 延迟 | < 20ms（含 fsync） | 考虑磁盘 IO 开销 |
| Important 场景 P99 延迟 | < 1ms | 异步写入，仅入队延迟 |
| RingBuffer 丢消息率 | < 0.001% | buffer full 时 fallback 同步写 |
| MQ 推送失败重试 | 3 次，指数退避 | 退避间隔：1s, 2s, 4s |

**磁盘要求**：Vital 场景建议使用 SSD，避免机械硬盘 IO 成为瓶颈

### NFR-2: 可靠性

- Vital 日志不丢失（同步刷盘）
- MQ 作为备份，不影响主路径
- MQ 顺序：仅保证同一 trace_id 内的顺序，不保证全局顺序

### NFR-3: 可扩展性

- OpenTelemetry 厂商无关
- 支持 Jaeger/Zipkin 等后端

### NFR-4: 部署

- 边车模式部署 Jaeger Agent
- 服务无需直接连接 Collector
- 服务名从环境变量 `SERVICE_NAME` 读取，支持容器化部署

### NFR-5: 可观测性补充

- **Health Check 端点**: `/debug/log/health` 返回日志系统状态（buffer 使用率、MQ 连接状态）
- **日志延迟监控**: Prometheus 记录日志写入到实际落盘的时间差（用于监控 async 队列积压）

### NFR-6: 安全

- 敏感信息脱敏处理
- 日志文件权限控制（640）

## 目录结构

```
pkg/logger/
├── logger.go           # 主 Logger 封装
├── config.go          # 配置定义 + 加载（mapstructure 标签）
├── rotation.go        # 日志轮转（兼容 Vital 双缓冲）
├── cleaner.go         # 过期清理（按文件名日期）
├── sampler.go         # 采样配置（支持 Error/Warn 采样/限流）
├── metrics.go         # Prometheus 指标
├── sensitive.go       # 敏感信息脱敏
├── archive.go         # 日志压缩与归档
├── health.go          # Health Check 端点
├── core/
│   ├── vital.go       # Vital 级别 Core（双缓冲 + Sync + Fallback）
│   ├── important.go   # Important 级别 Core（Async Batch）
│   └── normal.go      # Normal 级别 Core（Sampler）
├── scene/
│   ├── business.go    # 业务日志
│   ├── access.go      # 访问日志
│   ├── audit.go       # 审计日志 + AuditRecord
│   └── error.go       # 错误日志
├── storage/
│   └── logger.go      # 存储层独立 Logger（与全局级别解耦）
├── mq/
│   ├── producer.go    # MQ 生产者接口（抽象）
│   └── kafka.go       # Kafka 实现
├── tracing/
│   ├── otel.go        # OpenTelemetry 初始化
│   ├── gin.go         # Gin 中间件
│   ├── grpc.go        # gRPC 拦截器
│   └── storage.go     # 存储层 Hook（MySQL/Redis/MongoDB/ES）
├── context.go         # Context 传递
└── testutil/
    └── test_logger.go # 测试辅助工具
```

**依赖注入**：Core 只定义接口，MQ Producer 通过参数传入，避免循环依赖。

**Build Tag**：OpenTelemetry 相关依赖通过 build tag 控制（`//go:build otel`）。

```
//go:build otel
// +build otel

package logger
// OpenTelemetry 相关代码
```

## 依赖

### 核心依赖

| 依赖 | 用途 | 参考 |
|------|------|------|
| go.uber.org/zap | 日志库 | - |
| gopkg.in/yaml.v3 | YAML 配置解析 | - |
| gopkg.in/natefinch/lumberjack.v2 | 日志轮转 | - |
| github.com/prometheus/client_golang | Prometheus 指标 | - |
| github.com/IBM/sarama | Kafka 客户端 | - |

### OpenTelemetry 依赖

| 依赖 | 用途 | 参考 |
|------|------|------|
| go.opentelemetry.io/otel | OpenTelemetry API | [官方](https://opentelemetry.io/) |
| go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc | gRPC OTLP 导出器 | [grpcChain](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/google.golang.org/grpc/otelgrpc) |
| go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp | HTTP OTLP 导出器 | [ginChain](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gin-gonic/gin) |
| go.opentelemetry.io/otel/contrib/instrumentation/github.com/gin-gonic/gin/otelgin | Gin 中间件 | [otelgin](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gin-gonic/gin) |
| go.opentelemetry.io/otel/contrib/instrumentation/google.golang.org/grpc/otelgrpc | gRPC 拦截器/客户端 | [otelgrpc](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/google.golang.org/grpc/otelgrpc) |

### 存储层依赖

| 依赖 | 用途 | 参考 |
|------|------|------|
| gorm.io/gorm | ORM 库 | - |
| gorm.io/plugin/opentelemetry | GORM OpenTelemetry 插件 | [go-gorm/opentelemetry](https://github.com/go-gorm/opentelemetry) |
| github.com/redis/go-redis/v9 | Redis 客户端 | - |
| github.com/redis/go-redis/extra/redisotel/v9 | Redis OTel 插件 | [redisotel](https://github.com/redis/go-redis/tree/master/extra/redisotel) |
| go.mongodb.org/mongo-driver/mongo | MongoDB 驱动 | - |
| github.com/elastic/go-elasticsearch/v8 | Elasticsearch 客户端 | - |

### 可选依赖

| 依赖 | 用途 | 参考 |
|------|------|------|
| github.com/cloudwego/kitex | Kitex RPC 框架 | 字节内部框架 |

## 决策记录

| 日期 | 决策 | 原因 |
|------|------|------|
| 2026-04-25 | Vital 场景采用 RingBuffer + Sync 刷盘 | 兼顾性能和可靠性，1万 QPS 下每条 sync 不可行 |
| 2026-04-25 | Vital buffer_size 从 1000 增至 10000 | 避免 1万 QPS 下 0.1s 就触发一次 fsync |
| 2026-04-25 | MQ 作为异步备份而非主路径 | File 是 Source of Truth，MQ 挂了不影响业务 |
| 2026-04-25 | BFF 生成 trace_id，SRV 接收并传播 | BFF 是入口，唯一可信的 ID 生成点 |
| 2026-04-25 | SRV 无 trace 时自动生成 + warning | 兼容 BFF 未开启调用链的情况 |
| 2026-04-25 | MQ 顺序通过一致性哈希保证 | 相同 trace_id 到固定 partition，消费端保序 |
| 2026-04-25 | Service name 从环境变量读取 | 支持容器化部署 |
| 2026-04-25 | BFF 接口级别调用链开关 | 按需开启，减少不必要的开销 |
| 2026-04-25 | 采用 OpenTelemetry 而非直接用 Jaeger Client | 厂商无关，支持多后端 |
| 2026-04-25 | gRPC/Kitex 透传通过 OpenTelemetry 自动完成 | 标准机制，开箱即用 |
| 2026-04-25 | 存储层通过 OpenTelemetry 插件对接 | MySQL/Redis/MongoDB/ES 官方插件支持 |
| 2026-04-25 | 存储层日志默认 Debug 级别 | 避免大量存储日志淹没业务日志 |
| 2026-04-25 | Audit Log 使用结构化 AuditRecord | 确保字段一致性，便于合规查询 |
| 2026-04-25 | Error/Warn 也支持采样 | 1万 QPS 下 Error 5% 也有 500 QPS，需可配置采样率 |
| 2026-04-25 | 日志清理按文件名日期判断 | 防止 touch 命令篡改文件修改时间 |
| 2026-04-25 | 支持动态日志级别调整 | 生产环境排障至关重要 |
| 2026-04-25 | 敏感信息脱敏 | 审计日志合规要求 |
| 2026-04-25 | RingBuffer full 时 fallback 同步写 | 保证 Vital 不丢消息 |
| 2026-04-25 | AuditRecord.UserID 改为 interface{} | 支持 string/int64 等多种类型 |
| 2026-04-25 | 动态级别调整使用 atomic.Value | 保证并发读安全 |
| 2026-04-25 | Vital P99 延迟调整为 < 20ms | 考虑磁盘 IO 开销，SSD 建议 |
| 2026-04-25 | 新增故障排查手册和性能调优指南 | 提升可运维性 |
| 2026-04-25 | 新增默认脱敏规则集 | 预编译正则，性能优化 |
| 2026-04-25 | 采用双缓冲（Double Buffering）替代单 RingBuffer | fsync 期间不阻塞写入 |
| 2026-04-25 | fsync > 50ms 时降级为异步 + 告警 | 避免单次 fsync 阻塞业务 |
| 2026-04-25 | Vital 场景必须使用 SSD | 机械硬盘无法保证 P99 < 20ms |
| 2026-04-25 | Error 保持 100% 不采样 | 故障排查必需，使用限流替代采样 |
| 2026-04-25 | 存储层 Logger 与全局级别解耦 | 避免全局 Debug 关闭时存储日志不可见 |
| 2026-04-25 | MQ Producer 接口抽象 | 支持 Kafka/Pulsar/RabbitMQ 多实现 |
| 2026-04-25 | partition 数量固定（默认 64） | 不支持动态扩缩容，仅保证同 trace_id 内有序 |
| 2026-04-25 | OpenTelemetry 依赖通过 build tag 控制 | 无 OTel 配置时不编译 |
| 2026-04-25 | Health Check 端点提升至 P0 | 生产可观测性必需，无健康检查无法上线 |
| 2026-04-25 | atomic.Value map 修改使用全量替换模式 | 避免 map 内部修改的 data race |
| 2026-04-25 | 移除 sync_timeout，保留 fsync_timeout | 配置项语义清晰 |
| 2026-04-25 | MQ Push 接口接受 scene 而非 topic | Producer 内部拼接 topic_prefix + scene |
| 2026-04-25 | Error 限流使用令牌桶算法 | rate: 100条/秒，burst: 200，超限 aggregate |
| 2026-04-25 | Vital 场景禁用 lumberjack | 自研基于双缓冲的轮转，避免文件锁冲突 |
| 2026-04-25 | 存储层 Logger 提供 NewStorageLogger() API | 独立配置，与全局级别解耦 |

### P0（必须实现，阻塞发布）

- [ ] Vital Core 的双缓冲 + fsync 实现（buffer_size=10000）
- [ ] Important Core 的异步批量
- [ ] Access/Business/Error 场景
- [ ] OpenTelemetry 基础集成（Gin + gRPC）
- [ ] 配置文件加载 + 默认值
- [ ] AuditRecord 结构定义
- [ ] **Health Check 端点（`/debug/log/health`）** ← 从 P1 提升（P0 阻塞发布）

### P1（重要，第一版包含）

- [ ] Audit 场景 + Vital 保证
- [ ] MQ 备份（Kafka）+ 一致性哈希
- [ ] 日志轮转 + 过期清理（按文件名日期）
- [ ] Prometheus 指标
- [ ] 存储层 Hook（至少 GORM）
- [ ] 动态日志级别调整（`/debug/loglevel`）

### P2（可延后）

- [ ] Kitex 框架适配
- [ ] 敏感信息脱敏
- [ ] 日志压缩与归档
- [ ] Elasticsearch 存储层 Hook
- [ ] 存储层日志 Body 记录
- [ ] Adaptive Sampling
- [ ] 测试辅助工具（TestLogger）
- [ ] SRV 无 trace 时自动生成兼容 ← 从 P1 降至 P2（BFF 默认开启 OTel，兜底是边缘情况）

## 故障排查手册

### 故障 1: MQ 连接失败

```
现象：日志正常写入文件，但 MQ 备份失败

排查步骤：
1. 检查 Kafka Broker 是否可达：telnet <broker> 9092
2. 检查 Topic 是否存在：kafka-topics.sh --list
3. 查看指标：log_mq_push_total{status="error"} 是否上升

处理策略：
1. MQ 失败不影响主路径（文件写入正常）
2. 后台继续重试 3 次，指数退避
3. 3 次仍失败，写入本地 error_mq.log 供人工介入
```

### 故障 2: 磁盘满

```
现象：日志写入变慢或失败

排查步骤：
1. df -h 检查磁盘使用率
2. du -sh <log_dir>/* 查看日志目录大小
3. 检查归档任务是否正常执行

处理策略：
1. 紧急：切换到 stdout 输出，关闭文件写入
2. 清理过期日志（手动触发清理任务）
3. 配置磁盘告警阈值（> 80% 告警）
```

### 故障 3: Vital buffer full 告警

```
现象：log_vital_buffer_full_total 指标上升

排查步骤：
1. 检查 Vital 日志 QPS 是否超过预期
2. 检查 fsync 是否成为瓶颈（磁盘 IO 高）

处理策略：
1. 触发 fallback 同步写入，保证不丢
2. 降低 Vital 日志量（减少非必要字段）
3. 扩容或优化磁盘（SSD）
```

## 性能调优指南

### 场景 1: 高吞吐场景（5万+ QPS）

```yaml
# 配置优化建议
vital:
  buffer_size: 50000      # 增大 buffer
  sync_interval: 2s       # 延长同步间隔

sampling:
  error: 0.5             # 降低 Error 采样率
  warn: 0.8              # 降低 Warn 采样率
```

### 场景 2: 低延迟要求（P99 < 5ms）

```yaml
# 配置优化建议
vital:
  buffer_size: 5000
  sync_interval: 500ms

# 必须使用 SSD
```

### 场景 3: 合规优先（审计日志零丢失）

```yaml
# 配置优化建议
vital:
  buffer_size: 1000       # 较小 buffer，快照同步
  sync_interval: 100ms     # 频繁同步
  fallback_on_full: true   # 确保不丢

# 配合 SSD + RAID0
```
v4 版本基于第四轮评估优化，新增字段清单、MQ 消费端约定、Docker/K8s 部署模板、Sampler 接口定义、API 增强建议、验收测试矩阵和压测场景。Health Check 提升至 P0，SRV trace 兜底降至 P2。

建议开发顺序：

Week 1: 核心架构 + Vital/Important Core + 配置加载

Week 2: 场景封装 + 轮转清理 + Prometheus 指标

Week 3: MQ 备份 + OpenTelemetry 集成

Week 4: 动态级别调整 + 存储层 Hook + 测试

总工作量预估：3-4 周（1 人全职）或 2 周（2 人并行）

## 验收标准

| 指标 | 要求 |
|------|------|
| 1万 QPS 下 Vital 场景 P99 延迟 | < 20ms（含 fsync） |
| 1万 QPS 下 Important 场景 P99 延迟 | < 1ms |
| RingBuffer 丢消息率 | < 0.001% |
| MQ 推送失败重试 | 3 次，指数退避（1s, 2s, 4s） |

---

## Docker/K8s 部署模板

### Sidecar 模式（推荐 K8s 部署）

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  replicas: 3
  template:
    spec:
      containers:
        # 应用容器
        - name: app
          image: my-app:latest
          ports:
            - containerPort: 8080
          env:
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "http://jaeger-collector:4317"
            - name: SERVICE_NAME
              value: "my-service"

        # Jaeger Agent Sidecar（可选，若使用 Agent 模式）
        - name: jaeger-agent
          image: jaegertracing/jaeger-agent:latest
          ports:
            - containerPort: 6831
              protocol: UDP
          args:
            - "--reporter.grpc.host-port=jaeger-collector:14250"
            - "--tags=service.name=my-service"
```

### 本地 Docker 开发环境

```bash
# 启动完整可观测性栈
docker-compose -f docker-compose.yaml up -d jaeger-all-in-one prometheus grafana loki

# 查看 Jaeger UI
open http://localhost:16686

# 查看 Prometheus
open http://localhost:9090
```

---

## Sampler 接口定义

```go
// pkg/logger/sampler.go

// Sampler 接口（与 zapcore.Sampler 兼容）
type Sampler interface {
    // ShouldSample 决定是否采样
    ShouldSample(ent zapcore.Entry, fields []zapcore.Field) bool
    // Close 清理资源
    Close()
}

// DynamicSampler 动态采样器（按错误率/延迟动态调整）
type DynamicSampler struct {
    errorRate   atomic.Float64
    latencyP99  atomic.Int64
    baseRate    float64  // 基础采样率
    minRate     float64  // 最低采样率
    maxRate     float64  // 最高采样率
}

func (s *DynamicSampler) ShouldSample(ent zapcore.Entry, fields []zapcore.Field) bool {
    rate := s.calculateRate()
    return rand.Float64() < rate
}

// InfoSampler Zapcore 兼容采样器
type InfoSampler struct {
    initial    int   // 初始采样条数
    thereafter  int   // 之后每 tick 采样条数
    tick       time.Duration
    counter    atomic.Int64
}

func (s *InfoSampler) ShouldSample(ent zapcore.Entry, fields []zapcore.Field) bool {
    count := s.counter.Add(1)
    if count <= int64(s.initial) {
        return true
    }
    return (count-int64(s.initial))%int64(s.thereafter) == 0
}
```

---

## API 增强设计

### 建议增加的 API

```go
// pkg/logger/logger.go

// MustNew panic 版本，方便 main.go 快速启动
func MustNew(cfg *config.Config) *Logger {
    l, err := New(cfg)
    if err != nil {
        panic("failed to initialize logger: " + err.Error())
    }
    return l
}

// Audit.Logf 便捷方法
func (a *AuditLogger) Logf(action, resource, format string, args ...any) {
    a.Log(&AuditRecord{
        Action:   action,
        Resource: resource,
        Details:   map[string]any{"msg": fmt.Sprintf(format, args...)},
    })
}

// Producer.Healthy 健康检查
type Producer interface {
    Push(ctx context.Context, scene, key string, data []byte) error
    Healthy() bool  // 返回 MQ 连接是否健康
    Close() error
}

// SetLevels 原子批量设置
func (l *LevelManager) SetLevels(levels map[string]zapcore.Level) {
    old := m.load().(map[string]zapcore.Level)
    newMap := make(map[string]zapcore.Level, len(old)+len(levels))
    for k, v := range old {
        newMap[k] = v
    }
    for k, v := range levels {
        newMap[k] = v
    }
    m.levels.Store(newMap)
}
```

---

## 验收测试矩阵

| 测试项 | 方法 | 通过标准 |
|--------|------|----------|
| Vital 双缓冲不丢 | 注入 50000 条后 kill -9 | 重启后文件条数 == 50000 |
| Vital fsync 降级 | 模拟磁盘延迟 100ms | 降级日志出现 + P99 < 50ms（降级后） |
| MQ 备份 | 停掉 Kafka 后写入 | 文件正常 + error_mq.log 有记录 |
| 调用链注入 | Gin 请求 → 查日志 | `trace_id` 字段一致贯穿 B/A/A/E |
| 无 trace 兜底 | 关闭 BFF OTel → SRV | SRV 日志有 warning + 新 trace_id |
| 存储层级别解耦 | 全局 level=warn, storage=debug | MySQL debug 日志可见 |
| 采样 | Info 采样率 0.1 | 1000 条入 ≈ 100 条出 |
| Error 限流 | 1s 内注入 500 条 Error | 100 条写入 + 400 条聚合统计 |
| 脱敏 | AuditRecord.UserPhone="13812345678" | 输出 "138****5678" |
| 轮转 | 修改系统时间跨日 | 新文件创建 + 旧文件未丢失 |
| Health Check | 查询 `/debug/log/health` | JSON 含 buffer_usage, mq_status |
| 动态级别 | POST 改 audit → info | 后续 audit debug 不输出 |

---

## 压测场景

| 场景 | 参数 | 通过标准 |
|------|------|----------|
| 1万 QPS Mixed | B:A:A:E = 6:2:1:1 | Vital P99 < 20ms, Imp P99 < 1ms |
| 5万 QPS Important | 纯 Info | P99 < 2ms, 无 OOM |
| Vital 满 buffer | 10万条突发 | fallback 触发但不丢 |
| MQ 故障恢复 | Kafka 停 30s 后恢复 | 0 条丢失，补推成功 |

---

## 开发启动 Checklist

```
Week 1 交付物：
  [ ] config.go + 默认值 + YAML 加载
  [ ] core/vital.go (DoubleBuffer + fsync + fallback)
  [ ] core/important.go (AsyncBatch)
  [ ] 单元测试覆盖双缓冲 swap 路径

前置条件（开发前确认）：
  [ ] 确认云环境磁盘类型（SSD ✅ / 云盘 ⚠️ 需实测）
  [ ] 锁定 go.mod 版本（尤其 OTel 插件）
  [ ] 补充 Access/Business/Audit/Error 必选字段清单（已完成）
```
