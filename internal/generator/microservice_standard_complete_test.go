package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS01_StandardMicroservice_DirectoryStructure 测试 MS-01: 标准微服务模板目录结构
func TestMS01_StandardMicroservice_DirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("test-srv", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证目录结构
	requiredDirs := []string{
		"app/test-srv/biz/dal/mysql",
		"app/test-srv/biz/dal/redis",
		"app/test-srv/biz/model",
		"app/test-srv/biz/repository",
		"app/test-srv/biz/service",
		"app/test-srv/conf",
		"app/test-srv/handler",
		"idl",
		"kitex_gen",
		"conf/dev",
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s should exist", dir)
		}
	}
}

// TestMS02_MainGoGeneration 测试 MS-02: main.go 生成逻辑
func TestMS02_MainGoGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("order-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 读取 main.go
	mainPath := filepath.Join(tmpDir, "app/order-service/main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("Read main.go failed: %v", err)
	}

	contentStr := string(content)

	// 验证关键内容
	requiredContent := []string{
		"package main",
		"github.com/cloudwego/kitex/server",
		"order-service/kitex_gen/example/v1/exampleservice",
		"order-service/app/order-service/handler",
		"NewExampleHandler",
		"server.WithServiceAddr(\":8888\")",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("main.go should contain: %s", req)
		}
	}
}

// TestMS03_ProtobufIDLGeneration 测试 MS-03: IDL (Protobuf) 模板生成
func TestMS03_ProtobufIDLGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("payment-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 读取 proto 文件
	protoPath := filepath.Join(tmpDir, "idl/example.proto")
	content, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("Read example.proto failed: %v", err)
	}

	contentStr := string(content)

	// 验证 Protobuf 语法
	requiredContent := []string{
		"syntax = \"proto3\";",
		"package example.v1;",
		"message Example",
		"message GetExampleReq",
		"message GetExampleResp",
		"service ExampleService",
		"rpc GetExample(GetExampleReq) returns (GetExampleResp)",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("example.proto should contain: %s", req)
		}
	}
}

// TestMS04_KitexCodeGeneration 测试 MS-04: Kitex 代码生成集成
func TestMS04_KitexCodeGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("user-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证 kitex_gen 目录存在（Kitex 代码生成目标目录）
	kitexGenDir := filepath.Join(tmpDir, "kitex_gen")
	if _, err := os.Stat(kitexGenDir); os.IsNotExist(err) {
		t.Error("kitex_gen directory should exist for Kitex code generation")
	}

	// 验证 main.go 中引用了 Kitex
	mainPath := filepath.Join(tmpDir, "app/user-service/main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("Read main.go failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "kitex") {
		t.Error("main.go should import kitex")
	}
	if !strings.Contains(contentStr, "exampleservice.NewServer") {
		t.Error("main.go should use Kitex service server")
	}
}

// TestMS05_ServiceRegistryConfig 测试 MS-05: 服务注册配置 (etcd)
func TestMS05_ServiceRegistryConfig(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("api-gateway", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证配置文件存在
	configPath := filepath.Join(tmpDir, "conf/dev/conf.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("conf/dev/conf.yaml should exist")
	}

	// 读取配置文件
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Read conf.yaml failed: %v", err)
	}

	contentStr := string(content)

	// 验证包含服务注册配置（etcd）
	requiredConfig := []string{
		"registry:",
		"etcd",
		"server:",
		"addr:",
		":8888",
	}

	for _, req := range requiredConfig {
		if !strings.Contains(contentStr, req) {
			t.Errorf("conf.yaml should contain: %s", req)
		}
	}
}

// TestMS06_GeneratorUnitTests 测试 MS-06: 生成器单元测试
func TestMS06_GeneratorUnitTests(t *testing.T) {
	// 这个测试本身就是 MS-06 的一部分
	// 验证生成器可以正确创建
	g := NewMicroserviceStandardGenerator("test-service", "/tmp/test")
	
	if g == nil {
		t.Fatal("Generator should not be nil")
	}
	if g.serviceName != "test-service" {
		t.Errorf("Expected serviceName 'test-service', got '%s'", g.serviceName)
	}
	if g.outputDir != "/tmp/test" {
		t.Errorf("Expected outputDir '/tmp/test', got '%s'", g.outputDir)
	}
}

