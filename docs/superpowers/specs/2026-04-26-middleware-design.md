# BFF/SRV 中间件设计规格

## 一、概述

### 1.1 设计目标

为 gpx 脚手架的 BFF 层和 SRV 层提供完整的中间件/拦截器生成能力，支持 JWT 认证、限流、黑白名单等常用中间件。

### 1.2 命令接口

**BFF 层：**
```bash
gpx micro-bff --name myshop --http gin --middleware jwt,ratelimit,blacklist
gpx micro-bff --name myshop --http hertz --middleware jwt,ratelimit,blacklist
```

**SRV 层：**
```bash
gpx micro-srv --name myshop --rpc grpc --middleware ratelimit,retry,timeout,tracing
gpx micro-srv --name myshop --rpc kitex --middleware ratelimit,retry,timeout,tracing
```

### 1.3 设计原则

| 原则 | 说明 |
|------|------|
| Flag 控制生成 | `--middleware` flag 决定生成哪些文件 |
| YAML 控制运行时 | `middleware.yaml` 决定运行时启用哪些、顺序 |
| pkg 层共享 | 核心逻辑放在 `pkg/`，与框架无关 |
| 分离适配层 | `internal/middleware` 或 `internal/interceptor` 负责框架适配 |

---

## 二、BFF 层架构

### 2.1 生成文件结构

```
myshop/
├── pkg/
│   ├── jwt/
│   │   ├── jwt.go       # 核心 JWT 逻辑（生成/解析/刷新）
│   │   ├── options.go   # 配置选项函数
│   │   └── types.go     # Config, CustomClaims 定义
│   ├── ratelimit/
│   │   ├── ratelimit.go # 令牌桶/滑动窗口限流实现
│   │   └── types.go     # RateLimiter 接口定义
│   └── blacklist/
│       ├── blacklist.go # IP 黑白名单实现
│       └── types.go     # Blacklist 配置
└── bffH5/
    └── internal/
        └── middleware/
            ├── middleware.go  # Builder 入口
            ├── jwt.go         # Gin/Hertz 适配（按 --middleware 生成）
            ├── ratelimit.go   # Gin/Hertz 适配
            └── blacklist.go   # Gin/Hertz 适配
```

### 2.2 middleware.yaml 配置

```yaml
order:
  - jwt
  - ratelimit
  - blacklist

jwt:
  secret: "your-secret-key"
  expiry: "24h"
  issuer: "myshop"

ratelimit:
  qps: 100
  burst: 200

blacklist:
  mode: "block"
  ips:
    - "192.168.1.1"
    - "10.0.0.0/8"
```

### 2.3 Builder 模式

```go
// middleware.go
type Middleware struct {
    jwt        *jwt.JWTService
    ratelimit  *ratelimit.RateLimiter
    blacklist  *blacklist.Blacklist
}

func (m *Middleware) Build(cfg *middleware.Config) error {
    for _, name := range cfg.Order {
        switch name {
        case "jwt":
            m.jwt = jwt.New(cfg.JWT.Secret, jwt.WithExpiry(cfg.JWT.Expiry))
        case "ratelimit":
            m.ratelimit = ratelimit.New(cfg.RateLimit.QPS)
        case "blacklist":
            m.blacklist = blacklist.New(cfg.Blacklist.IPs)
        }
    }
    return nil
}
```

### 2.4 Gin 适配示例

```go
// jwt.go
func JWTMiddleware(jwtSvc *jwt.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        // 解析 token ...
        c.Next()
    }
}
```

### 2.5 中间件执行顺序

运行时按 `middleware.yaml` 中的 `order` 字段顺序执行：

```
请求 → JWT 认证 → RateLimit 限流 → Blacklist 黑白名单 → Handler
```

---

## 三、SRV 层架构

### 3.1 支持的拦截器

| 拦截器 | 说明 |
|--------|------|
| `ratelimit` | 服务端限流，保护服务不被压垮 |
| `retry` | 幂等重试，处理临时性故障 |
| `timeout` | 超时控制，避免慢调用阻塞 |
| `tracing` | 链路追踪，关联分布式日志 |

### 3.2 生成文件结构

```
myshop/srvProduct/
└── internal/
    └── interceptor/
        ├── interceptor.go  # Builder 入口
        ├── ratelimit.go    # gRPC/Kitex 适配
        ├── retry.go       # gRPC/Kitex 适配
        ├── timeout.go      # gRPC/Kitex 适配
        └── tracing.go      # gRPC/Kitex 适配
```

### 3.3 interceptor.yaml 配置

```yaml
order:
  - ratelimit
  - retry
  - timeout
  - tracing

ratelimit:
  qps: 100
  burst: 50

retry:
  maxAttempts: 3
  backoff: "exponential"

timeout:
  default: "500ms"
  methods:
    CreateProduct: "1s"

tracing:
  serviceName: "srvProduct"
```

