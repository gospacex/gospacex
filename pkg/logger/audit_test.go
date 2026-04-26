package logger

import (
	"context"
	"os"
	"testing"
	"time"

	"gpx/pkg/logger/scene"
)

func TestAuditRecordJSON(t *testing.T) {
	record := &scene.AuditRecord{
		Action:     "update_password",
		Resource:   "user",
		UserID:     12345,
		ResourceID: "user-12345",
		TraceID:    "trace-abc",
		SpanID:     "span-def",
		Details:    map[string]any{"ip": "192.168.1.1"},
		Timestamp:  time.Now(),
	}

	if record.Action != "update_password" {
		t.Errorf("expected Action 'update_password', got '%s'", record.Action)
	}
	if record.Resource != "user" {
		t.Errorf("expected Resource 'user', got '%s'", record.Resource)
	}
	if record.UserID != 12345 {
		t.Errorf("expected UserID 12345, got %v", record.UserID)
	}
}

func TestAuditLogMethods(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Vital.BufferSize = 100
	cfg.Vital.FsyncTimeout = 50 * time.Millisecond
	cfg.Vital.FallbackOnFull = true

	levelMgr := NewLevelManager()

	auditLogger, err := scene.NewAuditLogger(os.Stdout, &scene.Config{
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	record := &scene.AuditRecord{
		Action:    "test_action",
		Resource:  "test_resource",
		Timestamp: time.Now(),
	}

	auditLogger.Log(record)

	if record.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

func TestAuditLogContext(t *testing.T) {
	cfg := DefaultConfig()
	levelMgr := NewLevelManager()

	auditLogger, err := scene.NewAuditLogger(os.Stdout, &scene.Config{
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
	ctx = context.WithValue(ctx, "span_id", "span-456")

	record := &scene.AuditRecord{
		Action:    "test_action",
		Resource:  "test_resource",
		Timestamp: time.Now(),
	}

	auditLogger.LogContext(ctx, record)

	if record.TraceID != "trace-123" {
		t.Errorf("expected TraceID 'trace-123', got '%s'", record.TraceID)
	}
	if record.SpanID != "span-456" {
		t.Errorf("expected SpanID 'span-456', got '%s'", record.SpanID)
	}
}

func TestAuditLogf(t *testing.T) {
	cfg := DefaultConfig()
	levelMgr := NewLevelManager()

	auditLogger, err := scene.NewAuditLogger(os.Stdout, &scene.Config{
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	auditLogger.Logf("login", "user", "user %s logged in from %s", "john", "192.168.1.1")
}
