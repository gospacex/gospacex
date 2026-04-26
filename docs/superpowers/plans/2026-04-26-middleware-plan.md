# BFF/SRV 中间件实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 gpx 脚手架添加 BFF 层和 SRV 层中间件/拦截器生成能力

**架构：** BFF 层生成 pkg/jwt 等共享包 + internal/middleware 框架适配；SRV 层生成 internal/interceptor 拦截器；通过 --middleware flag 控制生成，middleware.yaml 控制运行时

**技术栈：** Go, github.com/golang-jwt/jwt/v5, golang.org/x/time/rate

---

## 文件结构

### 1. pkg 层模板（BFF/SRV 共用）

```
templates/pkg/
├── jwt/
│   ├── jwt.go.tmpl        # JWTService, New, GenerateToken, ParseToken, RefreshToken
│   ├── options.go.tmpl    # WithExpiry, WithIssuer 等选项函数
│   └── types.go.tmpl      # Config, CustomClaims
├── ratelimit/
│   ├── ratelimit.go.tmpl  # TokenBucketLimiter, Allow, AllowN
│   └── types.go.tmpl      # RateLimiter 接口
└── blacklist/
    ├── blacklist.go.tmpl   # IPBlacklist, IsBlocked, Add, Remove
    └── types.go.tmpl      # Blacklist 配置
```

### 2. BFF 层模板

```
templates/micro-app/bff/middleware/
├── middleware.go.tmpl      # Middleware Builder
├── gin_jwt.go.tmpl        # Gin JWT 适配
├── gin_ratelimit.go.tmpl  # Gin 限流适配
├── gin_blacklist.go.tmpl   # Gin 黑白名单适配
├── hertz_jwt.go.tmpl      # Hertz JWT 适配
├── hertz_ratelimit.go.tmpl # Hertz 限流适配
└── hertz_blacklist.go.tmpl # Hertz 黑白名单适配
```

### 3. SRV 层模板

```
templates/micro-app/srv/interceptor/
├── interceptor.go.tmpl    # Interceptor Builder
├── grpc_ratelimit.go.tmpl # gRPC 限流
├── grpc_retry.go.tmpl     # gRPC 重试
├── grpc_timeout.go.tmpl   # gRPC 超时
├── grpc_tracing.go.tmpl   # gRPC 链路追踪
├── kitex_ratelimit.go.tmpl # Kitex 限流
├── kitex_retry.go.tmpl    # Kitex 重试
├── kitex_timeout.go.tmpl  # Kitex 超时
└── kitex_tracing.go.tmpl  # Kitex 链路追踪
```

### 4. 配置文件模板

```
templates/
├── configs/
│   ├── middleware.yaml.tmpl    # BFF middleware.yaml
│   └── interceptor.yaml.tmpl   # SRV interceptor.yaml
```

### 5. CLI 修改

```
internal/cli/
├── microapp_new.go         # 添加 --middleware flag 解析和中间件生成逻辑
└── microbff.go             # 修改 BFF middleware.go 生成逻辑
```

---

## 任务 1：修复 Hertz 模板 Bug

**文件：**
- 修改：`templates/micro-app/bff/middleware/hertz_middleware.go.tmpl`

- [ ] **步骤 1：读取当前 hertz_middleware.go.tmpl 内容**

- [ ] **步骤 2：移除重复的代码块（lines 51-66）**

当前文件在 Logger 函数后又有一份重复的代码块，需要删除。

- [ ] **步骤 3：验证修复后的文件语法正确**

运行：`go run . micro-bff --name test --http hertz --modules product` 生成项目检查

---

## 任务 2：创建 pkg/jwt 模板

**文件：**
- 创建：`templates/pkg/jwt/types.go.tmpl`
- 创建：`templates/pkg/jwt/options.go.tmpl`
- 创建：`templates/pkg/jwt/jwt.go.tmpl`

- [ ] **步骤 1：创建 types.go.tmpl**

```go
package jwt

import "time"

type Config struct {
    Secret string
    Expiry time.Duration
    Issuer string
}

type CustomClaims struct {
    UserID   int64
    Username string
    Role     string
    jwt.RegisteredClaims
}
```

- [ ] **步骤 2：创建 options.go.tmpl**

```go
package jwt

type Option func(*JWTService)

func WithExpiry(expiry time.Duration) Option {
    return func(s *JWTService) {
        s.expiry = expiry
    }
}

func WithIssuer(issuer string) Option {
    return func(s *JWTService) {
        s.issuer = issuer
    }
}
```

