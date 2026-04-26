# pkg/logger 使用指南

## 概述

`pkg/logger` 是基于 uber-go/zap 封装的企业级日志库，提供四类日志场景。

## 配置

日志配置在 `configs/log.yaml` 中：

```yaml
env: dev           # dev/staging/prod
level: info       # debug/info/warn/error

sampling:
  initial: 100    # 初始采样数
  thereafter: 200 # 后续采样数
  tick: 1s        # 采样周期

rotation:
  enabled: true   # 启用日志轮转
  max_age_days: 7  # 保留天数

output:
  file: ./logs/app.log  # 日志文件路径
  stdout: true         # 同时输出到 stdout

prometheus:
  enabled: false       # 启用 Prometheus 指标
  namespace: app
  subsystem: log

mq:
  enabled: false       # 启用 MQ 推送
  type: kafka
  brokers:
    - localhost:9092
  topic: app-logs
  async: true
  batch_size: 100
  flush_interval: 3s
```

## 初始化

```go
package main

import (
    "myshop/pkg/config"
    "myshop/pkg/logger"
    "myshop/configs"
)

func main() {
    // 加载应用配置
    cfg, _ := config.Load("configs/config.yaml")
    
    // 加载日志配置并初始化
    logCfg, _ := logger.LoadConfig("configs/log.yaml")
    log, err := logger.NewLogger(logCfg)
    if err != nil {
        panic(err)
    }
    defer log.Sync()
    
    // 使用日志
    log.Business.Infow("application started")
}
```

## 四类日志

| 日志类型 | 用途 | 方法 |
|----------|------|-------|
| Business | 业务逻辑 | Infow, Warnw, Errorw |
| Access | 访问记录 | Infow |
| Audit | 审计追踪 | Log |
| Error | 错误追踪 | Errorw, WithStack |

### Business 业务日志

```go
logger.Business.Infow("create order", "order_id", "12345", "amount", 99.99)
logger.Business.Warnw("库存不足", "product_id", "P001", "stock", 0)
logger.Business.Errorw("创建订单失败", "error", err.Error())
```

### Access 访问日志

```go
logger.Access.Infow("request",
    "method", c.Request.Method,
    "path", c.Request.URL.Path,
    "status", 200,
    "latency_ms", latency,
)
```

### Audit 审计日志

```go
logger.Audit.Log("create", userID, "order", map[string]interface{}{
    "order_id": "12345",
    "amount": 99.99,
})
```

### Error 错误日志

```go
logger.Error.Errorw("database error", "error", err.Error())

// 带堆栈
logger.Error.WithStack(err).Errorw("critical error")
```

## BFF 层集成示例

`bffH5/internal/handler/productHandler.go`:

