package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS04_DDD_DirectoryStructure 测试 MS-04: DDD 模板目录结构
func TestMS04_DDD_DirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("ddd-test-srv", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证 DDD 分层目录结构
	requiredDirs := []string{
		// Domain 层
		"app/ddd-test-srv/domain/entity",
		"app/ddd-test-srv/domain/repository",
		"app/ddd-test-srv/domain/service",
		// Application 层
		"app/ddd-test-srv/application/service",
		"app/ddd-test-srv/application/dto",
		// Infrastructure 层
		"app/ddd-test-srv/infrastructure/persistence/mysql",
		"app/ddd-test-srv/infrastructure/persistence/redis",
		"app/ddd-test-srv/infrastructure/logger",
		// Interfaces 层
		"app/ddd-test-srv/interfaces/rpc",
		"app/ddd-test-srv/interfaces/http",
		// 其他
		"idl",
		"kitex_gen",
		"conf/dev",
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("DDD directory %s should exist", dir)
		}
	}
}

// TestMS05_DomainEntityGeneration 测试 MS-05: 领域实体生成
func TestMS05_DomainEntityGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("order-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 读取领域实体文件
	entityPath := filepath.Join(tmpDir, "app/order-service/domain/entity/example.go")
	content, err := os.ReadFile(entityPath)
	if err != nil {
		t.Fatalf("Read entity.go failed: %v", err)
	}

	contentStr := string(content)

	// 验证实体定义
	requiredContent := []string{
		"package entity",
		"type Example struct",
		"ID        int64",
		"Name      string",
		"Data      string",
		"CreatedAt time.Time",
		"UpdatedAt time.Time",
		"NewExample",
		"Validate",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("entity.go should contain: %s", req)
		}
	}
}

// TestMS06_ApplicationServiceGeneration 测试 MS-06: 用例层（应用服务）生成
func TestMS06_ApplicationServiceGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("user-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 读取应用服务文件
	appServicePath := filepath.Join(tmpDir, "app/user-service/application/service/example.go")
	content, err := os.ReadFile(appServicePath)
	if err != nil {
		t.Fatalf("Read example.go failed: %v", err)
	}

	contentStr := string(content)

	// 验证应用服务定义
	requiredContent := []string{
		"package service",
		"type ExampleAppService struct",
		"NewExampleAppService",
		"CreateExample",
		"GetExample",
		"UpdateExample",
		"DeleteExample",
		"context.Context",
		"repository.ExampleRepository",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("application service should contain: %s", req)
		}
	}
}

// TestMS07_InfrastructureGeneration 测试 MS-07: 基础设施层生成
func TestMS07_InfrastructureGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("product-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证 MySQL 仓储实现
	mysqlRepoPath := filepath.Join(tmpDir, "app/product-service/infrastructure/persistence/mysql/example.go")
	mysqlContent, err := os.ReadFile(mysqlRepoPath)
	if err != nil {
		t.Fatalf("Read mysql repository failed: %v", err)
	}

	mysqlStr := string(mysqlContent)
	requiredRepoContent := []string{
		"package mysql",
		"type ExampleRepositoryImpl struct",
		"NewExampleRepository",
		"Create",
		"GetByID",
		"Update",
		"Delete",
		"FindByName",
		"gorm.DB",
	}

	for _, req := range requiredRepoContent {
		if !strings.Contains(mysqlStr, req) {
			t.Errorf("mysql repository should contain: %s", req)
		}
	}

	// 验证 MySQL 初始化
	mysqlInitPath := filepath.Join(tmpDir, "app/product-service/infrastructure/persistence/mysql/init.go")
	initContent, err := os.ReadFile(mysqlInitPath)
	if err != nil {
		t.Fatalf("Read mysql init failed: %v", err)
	}

	initStr := string(initContent)
	requiredInitContent := []string{
		"package mysql",
		"var DB *gorm.DB",
		"func Init()",
		"func Close()",
		"gorm.Open",
	}

	for _, req := range requiredInitContent {
		if !strings.Contains(initStr, req) {
			t.Errorf("mysql init should contain: %s", req)
		}
	}
}

