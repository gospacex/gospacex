package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRotation_FileNaming(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("filename includes date", func(t *testing.T) {
		maxAgeDays := 7

		rotator, err := NewVitalRotator("audit", tmpDir, maxAgeDays)
		if err != nil {
			t.Fatalf("failed to create rotator: %v", err)
		}

		filename := rotator.currentName
		if !strings.Contains(filename, "audit-") {
			t.Errorf("filename %q should contain scene prefix", filename)
		}
		if !strings.Contains(filename, time.Now().Format("2006-01-02")) {
			t.Errorf("filename %q should contain today's date", filename)
		}
		if !strings.HasSuffix(filename, ".log") {
			t.Errorf("filename %q should have .log extension", filename)
		}

		rotator.Close()
	})
}

func TestRotation_DateExtraction(t *testing.T) {
	t.Run("extracts date from filename", func(t *testing.T) {
		rotator := &VitalRotator{
			scene:    "audit",
			dir:      "/tmp/logs",
			maxAgeDays: 7,
		}

		tests := []struct {
			filename    string
			expectDate  string
			expectValid bool
		}{
			{"audit-2026-04-26.log", "2026-04-26", true},
			{"audit-2026-04-25.log", "2026-04-25", true},
			{"business-2026-04-20.log", "2026-04-20", true},
			{"audit-invalid.log", "", false},
			{"audit-2026.log", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.filename, func(t *testing.T) {
				date, err := rotator.extractDateFromFilename(tt.filename)
				if tt.expectValid {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if date.Format("2006-01-02") != tt.expectDate {
						t.Errorf("date = %q, want %q", date.Format("2006-01-02"), tt.expectDate)
					}
				}
			})
		}
	})
}

func TestRotation_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("deletes files older than retention", func(t *testing.T) {
		maxAgeDays := 7

		rotator, err := NewVitalRotator("audit", tmpDir, maxAgeDays)
		if err != nil {
			t.Fatalf("failed to create rotator: %v", err)
		}
		rotator.maxAgeDays = 7

		oldFile := filepath.Join(tmpDir, "audit-2026-04-10.log")
		os.WriteFile(oldFile, []byte("old log content"), 0644)

		recentFile := filepath.Join(tmpDir, "audit-2026-04-25.log")
		os.WriteFile(recentFile, []byte("recent log content"), 0644)

		ctx := context.Background()
		deleted, err := rotator.cleanup(ctx)
		if err != nil {
			t.Errorf("cleanup failed: %v", err)
		}

		if deleted != 1 {
			t.Errorf("deleted %d files, want 1", deleted)
		}

		if _, err := os.Stat(recentFile); os.IsNotExist(err) {
			t.Error("recent file should not be deleted")
		}

		rotator.Close()
	})

	t.Run("uses filename date not modification time", func(t *testing.T) {
		maxAgeDays := 7

		rotator, err := NewVitalRotator("audit", tmpDir, maxAgeDays)
		if err != nil {
			t.Fatalf("failed to create rotator: %v", err)
		}
		rotator.maxAgeDays = 7

		touchedFile := filepath.Join(tmpDir, "audit-2026-04-10.log")
		os.WriteFile(touchedFile, []byte("touched content"), 0644)

		now := time.Now()
		os.Chtimes(touchedFile, now, now)

		ctx := context.Background()
		deleted, _ := rotator.cleanup(ctx)

		if deleted != 1 {
			t.Errorf("should delete file based on filename date, not mod time")
		}

		rotator.Close()
	})
}

func TestRotation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	maxAgeDays := 7

	rotator, err := NewVitalRotator("audit", tmpDir, maxAgeDays)
	if err != nil {
		t.Fatalf("failed to create rotator: %v", err)
	}

	for i := 0; i < 100; i++ {
		rotator.Write([]byte("test log entry\n"))
	}

	ctx := context.Background()
	deleted, err := rotator.cleanup(ctx)
	if err != nil {
		t.Errorf("cleanup failed: %v", err)
	}

	t.Logf("deleted %d old files", deleted)

	rotator.Close()
}