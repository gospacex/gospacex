# GPX MCP 工具使用指南

> 本文档说明如何使用 GPX 项目提供的 MCP (Model Context Protocol) 工具

---

## 概述

GPX 项目集成了 MCP 工具，允许 AI 助手（如 Claude Code）与项目进行深度集成，提供智能化的开发辅助功能。

## 工具列表

### 1. 项目初始化工具

**工具名称：** `init`

**功能：** 初始化新的 GPX 项目配置

**参数：**
- `project_name` (string, 必需): 项目名称
- `project_type` (string, 可选): 项目类型，支持 "microservice" 或 "bff"
- `output_dir` (string, 可选): 输出目录，默认为当前目录

**使用示例：**
```bash
# 交互式初始化
gpx init

# 命令行模式初始化微服务项目
gpx init --project-name=user-service --project-type=microservice --output-dir=./services
```

### 2. 代码生成工具

**工具名称：** `generate`

**功能：** 生成项目代码和配置文件

**参数：**
- `template` (string, 必需): 模板类型，可选值：
  - `microservice`: 微服务模板
  - `bff`: BFF 层模板
  - `config`: 配置文件模板
  - `docker`: Docker 配置模板
- `output` (string, 可选): 输出路径
- `force` (boolean, 可选): 是否强制覆盖已存在文件

**使用示例：**
```bash
# 生成微服务模板
gpx generate --template=microservice --output=./services/user

# 生成 BFF 层配置
gpx generate --template=bff --output=./bff

# 强制覆盖现有文件
gpx generate --template=config --force
```

### 3. 配置管理工具

**工具名称：** `config`

**功能：** 管理项目配置

**参数：**
- `action` (string, 必需): 操作类型，可选值：
  - `get`: 获取配置
  - `set`: 设置配置
  - `list`: 列出所有配置
  - `validate`: 验证配置
- `key` (string, 可选): 配置键
- `value` (string, 可选): 配置值
- `file` (string, 可选): 配置文件路径

**使用示例：**
```bash
# 列出所有配置
gpx config --action=list

# 获取特定配置
gpx config --action=get --key=database.url

# 设置配置值
gpx config --action=set --key=server.port --value=8080

# 验证配置文件
gpx config --action=validate --file=config.yaml
```

### 4. 依赖管理工具

**工具名称：** `deps`

**功能：** 管理项目依赖

**参数：**
- `action` (string, 必需): 操作类型，可选值：
  - `add`: 添加依赖
  - `remove`: 移除依赖
  - `update`: 更新依赖
  - `list`: 列出依赖
  - `graph`: 显示依赖关系图
- `package` (string, 可选): 包名
- `version` (string, 可选): 版本号

**使用示例：**
```bash
# 列出所有依赖
gpx deps --action=list

# 添加新依赖
gpx deps --action=add --package=github.com/gin-gonic/gin --version=v1.9.1

# 移除依赖
gpx deps --action=remove --package=unused-package

# 显示依赖关系图
gpx deps --action=graph --output=dot
```

### 5. 项目构建工具

**工具名称：** `build`

**功能：** 构建项目

**参数：**
- `target` (string, 可选): 构建目标，可选值：
  - `all`: 构建所有服务
  - `service`: 构建单个服务
  - `bff`: 构建 BFF 层
- `service_name` (string, 可选): 服务名称（当 target=service 时必需）
- `output` (string, 可选): 输出目录
- `platform` (string, 可选): 目标平台，如 `linux/amd64`, `darwin/arm64`

**使用示例：**
```bash
# 构建所有服务
gpx build --target=all

# 构建特定服务
gpx build --target=service --service_name=user-service

# 跨平台构建
gpx build --target=all --platform=linux/amd64 --output=./dist
```

### 6. 测试工具

**工具名称：** `test`

**功能：** 运行测试

**参数：**
- `type` (string, 可选): 测试类型，可选值：
  - `unit`: 单元测试
  - `integration`: 集成测试
  - `e2e`: 端到端测试
  - `all`: 所有测试
- `service` (string, 可选): 指定服务名称
- `coverage` (boolean, 可选): 是否生成覆盖率报告
- `output` (string, 可选): 输出格式，可选 `json`, `xml`, `html`

**使用示例：**
```bash
# 运行所有测试
gpx test --type=all

# 运行单元测试并生成覆盖率报告
gpx test --type=unit --coverage --output=html

# 测试特定服务
gpx test --type=integration --service=user-service
```

### 7. 部署工具

**工具名称：** `deploy`

**功能：** 部署项目

**参数：**
- `environment` (string, 必需): 部署环境，如 `dev`, `staging`, `prod`
- `strategy` (string, 可选): 部署策略，可选值：
  - `rolling`: 滚动更新
  - `blue-green`: 蓝绿部署
  - `canary`: 金丝雀发布
- `dry_run` (boolean, 可选): 预演模式，不实际执行
- `timeout` (integer, 可选): 超时时间（秒）

**使用示例：**
```bash
# 部署到开发环境
gpx deploy --environment=dev

# 生产环境蓝绿部署
gpx deploy --environment=prod --strategy=blue-green --timeout=300

# 预演部署
gpx deploy --environment=staging --dry_run
```