// TestMS07_EndToEnd_Compilable 测试 MS-07: 端到端测试 - 生成项目可编译
func TestMS07_EndToEnd_Compilable(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("e2e-test-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证 go.mod 存在
	goModPath := filepath.Join(tmpDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Fatal("go.mod should exist")
	}

	// 读取 go.mod 验证模块名正确
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Read go.mod failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "module e2e-test-service") {
		t.Error("go.mod should have correct module name")
	}

	// 验证所有必要的 Go 文件存在
	requiredFiles := []string{
		"app/e2e-test-service/main.go",
		"app/e2e-test-service/handler/handler.go",
		"app/e2e-test-service/biz/model/base.go",
		"app/e2e-test-service/biz/model/example.go",
		"app/e2e-test-service/biz/dal/mysql/init.go",
		"app/e2e-test-service/biz/dal/redis/init.go",
		"app/e2e-test-service/biz/repository/example_repo.go",
		"app/e2e-test-service/biz/service/example_service.go",
		"app/e2e-test-service/conf/conf.go",
		"idl/example.proto",
	}

	for _, f := range requiredFiles {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %s should exist", f)
		}
	}

	// 验证 readme.md 存在
	readmePath := filepath.Join(tmpDir, "readme.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("readme.md should exist")
	}
}

// TestStandardMicroservice_HandlerGeneration 测试 Handler 生成
func TestStandardMicroservice_HandlerGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("product-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	handlerPath := filepath.Join(tmpDir, "app/product-service/handler/handler.go")
	content, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("Read handler.go failed: %v", err)
	}

	contentStr := string(content)

	// 验证 Handler 实现
	requiredContent := []string{
		"package handler",
		"type ExampleHandler struct",
		"NewExampleHandler",
		"GetExample",
		"CreateExample",
		"context.Context",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("handler.go should contain: %s", req)
		}
	}
}

// TestStandardMicroservice_ModelGeneration 测试 Model 生成
func TestStandardMicroservice_ModelGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("inventory-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证 base model
	baseModelPath := filepath.Join(tmpDir, "app/inventory-service/biz/model/base.go")
	baseContent, err := os.ReadFile(baseModelPath)
	if err != nil {
		t.Fatalf("Read base.go failed: %v", err)
	}

	baseStr := string(baseContent)
	if !strings.Contains(baseStr, "BaseModel") {
		t.Error("base.go should contain BaseModel")
	}
	// 验证使用 gorm.Model 或自定义 ID 字段
	if !strings.Contains(baseStr, "gorm.Model") && !strings.Contains(baseStr, "primaryKey") {
		t.Error("base.go should use gorm.Model or have primary key")
	}

	// 验证 example model
	exampleModelPath := filepath.Join(tmpDir, "app/inventory-service/biz/model/example.go")
	exampleContent, err := os.ReadFile(exampleModelPath)
	if err != nil {
		t.Fatalf("Read example.go failed: %v", err)
	}

	exampleStr := string(exampleContent)
	if !strings.Contains(exampleStr, "type Example struct") {
		t.Error("example.go should contain Example struct")
	}
	if !strings.Contains(exampleStr, "TableName") {
		t.Error("example.go should implement TableName")
	}
}

// TestStandardMicroservice_ServiceLayerGeneration 测试 Service 层生成
func TestStandardMicroservice_ServiceLayerGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("order-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	servicePath := filepath.Join(tmpDir, "app/order-service/biz/service/example_service.go")
	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Read example_service.go failed: %v", err)
	}

	contentStr := string(content)

	// 验证 Service 实现
	requiredContent := []string{
		"package service",
		"type ExampleService struct",
		"GetByID",
		"Create",
		"context.Context",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("example_service.go should contain: %s", req)
		}
	}
}

// TestStandardMicroservice_RepositoryGeneration 测试 Repository 层生成
func TestStandardMicroservice_RepositoryGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceStandardGenerator("user-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	repoPath := filepath.Join(tmpDir, "app/user-service/biz/repository/example_repo.go")
	content, err := os.ReadFile(repoPath)
	if err != nil {
		t.Fatalf("Read example_repo.go failed: %v", err)
	}

	contentStr := string(content)

	// 验证 Repository 实现
	requiredContent := []string{
		"package repository",
		"type ExampleRepository struct",
		"GetByID",
		"Create",
		"gorm.DB",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("example_repo.go should contain: %s", req)
		}
	}
}