// TestMS08_DDDLayeredArchitecture 测试 MS-08: DDD 分层架构验证
func TestMS08_DDDLayeredArchitecture(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("inventory-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证依赖方向：Application 依赖 Domain
	appServicePath := filepath.Join(tmpDir, "app/inventory-service/application/service/example.go")
	appContent, err := os.ReadFile(appServicePath)
	if err != nil {
		t.Fatalf("Read application service failed: %v", err)
	}

	appStr := string(appContent)
	if !strings.Contains(appStr, "domain/repository") {
		t.Error("Application service should depend on domain/repository")
	}
	if !strings.Contains(appStr, "domain/entity") {
		t.Error("Application service should depend on domain/entity")
	}

	// 验证依赖方向：Infrastructure 依赖 Domain
	infraPath := filepath.Join(tmpDir, "app/inventory-service/infrastructure/persistence/mysql/example.go")
	infraContent, err := os.ReadFile(infraPath)
	if err != nil {
		t.Fatalf("Read infrastructure failed: %v", err)
	}

	infraStr := string(infraContent)
	if !strings.Contains(infraStr, "domain/entity") {
		t.Error("Infrastructure should depend on domain/entity")
	}
	if !strings.Contains(infraStr, "domain/repository") {
		t.Error("Infrastructure should depend on domain/repository")
	}

	// 验证 Repository 接口定义在 Domain 层
	repoInterfacePath := filepath.Join(tmpDir, "app/inventory-service/domain/repository/example.go")
	repoContent, err := os.ReadFile(repoInterfacePath)
	if err != nil {
		t.Fatalf("Read repository interface failed: %v", err)
	}

	repoStr := string(repoContent)
	if !strings.Contains(repoStr, "type ExampleRepository interface") {
		t.Error("Domain layer should define repository interface")
	}
}

// TestMS09_DDDUnitTests 测试 MS-09: DDD 生成器单元测试
func TestMS09_DDDUnitTests(t *testing.T) {
	// 验证生成器初始化
	g := NewMicroserviceDDDGenerator("test-ddd-service", "/tmp/test-ddd")

	if g == nil {
		t.Fatal("DDD Generator should not be nil")
	}
	if g.serviceName != "test-ddd-service" {
		t.Errorf("Expected serviceName 'test-ddd-service', got '%s'", g.serviceName)
	}
	if g.outputDir != "/tmp/test-ddd" {
		t.Errorf("Expected outputDir '/tmp/test-ddd', got '%s'", g.outputDir)
	}
}

// TestDDD_RepositoryInterface 测试仓储接口定义
func TestDDD_RepositoryInterface(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("payment-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	repoPath := filepath.Join(tmpDir, "app/payment-service/domain/repository/example.go")
	content, err := os.ReadFile(repoPath)
	if err != nil {
		t.Fatalf("Read repository interface failed: %v", err)
	}

	contentStr := string(content)

	// 验证仓储接口方法
	requiredMethods := []string{
		"Create(ctx context.Context, e *entity.Example) error",
		"GetByID(ctx context.Context, id int64) (*entity.Example, error)",
		"Update(ctx context.Context, e *entity.Example) error",
		"Delete(ctx context.Context, id int64) error",
		"FindByName(ctx context.Context, name string) (*entity.Example, error)",
	}

	for _, method := range requiredMethods {
		if !strings.Contains(contentStr, method) {
			t.Errorf("Repository interface should contain method: %s", method)
		}
	}
}

// TestDDD_DomainService 测试领域服务
func TestDDD_DomainService(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("order-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	domainServicePath := filepath.Join(tmpDir, "app/order-service/domain/service/example.go")
	content, err := os.ReadFile(domainServicePath)
	if err != nil {
		t.Fatalf("Read domain service failed: %v", err)
	}

	contentStr := string(content)

	// 验证领域服务定义
	requiredContent := []string{
		"package service",
		"type ExampleDomainService struct",
		"NewExampleDomainService",
		"ValidateExample",
		"ErrInvalidName",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("domain service should contain: %s", req)
		}
	}
}

// TestDDD_Configuration 测试配置文件
func TestDDD_Configuration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("api-gateway", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, "conf/dev/conf.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Read config failed: %v", err)
	}

	contentStr := string(content)

	// 验证配置内容
	requiredConfig := []string{
		"server:",
		"service_name: api-gateway",
		"address: \":8888\"",
		"registry:",
		"type: etcd",
		"mysql:",
		"max_open_conns: 100",
		"max_idle_conns: 10",
	}

	for _, req := range requiredConfig {
		if !strings.Contains(contentStr, req) {
			t.Errorf("conf.yaml should contain: %s", req)
		}
	}
}

// TestDDD_ProtoIDL 测试 Protobuf IDL
func TestDDD_ProtoIDL(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("user-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	protoPath := filepath.Join(tmpDir, "idl/example.proto")
	content, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("Read proto failed: %v", err)
	}

	contentStr := string(content)

	// 验证 Protobuf 定义
	requiredContent := []string{
		"syntax = \"proto3\";",
		"package example.v1;",
		"message Example",
		"message GetExampleReq",
		"message GetExampleResp",
		"message CreateExampleReq",
		"message CreateExampleResp",
		"service ExampleService",
		"rpc GetExample",
		"rpc CreateExample",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("example.proto should contain: %s", req)
		}
	}
}

// TestDDD_Readme 测试 README 文档
func TestDDD_Readme(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("demo-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	readmePath := filepath.Join(tmpDir, "readme.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Read readme.md failed: %v", err)
	}

	contentStr := string(content)

	// 验证 README 内容
	requiredContent := []string{
		"DDD",
		"domain/",
		"application/",
		"infrastructure/",
		"interfaces/",
		"kitex",
	}

	for _, req := range requiredContent {
		if !strings.Contains(contentStr, req) {
			t.Errorf("readme.md should contain: %s", req)
		}
	}
}

// TestDDD_EndToEnd 端到端测试
func TestDDD_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewMicroserviceDDDGenerator("e2e-ddd-service", tmpDir)
	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证 go.mod 存在
	goModPath := filepath.Join(tmpDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Fatal("go.mod should exist")
	}

	// 验证 go.mod 内容
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Read go.mod failed: %v", err)
	}

	goModStr := string(goModContent)
	if !strings.Contains(goModStr, "module e2e-ddd-service") {
		t.Error("go.mod should have correct module name")
	}
	if !strings.Contains(goModStr, "github.com/cloudwego/kitex") {
		t.Error("go.mod should include kitex dependency")
	}
	if !strings.Contains(goModStr, "gorm.io/gorm") {
		t.Error("go.mod should include gorm dependency")
	}

	// 验证所有必要文件存在
	requiredFiles := []string{
		"app/e2e-ddd-service/domain/entity/example.go",
		"app/e2e-ddd-service/domain/repository/example.go",
		"app/e2e-ddd-service/domain/service/example.go",
		"app/e2e-ddd-service/application/service/example.go",
		"app/e2e-ddd-service/infrastructure/persistence/mysql/example.go",
		"app/e2e-ddd-service/infrastructure/persistence/mysql/init.go",
		"app/e2e-ddd-service/interfaces/rpc/handler.go",
		"app/e2e-ddd-service/main.go",
		"idl/example.proto",
		"conf/dev/conf.yaml",
		"readme.md",
	}

	for _, f := range requiredFiles {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %s should exist", f)
		}
	}
}
