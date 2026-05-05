package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStandardMicroservice_Generate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &ProjectConfig{
		ProjectType: "microservice",
		Style:       "standard",
		ServiceName: "test-service",
		ModuleName:  "github.com/example/test-service",
		OutputDir:   tmpDir,
	}

	g := NewMicroserviceGenerator(cfg)
	err := g.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Verify generated files (matching actual implementation paths)
	files := []string{
		"app/test-service/main.go",
		"app/test-service/handler/handler.go",
		"app/test-service/biz/model/base.go",
		"app/test-service/biz/dal/mysql/init.go",
		"app/test-service/biz/dal/redis/init.go",
		"idl/example.proto",
		"go.mod",
		"readme.md",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s should exist", f)
		}
	}
}

func TestProtobufGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewProtobufGenerator(tmpDir)
	err := g.Generate("test-service", "api", "github.com/example/test-service")
	if err != nil {
		t.Fatalf("generate proto failed: %v", err)
	}

	// 验证 proto 文件存在
	path := filepath.Join(tmpDir, "api", "test-service.proto")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("proto file should exist")
	}
}

func TestKitexGenerator(t *testing.T) {
	// 只测试结构，不实际调用 kitex 命令
	g := NewKitexGenerator("/tmp", "test-module")
	if g.OutputDir != "/tmp" {
		t.Error("OutputDir should be /tmp")
	}
	if g.ModuleName != "test-module" {
		t.Error("ModuleName should be test-module")
	}
}
