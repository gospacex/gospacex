# gpx micro 命令使用手册

> 命令别名：`micro-app` / `micro`
>
> 用于生成完整的微服务项目（BFF 层 + 多个微服务）

---

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
  - [交互式模式](#交互式模式)
  - [命令行模式](#命令行模式)
- [参数详解](#参数详解)
  - [必填参数](#必填参数)
  - [可选参数](#可选参数)
  - [数据库参数](#数据库参数)
  - [联表查询参数](#联表查询参数)
  - [进阶功能参数](#进阶功能参数)
  - [配置文件参数](#配置文件参数)
- [生成的项目结构](#生成的项目结构)
- [功能特性详解](#功能特性详解)
  - [F1：基础微服务项目生成](#f1基础微服务项目生成)
  - [F2：数据库反向工程](#f2数据库反向工程)
  - [F3：联表查询代码生成](#f3联表查询代码生成)
  - [F4：IDL 类型支持](#f4idl-类型支持)
  - [F5：HTTP 框架选择](#f5http-框架选择)
  - [F6：中间件/拦截器生成](#f6中间件拦截器生成)
  - [F7：Nacos 配置中心集成](#f7nacos-配置中心集成)
  - [F8：测试代码生成](#f8测试代码生成)
  - [F9：服务注册中心](#f9服务注册中心)
  - [F10：配置文件驱动生成](#f10配置文件驱动生成)
- [使用示例](#使用示例)
- [相关子命令](#相关子命令)
  - [micro-bff：为已有项目添加 BFF 层](#micro-bff为已有项目添加-bff-层)
  - [gen-proto：从数据库表生成 Proto 文件](#gen-proto从数据库表生成-proto-文件)
  - [gen-grpc：从数据库表生成完整 gRPC 代码](#gen-grpc从数据库表生成完整-grpc-代码)

---

## 概述

`micro`（`micro-app`）是 gpx 的核心命令，用于一键生成包含 **BFF 层**和多个**微服务**的完整 Go 项目骨架。

生成项目的技术栈：

| 层次 | 默认技术 | 可选技术 |
|------|----------|----------|
| BFF HTTP 框架 | Gin | Hertz |
| 微服务通信协议 | gRPC | Kitex |
| IDL 描述语言 | Protobuf | Thrift |
| 服务注册中心 | 直连（无） | Consul、etcd |
| 配置中心 | 本地 YAML | Nacos |

---


## 快速开始

### 安装

go install github.com/gospacex/gpx@latest

### 交互式模式

直接运行 `micro` 命令（不带任何参数）进入交互式向导：

```bash
go run . micro
│
├─ 有参数 ──→ 直接处理参数，跳过交互
│
└─ 无参数 ──→ 显示选项：A. 默认标准微服 / B. 自定义选型

```

无参数向导模式分两种模式：

**A. 默认标准模式**（快速生成，使用默认配置）

```
请选择生成模式：
  A. 默认标准微服（快速生成，使用默认配置）
  B. 自定义配置（DIY 配置看板）

请输入选项 [A/B]: A
```

默认配置：`standard` 架构 + `proto` IDL + `sql` + `cache`，不启用注册中心/配置中心。

**B. DIY 配置看板**（完整交互式配置）

依次配置 9 个阶段：

| 阶段 | 内容 |
|------|------|
| 阶段 1 | 微服务类型（standard/ddd/serviceMesh）、IDL 类型（proto/thrift） |
| 阶段 2 | 项目名称、BFF 名称、模块列表 |
| 阶段 3 | 数据存储选型（sql/cache/nosql/es/mq，多选） |
| 阶段 4 | 注册中心（nacos/consul/etcd/zookeeper/polaris） |
| 阶段 5 | 配置中心（nacos/apollo/consul/etcd/zookeeper） |
| 阶段 6 | SQL 配置（mysql/pg，主机/端口/用户/密码/数据库/表） |
| 阶段 7 | Cache 配置（redis/memcached/dragonfly/keydb） |
| 阶段 8 | MQ 配置（rabbitmq/rocketmq/kafka/pulsar/redis-stream） |
| 阶段 9 | 进阶特性（DTM 分布式事务、调用链追踪、测试代码） |

```aiignore
hyx ~/work/gowork/src/gpx [dev] ➜ go run . micro
╔══════════════════════════════════════════════════════════════════╗
║                    欢迎使用微应用生成器                              ║
║              GoSpaceX Micro-App Generator v1.0                    ║
╚══════════════════════════════════════════════════════════════════╝

请选择生成模式：
A. 默认标准微服（快速生成，使用默认配置）
B. 自定义配置（DIY 配置看板）

请输入选项 [A/B]: B

╔══════════════════════════════════════════════════════════════════╗
║                        DIY 微服务配置看板                           ║
╚══════════════════════════════════════════════════════════════════╝

【阶段 1】基础信息
微服务类型:
1. standard
2. ddd
3. serviceMesh
请选择 [默认: standard]:

```


### 命令行模式

```bash
go run . micro \
--name myshop \
--output output \
--bff h5 \
--modules product \
--db-host 127.0.0.1 \
--db-port 3306 \
--db-user root \
--db-password 123456 \
--db-name gospacex \
--db-table eb_store_product,eb_store_product_description,eb_store_product_attr \
--test
--otel
-----------------------
添加入参--otel 开启微服调用链，否则生成的项目pkg里有调用链逻辑但生成的bff和srv的业务代码里不要有调用链嵌入
生成项目确认日志功能是否正常
-----------------------
使用superpowers TDD驱动开发


go run . micro --name myshop --output output --bff h5 --modules product --db-host 127.0.0.1 --db-port 3306 --db-user root --db-password 123456 --db-name gospacex --db-table eb_store_product --test

gpx micro --name myshop --output output --bff h5 --modules product --db-host 127.0.0.1 --db-port 3306 --db-user root --db-password 123456 --db-name gospacex --db-table eb_store_product --test

go run . micro-app --name myshop --output output --bff h5 --modules product --db-host 127.0.0.1 --db-port 3306 --db-user root --db-password 123456 --db-name gospacex --db-table eb_store_product --test


```

---

## 参数详解

### 必填参数

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--name` | — | 微应用名称（项目名） | `--name myShop` |
| `--output` | `-o` | 项目输出目录 | `--output ./output` |
| `--bff` | — | BFF 层名称 | `--bff h5` |
| `--modules` | — | 微服务列表，可多次指定或用英文/中文逗号分隔 | `--modules product,order,user` |

> **注意**：`--srvs` 是 `--modules` 的别名，两者可混用，效果叠加。

### 可选参数

| 参数 | 默认值 | 说明 | 可选值 |
|------|--------|------|--------|
| `--style` | `standard` | 微服务架构风格 | `standard` |
| `--idl` | `proto` | IDL 描述语言 | `proto`、`thrift` |
| `--protocol` | `grpc` | 微服务通信协议 | `grpc`、`kitex` |
| `--http` | `gin` | BFF HTTP 框架 | `gin`、`hertz` |
| `--register` | —（直连） | 服务注册中心 | `consul`、`etcd` |
| `--test` | `false` | 生成 BFF 和微服层的接口测试代码 | `true`/`false` |
| `--middleware` | `false` | 生成中间件（BFF）/拦截器（SRV） | `true`/`false` |
| `--config` | —（本地 YAML） | 配置中心类型 | `nacos`、`viper` |

### 数据库参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--db-host` | `127.0.0.1` | 数据库主机 |
| `--db-port` | `3306` | 数据库端口 |
| `--db-user` | `root` | 数据库用户名 |
| `--db-password` | `123456` | 数据库密码 |
| `--db-name` | `myshop` | 数据库名 |
| `--db-table` | —（不指定则不反向工程） | 表名，支持英文/中文逗号分隔多表 |

当指定 `--db-table` 时，工具会连接数据库读取表结构，并基于真实字段生成 Proto、Model、Repository、Service、Handler 等全套代码。

**表数与 module 数的对应关系：**

- 1 表 : 1 module → 一一对应
- N 表 : 1 module（N > 1）→ 第一张表作为主表，其余表生成附属 model 文件
- N 表 : N module → 按顺序一一对应

**`is_` 前缀字段自动映射为 `bool`：**

当字段名满足以下条件时，Proto 类型自动映射为 `bool`（仅对 `tinyint` 类型有效）：
- 以 `is_` 开头（如 `is_deleted`、`is_active`）
- 以 `_flag` 结尾
- 名称为 `enabled`、`disabled`、`active`、`deleted`、`status` 等布尔语义词

### 联表查询参数

用于生成多表 JOIN 查询代码，支持同时指定多组联表关系。

| 参数 | 简写 | 说明 | 格式 |
|------|------|------|------|
| `--db-join-condition` / `--db-join-key` / `--djc` | — | 联表条件（三者等价，可多次指定） | `table1.field1=table2.field2` |
| `--db-join-style` / `--djs` | — | 联表关系类型（与 condition 一一对应） | `table1:table2=<style>` |

联表关系类型（`style`）：

| 值 | 含义 |
|----|------|
| `1t1` | 一对一 |
| `1tn` | 一对多 |
| `nt1` | 多对一 |
| `ntn` | 多对多 |

> `--db-join-condition` 和 `--db-join-style` 必须数量一致，一一对应。

### 进阶功能参数

| 参数 | 说明 |
|------|------|
| `--srvs` | `--modules` 的别名，两者效果叠加 |
| `--test` | 同时生成 BFF 层和微服层的接口测试代码，以及 Shell 脚本 |
| `--middleware` | BFF 生成 middleware，SRV 生成 interceptor |
| `--config nacos` | 在 `pkg/config/` 生成 Nacos 支持代码，并更新各服务 `config.yaml` |

### 配置文件参数

| 参数 | 说明 | 支持格式 |
|------|------|----------|
| `--config-file` | 从配置文件读取所有参数，文件中字段会覆盖命令行同名参数 | `yaml`、`json`、`toml` |

配置文件结构（YAML 示例）：

```yaml
name: myShop
output: ./output
bff: h5
modules:
  - product
  - order
  - user
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: secret
  name: myshop
  tables:
    - eb_product
    - eb_order
```

---

## 生成的项目结构

```
<output>/<name>/
├── bff<BFF>/                     # BFF 层（如 bffH5）
│   ├── cmd/
│   │   └── main.go               # 入口
│   ├── configs/
│   │   └── config.yaml           # 配置文件
│   ├── internal/
│   │   ├── handler/              # HTTP Handler（每个 module 一个文件）
│   │   ├── middleware/           # 中间件（--middleware 时生成）
│   │   ├── rpcClient/            # gRPC/Kitex 客户端
│   │   └── router/               # 路由注册
│   └── test/                     # 接口测试（--test 时生成）
│
├── srv<Module>/                  # 微服务（如 srvProduct、srvOrder）
│   ├── cmd/
│   │   └── main.go
│   ├── configs/
│   │   └── config.yaml
│   ├── internal/
│   │   ├── handler/              # gRPC/Kitex Handler
│   │   ├── interceptor/          # 拦截器（--middleware 时生成）
│   │   ├── model/                # 数据模型（基于表结构时包含 GORM 字段）
│   │   ├── repository/           # 数据访问层
│   │   └── service/              # 业务逻辑层
│   └── test/                     # 单元测试（--test 时生成）
│
├── common/
│   ├── idl/
│   │   └── <module>.proto        # Proto/Thrift IDL 文件
│   ├── kitexGen/                 # 生成的 RPC 代码
│   ├── errors/                   # 错误码定义
│   └── constants/                # 常量定义
│
├── pkg/
│   ├── config/                   # 配置加载（含 Nacos 支持）
│   ├── database/                 # 数据库初始化
│   ├── logger/                   # 日志封装
│   └── utils/                    # 工具函数
│
├── tests/                        # Shell 接口测试脚本（--test 时生成）
├── scripts/                      # 启动/停止脚本
├── Makefile                      # 构建脚本
├── README.md                     # 项目说明
├── go.mod
└── .gitignore
```

---

## 功能特性详解

### F1：基础微服务项目生成

最小化命令，生成默认脚手架（无数据库反向工程）：

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product,order,user
```

生成内容：默认 Proto IDL（含基础 CRUD 接口）、BFF 路由/Handler 骨架、微服务 Handler/Service/Repository 骨架。

---

### F2：数据库反向工程

指定 `--db-table` 后，自动连接 MySQL 读取真实表结构，生成完整 CRUD 代码：

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --db-host 127.0.0.1 \
  --db-port 3306 \
  --db-user root \
  --db-password secret \
  --db-name myshop \
  --db-table eb_product
```

**多表反向工程（附属表）：**

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --db-name myshop \
  --db-table eb_product,eb_product_description,eb_product_attr
  # 3 张表 → 1 个 module：主表 eb_product + 2 个附属 model
```

**多模块多表反向工程（一一对应）：**

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product,order \
  --db-name myshop \
  --db-table eb_product,eb_order
  # 2 张表 ↔ 2 个 module
```

---

### F3：联表查询代码生成

在反向工程基础上，生成多表 JOIN 查询代码：

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --db-name myshop \
  --db-table eb_product,eb_product_description \
  --db-join-condition "eb_product.id=eb_product_description.product_id" \
  --db-join-style "eb_product:eb_product_description=1t1"
```

多组联表（每组 condition + style 一一对应）：

```bash
go run . micro \
  ... \
  --db-join-condition "order.user_id=user.id" \
  --db-join-condition "order.product_id=product.id" \
  --db-join-style "order:user=nt1" \
  --db-join-style "order:product=nt1"
```

---

### F4：IDL 类型支持

**Protobuf（默认）：**

```bash
go run . micro --idl proto ...
```

生成 `.proto` 文件，包含完整的 Service 定义和 Message 结构。

**Thrift：**

```bash
go run . micro --idl thrift ...
```

生成 `.thrift` 文件，适用于 Kitex 框架。

---

### F5：HTTP 框架选择

**Gin（默认）：**

```bash
go run . micro --http gin ...
```

**Hertz：**

```bash
go run . micro --http hertz ...
```

BFF 层的 Handler 和 middleware 代码将使用对应框架的 API 生成。

---

### F6：中间件/拦截器生成

启用 `--middleware` 后：

- BFF 层生成 `middleware.go`（包含日志、Recovery 等中间件）
- SRV 层每个微服务生成 `interceptor.go`（gRPC 拦截器或 Kitex 中间件）

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --middleware
```

生成路径：
- `bffH5/internal/middleware/middleware.go`
- `srvProduct/internal/interceptor/interceptor.go`

---

### F7：Nacos 配置中心集成

启用 `--config nacos` 后，自动：
1. 在 `pkg/config/nacos_config.go` 生成 Nacos 客户端代码
2. 在 `pkg/config/config.go` 的 `Config` struct 中添加 `NacosConfig` 字段
3. 更新 BFF 和所有 SRV 的 `configs/config.yaml`，追加 Nacos 配置段

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --config nacos
```

生成的 `config.yaml` 会追加：

```yaml
nacos:
  server_addr: 127.0.0.1
  port: 8848
  namespace: ""
  group: DEFAULT_GROUP
  data_id: myShop.yaml
```

---

### F8：测试代码生成

启用 `--test` 后：

- `bff<BFF>/test/` → BFF 接口测试文件（基于 `net/http/httptest`）
- `srv<Module>/test/` → 微服务单元测试文件
- `tests/*.sh` → curl/grpcurl Shell 测试脚本

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --test
```

---

### F9：服务注册中心

指定 `--register` 后，BFF 通过注册中心发现微服务地址（否则直连）：

```bash
# 使用 Consul
go run . micro ... --register consul

# 使用 etcd
go run . micro ... --register etcd
```

注册中心配置地址在生成的 `configs/config.yaml` 中指定。

---

### F10：配置文件驱动生成

将所有参数写入配置文件，通过 `--config-file` 指定（支持 YAML、JSON、TOML）：

```bash
go run . micro --config-file ./myapp.yaml
```

`myapp.yaml` 示例：

```yaml
name: myShop
output: ./output
bff: h5
modules:
  - product
  - order
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""
  name: myshop
  tables:
    - eb_product
    - eb_order
```

`myapp.toml` 示例：

```toml
name = "myShop"
output = "./output"
bff = "h5"
modules = ["product", "order"]

[database]
host = "127.0.0.1"
port = 3306
user = "root"
password = ""
name = "myshop"
tables = ["eb_product", "eb_order"]
```

---

## 使用示例

### 示例 1：最简生成

```bash
go run . micro \
  --name shop \
  --output ./output \
  --bff h5 \
  --modules product
```

### 示例 2：基于数据库表结构生成

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product,order,user \
  --db-host 127.0.0.1 \
  --db-port 3306 \
  --db-user root \
  --db-password 123456 \
  --db-name myshop \
  --db-table eb_product,eb_order,eb_user
```

### 示例 3：Hertz + Kitex + Thrift

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff api \
  --modules product,order \
  --http hertz \
  --protocol kitex \
  --idl thrift
```

### 示例 4：完整功能（数据库反向工程 + 中间件 + 注册中心 + 测试）

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product,order \
  --db-name myshop \
  --db-password 123456 \
  --db-table eb_product,eb_order \
  --register consul \
  --middleware \
  --test \
  --config nacos
```

### 示例 5：联表查询

```bash
go run . micro \
  --name myShop \
  --output ./output \
  --bff h5 \
  --modules product \
  --db-name myshop \
  --db-password 123456 \
  --db-table eb_store_product,eb_store_product_description \
  --djc "eb_store_product.id=eb_store_product_description.product_id" \
  --djs "eb_store_product:eb_store_product_description=1tn"
```

### 示例 6：通过配置文件生成

```bash
go run . micro --config-file ./myshop.yaml
```

---

## 相关子命令

### micro-bff：为已有项目添加 BFF 层

向已有微服务项目添加新的 BFF 层，不会重新生成整个项目。

```bash
go run . micro-bff \
  --name web \
  --output ./myShop \
  --modules product,order,user
```

**参数：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name` | ✓ | BFF 名称（如 `h5`、`web`、`admin`） |
| `--output` / `-o` | ✓ | 已有微服务项目的根目录 |
| `--modules` | ✓ | 微服务列表（逗号分隔） |
| `--db-host` | — | 数据库主机（默认 `127.0.0.1`） |
| `--db-port` | — | 数据库端口（默认 `3306`） |
| `--db-user` | — | 数据库用户（默认 `root`） |
| `--db-password` | — | 数据库密码（默认 `123456`） |
| `--db-name` | — | 数据库名（默认 `gospacex`） |

**生成的 BFF 目录结构：**

```
<output>/bff_<name>/
├── cmd/
│   └── main.go
├── configs/
│   └── config.yaml
└── internal/
    ├── dto/             # 请求/响应结构
    ├── handler/         # HTTP Handler
    ├── middleware/       # 中间件
    ├── rpc_client/      # gRPC 客户端
    └── router/          # 路由注册
```

---

### gen-proto：从数据库表生成 Proto 文件

```bash
go run . gen-proto \
  --table user \
  --output ./idl/user.proto \
  --host 127.0.0.1 \
  --port 3306 \
  --user root \
  --password "" \
  --database mydb
```

**参数：**

| 参数 | 简写 | 必填 | 说明 |
|------|------|------|------|
| `--table` | `-t` | ✓ | 数据库表名 |
| `--output` | `-o` | — | 输出文件路径（默认 `./<table>.proto`） |
| `--host` | — | — | 数据库主机（默认 `127.0.0.1`） |
| `--port` | — | — | 数据库端口（默认 `3306`） |
| `--user` | — | — | 数据库用户（默认 `root`） |
| `--password` | — | — | 数据库密码 |
| `--database` | — | — | 数据库名 |
| `--dry-run` | — | — | 仅预览，不写入文件 |

---

### gen-grpc：从数据库表生成完整 gRPC 代码

```bash
go run . gen-grpc \
  --db-dsn="root:password@tcp(localhost:3306)/mydb" \
  --tables="users,orders" \
  --idl-path ./common/idl \
  --srv-path ./srv \
  --bff-path ./bff
```

**参数：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `--db-dsn` | ✓ | 数据库连接字符串（`user:pass@tcp(host:port)/dbname`） |
| `--tables` | — | 指定表名，逗号分隔；不指定则处理所有表 |
| `--idl-path` | — | Proto 文件输出路径 |
| `--srv-path` | — | 微服务代码输出路径 |
| `--bff-path` | — | BFF 层代码输出路径 |
| `--proto-import` | — | Proto 导入路径 |
| `--srv-port` | — | 微服务 gRPC 端口（默认 `50051`） |
| `--bff-port` | — | BFF HTTP 端口（默认 `8080`） |
| `--dry-run` | — | 预览模式，不写入文件 |

---

## 附录：命令速查

```
gpx micro                              # 交互式模式
gpx micro --name <n> --output <o> --bff <b> --modules <m,...>  # 最简模式
gpx micro ... --idl thrift             # 使用 Thrift IDL
gpx micro ... --http hertz             # 使用 Hertz HTTP 框架
gpx micro ... --protocol kitex         # 使用 Kitex 协议
gpx micro ... --db-table <t>           # 数据库反向工程
gpx micro ... --djc t1.f1=t2.f2 --djs t1:t2=1tn   # 联表查询
gpx micro ... --middleware             # 生成中间件/拦截器
gpx micro ... --config nacos           # Nacos 配置中心
gpx micro ... --register consul        # Consul 注册中心
gpx micro ... --test                   # 生成测试代码
gpx micro ... --config-file x.yaml     # 配置文件驱动

gpx micro-bff --name <b> --output <project> --modules <m,...>  # 为已有项目添加 BFF
gpx gen-proto -t <table> --database <db>                       # 生成单表 Proto
gpx gen-grpc --db-dsn="..." --srv-path ./srv --bff-path ./bff  # 生成完整 gRPC 代码
```
