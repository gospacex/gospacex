package generator

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMySQLGenerator_Generate tests MySQL template generation
func TestMySQLGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	g := NewMySQLGenerator(tmpDir)

	err := g.Generate("User")
	if err != nil {
		t.Fatalf("MySQL generate failed: %v", err)
	}

	// Verify files exist
	files := []string{
		"templates/database/mysql/config.go.tmpl",
		"templates/database/mysql/client.go.tmpl",
		"templates/database/mysql/crud.go.tmpl",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s should exist", f)
		}
	}
}

// TestRedisGenerator_Generate tests Redis template generation
func TestRedisGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	g := NewRedisGenerator(tmpDir)

	err := g.Generate()
	if err != nil {
		t.Fatalf("Redis generate failed: %v", err)
	}

	// Verify files exist
	files := []string{
		"templates/database/redis/config.go.tmpl",
		"templates/database/redis/client.go.tmpl",
		"templates/database/redis/cache.go.tmpl",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s should exist", f)
		}
	}
}
