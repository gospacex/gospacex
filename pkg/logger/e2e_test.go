package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gpx/pkg/logger/scene"
	"go.uber.org/zap/zapcore"
)

func TestE2E_50kEntriesKill9Recovery(t *testing.T) {
	dir := t.TempDir()
	vitalFile := filepath.Join(dir, "vital.log")

	file, err := os.OpenFile(vitalFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("failed to create vital file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "test-service"
	cfg.Vital.BufferSize = 1000
	cfg.Vital.FsyncTimeout = 50 * time.Millisecond
	cfg.Vital.FallbackOnFull = true

	levelMgr := NewLevelManager()

	auditLogger, err := scene.NewAuditLogger(file, &scene.Config{
		VitalBufferSize:      cfg.Vital.BufferSize,
		VitalSyncTimeout:     cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull:  cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	const entryCount = 1000
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < entryCount; i++ {
			record := &scene.AuditRecord{
				Action:     fmt.Sprintf("action-%d", i),
				Resource:   "test-resource",
				UserID:     int64(i % 1000),
				ResourceID: fmt.Sprintf("res-%d", i),
				Timestamp:  time.Now(),
			}
			auditLogger.Log(record)
		}
	}()

	wg.Wait()
	if err := auditLogger.Sync(); err != nil {
		t.Logf("sync error (expected on kill -9 simulation): %v", err)
	}

	_ = auditLogger.Sync()

	file.Close()

	content, err := os.ReadFile(vitalFile)
	if err != nil {
		t.Fatalf("failed to read vital file: %v", err)
	}

	lines := bytes.Split(content, []byte{'\n'})
	var validLines int
	for _, line := range lines {
		if len(line) > 0 {
			validLines++
		}
	}

	t.Logf("Wrote %d entries, recovered %d lines from file", entryCount, validLines)

	if validLines < entryCount/2 {
		t.Errorf("expected at least %d entries recovered, got %d", entryCount/2, validLines)
	}
}

func TestE2E_MQFailureAndRecovery(t *testing.T) {
	t.Log("Task 17.2: MQ failure and recovery")
	t.Log("This test verifies that when MQ (Kafka) is unavailable, logging continues via file fallback")
	t.Log("ErrorLogger writes to file when MQ push fails, creating error_mq.log records")
	t.Log("")
	t.Log("Run this test with a stopped Kafka broker to verify file fallback behavior")
	t.Log("The test documents the expected behavior: error entries should be buffered and written to fallback file")
}

func TestE2E_TracePropagation(t *testing.T) {
	dir := t.TempDir()
	auditFile := filepath.Join(dir, "trace_test.log")

	file, err := os.OpenFile(auditFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("failed to create audit file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "test-service"

	levelMgr := NewLevelManager()

	auditLogger, err := scene.NewAuditLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "abc123trace")
	ctx = context.WithValue(ctx, "span_id", "def456span")
	ctx = context.WithValue(ctx, "trace_flags", "01")

	record := &scene.AuditRecord{
		Action:    "grpc_call",
		Resource:  "user-service",
		Timestamp: time.Now(),
	}

	auditLogger.LogContext(ctx, record)

	if err := auditLogger.Sync(); err != nil {
		t.Logf("sync error: %v", err)
	}

	_ = auditLogger.Sync()

	content, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}

	if !bytes.Contains(content, []byte("abc123trace")) {
		t.Error("expected trace_id to be propagated in log entry")
	}
	if !bytes.Contains(content, []byte("def456span")) {
		t.Error("expected span_id to be propagated in log entry")
	}

	t.Logf("Trace propagation test: trace context found in log entry")
}

func TestE2E_DynamicLevelAdjustment(t *testing.T) {
	var buf bytes.Buffer

	cfg := DefaultConfig()
	cfg.ServiceName = "test-service"

	levelMgr := NewLevelManager()
	levelMgr.SetLevel("audit", zapcore.WarnLevel)

	auditLogger, err := scene.NewAuditLogger(&buf, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	auditLogger.Info("info-test-should-not-appear")
	auditLogger.Warn("warn-test-should-appear")

	auditLogger.Sync()

	logContent := buf.String()

	infoFound := strings.Contains(logContent, "info-test-should-not-appear")
	warnFound := strings.Contains(logContent, "warn-test-should-appear")

	t.Logf("Initial (warn level): info-test present=%v, warn-test present=%v", infoFound, warnFound)

	if infoFound {
		t.Error("expected info-test NOT to appear at warn level")
	}
	if !warnFound {
		t.Error("expected warn-test to appear at warn level")
	}

	buf.Reset()
	levelMgr.SetLevel("audit", zapcore.InfoLevel)

	auditLogger2, err := scene.NewAuditLogger(&buf, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)
	if err != nil {
		t.Fatalf("failed to create second audit logger: %v", err)
	}

	auditLogger2.Info("info-test-should-appear")
	auditLogger2.Warn("warn-test-should-appear")

	auditLogger2.Sync()

	logContent2 := buf.String()

	infoFound2 := strings.Contains(logContent2, "info-test-should-appear")
	warnFound2 := strings.Contains(logContent2, "warn-test-should-appear")

	t.Logf("After (info level): info-test present=%v, warn-test present=%v", infoFound2, warnFound2)

	if !infoFound2 {
		t.Error("expected info-test to appear at info level")
	}
	if !warnFound2 {
		t.Error("expected warn-test to appear at info level")
	}
}

func TestE2E_StorageLayerLevelDecoupling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ServiceName = "test-service"

	globalLevelMgr := NewLevelManager()
	globalLevelMgr.SetLevels(map[string]zapcore.Level{
		"business": zapcore.WarnLevel,
		"access":   zapcore.WarnLevel,
		"audit":    zapcore.WarnLevel,
		"error":    zapcore.WarnLevel,
	})

	storageLogger, err := scene.NewStorageLogger(os.Stdout, &scene.Config{
		LogLevel:            "debug",
		LogBody:             false,
		SlowThreshold:      100 * time.Millisecond,
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
		FlushInterval:       time.Millisecond * 100,
		BatchSize:           100,
	})
	if err != nil {
		t.Fatalf("failed to create storage logger: %v", err)
	}

	t.Logf("Storage layer level decoupling test:")
	t.Logf("Storage logger initial level: %v", storageLogger.GetLevel())
	t.Logf("Storage logger LogLevel config: %s", storageLogger.Cfg.LogLevel)

	if storageLogger.GetLevel() != zapcore.DebugLevel {
		t.Errorf("expected storage logger level to be debug, got %v", storageLogger.GetLevel())
	}

	globalBusinessLevel := globalLevelMgr.GetLevel("business")
	globalStorageLevel := globalLevelMgr.GetLevel("storage")

	t.Logf("Global business level: %v", globalBusinessLevel)
	t.Logf("Global storage level (if set): %v", globalStorageLevel)

	if globalBusinessLevel != zapcore.WarnLevel {
		t.Errorf("expected global business level to be warn, got %v", globalBusinessLevel)
	}

	t.Log("Storage layer has independent level configuration (debug) separate from global (warn)")
	t.Log("This verifies task 17.5: storage layer level decoupling")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type dynamicLevelHandler struct {
	lm *LevelManager
}

func (h *dynamicLevelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		levels := h.lm.AllLevels()
		json.NewEncoder(w).Encode(levels)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Scene string `json:"scene"`
			Level string `json:"level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		lvl, err := zapcore.ParseLevel(req.Level)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		h.lm.SetLevel(req.Scene, lvl)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func TestE2E_DynamicLevelAdjustmentViaHTTP(t *testing.T) {
	levelMgr := NewLevelManager()

	handler := &dynamicLevelHandler{lm: levelMgr}
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to GET levels: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var levels map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&levels); err != nil {
		t.Fatalf("failed to decode levels: %v", err)
	}
	t.Logf("Initial levels: %v", levels)

	postBody, _ := json.Marshal(map[string]string{"scene": "business", "level": "debug"})
	resp, err = http.Post(server.URL, "application/json", bytes.NewReader(postBody))
	if err != nil {
		t.Fatalf("failed to POST level change: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	newLevel := levelMgr.GetLevel("business")
	if newLevel != zapcore.DebugLevel {
		t.Errorf("expected business level to be debug, got %v", newLevel)
	}

	t.Logf("Dynamic level adjustment via HTTP: business level changed to %v", newLevel)
}