- [ ] **步骤 3：创建 jwt.go.tmpl**

```go
package jwt

import (
    "errors"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrExpiredToken = errors.New("token expired")
)

type JWTService struct {
    secret []byte
    expiry time.Duration
    issuer string
}

func New(secret string, opts ...Option) *JWTService {
    s := &JWTService{
        secret: []byte(secret),
        expiry: 24 * time.Hour,
        issuer: "app",
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

func (s *JWTService) GenerateToken(claims *CustomClaims) (string, error) {
    claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(s.expiry))
    claims.Issuer = s.issuer
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secret)
}

func (s *JWTService) ParseToken(tokenString string) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return s.secret, nil
    })
    if err != nil {
        return nil, ErrInvalidToken
    }
    claims, ok := token.Claims.(*CustomClaims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    return claims, nil
}

func (s *JWTService) RefreshToken(oldToken string) (string, error) {
    claims, err := s.ParseToken(oldToken)
    if err != nil {
        return "", err
    }
    return s.GenerateToken(claims)
}
```

- [ ] **步骤 4：Commit**

```bash
git add templates/pkg/jwt/
git commit -m "feat: add pkg/jwt templates"
```

---

## 任务 3：创建 pkg/ratelimit 模板

**文件：**
- 创建：`templates/pkg/ratelimit/types.go.tmpl`
- 创建：`templates/pkg/ratelimit/ratelimit.go.tmpl`

- [ ] **步骤 1：创建 types.go.tmpl**

```go
package ratelimit

import "time"

type RateLimiter interface {
    Allow() bool
}

type Config struct {
    QPS   int
    Burst int
}
```

- [ ] **步骤 2：创建 ratelimit.go.tmpl**

```go
package ratelimit

import (
    "context"
    "time"

    "golang.org/x/time/rate"
)

type TokenBucketLimiter struct {
    limiter *rate.Limiter
}

func New(qps int, burst int) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        limiter: rate.NewLimiter(rate.Limit(qps), burst),
    }
}

func (l *TokenBucketLimiter) Allow() bool {
    return l.limiter.Allow()
}

func (l *TokenBucketLimiter) AllowN(ctx context.Context, n int) bool {
    return l.limiter.AllowN(time.Now(), time.Second, int64(n))
}
```

- [ ] **步骤 3：Commit**

```bash
git add templates/pkg/ratelimit/
git commit -m "feat: add pkg/ratelimit templates"
```

---

## 任务 4：创建 pkg/blacklist 模板

**文件：**
- 创建：`templates/pkg/blacklist/types.go.tmpl`
- 创建：`templates/pkg/blacklist/blacklist.go.tmpl`

- [ ] **步骤 1：创建 types.go.tmpl**

```go
package blacklist

type Config struct {
    Mode string
    IPs  []string
}
```

- [ ] **步骤 2：创建 blacklist.go.tmpl**

```go
package blacklist

import (
    "net"
    "networks"
)

type Blacklist interface {
    IsBlocked(ip string) bool
}

type IPBlacklist struct {
    networks *networks.Networks
}

func New(ipList []string) (*IPBlacklist, error) {
    return &IPBlacklist{
        networks: networks.MustParseCIDR(ipList),
    }, nil
}

func (b *IPBlacklist) IsBlocked(ip string) bool {
    return b.networks.Contains(net.ParseIP(ip))
}
```

- [ ] **步骤 3：Commit**

```bash
git add templates/pkg/blacklist/
git commit -m "feat: add pkg/blacklist templates"
```

---

## 任务 5：创建 BFF middleware Builder 模板

**文件：**
- 创建：`templates/micro-app/bff/middleware/middleware.go.tmpl`
- 创建：`templates/micro-app/bff/middleware/gin_jwt.go.tmpl`
- 创建：`templates/micro-app/bff/middleware/gin_ratelimit.go.tmpl`
- 创建：`templates/micro-app/bff/middleware/gin_blacklist.go.tmpl`
- 创建：`templates/micro-app/bff/middleware/hertz_jwt.go.tmpl`
- 创建：`templates/micro-app/bff/middleware/hertz_ratelimit.go.tmpl`
- 创建：`templates/micro-app/bff/middleware/hertz_blacklist.go.tmpl`

- [ ] **步骤 1：创建 middleware.go.tmpl（Builder 入口）**