### 3.4 gRPC 适配示例

```go
// ratelimit.go
func RateLimitInterceptor(rl *ratelimit.RateLimiter) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if !rl.Allow() {
            return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
        }
        return handler(ctx, req)
    }
}
```

---

## 四、pkg 层设计

### 4.1 pkg/jwt

```go
// jwt.go
type JWTService struct {
    secret []byte
    expiry time.Duration
    issuer string
}

func New(secret string, opts ...Option) *JWTService
func (s *JWTService) GenerateToken(claims *CustomClaims) (string, error)
func (s *JWTService) ParseToken(tokenString string) (*CustomClaims, error)
func (s *JWTService) RefreshToken(oldToken string) (string, error)
```

### 4.2 pkg/ratelimit

```go
// ratelimit.go
type RateLimiter interface {
    Allow() bool
    AllowN(ctx context.Context, n int) bool
}

type TokenBucketLimiter struct { ... }
func NewTokenBucket(qps int, burst int) *TokenBucketLimiter

type SlidingWindowLimiter struct { ... }
func NewSlidingWindow(qps int, window time.Duration) *SlidingWindowLimiter
```

### 4.3 pkg/blacklist

```go
// blacklist.go
type Blacklist interface {
    IsBlocked(ip string) bool
    Add(ip string) error
    Remove(ip string) error
}

type IPBlacklist struct { ... }
func New(ipList []string) (*IPBlacklist, error)
```

---

## 五、框架适配

### 5.1 BFF 层适配

| HTTP 引擎 | 文件 | 适配要点 |
|-----------|------|----------|
| Gin | `middleware/gin_jwt.go` | `gin.HandlerFunc`，使用 `c.GetHeader()`, `c.Set()` |
| Hertz | `middleware/hertz_jwt.go` | `app.HandlerFunc`，使用 `c.GetHeader()`, `c.Set()` |

### 5.2 SRV 层适配

| RPC 引擎 | 文件 | 适配要点 |
|----------|------|----------|
| gRPC | `interceptor/grpc_ratelimit.go` | `grpc.UnaryServerInterceptor`，从 `metadata` 提取信息 |
| Kitex | `interceptor/kitex_ratelimit.go` | `endpoint.Middleware`，从 `rpcinfo` 获取信息 |

---

## 六、模板文件清单

### 6.1 BFF 层模板

```
templates/micro-app/bff/middleware/
├── middleware.go.tmpl      # Builder 入口
├── gin_jwt.go.tmpl         # Gin JWT 适配
├── gin_ratelimit.go.tmpl   # Gin 限流适配
├── gin_blacklist.go.tmpl   # Gin 黑白名单适配
├── hertz_jwt.go.tmpl       # Hertz JWT 适配
├── hertz_ratelimit.go.tmpl # Hertz 限流适配
└── hertz_blacklist.go.tmpl # Hertz 黑白名单适配
```

### 6.2 SRV 层模板

```
templates/micro-app/srv/interceptor/
├── interceptor.go.tmpl     # Builder 入口
├── grpc_ratelimit.go.tmpl  # gRPC 限流适配
├── grpc_retry.go.tmpl      # gRPC 重试适配
├── grpc_timeout.go.tmpl    # gRPC 超时适配
├── grpc_tracing.go.tmpl    # gRPC 链路追踪适配
├── kitex_ratelimit.go.tmpl  # Kitex 限流适配
├── kitex_retry.go.tmpl      # Kitex 重试适配
├── kitex_timeout.go.tmpl    # Kitex 超时适配
└── kitex_tracing.go.tmpl    # Kitex 链路追踪适配
```

### 6.3 pkg 层模板

```
templates/pkg/
├── jwt/
│   ├── jwt.go.tmpl
│   ├── options.go.tmpl
│   └── types.go.tmpl
├── ratelimit/
│   ├── ratelimit.go.tmpl
│   └── types.go.tmpl
└── blacklist/
    ├── blacklist.go.tmpl
    └── types.go.tmpl
```

---

## 七、后续扩展

### 7.1 Phase 1（当前）
- BFF: JWT、RateLimit、Blacklist 中间件
- SRV: Ratelimit、Retry、Timeout、Tracing 拦截器

### 7.2 Phase 2
- BFF: CORS、RequestID、Gzip 中间件
- SRV: 熔断、舱壁隔离

### 7.3 Phase 3
- BFF: 请求签名、CSRF 防护
- SRV: 服务认证、审计日志

---

## 八、依赖库

| 库 | 版本 | 用途 |
|----|------|------|
| github.com/golang-jwt/jwt/v5 | v5.x | JWT 生成和校验 |
| golang.org/x/time/rate | latest | 令牌桶限流 |
| go.opentelemetry.io/otel | latest | 链路追踪 |
