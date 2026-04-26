package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"gpx/pkg/logger/scene"
)

func Benchmark10kQPS_MixedBusinessAccessAuditError(b *testing.B) {
	dir := b.TempDir()
	logFile := filepath.Join(dir, "bench_10k.log")

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		b.Fatalf("failed to create log file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "bench-service"

	levelMgr := NewLevelManager()

	businessLogger, _ := scene.NewBusinessLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)

	accessLogger, _ := scene.NewAccessLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)

	auditLogger, _ := scene.NewAuditLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)

	errorLogger, _ := scene.NewErrorLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
		Rate:                cfg.RateLimit.Error.Rate,
		Burst:               cfg.RateLimit.Error.Burst,
		OverflowAction:      cfg.RateLimit.Error.OverflowAction,
	}, levelMgr)

	b.ResetTimer()
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			sceneType := i % 10
			switch sceneType {
			case 0, 1, 2, 3, 4, 5:
				businessLogger.Info(fmt.Sprintf("business log %d", i))
			case 6, 7:
				accessLogger.Info(fmt.Sprintf("access log %d", i))
			case 8:
				auditLogger.Log(&scene.AuditRecord{
					Action:    fmt.Sprintf("audit-%d", i),
					Resource:  "test",
					Timestamp: time.Now(),
				})
			case 9:
				errorLogger.Error(fmt.Sprintf("error log %d", i))
			}
			atomic.AddInt64(&i, 1)
		}
	})
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "QPS")
}

func BenchmarkVitalP99Latency(b *testing.B) {
	dir := b.TempDir()
	vitalFile := filepath.Join(dir, "vital_p99.log")

	file, err := os.OpenFile(vitalFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		b.Fatalf("failed to create vital file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "bench-service"
	cfg.Vital.BufferSize = 10000
	cfg.Vital.FsyncTimeout = 50 * time.Millisecond
	cfg.Vital.FallbackOnFull = true

	levelMgr := NewLevelManager()

	auditLogger, _ := scene.NewAuditLogger(file, &scene.Config{
		VitalBufferSize:      cfg.Vital.BufferSize,
		VitalSyncTimeout:     cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull:  cfg.Vital.FallbackOnFull,
	}, levelMgr)

	latencies := make([]time.Duration, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		auditLogger.Log(&scene.AuditRecord{
			Action:    fmt.Sprintf("action-%d", i),
			Resource:  "test-resource",
			Timestamp: time.Now(),
		})
		latencies[i] = time.Since(start)
	}
	b.StopTimer()

	auditLogger.Sync()

	p99 := calculateP99(latencies)
	p999 := calculateP999(latencies)

	b.ReportMetric(p99.Seconds()*1000, "p99_latency_ms")
	b.ReportMetric(p999.Seconds()*1000, "p999_latency_ms")

	if p99 > 20*time.Millisecond {
		b.Logf("WARNING: Vital P99 latency %v exceeds 20ms target", p99)
	}
}

func BenchmarkImportantP99Latency(b *testing.B) {
	dir := b.TempDir()
	logFile := filepath.Join(dir, "important_p99.log")

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		b.Fatalf("failed to create log file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "bench-service"

	levelMgr := NewLevelManager()

	businessLogger, _ := scene.NewBusinessLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)

	latencies := make([]time.Duration, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		businessLogger.Info(fmt.Sprintf("business log %d", i))
		latencies[i] = time.Since(start)
	}
	b.StopTimer()

	businessLogger.Sync()

	p99 := calculateP99(latencies)

	b.ReportMetric(p99.Seconds()*1000, "p99_latency_ms")

	if p99 > 1*time.Millisecond {
		b.Logf("WARNING: Important P99 latency %v exceeds 1ms target", p99)
	}
}

func Benchmark50kQPS_ImportantOnly(b *testing.B) {
	dir := b.TempDir()
	logFile := filepath.Join(dir, "50k_important.log")

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		b.Fatalf("failed to create log file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "bench-service"
	cfg.Vital.BufferSize = 50000

	levelMgr := NewLevelManager()

	businessLogger, _ := scene.NewBusinessLogger(file, &scene.Config{
		VitalBufferSize:     cfg.Vital.BufferSize,
		VitalSyncTimeout:    cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull: cfg.Vital.FallbackOnFull,
	}, levelMgr)

	b.ResetTimer()
	b.SetParallelism(500)

	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			businessLogger.Info(fmt.Sprintf("important log %d", atomic.AddInt64(&counter, 1)))
		}
	})
	b.StopTimer()

	businessLogger.Sync()

	qps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(qps, "QPS")

	if qps < 50000 {
		b.Logf("WARNING: Achieved %.0f QPS, target is 50k QPS", qps)
	}
}