```go
package middleware

import (
    "{{.AppName}}/pkg/jwt"
    "{{.AppName}}/pkg/ratelimit"
    "{{.AppName}}/pkg/blacklist"
)

type Middleware struct {
    jwt        *jwt.JWTService
    ratelimit  *ratelimit.TokenBucketLimiter
    blacklist  *blacklist.IPBlacklist
}

type Config struct {
    Order      []string
    JWT        *jwt.Config
    RateLimit  *ratelimit.Config
    Blacklist  *blacklist.Config
}

func New(cfg *Config) (*Middleware, error) {
    m := &Middleware{}

    for _, name := range cfg.Order {
        switch name {
        case "jwt":
            if cfg.JWT != nil {
                m.jwt = jwt.New(cfg.JWT.Secret,
                    jwt.WithExpiry(cfg.JWT.Expiry),
                    jwt.WithIssuer(cfg.JWT.Issuer),
                )
            }
        case "ratelimit":
            if cfg.RateLimit != nil {
                m.ratelimit = ratelimit.New(cfg.RateLimit.QPS, cfg.RateLimit.Burst)
            }
        case "blacklist":
            if cfg.Blacklist != nil {
                m.blacklist, _ = blacklist.New(cfg.Blacklist.IPs)
            }
        }
    }

    return m, nil
}
```

- [ ] **步骤 2：创建 gin_jwt.go.tmpl**

```go
package middleware

import (
    "strings"

    "{{.AppName}}/pkg/jwt"
    "github.com/gin-gonic/gin"
)

func JWTMiddleware(jwtSvc *jwt.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"code": 401, "message": "missing token"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtSvc.ParseToken(tokenString)
        if err != nil {
            c.JSON(401, gin.H{"code": 401, "message": "invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Next()
    }
}
```

- [ ] **步骤 3：创建 gin_ratelimit.go.tmpl**

```go
package middleware

import (
    "{{.AppName}}/pkg/ratelimit"
    "github.com/gin-gonic/gin"
)

func RateLimitMiddleware(rl *ratelimit.TokenBucketLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !rl.Allow() {
            c.JSON(429, gin.H{"code": 429, "message": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

- [ ] **步骤 4：创建 gin_blacklist.go.tmpl**

```go
package middleware

import (
    "{{.AppName}}/pkg/blacklist"
    "github.com/gin-gonic/gin"
)

