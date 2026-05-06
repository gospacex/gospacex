# GPX - Go 项目脚手架生成器

<!-- Badges -->
<div align="center">

[![License](https://img.shields.io/github/license/gospacex/gpx?style=flat-square&logo=apache)
](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26.2-blue?style=flat-square&logo=go)
](go.mod)
[![Status](https://img.shields.io/badge/status-active-brightgreen?style=flat-square)
]()
[![Last Commit](https://img.shields.io/github/last-commit/gospacex/gpx?style=flat-square&logo=git)
]()
[![Stars](https://img.shields.io/github/stars/gospacex/gpx?style=flat-square&logo=github)
]()

</div>

## 📖 项目简介

**GPX** (Go Project scaffold Generator) 是一个基于 [Cobra CLI](https://github.com/spf13/cobra) 开发的通用脚手架工具，用于快速生成 Go 项目结构。

支持生成四类项目模板：

| 类型 | 架构 | 特性 |
|------|------|------|
| **微服务 (MicroApp)** | 标准架构 / DDD | Istio、Protobuf/Thrift IDL 支持 |
| **BFF (MicroBff)** | 微服务聚合层 | API 聚合、协议转换 |
| **单体 (Monolith)** | 传统 MVC | 快速启动、轻量级服务 |
| **脚本中心 (ScriptCenter)** | 定时任务框架 | 基于 gocron 的调度系统 |
| **Agent** | AI Agent | 基于 CloudWeGo Eino |

## 🚀 快速开始

### 安装

```bash
git clone https://github.com/gospacex/gpx.git
cd gpx
go build -o gpx
```

### 使用

```bash
# 查看帮助
./gpx --help

# 生成微服务项目
./gpx microapp new my-service

# 生成单体项目
./gpx monolith new my-app

# 生成脚本中心
./gpx scriptcenter new my-scripts

# 生成 CRUD 模块
./gpx crud generate --table users

# 生成 Protobuf 文件
./gpx gen proto --idl ./proto/user.proto

# 生成 gRPC 代码
./gpx gen grpc --proto ./proto/*.proto
```

```
# 微服多表
gpx micro --name myshop4 --output output --bff h5 --srvs product --db-host 127.0.0.1 --db-port 3306 --db-user root --db-password 123456 --db-name gospacex --db-table eb_store_product,eb_store_product_attr --test
# 微服连表
gpx micro --name myshop4 --output output --bff h5 --srvs product --db-host 127.0.0.1 --db-port 3306 --db-user root --db-password 123456 --db-name gospacex --db-table eb_store_product,eb_store_product_attr --db-join-condition eb_store_product.id=eb_store_product_attr.product_id --db-join-style  eb_store_product:eb_store_product_attr=1t1 --test
```

## 📦 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| [go-elasticsearch](https://github.com/elastic/go-elasticsearch) | v7.17.10 | Elasticsearch 客户端 |
| [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | v1.9.3 | MySQL 数据库驱动 |
| [spf13/cobra](https://github.com/spf13/cobra) | v1.10.2 | CLI 框架 |
| [stretchr/testify](https://github.com/stretchr/testify) | v1.11.1 | 单元测试 |
| [jinzhu/copier](https://github.com/jinzhu/copier) | v0.4.0 | 对象拷贝 |
| [pelletier/go-toml](https://github.com/pelletier/go-toml) | v2.2.4 | TOML 配置解析 |

## 🏗️ 项目结构

```
gpx/
├── main.go                 # 入口文件
├── internal/
│   ├── cli/                # CLI 命令定义
│   ├── config/             # 配置管理
│   ├── generator/          # 代码生成器核心
│   └── template/          # 模板引擎
├── templates/              # 项目模板
│   ├── agent/             # Agent 项目模板
│   ├── monolith/          # 单体项目模板
│   ├── microbff/          # BFF 项目模板
│   └── scriptcenter/      # 脚本中心模板
├── pkg/                    # 公共包
├── tests/                  # 测试用例
├── data/                   # 数据文件
├── docs/                   # 文档
└── docker-compose.yaml    # 开发基础设施
```

## 🐳 开发环境

项目提供了完整的 Docker Compose 开发环境：

```bash
# 启动所有服务
docker-compose up -d
```

**基础设施服务：**

| 服务 | 端口 | 描述 |
|------|------|------|
| MySQL | 3306 | 关系型数据库 |
| Redis | 6379 | 缓存服务 |
| Elasticsearch | 9200 | 搜索引擎 |
| Kafka | 9092 | 消息队列 |
| RabbitMQ | 5672 | 消息代理 |
| Consul | 8500 | 服务发现 |
| Nacos | 8848 | 配置中心 |
| Jaeger | 16686 | 链路追踪 |
| Prometheus | 9090 | 监控指标 |
| Grafana | 3000 | 可视化面板 |

## 📚 命令列表

| 命令 | 说明 |
|------|------|
| `gpx microapp new <name>` | 创建微服务项目 |
| `gpx microbff new <name>` | 创建 BFF 项目 |
| `gpx monolith new <name>` | 创建单体项目 |
| `gpx scriptcenter new <name>` | 创建脚本中心 |
| `gpx crud generate <table>` | 生成 CRUD 代码 |
| `gpx gen proto --idl <path>` | 生成 Protobuf 文件 |
| `gpx gen grpc --proto <path>` | 生成 gRPC 代码 |
| `gpx pkg new <name>` | 创建共享包 |

## 📄 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可证。