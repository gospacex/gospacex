# Logger BFF/SRV 层自动集成实现计划

> **面向 AI 代理的工作者：** 建议使用 subagent-driven-development 逐任务实现此计划。

**目标：** 在生成的 BFF/SRV 层代码中自动添加 logger 调用

**架构：** 修改 16 个模板文件，在 BFF handler/middleware/main 和 SRV handler/interceptor/main/repo 中添加 logger 调用

**技术栈：** Go, Uber Zap, Gin, Hertz, gRPC, Kitex

---

## 文件列表

### BFF 层 (Gin) - 4 个文件
- `templates/micro-app/bff/handler/handler.go.tmpl` - CRUD handler 模板
- `templates/micro-app/bff/middleware/gin_middleware.go.tmpl` - 中间件模板
- `templates/micro-app/bff/main/gin_main.go.tmpl` - main 模板
- `templates/micro-app/bff/main/gin_main_schema.go.tmpl` - 从 schema 生成 main

### BFF 层 (Hertz) - 5 个文件
- `templates/micro-app/bff/handler/handler_hertz.go.tmpl` - Hertz handler 模板
- `templates/micro-app/bff/handler/handler_hertz_crud.go.tmpl` - Hertz CRUD handler
- `templates/micro-app/bff/middleware/hertz_middleware.go.tmpl` - Hertz 中间件
- `templates/micro-app/bff/main/hertz_main.go.tmpl` - Hertz main
- `templates/micro-app/bff/main/hertz_main_schema.go.tmpl` - 从 schema 生成

### SRV 层 - 7 个文件
- `templates/micro-app/srv/handler/handler.go.tmpl` - SRV handler
- `templates/micro-app/srv/interceptor/grpc_interceptor.go.tmpl` - gRPC 拦截器
- `templates/micro-app/srv/interceptor/kitex_interceptor.go.tmpl` - Kitex 拦截器
- `templates/micro-app/srv/main/main_consul.go.tmpl` - Consul 注册
- `templates/micro-app/srv/main/main_direct.go.tmpl` - Direct 模式
- `templates/micro-app/srv/main/main_etcd.go.tmpl` - Etcd 注册
- `templates/micro-app/srv/repo/repository.go.tmpl` - Repository

---

## 任务 1：更新 BFF Gin Handler 模板

**文件：** `templates/micro-app/bff/handler/handler.go.tmpl`

- [ ] **步骤 1：读取现有模板**

```bash
cat templates/micro-app/bff/handler/handler.go.tmpl
```

- [ ] **步骤 2：修改 import 添加 logger**

```go
import (
	"net/http"
	"time"

	"{{.AppName}}/pkg/logger"
	"{{.AppName}}/{{.BffDirName}}/internal/rpcClient"
	pb "{{.AppName}}/common/kitexGen/{{.Module}}"

	"github.com/gin-gonic/gin"
)
```

- [ ] **步骤 3：修改 NewXxxHandler 添加初始化日志**

```go
func New{{.UpperName}}Handler() *{{.UpperName}}Handler {
	cli, err := rpcclient.New{{.UpperName}}Client("127.0.0.1:{{.SrvPort}}")
	if err != nil {
		logger.Business.Errorw("failed to create {{.LowerName}} client", "error", err.Error())
		panic("failed to create {{.LowerName}} client: " + err.Error())
	}
	logger.Business.Infow("{{.UpperName}}Handler initialized", "addr", "127.0.0.1:{{.SrvPort}}")
	return &{{.UpperName}}Handler{client: cli}
}
```

- [ ] **步骤 4：修改 Create 方法添加日志**

```go
func (h *{{.UpperName}}Handler) Create(c *gin.Context) {
	start := time.Now()
	
	// ... request binding ...
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Business.Errorw("bind create {{.LowerName}} request failed", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Business.Infow("create {{.LowerName}} request",
		// ... fields ...
	)

	// ... RPC call ...
	if err != nil {
		logger.Business.Errorw("create {{.LowerName}} failed", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Business.Infow("create {{.LowerName}} success",
		"id", resp.Id,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	logger.Audit.Log("create", "system", "{{.LowerName}}", map[string]interface{}{
		"id": resp.Id,
	})

	c.JSON(http.StatusOK, gin.H{"id": resp.Id})
}
```

