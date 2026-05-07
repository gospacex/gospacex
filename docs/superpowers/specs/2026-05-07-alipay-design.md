# 支付宝对接功能设计文档

## 概述

在 gpx 脚手架生成器中添加 `--play alipay` 参数，支持生成带支付宝支付功能的微服务项目。支付宝功能将注入到 `--srvs` 指定的服务模块（如 product）中，在 SRV 层和 BFF 层分别实现业务逻辑和 HTTP 接口。

## 1. CLI 参数扩展

**目标：** 在 `go run micro` 命令中添加 `--play alipay` 参数

**设计：**
- 在 `internal/cli/` 的 micro 命令中添加 `--play` 参数（类型：string，支持逗号分隔多个支付平台）
- 参数解析后，将 `Alipay bool` 和 `AlipayConfig` 传递到 generator
- `AlipayConfig` 包含：AppID、PrivateKey、PublicKey（从配置文件读取，生成时写入模板）

**影响文件：**
- `internal/cli/micro.go` — 添加 Cobra 参数定义
- `internal/generator/microapp_generator.go` — 接收参数并传递到模板上下文

## 2. SRV 层 + IDL（proto）设计

**SRV 层（product 服务）改动：**

1. **新增文件：**
   - `internal/alipay/config.go` — 支付宝配置（AppID、私钥、公钥）
   - `internal/alipay/service.go` — 业务逻辑（生成支付链接、处理回调）

2. **proto 定义（若项目使用 Protobuf）：**
   在 `api/alipay/` 下新增 `alipay.proto`：
   ```protobuf
   service AlipayService {
     rpc GeneratePayURL(GeneratePayURLRequest) returns (GeneratePayURLResponse);
     rpc HandleNotify(HandleNotifyRequest) returns (HandleNotifyResponse);
   }

   message GeneratePayURLRequest {
     string order_id = 1;
     string total_amount = 2;
     string subject = 3;
   }

   message GeneratePayURLResponse {
     string pay_url = 1;
   }

   message HandleNotifyRequest {
     map<string, string> params = 1;
   }

   message HandleNotifyResponse {
     string status = 1;
   }
   ```

3. **业务逻辑（service.go）：**
   - `GeneratePayURL`：调用 `gopay/alipay` SDK 发起 `TradePagePay` 请求，返回支付 URL
   - `HandleNotify`：验证签名、解析通知、更新订单状态

**影响文件：**
- `api/alipay/alipay.proto` — 新增（若用 Protobuf）
- `internal/alipay/` — 新增目录和文件
- `internal/service/order.go` — 注入 AlipayService

## 3. BFF 层设计

**BFF 层（h5）改动：**

1. **新增/修改文件：**
   - `internal/handler/orderHandler.go` — 处理 HTTP 请求，直接调用 rpcClient
   - `internal/rpcClient/orderRpcClient.go` — RPC 客户端，调用 SRV 层的 AlipayService

2. **HTTP 接口定义：**
   ```go
   // POST /api/order/pay
   // 入参：{ "orderId": "xxx", "totalAmount": "100.00", "subject": "订单标题" }
   // 返回：{ "payUrl": "https://openapi.alipay.com/..." }

   // POST /api/order/notify
   // 入参：支付宝回调参数（form-data）
   // 返回：字符串 "success" 或 "fail"
   ```

3. **调用链：**
   - `OrderHandler.GeneratePayURL` → `OrderRpcClient.GeneratePayURL()` → SRV `AlipayService`
   - `OrderHandler.HandleNotify` → `OrderRpcClient.HandleNotify()` → SRV `AlipayService`

4. **路由注册：**
   - `POST /api/order/pay` → `OrderHandler.GeneratePayURL`
   - `POST /api/order/notify` → `OrderHandler.HandleNotify`

## 4. 配置管理

**方式：** 使用配置文件（参考 go-alipay 的 config.go）

**配置文件位置：** `config/alipay.yaml`（生成到项目的 config/ 目录）

**配置内容：**
```yaml
alipay:
  app_id: "2021000xxxxx"
  private_key: "MIIEpAIBAAKCAQEA..."
  public_key: "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A..."
  notify_url: "https://myshop.com/api/order/notify"  # SDK使用的回调地址，需在支付宝商户平台配置相同地址
```

**同步返回地址（return_url）：** 不传递，使用支付宝默认行为（用户支付完成后停留在支付宝页面）

## 5. 测试设计

### 单元测试

1. **SRV 层测试：**
   - `internal/alipay/service_test.go`
     - `TestGeneratePayURL` — mock `gopay/alipay` SDK，验证生成的支付链接格式
     - `TestHandleNotify` — 模拟支付宝回调参数，验证签名验证和订单状态更新

2. **BFF 层测试：**
   - `internal/handler/orderHandler_test.go`
     - `TestGeneratePayURLHandler` — mock `OrderRpcClient`，验证 HTTP 接口返回正确的 `payUrl`
     - `TestHandleNotifyHandler` — 模拟支付宝 form-data 回调，验证返回 "success"

### 端到端测试（E2E）

- `tests/alipay/pay_test.go`
  - 启动测试环境（MySQL、SRV、BFF）
  - 调用 `POST /api/order/pay` 接口，验证返回支付链接
  - 模拟支付宝回调 `POST /api/order/notify`，验证订单状态更新
  - 使用 `httptest.NewRecorder()` 模拟 HTTP 请求

### 测试依赖

- `github.com/stretchr/testify` — 断言库
- `github.com/golang/mock` 或 `github.com/uber-go/mock` — mock 生成

## 6. 实现步骤

1. 在 `internal/cli/micro.go` 添加 `--play` 参数
2. 在 `templates/microservice/` 添加支付宝相关模板文件
3. 在 `templates/micro-bff/` 添加 BFF 层模板文件
4. 修改 `internal/generator/microapp_generator.go` 支持支付宝代码注入
5. 编写单元测试和 E2E 测试
6. 验证生成的项目可以正常编译和运行

## 7. 参考资料

- 参考项目：`/Users/hyx/work/gowork/src/gospacex/go-alipay`
- SDK 文档：`github.com/go-pay/gopay/alipay`
- 支付宝开放平台：https://open.alipay.com/