func BenchmarkVitalBufferOverflowFallback(b *testing.B) {
	dir := b.TempDir()
	vitalFile := filepath.Join(dir, "vital_overflow.log")

	file, err := os.OpenFile(vitalFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		b.Fatalf("failed to create vital file: %v", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	cfg.ServiceName = "bench-service"
	cfg.Vital.BufferSize = 100
	cfg.Vital.FsyncTimeout = 1 * time.Millisecond
	cfg.Vital.FallbackOnFull = true

	levelMgr := NewLevelManager()

	auditLogger, _ := scene.NewAuditLogger(file, &scene.Config{
		VitalBufferSize:      cfg.Vital.BufferSize,
		VitalSyncTimeout:     cfg.Vital.FsyncTimeout,
		VitalFallbackOnFull:  cfg.Vital.FallbackOnFull,
	}, levelMgr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auditLogger.Log(&scene.AuditRecord{
			Action:    fmt.Sprintf("action-%d", i),
			Resource:  "test-resource",
			Timestamp: time.Now(),
		})
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "write_QPS")
	b.Logf("Buffer overflow fallback test completed: %d entries in %v", b.N, b.Elapsed())
}

func calculateP99(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	quickSort(sorted)

	idx := int(float64(len(sorted)) * 0.99)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func calculateP999(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	quickSort(sorted)

	idx := int(float64(len(sorted)) * 0.999)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func quickSort(a []time.Duration) {
	if len(a) < 2 {
		return
	}
	pivot := a[len(a)/2]
	i, j := 0, len(a)-1
	for i <= j {
		for a[i] < pivot {
			i++
		}
		for a[j] > pivot {
			j--
		}
		if i <= j {
			a[i], a[j] = a[j], a[i]
			i++
			j--
		}
	}
	quickSort(a[:j+1])
	quickSort(a[i:])
}

func TestPerformance_10kQPS_Documentation(t *testing.T) {
	t.Log("=== Performance Test 18.1: 10k QPS Mixed Load ===")
	t.Log("Ratio: Business:Access:Audit:Error = 6:2:1:1")
	t.Log("Run: go test -bench=Benchmark10kQPS_MixedBusinessAccessAuditError -benchmem")
}

func TestPerformance_VitalP99_Documentation(t *testing.T) {
	t.Log("=== Performance Test 18.2: Vital P99 < 20ms ===")
	t.Log("Run: go test -bench=BenchmarkVitalP99Latency -benchmem")
	t.Log("Target: P99 latency < 20ms")
}

func TestPerformance_ImportantP99_Documentation(t *testing.T) {
	t.Log("=== Performance Test 18.3: Important P99 < 1ms ===")
	t.Log("Run: go test -bench=BenchmarkImportantP99Latency -benchmem")
	t.Log("Target: P99 latency < 1ms")
}

func TestPerformance_50kQPS_Stress_Documentation(t *testing.T) {
	t.Log("=== Performance Test 18.4: 50k QPS Important-only Stress Test ===")
	t.Log("Run: go test -bench=Benchmark50kQPS_ImportantOnly -benchmem")
	t.Log("Target: Sustained 50k QPS")
}

func TestPerformance_VitalOverflow_Documentation(t *testing.T) {
	t.Log("=== Performance Test 18.5: Vital Buffer Overflow Fallback ===")
	t.Log("Run: go test -bench=BenchmarkVitalBufferOverflowFallback -benchmem")
	t.Log("Expected: Fallback to sync write when buffer full")
}