- [ ] **步骤 5：修改 Get/List/Update/Delete 方法添加日志**（同上模式）

- [ ] **步骤 6：验证修改**

```bash
head -20 templates/micro-app/bff/handler/handler.go.tmpl
```

---

## 任务 2：更新 BFF Gin Middleware 模板

**文件：** `templates/micro-app/bff/middleware/gin_middleware.go.tmpl`

- [ ] **步骤 1：读取现有模板**

- [ ] **步骤 2：添加 AccessLogger 中间件**

```go
func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		
		logger.Access.Infow("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", latency,
			"client_ip", c.ClientIP(),
		)
		
		if status >= 500 {
			logger.Error.Errorw("server error",
				"status", status,
				"path", c.Request.URL.Path,
			)
		}
	}
}
```

- [ ] **步骤 3：验证**

---

## 任务 3：更新 BFF Gin Main 模板

**文件：** 
- `templates/micro-app/bff/main/gin_main.go.tmpl`
- `templates/micro-app/bff/main/gin_main_schema.go.tmpl`

- [ ] **步骤 1：添加 logger 初始化**

```go
	// 初始化日志
	logCfg, err := logger.LoadConfig("configs/log.yaml")
	if err != nil {
		log.Fatalf("Failed to load log config: %v", err)
	}
	log, err := logger.NewLogger(logCfg)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Sync()
	
	logger.Business.Infow("BFF service starting",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)
```

---

## 任务 4-9：BFF Hertz 层（与 Gin 类似结构）

使用相同模式修改：
- handler_hertz.go.tmpl
- handler_hertz_crud.go.tmpl
- hertz_middleware.go.tmpl
- hertz_main.go.tmpl
- hertz_main_schema.go.tmpl

---

## 任务 10-16：SRV 层

### SRV Handler
- `templates/micro-app/srv/handler/handler.go.tmpl`

```go
// 在每个方法中添加
logger.Business.Infow("handle create {{.LowerName}}", /* fields */)
logger.Business.Errorw("create failed", "error", err.Error())
```

### SRV Interceptor
- `templates/micro-app/srv/interceptor/grpc_interceptor.go.tmpl`
- `templates/micro-app/srv/interceptor/kitex_interceptor.go.tmpl`

```go
logger.Access.Infow("gRPC call",
	"method", method,
	"status", status,
	"latency_ms", latency,
)
```

### SRV Main
- main_consul.go.tmpl, main_direct.go.tmpl, main_etcd.go.tmpl

添加 logger 初始化代码

### SRV Repository
- `templates/micro-app/srv/repo/repository.go.tmpl`

```go
logger.Business.Infow("query {{.LowerName}}", /* fields */)
logger.Business.Errorw("query failed", "error", err.Error())
```

---

## 验证步骤

全部修改完成后：

```bash
# 重新生成项目
cd /Users/hyx/work/gowork/src/gpx && rm -rf output/test && go run . micro --name test --output output --bff h5 --modules product --db-host 127.0.0.1 --db-port 3306 --db-user root --db-password 123456 --db-name testdb --db-table products --test

# 检查生成的代码是否包含 logger
grep -r "logger\." output/test/bffH5/internal/handler/
grep -r "logger\." output/test/bffH5/internal/middleware/
grep -r "logger\." output/test/srvProduct/internal/handler/

# 编译测试
cd output/test && go build ./...
```

---

## 自检清单

- [ ] ��有 16 个模板文件都已修改
- [ ] import 语句包含 logger 包
- [ ] 每个方法都有对应的日志调用
- [ ] 生成的代码可以编译通过
- [ ] 四类日志（Business/Access/Audit/Error）都有使用