func BlacklistMiddleware(bl *blacklist.IPBlacklist) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        if bl.IsBlocked(clientIP) {
            c.JSON(403, gin.H{"code": 403, "message": "ip blocked"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

- [ ] **步骤 5：创建 hertz_jwt.go.tmpl**

```go
package middleware

import (
    "context"
    "strings"

    "{{.AppName}}/pkg/jwt"
    "github.com/cloudwego/hertz/pkg/app"
)

func JWTMiddleware(jwtSvc *jwt.JWTService) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        authHeader := string(c.GetHeader("Authorization"))
        if authHeader == "" {
            c.JSON(401, map[string]interface{}{"code": 401, "message": "missing token"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtSvc.ParseToken(tokenString)
        if err != nil {
            c.JSON(401, map[string]interface{}{"code": 401, "message": "invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Next(ctx)
    }
}
```

- [ ] **步骤 6：创建 hertz_ratelimit.go.tmpl**

```go
package middleware

import (
    "context"

    "{{.AppName}}/pkg/ratelimit"
    "github.com/cloudwego/hertz/pkg/app"
)

func RateLimitMiddleware(rl *ratelimit.TokenBucketLimiter) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        if !rl.Allow() {
            c.JSON(429, map[string]interface{}{"code": 429, "message": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next(ctx)
    }
}
```

- [ ] **步骤 7：创建 hertz_blacklist.go.tmpl**

```go
package middleware

import (
    "context"

    "{{.AppName}}/pkg/blacklist"
    "github.com/cloudwego/hertz/pkg/app"
)

func BlacklistMiddleware(bl *blacklist.IPBlacklist) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        clientIP := string(c.ClientIP())
        if bl.IsBlocked(clientIP) {
            c.JSON(403, map[string]interface{}{"code": 403, "message": "ip blocked"})
            c.Abort()
            return
        }
        c.Next(ctx)
    }
}
```

- [ ] **步骤 8：Commit**

```bash
git add templates/micro-app/bff/middleware/
git commit -m "feat: add BFF middleware templates"
```

---

## 任务 6：创建 SRV interceptor Builder 模板

**文件：**
- 创建：`templates/micro-app/srv/interceptor/interceptor.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/grpc_ratelimit.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/grpc_retry.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/grpc_timeout.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/grpc_tracing.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/kitex_ratelimit.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/kitex_retry.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/kitex_timeout.go.tmpl`
- 创建：`templates/micro-app/srv/interceptor/kitex_tracing.go.tmpl`

- [ ] **步骤 1：创建 interceptor.go.tmpl（Builder 入口）**

```go
package interceptor

import (
    "{{.AppName}}/pkg/ratelimit"
    "golang.org/x/time/rate"
    "google.golang.org/grpc"
)

type Interceptor struct {
    ratelimit *ratelimit.TokenBucketLimiter
}

type Config struct {
    Order      []string
    RateLimit  *ratelimit.Config
}

func New(cfg *Config) (*Interceptor, error) {
    i := &Interceptor{}

    for _, name := range cfg.Order {
        switch name {
        case "ratelimit":
            if cfg.RateLimit != nil {
                i.ratelimit = ratelimit.New(cfg.RateLimit.QPS, cfg.RateLimit.Burst)
            }
        }
    }

    return i, nil
}

func (i *Interceptor) UnaryServer() []grpc.UnaryServerInterceptor {
    var interceptors []grpc.UnaryServerInterceptor

    for _, name := range []string{"ratelimit", "retry", "timeout", "tracing"} {
        switch name {
        case "ratelimit":
            interceptors = append(interceptors, i.ratelimitInterceptor())
        case "retry":
            interceptors = append(interceptors, i.retryInterceptor())
        case "timeout":
            interceptors = append(interceptors, i.timeoutInterceptor())
        case "tracing":
            interceptors = append(interceptors, i.tracingInterceptor())
        }
    }

    return interceptors
}
```

- [ ] **步骤 2：创建 grpc_ratelimit.go.tmpl**

```go
package interceptor

import (
    "{{.AppName}}/pkg/ratelimit"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func (i *Interceptor) ratelimitInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx interface{}, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if i.ratelimit != nil && !i.ratelimit.Allow() {
            return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
        }
        return handler(ctx, req)
    }
}
```

- [ ] **步骤 3：创建 grpc_retry.go.tmpl**

```go
package interceptor

import (
    "context"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type retryConfig struct {
    maxAttempts int
    backoff    time.Duration
}

func (i *Interceptor) retryInterceptor() grpc.UnaryServerInterceptor {
    cfg := &retryConfig{maxAttempts: 3, backoff: 100 * time.Millisecond}

    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        var lastErr error
        for attempt := 0; attempt < cfg.maxAttempts; attempt++ {
            if attempt > 0 {
                time.Sleep(cfg.backoff * time.Duration(attempt))
            }
            resp, err := handler(ctx, req)
            if err == nil {
                return resp, nil
            }
            if !isRetryable(err) {
                return nil, err
            }
            lastErr = err
        }
        return nil, lastErr
    }
}

func isRetryable(err error) bool {
    st, ok := status.FromError(err)
    if !ok {
        return false
    }
    switch st.Code() {
    case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted:
        return true
    default:
        return false
    }
}
```

- [ ] **步骤 4：创建 grpc_timeout.go.tmpl**

```go
package interceptor

import (
    "context"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type timeoutConfig struct {
    defaultTimeout time.Duration
}

func (i *Interceptor) timeoutInterceptor() grpc.UnaryServerInterceptor {
    cfg := &timeoutConfig{defaultTimeout: 500 * time.Millisecond}

    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        ctx, cancel := context.WithTimeout(ctx, cfg.defaultTimeout)
        defer cancel()

        resp, err := handler(ctx, req)
        if err != nil {
            if ctx.Err() == context.DeadlineExceeded {
                return nil, status.Error(codes.DeadlineExceeded, "timeout exceeded")
            }
        }
        return resp, err
    }
}
```

- [ ] **步骤 5：创建 grpc_tracing.go.tmpl**

```go
package interceptor

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

func (i *Interceptor) tracingInterceptor() grpc.UnaryServerInterceptor {
    tracer := otel.Tracer("")
    propagator := otel.GetTextMapPropagator()

    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if ok {
            ctx = propagator.Extract(ctx, propagation.Metadata(md))
        }

        spanName := info.FullMethod
        ctx, span := tracer.Start(ctx, spanName)
        defer span.End()

        resp, err := handler(ctx, req)
        if err != nil {
            span.RecordError(err)
        }
        return resp, err
    }
}
```

- [ ] **步骤 6-9：创建 Kitex 拦截器模板**

Kitex 使用 `endpoint.Middleware`，需要适配其接口。参考 grpc 拦截器模式创建对应的 kitex 版本。

- [ ] **步骤 10：Commit**

```bash
git add templates/micro-app/srv/interceptor/
git commit -m "feat: add SRV interceptor templates"
```

---

## 任务 7：创建 configs 模板

**文件：**
- 创建：`templates/configs/middleware.yaml.tmpl`
- 创建：`templates/configs/interceptor.yaml.tmpl`

- [ ] **步骤 1：创建 middleware.yaml.tmpl**

```yaml
order:
  - jwt
  - ratelimit
  - blacklist

jwt:
  secret: "your-secret-key-change-in-production"
  expiry: "24h"
  issuer: "{{.AppName}}"

ratelimit:
  qps: 100
  burst: 200

blacklist:
  mode: "block"
  ips:
    - "192.168.1.1"
    - "10.0.0.0/8"
```

- [ ] **步骤 2：创建 interceptor.yaml.tmpl**

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
  backoff: "100ms"

timeout:
  default: "500ms"

tracing:
  serviceName: "{{.Module}}"
```

- [ ] **步骤 3：Commit**

```bash
git add templates/configs/
git commit -m "feat: add middleware and interceptor config templates"
```

---

## 任务 8：修改 CLI 支持 --middleware flag

**文件：**
- 修改：`internal/cli/microapp_new.go`

- [ ] **步骤 1：添加 microAppMiddleware 变量**

在文件顶部添加：
```go
var microAppMiddleware string
```

- [ ] **步骤 2：在 init() 中添加 flag**

```go
newMicroAppCmd.Flags().BoolVar(&microAppMiddleware, "middleware", false, "生成中间件（jwt,ratelimit,blacklist）")
```

- [ ] **步骤 3：解析 middleware flag 为切片**

在 genMiddleware 函数开始处添加：
```go
middlewareList := strings.Split(microAppMiddleware, ",")
```

- [ ] **步骤 4：修改 genMiddleware 函数逻辑**

根据 middlewareList 决定生成哪些中间件文件：
```go
for _, m := range middlewareList {
    switch m {
    case "jwt":
        // 生成 jwt 相关文件
    case "ratelimit":
        // 生成 ratelimit 相关文件
    case "blacklist":
        // 生成 blacklist 相关文件
    }
}
```

- [ ] **步骤 5：生成 middleware.yaml**

在 genMiddleware 函数中添加生成 configs/middleware.yaml 的逻辑

- [ ] **步骤 6：Commit**

```bash
git add internal/cli/microapp_new.go
git commit -m "feat: add --middleware flag support"
```

---

## 任务 9：端到端测试

- [ ] **步骤 1：生成测试项目**

```bash
go run . micro --name testshop --output /tmp/testshop --bff h5 --modules product --http gin --middleware jwt,ratelimit,blacklist --test
```

- [ ] **步骤 2：验证生成的文件结构**

```
/tmp/testshop/
├── pkg/
│   ├── jwt/
│   ├── ratelimit/
│   └── blacklist/
├── bffH5/
│   └── internal/
│       └── middleware/
│           ├── middleware.go
│           ├── gin_jwt.go
│           ├── gin_ratelimit.go
│           └── gin_blacklist.go
└── configs/
    └── middleware.yaml
```

- [ ] **步骤 3：验证项目编译**

```bash
cd /tmp/testshop && go build ./...
```

- [ ] **步骤 4：Commit**

```bash
git add -A && git commit -m "test: verify middleware generation"
```

---

## 规格覆盖检查

| 需求 | 对应任务 |
|------|----------|
| pkg/jwt 模板 | 任务 2 |
| pkg/ratelimit 模板 | 任务 3 |
| pkg/blacklist 模板 | 任务 4 |
| BFF middleware Builder | 任务 5 |
| Gin 适配器 | 任务 5 |
| Hertz 适配器 | 任务 5 |
| SRV interceptor Builder | 任务 6 |
| gRPC 拦截器 | 任务 6 |
| Kitex 拦截器 | 任务 6 |
| middleware.yaml 配置 | 任务 7 |
| interceptor.yaml 配置 | 任务 7 |
| --middleware flag | 任务 8 |
| 端到端测试 | 任务 9 |