### 8. 监控工具

**工具名称：** `monitor`

**功能：** 监控服务状态

**参数：**
- `action` (string, 必需): 操作类型，可选值：
  - `status`: 查看状态
  - `logs`: 查看日志
  - `metrics`: 查看指标
  - `health`: 健康检查
- `service` (string, 可选): 服务名称
- `since` (string, 可选): 时间范围，如 `5m`, `1h`, `24h`
- `format` (string, 可选): 输出格式

**使用示例：**
```bash
# 查看所有服务状态
gpx monitor --action=status

# 查看特定服务日志
gpx monitor --action=logs --service=user-service --since=1h

# 查看指标
gpx monitor --action=metrics --format=json
```

## MCP 集成特性

### 智能代码分析
- 自动识别项目结构
- 检测代码质量问题
- 建议优化方案

### 上下文感知
- 理解项目依赖关系
- 识别微服务边界
- 推荐合适的模板

### 实时反馈
- 构建状态实时更新
- 测试结果即时显示
- 错误信息智能提示

## 配置文件

MCP 工具的配置文件位于项目根目录的 `.gpx/config.yaml`：

```yaml
mcp:
  enabled: true
  tools:
    - init
    - generate
    - config
    - deps
    - build
    - test
    - deploy
    - monitor

  settings:
    auto_save: true
    backup_enabled: true
    max_backups: 5
    log_level: info

  integrations:
    git:
      enabled: true
      auto_commit: false
    docker:
      enabled: true
      registry: "docker.io"
    kubernetes:
      enabled: false
```

## 环境变量

可通过环境变量配置 MCP 工具行为：

```bash
# 启用调试模式
export GPX_MCP_DEBUG=true

# 设置日志级别
export GPX_LOG_LEVEL=debug

# 配置 MCP 服务器地址
export GPX_MCP_SERVER=http://localhost:8080

# 设置 API 密钥
export GPX_MCP_API_KEY=your-api-key
```

## 故障排除

### 常见问题

**1. 工具无法加载**
```
问题：启动时提示 "MCP tool not found"
解决：确保 GPX 二进制文件在 PATH 中，或使用绝对路径
```

**2. 权限错误**
```
问题：提示 "Permission denied"
解决：检查文件权限，确保可执行权限：
chmod +x /path/to/gpx
```

**3. 配置加载失败**
```
问题：配置文件读取错误
解决：检查配置文件格式是否正确，运行：
gpx config --action=validate --file=.gpx/config.yaml
```

### 调试模式

启用调试模式获取详细信息：

```bash
# 设置调试环境变量
export GPX_MCP_DEBUG=true

# 运行工具并查看详细输出
gpx --verbose test --type=all
```

### 查看日志

日志文件位置：
- Linux/Mac: `~/.gpx/logs/`
- Windows: `%USERPROFILE%\.gpx\logs\`

## 最佳实践

### 1. 项目初始化
```bash
# 推荐：使用交互模式初始化
gpx init

# 然后逐步添加服务
gpx generate --template=microservice --output=./services/user
gpx generate --template=microservice --output=./services/order
```

### 2. 依赖管理
```bash
# 定期更新依赖
gpx deps --action=update

# 使用精确版本
gpx deps --action=add --package=package --version=v1.2.3
```

### 3. 测试策略
```bash
# 开发时运行单元测试
gpx test --type=unit --coverage

# 提交前运行所有测试
gpx test --type=all
```

### 4. 部署流程
```bash
# 先预演
gpx deploy --environment=staging --dry_run

# 确认无误后正式部署
gpx deploy --environment=prod --strategy=rolling
```

## 进阶用法

### 自定义模板

创建自定义模板目录：
```bash
mkdir -p ~/.gpx/templates/custom
```

在模板中可使用变量：
- `{{.ProjectName}}`: 项目名称
- `{{.ServiceName}}`: 服务名称
- `{{.Version}}`: 版本号
- `{{.Author}}`: 作者信息

### 插件开发

GPX 支持通过插件扩展 MCP 工具功能：

```go
// 示例插件结构
type Plugin interface {
    Name() string
    Execute(args []string) error
    RegisterCommands() []Command
}
```

### CI/CD 集成

在 CI 脚本中使用：

```bash
#!/bin/bash
# 初始化项目
gpx init --project-name=myapp --project-type=microservice --output-dir=.

# 安装依赖
gpx deps --action=add --package=all

# 运行测试
gpx test --type=all --coverage --output=json > test-report.json

# 构建
gpx build --target=all --platform=linux/amd64

# 部署
gpx deploy --environment=prod --strategy=blue-green
```

## 相关资源

- [GPX 主文档](../README.md)
- [微服务命令手册](./micro-command.md)
- [GPX GitHub 仓库](https://github.com/gospacex/gpx)
- [MCP 官方文档](https://modelcontextprotocol.io/)

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2024-05-05 | 初始版本，支持 8 个核心 MCP 工具 |

---

**注意：** 本文档基于 GPX v1.0.0+ 版本。 older versions may have different tool names or parameters.
