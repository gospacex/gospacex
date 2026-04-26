# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-04-26

### Added

- Multi-scene logging support (Business, Access, Audit, Error)
- Reliability levels (Vital, Important, Normal)
- DoubleBuffer implementation for Vital scene with Fsync
- AsyncBatch for Important scene
- Token bucket rate limiting for Error logger
- Kafka async producer with partition-aware routing
- OpenTelemetry trace context propagation
- Prometheus metrics (log_entries_total, mq_push_total, fsync_latency_seconds, buffer_usage_ratio)
- Dynamic log level adjustment via HTTP handler
- Health check endpoint (/debug/log/health)
- GORM storage layer hooks with slow query detection
- Redis tracing integration via redisotel
- Gin and gRPC tracing middleware
- Log rotation with cleanup
- Field-level sensitive data masking (phone, idcard, bankcard, email)
- YAML configuration support

### Features

- Business Logger: Important reliability, async batch to Kafka
- Access Logger: Important reliability, async batch to Kafka
- Audit Logger: Vital reliability, double buffer with Fsync
- Error Logger: Important reliability, token bucket rate limiting
- Storage Logger: Independent log level configuration

### Architecture

- Double buffer with active/standby swap and Fsync
- Async batch with configurable batch size and flush interval
- Consistent hashing for Kafka partition selection
- Exponential backoff retry (1s, 2s, 4s)
- Sampling support (warn/info/debug levels)
