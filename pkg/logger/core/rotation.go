package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type VitalRotator struct {
	scene       string
	dir         string
	maxAgeDays  int
	file        *os.File
	currentName string
}

func NewVitalRotator(scene, dir string, maxAgeDays int) (*VitalRotator, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	rotator := &VitalRotator{
		scene:      scene,
		dir:        dir,
		maxAgeDays: maxAgeDays,
	}

	filename := rotator.generateFilename()
	f, err := os.OpenFile(filepath.Join(dir, filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	rotator.file = f
	rotator.currentName = filename

	return rotator, nil
}

func (r *VitalRotator) generateFilename() string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s-%s.log", r.scene, date)
}

func (r *VitalRotator) currentFile() string {
	return r.currentName
}

func (r *VitalRotator) Write(p []byte) (int, error) {
	if r.file == nil {
		return 0, fmt.Errorf("rotator closed")
	}

	filename := r.generateFilename()
	if filename != r.currentName {
		r.rotate(filename)
	}

	return r.file.Write(p)
}

func (r *VitalRotator) rotate(newFilename string) error {
	oldFile := r.file

	f, err := os.OpenFile(filepath.Join(r.dir, newFilename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open new file: %w", err)
	}

	if oldFile != nil {
		if err := oldFile.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "vital: sync old file failed: %v\n", err)
		}
		if err := oldFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "vital: close old file failed: %v\n", err)
		}
	}

	r.file = f
	r.currentName = newFilename

	return nil
}

func (r *VitalRotator) extractDateFromFilename(filename string) (time.Time, error) {
	name := filepath.Base(filename)

	if !strings.HasSuffix(name, ".log") {
		return time.Time{}, fmt.Errorf("invalid filename format: %s", filename)
	}

	name = strings.TrimSuffix(name, ".log")

	if len(name) < 10 {
		return time.Time{}, fmt.Errorf("filename too short: %s", filename)
	}

	dateStr := name[len(name)-10:]

	for i := 0; i < len(dateStr); i++ {
		if i == 4 || i == 7 {
			if dateStr[i] != '-' {
				return time.Time{}, fmt.Errorf("invalid date format in filename: %s", filename)
			}
			continue
		}
		if dateStr[i] < '0' || dateStr[i] > '9' {
			return time.Time{}, fmt.Errorf("invalid character in date: %s", filename)
		}
	}

	year, err := strconv.Atoi(dateStr[0:4])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %w", err)
	}
	month, err := strconv.Atoi(dateStr[5:7])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %w", err)
	}
	day, err := strconv.Atoi(dateStr[8:10])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %w", err)
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}

func (r *VitalRotator) cleanup(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return 0, fmt.Errorf("read dir: %w", err)
	}

	var logFiles []os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), r.scene+"-") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		logFiles = append(logFiles, info)
	}

	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Name() < logFiles[j].Name()
	})

	cutoff := time.Now().AddDate(0, 0, -r.maxAgeDays)
	deleted := 0

	for _, info := range logFiles {
		fileDate, err := r.extractDateFromFilename(info.Name())
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			filePath := filepath.Join(r.dir, info.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Fprintf(os.Stderr, "vital: failed to delete old log %s: %v\n", filePath, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "vital: deleted old log %s (date: %s)\n", filePath, fileDate.Format("2006-01-02"))
			deleted++
		}
	}

	return deleted, nil
}

func (r *VitalRotator) Sync() error {
	if r.file == nil {
		return nil
	}
	return r.file.Sync()
}

func (r *VitalRotator) StartCleanupRoutine(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := r.cleanup(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "vital: cleanup failed: %v\n", err)
				}
			}
		}
	}()
}

func (r *VitalRotator) Close() error {
	if r.file == nil {
		return nil
	}

	if err := r.file.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	err := r.file.Close()
	r.file = nil
	return err
}
