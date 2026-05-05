package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type StorageLoggerInterface interface {
	Warn(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
}

type GORMHook struct {
	logger        StorageLoggerInterface
	slowThreshold time.Duration
	logBody       bool
}

func NewGORMHook(logger StorageLoggerInterface, slowThreshold time.Duration, logBody bool) *GORMHook {
	return &GORMHook{
		logger:        logger,
		slowThreshold: slowThreshold,
		logBody:       logBody,
	}
}

func (h *GORMHook) BeforeQuery(db *gorm.DB) {
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	db.Set("query_start_time", start)

	_, span := otel.Tracer("gorm").Start(ctx, "gorm.query",
		trace.WithAttributes(
			attribute.String("db.system", "mysql"),
			attribute.String("db.operation", "query"),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	db.Set("otlp_span", span)
}

func (h *GORMHook) AfterQuery(db *gorm.DB) {
	startVal, ok := db.Get("query_start_time")
	if !ok {
		return
	}
	start, ok := startVal.(time.Time)
	if !ok {
		return
	}
	duration := time.Since(start)

	ctx := db.Statement.Context
	traceID, spanID := h.getTraceIDFromContext(ctx)

	if span, ok := db.Get("otlp_span"); ok {
		if s, ok := span.(interface{ End() }); ok {
			s.End()
		}
	}

	fields := []interface{}{
		"duration", duration.String(),
		"rows", db.RowsAffected,
		"trace_id", traceID,
		"span_id", spanID,
	}

	if duration > h.slowThreshold {
		fields = append(fields, "slow_query", true)
		h.logger.Warn("slow query detected", fields...)
	} else {
		h.logger.Debug("query executed", fields...)
	}
}

func (h *GORMHook) getTraceIDFromContext(ctx context.Context) (string, string) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String(), spanCtx.SpanID().String()
	}
	return "", ""
}

func (h *GORMHook) Name() string {
	return "OpenTelemetryGORMHook"
}

type RedisHook struct {
	logger        StorageLoggerInterface
	slowThreshold time.Duration
}

func NewRedisHook(logger StorageLoggerInterface, slowThreshold time.Duration) *RedisHook {
	return &RedisHook{
		logger:        logger,
		slowThreshold: slowThreshold,
	}
}

func InstrumentRedisTracing(client *redis.Client) error {
	return redisotel.InstrumentTracing(client)
}

func InstrumentRedisMetrics(client *redis.Client) error {
	return redisotel.InstrumentMetrics(client)
}

type SRVTraceFallback struct {
	logger StorageLoggerInterface
}

func NewSRVTraceFallback(logger StorageLoggerInterface) *SRVTraceFallback {
	return &SRVTraceFallback{logger: logger}
}

func StoreTraceIDInContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDContextKey, traceID)
}

func (f *SRVTraceFallback) GenerateTraceID(ctx context.Context) context.Context {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return ctx
	}

	_, span := otel.Tracer("srv-fallback").Start(ctx, "srv-generated-trace",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(attribute.Bool("srv.trace_fallback", true))

	traceID := span.SpanContext().TraceID().String()
	span.End()

	return StoreTraceIDInContext(ctx, traceID)
}

func WithSRVTraceFallback(ctx context.Context, logger StorageLoggerInterface) context.Context {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return ctx
	}

	logger.Warn("no trace context found, using srv fallback")
	return ctx
}

type contextKey string

const (
	TraceIDContextKey contextKey = "trace_id"
	SpanIDContextKey contextKey = "span_id"
)

func GetTraceIDFromContext(ctx context.Context) string {
	if v := ctx.Value(TraceIDContextKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}

func GetSpanIDFromContext(ctx context.Context) string {
	if v := ctx.Value(SpanIDContextKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.SpanID().String()
	}
	return ""
}

func FormatTraceFields(ctx context.Context) []interface{} {
	traceID := GetTraceIDFromContext(ctx)
	spanID := GetSpanIDFromContext(ctx)

	if traceID == "" && spanID == "" {
		return nil
	}

	fields := []interface{}{"trace_id", traceID}
	if spanID != "" {
		fields = append(fields, "span_id", spanID)
	}
	return fields
}

func InjectTraceFields(ctx context.Context, msg string) string {
	fields := FormatTraceFields(ctx)
	if len(fields) == 0 {
		return msg
	}
	return fmt.Sprintf("%s | trace_id=%s", msg, fields[1])
}

func TraceFromContext(ctx context.Context) (traceID, spanID string) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String(), spanCtx.SpanID().String()
	}
	return
}
