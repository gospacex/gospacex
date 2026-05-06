# Project Instructions

This file provides context for AI assistants working on this project.

## Project Overview

**gpx** is a Go project scaffold generator (脚手架生成器) built with Cobra CLI. It generates four types of Go projects:

- **微服务项目** (Microservices) — standard/DDD architecture, istio, Protobuf/Thrift IDL support
- **单体项目** (Monolith) — traditional MVC architecture
- **脚本中心** (Script Center) — gocron-based scheduled task framework
- **Agent 项目** (Agent) — CloudWeGo Eino-based

## Commands

- Build: `go build`
- Run: `go run .`
- Format: `go fmt ./...`
- Test: `go test ./...`

## Architecture

- **Entry point**: `main.go` → `internal/cli.Execute()`
- **CLI layer**: `internal/cli/` — Cobra command definitions
- **Generator core**: `internal/generator/` — code generation logic
- **Template engine**: `internal/template/` — Go template processing
- **Generated templates**: `templates/` — project templates (agent/, monolith/, etc.)
- **Logger module**: `pkg/logger/` — separate Go module with own `go.mod` (Go 1.25.0)

## Known Issues

- `internal/generator/microapp_generator.go` has format string bugs (mismatched printf arguments) — build succeeds but tests fail
- `internal/cli/root_test.go:81` expects version `0.1.0` but gets `"0.0.10"`
- `tests/es/crud_test.go` has undefined references (incomplete test file)

## Infrastructure

`docker-compose.yaml` provides dev infrastructure: MySQL, Redis, Elasticsearch, Kafka, RabbitMQ, RocketMQ, Consul, Nacos, Jaeger, Prometheus, Grafana, Loki, etc.

## Guidelines

- Follow existing code style and patterns
- Write tests for new functionality
- Keep changes focused and atomic
- Document public APIs
