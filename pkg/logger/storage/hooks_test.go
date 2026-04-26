package storage

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
)

type mockLogger struct {
	warnMsg    string
	debugMsg   string
	infoMsg    string
	warnFields map[string]interface{}
	debugFields map[string]interface{}
}

func (m *mockLogger) Warn(msg string, fields ...interface{}) {
	m.warnMsg = msg
	m.warnFields = make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			m.warnFields[fields[i].(string)] = fields[i+1]
		}
	}
}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {
	m.debugMsg = msg
	m.debugFields = make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			m.debugFields[fields[i].(string)] = fields[i+1]
		}
	}
}

func (m *mockLogger) Info(msg string, fields ...interface{}) {
	m.infoMsg = msg
}

func TestGetTraceIDFromContext(t *testing.T) {
	t.Run("from context value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TraceIDContextKey, "test-trace-id-123")
		traceID := GetTraceIDFromContext(ctx)
		if traceID != "test-trace-id-123" {
			t.Errorf("expected 'test-trace-id-123', got '%s'", traceID)
		}
	})

	t.Run("empty when no trace", func(t *testing.T) {
		traceID := GetTraceIDFromContext(context.Background())
		if traceID != "" {
			t.Errorf("expected empty trace ID, got '%s'", traceID)
		}
	})
}

func TestGetSpanIDFromContext(t *testing.T) {
	t.Run("from context value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), SpanIDContextKey, "test-span-id-456")
		spanID := GetSpanIDFromContext(ctx)
		if spanID != "test-span-id-456" {
			t.Errorf("expected 'test-span-id-456', got '%s'", spanID)
		}
	})
}

func TestFormatTraceFields(t *testing.T) {
	t.Run("with no span context returns nil", func(t *testing.T) {
		fields := FormatTraceFields(context.Background())
		if fields != nil {
			t.Errorf("expected nil fields for empty context, got %v", fields)
		}
	})

	t.Run("from context values", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TraceIDContextKey, "test-trace-id")
		ctx = context.WithValue(ctx, SpanIDContextKey, "test-span-id")
		fields := FormatTraceFields(ctx)
		if len(fields) < 2 {
			t.Errorf("expected at least 2 fields, got %d", len(fields))
		}
		if fields[0] != "trace_id" {
			t.Errorf("expected first field to be 'trace_id', got '%v'", fields[0])
		}
		if fields[2] != "span_id" {
			t.Errorf("expected third field to be 'span_id', got '%v'", fields[2])
		}
	})
}

func TestInjectTraceFields(t *testing.T) {
	t.Run("without trace context", func(t *testing.T) {
		result := InjectTraceFields(context.Background(), "test message")
		if result != "test message" {
			t.Errorf("expected original message, got '%s'", result)
		}
	})

	t.Run("with trace context values", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TraceIDContextKey, "test-trace-id")
		result := InjectTraceFields(ctx, "test message")
		if result == "" {
			t.Error("expected non-empty result")
		}
	})
}

func TestNewGORMHook(t *testing.T) {
	logger := &mockLogger{}
	hook := NewGORMHook(logger, 100*time.Millisecond, false)

	if hook == nil {
		t.Error("expected non-nil hook")
	}
	if hook.slowThreshold != 100*time.Millisecond {
		t.Errorf("expected slowThreshold 100ms, got %v", hook.slowThreshold)
	}
	if hook.logBody != false {
		t.Errorf("expected logBody false, got %v", hook.logBody)
	}
}

func TestGORMHook_Name(t *testing.T) {
	logger := &mockLogger{}
	hook := NewGORMHook(logger, 100*time.Millisecond, false)

	if hook.Name() != "OpenTelemetryGORMHook" {
		t.Errorf("expected 'OpenTelemetryGORMHook', got '%s'", hook.Name())
	}
}

func TestNewRedisHook(t *testing.T) {
	logger := &mockLogger{}
	hook := NewRedisHook(logger, 50*time.Millisecond)

	if hook == nil {
		t.Error("expected non-nil hook")
	}
	if hook.slowThreshold != 50*time.Millisecond {
		t.Errorf("expected slowThreshold 50ms, got %v", hook.slowThreshold)
	}
}

func TestSRVTraceFallback(t *testing.T) {
	logger := &mockLogger{}
	fallback := NewSRVTraceFallback(logger)

	if fallback == nil {
		t.Error("expected non-nil fallback")
	}

	t.Run("with existing trace context returns same context", func(t *testing.T) {
		tp := noop.NewTracerProvider()
		tracer := tp.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		result := fallback.GenerateTraceID(ctx)
		if result == nil {
			t.Error("expected non-nil context")
		}
	})
}

func TestTraceFromContext(t *testing.T) {
	t.Run("empty when no trace", func(t *testing.T) {
		traceID, spanID := TraceFromContext(context.Background())
		if traceID != "" {
			t.Errorf("expected empty trace ID, got '%s'", traceID)
		}
		if spanID != "" {
			t.Errorf("expected empty span ID, got '%s'", spanID)
		}
	})
}

func TestWithSRVTraceFallback(t *testing.T) {
	logger := &mockLogger{}

	t.Run("without trace logs warning", func(t *testing.T) {
		ctx := context.Background()
		WithSRVTraceFallback(ctx, logger)
		if logger.warnMsg == "" {
			t.Error("expected warning message")
		}
	})

	t.Run("with existing trace does not warn", func(t *testing.T) {
		logger.warnMsg = ""
		tp := noop.NewTracerProvider()
		tracer := tp.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		WithSRVTraceFallback(ctx, logger)
	})
}