```go
package handler

import (
    "net/http"
    "strconv"
    "time"
    "myshop/pkg/logger"
    "myshop/bffH5/internal/rpcClient"
    pb "myshop/common/kitexGen/product"

    "github.com/gin-gonic/gin"
)

type ProductHandler struct {
    client *rpcclient.ProductClient
}

func NewProductHandler() *ProductHandler {
    cli, err := rpcclient.NewProductClient("127.0.0.1:8001")
    if err != nil {
        logger.Business.Errorw("failed to create product client", "error", err.Error())
        panic("failed to create product client: " + err.Error())
    }
    return &ProductHandler{client: cli}
}

func (h *ProductHandler) Create(c *gin.Context) {
    start := time.Now()
    
    var req struct {
        // ... fields
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        logger.Business.Errorw("bind request failed", "error", err.Error())
        c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
        return
    }
    
    logger.Business.Infow("create product request",
        "store_name", req.StoreName,
        "price", req.Price,
    )
    
    resp, err := h.client.CreateProduct(c.Request.Context(), &pb.CreateRequest{})
    if err != nil {
        logger.Business.Errorw("create product failed", "error", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
        return
    }
    
    logger.Business.Infow("create product success",
        "id", resp.Id,
        "duration_ms", time.Since(start).Milliseconds(),
    )
    
    logger.Audit.Log("create", "user_id", "product", map[string]interface{}{
        "id": resp.Id,
    })
    
    c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) List(c *gin.Context) {
    start := time.Now()
    
    // Access 日志
    logger.Access.Infow("list request",
        "method", c.Request.Method,
        "path", c.Request.URL.Path,
        "query", c.Request.URL.Query(),
    )
    
    resp, err := h.client.ListProduct(c.Request.Context(), &pb.ListRequest{})
    if err != nil {
        logger.Error.Errorw("list product failed", "error", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
        return
    }
    
    logger.Business.Infow("list product success",
        "count", len(resp.List),
        "duration_ms", time.Since(start).Milliseconds(),
    )
    
    c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) Get(c *gin.Context) {
    id := c.Param("id")
    productID, _ := strconv.ParseInt(id, 10, 64)
    
    logger.Business.Infow("get product",
        "id", productID,
    )
    
    resp, err := h.client.GetProduct(c.Request.Context(), &pb.GetRequest{Id: productID})
    if err != nil {
        logger.Business.Errorw("get product failed", "id", productID, "error", err.Error())
        c.JSON(http.StatusNotFound, gin.H{"msg": "product not found"})
        return
    }
    
    c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) Update(c *gin.Context) {
    id := c.Param("id")
    productID, _ := strconv.ParseInt(id, 10, 64)
    
    var req struct {
        StoreName string  `json:"store_name"`
        Price    float64 `json:"price"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        logger.Business.Errorw("bind request failed", "error", err.Error())
        c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
        return
    }
    
    logger.Business.Infow("update product",
        "id", productID,
        "store_name", req.StoreName,
        "price", req.Price,
    )
    
    resp, err := h.client.UpdateProduct(c.Request.Context(), &pb.UpdateRequest{
        Id: productID,
        // ...
    })
    if err != nil {
        logger.Business.Errorw("update product failed", "error", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
        return
    }
    
    logger.Audit.Log("update", "user_id", "product", map[string]interface{}{
        "id": productID,
        "store_name": req.StoreName,
    })
    
    c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    productID, _ := strconv.ParseInt(id, 10, 64)
    
    logger.Business.Infow("delete product", "id", productID)
    
    _, err := h.client.DeleteProduct(c.Request.Context(), &pb.DeleteRequest{Id: productID})
    if err != nil {
        logger.Business.Errorw("delete product failed", "error", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
        return
    }
    
    logger.Audit.Log("delete", "user_id", "product", map[string]interface{}{
        "id": productID,
    })
    
    c.JSON(http.StatusOK, gin.H{})
}
```

## SRV 层集成示例

`srvProduct/internal/handler/productHandler.go`:

```go
package handler

import (
    "context"
    "myshop/pkg/logger"
    "myshop/srvProduct/internal/repository"
    "myshop/srvProduct/internal/service"

    pb "myshop/common/kitexGen/product"
)

type ProductHandler struct {
    repo *repository.ProductRepo
    svc  *service.ProductService
}

func NewProductHandler(db interface{}) *ProductHandler {
    repo := repository.NewProductRepo(db)
    return &ProductHandler{
        repo: repo,
        svc:  service.NewProductService(repo),
    }
}

func (h *ProductHandler) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
    logger.Business.Infow("handle create product",
        "store_name", req.StoreName,
        "price", req.Price,
    )
    
    resp, err := h.svc.Create(ctx, req)
    if err != nil {
        logger.Business.Errorw("create product failed", "error", err.Error())
        return nil, err
    }
    
    logger.Business.Infow("create product success", "id", resp.Id)
    return resp, nil
}

func (h *ProductHandler) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
    logger.Business.Infow("handle get product", "id", req.Id)
    
    product, err := h.repo.FindByID(ctx, req.Id)
    if err != nil {
        logger.Business.Errorw("get product failed", "id", req.Id, "error", err.Error())
        return nil, err
    }
    
    return &pb.GetResponse{Product: product}, nil
}
```

## Middleware 集成示例

`bffH5/internal/middleware/logger.go`:

```go
package middleware

import (
    "time"
    "myshop/pkg/logger"

    "github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
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

## 依赖

| 依赖 | 用途 |
|------|------|
| go.uber.org/zap | 日志库 |
| gopkg.in/yaml.v3 | 配置解析 |
| gopkg.in/natefinch/lumberjack.v2 | 日志轮转 |
| github.com/prometheus/client_golang | Prometheus 指标 |
| github.com/IBM/sarama | Kafka 客户端（MQ 功能）|