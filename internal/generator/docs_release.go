package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// DocsReleaseGenerator 文档和发布生成器
type DocsReleaseGenerator struct {
	projectName string
	outputDir   string
}

// NewDocsReleaseGenerator creates new docs and release generator
func NewDocsReleaseGenerator(projectName, outputDir string) *DocsReleaseGenerator {
	return &DocsReleaseGenerator{
		projectName: projectName,
		outputDir:   outputDir,
	}
}

// Generate generates docs and release files
func (g *DocsReleaseGenerator) Generate() error {
	dirs := []string{
		"docs",
		"docs/guides",
		"docs/api",
		"docs/deploy",
		".github",
		".github/ISSUE_TEMPLATE",
		".github/PULL_REQUEST_TEMPLATE",
		"releases",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"README.md":                         g.readmeContent(),
		"docs/guides/quickstart.md":         g.quickstartContent(),
		"docs/guides/installation.md":       g.installationContent(),
		"docs/api/api.md":                   g.apiContent(),
		"docs/deploy/docker.md":             g.dockerDeployContent(),
		"docs/deploy/k8s.md":                g.k8sDeployContent(),
		"CONTRIBUTING.md":                   g.contributingContent(),
		"CODE_OF_CONDUCT.md":                g.codeOfConductContent(),
		"LICENSE":                           g.licenseContent(),
		".github/ISSUE_TEMPLATE/bug.md":     g.bugTemplateContent(),
		".github/ISSUE_TEMPLATE/feature.md": g.featureTemplateContent(),
		"releases/v0.1.0.md":                g.releaseV010Content(),
	}

	for path, content := range files {
		fullPath := filepath.Join(g.outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (g *DocsReleaseGenerator) readmeContent() string {
	return fmt.Sprintf(`# %s

强大的 Go 项目脚手架生成器

[![Go Version](https://img.shields.io/badge/go-1.26.2-blue.svg)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache2.0-blue.svg)](LICENSE)
[![Tests](https://github.com/gospacex/gpx/actions/workflows/test.yml/badge.svg)](https://github.com/gospacex/gpx/actions)

## 特性

- ✅ **6 种项目类型** - 微服务/单体/Agent/脚本中心
- ✅ **5 种数据库** - MySQL/PostgreSQL/Redis/Elasticsearch/MongoDB
- ✅ **6+ 中间件** - Jaeger/Kafka/RocketMQ/Nacos/Apollo/Consul
- ✅ **分布式事务** - DTM (SAGA/TCC/Workflow)
- ✅ **完整测试** - 单元测试/集成测试/E2E 测试
- ✅ **K8s 部署** - Docker/Kubernetes/Istio

## 快速开始

`+"```bash"+`
# 安装
go install github.com/gospacex/gpx@latest

# 生成微服务项目
gpx new microservice --style standard --output order-service

# 生成单体项目
gpx new monolith --output admin-dashboard

# 生成 Agent 项目
gpx new agent --output my-agent
`+"```"+`

## 文档

- [快速开始](docs/guides/quickstart.md)
- [安装指南](docs/guides/installation.md)
- [API 文档](docs/api/api.md)
- [Docker 部署](docs/deploy/docker.md)
- [K8s 部署](docs/deploy/k8s.md)

## 项目结构

`+"```"+`
gpx/
├── cmd/                  # CLI 命令
├── internal/             # 内部实现
│   └── generator/        # 生成器
├── docs/                 # 文档
├── tests/                # 测试
└── releases/             # 发布说明
`+"```"+`

## 贡献

详见 [CONTRIBUTING.md](CONTRIBUTING.md)

## 许可证

Apache License 2.0
`, g.projectName)
}

func (g *DocsReleaseGenerator) quickstartContent() string {
	return `# 快速开始

## 安装

` + "```bash" + `
go install github.com/gospacex/gpx@latest
` + "```" + `

## 生成项目

### 微服务项目

` + "```bash" + `
gpx new microservice \
  --style standard \
  --db mysql,redis \
  --output order-service
` + "```" + `

### 单体项目

` + "```bash" + `
gpx new monolith \
  --orm gorm \
  --output admin-dashboard
` + "```" + `

### Agent 项目

` + "```bash" + `
gpx new agent \
  --output my-agent
` + "```" + `

## 运行生成的项目

` + "```bash" + `
cd order-service
go mod tidy
go run app/order-service/main.go
` + "```" + `
`
}

func (g *DocsReleaseGenerator) installationContent() string {
	return `# 安装指南

## 系统要求

- Go 1.26.2
- Git

## 安装方式

### Go install (推荐)

` + "```bash" + `
go install github.com/gospacex/gpx@latest
` + "```" + `

### 源码安装

` + "```bash" + `
git clone https://github.com/gospacex/gpx.git
cd gpx
go build -o gpx ./cmd/gpx
sudo mv gpx /usr/local/bin/
` + "```" + `

## 验证安装

` + "```bash" + `
gpx --version
` + "```" + `
`
}

func (g *DocsReleaseGenerator) apiContent() string {
	return `# API 文档

## 命令

### gpx new

生成新项目

` + "```bash" + `
gpx new [project-type] [flags]
` + "```" + `

**Flags:**

| Flag | 描述 | 默认值 |
|------|------|--------|
| --style | 架构风格 | standard |
| --db | 数据库 | - |
| --orm | ORM 框架 | gorm |
| --output | 输出目录 | - |

**项目类型:**

- microservice - 微服务项目
- monolith - 单体项目
- agent - Agent 项目
- script - 脚本中心

## 示例

` + "```bash" + `
# 生成标准微服务
gpx new microservice --style standard --output my-service

# 生成 DDD 微服务
gpx new microservice --style ddd --output my-service

# 生成带数据库的项目
gpx new microservice --db mysql,redis --output my-service
` + "```" + `
`
}

func (g *DocsReleaseGenerator) dockerDeployContent() string {
	return `# Docker 部署

## 构建镜像

` + "```bash" + `
docker build -t my-service:latest .
` + "```" + `

## 运行容器

` + "```bash" + `
docker run -d \
  -p 8080:8080 \
  -e MYSQL_HOST=mysql \
  -e REDIS_ADDR=redis:6379 \
  my-service:latest
` + "```" + `

## Docker Compose

` + "```yaml" + `
version: '3'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MYSQL_HOST=mysql
      - REDIS_ADDR=redis:6379
    depends_on:
      - mysql
      - redis
  
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=mydb
  
  redis:
    image: redis:alpine
` + "```" + `
`
}

func (g *DocsReleaseGenerator) k8sDeployContent() string {
	return `# Kubernetes 部署

## 部署

` + "```bash" + `
kubectl apply -f deploy/k8s/
` + "```" + `

## 使用 Kustomize

` + "```bash" + `
kubectl apply -k deploy/k8s/
` + "```" + `

## Helm Chart

` + "```bash" + `
helm install my-service ./deploy/helm
` + "```" + `
`
}

func (g *DocsReleaseGenerator) contributingContent() string {
	return `# 贡献指南

## 开发环境

1. Fork 项目
2. Clone 到本地
3. 创建分支
4. 提交 PR

## 代码规范

- 遵循 Go 官方规范
- 通过 golangci-lint 检查
- 测试覆盖率 >= 80%%

## 提交流程

1. 创建 Issue
2. Fork & 分支
3. 提交代码
4. 创建 PR
5. Code Review
6. 合并
`
}

func (g *DocsReleaseGenerator) codeOfConductContent() string {
	return `# 行为准则

## 承诺

我们致力于提供友好、包容的社区环境。

## 行为标准

- 使用友好、包容的语言
- 尊重不同观点和经验
- 优雅地接受建设性批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

## 执行

不当行为可通过 [project-email] 举报。
`
}

func (g *DocsReleaseGenerator) licenseContent() string {
	return `Apache License
Version 2.0, January 2004
http://www.apache.org/licenses/

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
`
}

func (g *DocsReleaseGenerator) bugTemplateContent() string {
	return `---
name: Bug Report
about: 报告 bug
title: '[BUG] '
labels: bug
---

## Bug 描述

简要描述 bug。

## 复现步骤

1. 步骤 1
2. 步骤 2
3. ...

## 期望行为

描述期望的行为。

## 环境信息

- OS: [e.g. macOS]
- Go Version: [e.g. 1.26.2]
- GPX Version: [e.g. v0.1.0]
`
}

func (g *DocsReleaseGenerator) featureTemplateContent() string {
	return `---
name: Feature Request
about: 功能建议
title: '[FEATURE] '
labels: enhancement
---

## 功能描述

简要描述建议的功能。

## 使用场景

描述使用场景。

## 实现建议

描述实现建议。
`
}

func (g *DocsReleaseGenerator) releaseV010Content() string {
	return `# v0.1.0 - Initial Release

## 新功能

- ✅ 6 种项目类型生成器
- ✅ 5 种数据库集成
- ✅ 6+ 中间件集成
- ✅ DTM 分布式事务
- ✅ 完整测试框架
- ✅ K8s/Istio 部署

## Bug 修复

- Initial release

## 变更

- Initial release

## 致谢

感谢所有贡献者！
`
}
