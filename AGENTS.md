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


<claude-mem-context>
# Memory Context

# [gpx] recent context, 2026-05-13 12:33am GMT+8

Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision 🚨security_alert 🔐security_note
Format: ID TIME TYPE TITLE
Fetch details: get_observations([IDs]) | Search: mem-search skill

Stats: 50 obs (12,733t read) | 3,342,959t work | 100% savings

### May 2, 2026
205 9:30a 🔵 Port check: all test ports 8081-8086/8001-8006 available
207 9:40a 🟣 Implemented ES/MQ/Tracing templates and test plan — commit efd9b66
208 " 🔵 User queried location of saved test plan
209 9:53a ⚖️ Restructured test execution approach: per-project manual verification
210 " ✅ Major test plan refactor: manual step-by-step workflow replacing automated scripts
212 " 🔵 test-execution-plan reverted to old content after commit d0fe227
211 9:55a ✅ Major test plan refactor: manual step-by-step workflow replacing automated scripts
216 10:12a 🔵 GPX test execution plan documented for all US cases
217 10:16a 🔵 Test infrastructure partial availability: ES/Redis/MySQL/Nacos running, Consul/Kafka/Jaeger missing
219 " 🔵 Docker registry access blocked: cannot pull Consul/Kafka/Jaeger images
220 " 🔵 micro-app command uses --modules (not --srvs) for microservice list
218 " 🔵 gpx CLI commands updated: micro-app replaces micro
221 10:20a ✅ GPX test execution task created for 6 US scenarios
222 " 🔵 GPX test tables exist in gospacex database: eb_store_product and eb_store_product_attr
224 10:21a 🟣 US-003: test-es project generated with Elasticsearch integration
225 " 🔴 US-004: test-mq project generated with Kafka warning - unknown mq type
226 " 🟣 US-005: test-trace project generated with OpenTelemetry tracing
223 " 🟣 US-001: test-registry project generated with Consul registry integration
227 10:22a 🟣 US-006: test-join project generated with one-to-one join query
228 " 🟣 US-007: test-middleware project generated with BFF middleware and SRV interceptors
229 " 🟣 All 6 GPX test projects successfully generated from test-execution-plan-20260502.md
230 10:23a 🟣 All 6 GPX test projects verified and pass go mod tidy
231 10:24a 🔵 Bug root cause: ES/tracing/registry config values empty due to flag variable parsing
237 " 🔴 Duplicate --registry flag causes URL parsing to fail
235 10:34a 🟣 Nacos config center design doc created — 5th design document this session
236 10:37a 🟣 Config center design doc committed — 5th design doc, 8-item implementation checklist
239 10:40a 🟣 Autopilot session initiated
240 " 🔵 Config Center Integration Design Document Created
241 10:41a 🟣 Autopilot Phase 0 Expansion started for Nacos config center implementation
242 " 🔵 Nacos config integration already partially implemented
243 " ✅ Implementation TODO created for Nacos config center
244 10:43a 🟣 Nacos config center integration project started
249 6:05p 🔵 Model switching inquiry via free-claude-code
250 " 🔵 free-claude-code command not found
252 6:07p 🔵 free-claude-code is a local skills project, not a system CLI
253 " 🔵 free-claude-code is a proxy server, not a CLI tool
251 " 🔵 free-claude-code not installed on system
254 6:08p 🔵 free-claude-code .env configuration inspected
255 6:16p ✅ Configuring NVIDIA NIM models into free-claude-code .env
256 " 🔵 MODEL already configured in .env as nvidia_nim/z-ai/glm4.7
257 6:18p ✅ Configured per-tier NVIDIA NIM models in free-claude-code .env
258 " 🔵 /model picker shows default models only, not NVIDIA NIM models
259 6:25p ✅ Commented out MODEL_OPUS and MODEL_HAIKU in free-claude-code .env
260 6:39p 🔴 gRPC get query returns error instead of empty result when no records found
261 9:07p 🔵 gRPC Get handler propagates service errors directly to client
262 " 🔵 Repository GetByID uses GORM First which returns error on no results
263 " 🔵 Service layer JoinProduct2ProductAttr already handles no-results case correctly
264 " 🔴 Fixed gRPC Get query to return empty response instead of error when no records found
S264 GPX框架handler模板探索及响应格式标准化研究 (May 2 at 10:15 PM)
S265 Initial greeting and project inquiry (May 2 at 10:15 PM)
S269 Brainstorming session initiation for GPX microservice scaffolding tool (May 2 at 10:16 PM)
S270 Brainstorming session initiation for GPX microservice scaffolding tool (May 2 at 10:17 PM)
S267 Brainstorming session initiation for GPX microservice scaffolding tool (May 2 at 10:18 PM)
S273 Design review of SRV layer and BFF layer return specifications for GPX project (May 2 at 10:20 PM)
S272 Brainstorming session initiation for GPX microservice scaffolding tool (May 2 at 10:24 PM)
S271 Brainstorming session initiation for GPX microservice scaffolding tool (May 2 at 10:24 PM)
265 10:27p ✅ SRV and BFF layer response design specification document created
S274 SRV and BFF layer response design specification document created and saved (May 2 at 10:31 PM)
266 11:16p ✅ Session initialization
S275 Session initialization and project context acknowledgment (May 2 at 11:16 PM)
**Learned**: The project is a Go-based microservice scaffolding tool called "gpx" with established functionality around configuration management, service layers, and various infrastructure templates. The environment uses oh-my-claudecode for multi-agent orchestration.

**Completed**: Only session initialization completed. No substantive work performed yet - the agent has merely greeted the user and restated known project context while awaiting direction.

**Next Steps**: Awaiting user's specification of what they want to work on regarding the gpx project. The agent is ready to assist with development, debugging, planning, or other tasks related to the microservice scaffolding tool.


Access 3343k tokens of past work via get_observations([IDs]) or mem-search skill.
</claude-mem-context>