package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
)

// TestGeneratedContent tests generated file content
func TestGeneratedContent(t *testing.T) {
	testDir := "/tmp/gpx-content-test"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)
	
	t.Run("MicroserviceStandardContent", func(t *testing.T) {
		g := NewMicroserviceStandardGenerator("test-svc", filepath.Join(testDir, "standard"))
		g.Generate()
		
		// Check main.go content
		content, _ := os.ReadFile(filepath.Join(testDir, "standard", "app/test-svc/main.go"))
		assert.Contains(t, string(content), "package main")
		assert.Contains(t, string(content), "kitex")
		
		// Check handler content
		content, _ = os.ReadFile(filepath.Join(testDir, "standard", "app/test-svc/handler/handler.go"))
		assert.Contains(t, string(content), "Handler")
		assert.Contains(t, string(content), "service")
	})
	
	t.Run("DDDStructure", func(t *testing.T) {
		g := NewMicroserviceDDDGenerator("test-ddd", filepath.Join(testDir, "ddd"))
		g.Generate()
		
		// Check DDD entity
		content, _ := os.ReadFile(filepath.Join(testDir, "ddd", "app/test-ddd/domain/entity/example.go"))
		assert.Contains(t, string(content), "entity")
		assert.Contains(t, string(content), "Example")
		
		// Check DDD repository interface
		content, _ = os.ReadFile(filepath.Join(testDir, "ddd", "app/test-ddd/domain/repository/example.go"))
		assert.Contains(t, string(content), "interface")
		assert.Contains(t, string(content), "Create")
		assert.Contains(t, string(content), "GetByID")
	})
	
	t.Run("IstioManifests", func(t *testing.T) {
		g := NewMicroserviceIstioGenerator("test-istio", filepath.Join(testDir, "istio"))
		g.Generate()
		
		// Check VirtualService
		content, _ := os.ReadFile(filepath.Join(testDir, "istio", "manifest/istio/virtual-service.yaml"))
		assert.Contains(t, string(content), "VirtualService")
		assert.Contains(t, string(content), "networking.istio.io")
		
		// Check DestinationRule
		content, _ = os.ReadFile(filepath.Join(testDir, "istio", "manifest/istio/destination-rule.yaml"))
		assert.Contains(t, string(content), "DestinationRule")
		assert.Contains(t, string(content), "subsets")
	})
	
	t.Run("ThriftIDL", func(t *testing.T) {
		g := NewMicroserviceThriftGenerator("test-thrift", filepath.Join(testDir, "thrift"))
		g.Generate()
		
		content, _ := os.ReadFile(filepath.Join(testDir, "thrift", "idl/thrift/example.thrift"))
		assert.Contains(t, string(content), "namespace go")
		assert.Contains(t, string(content), "struct Example")
		assert.Contains(t, string(content), "service ExampleService")
	})
	
	t.Run("DatabaseInit", func(t *testing.T) {
		g := NewDatabaseIntegrationGenerator(filepath.Join(testDir, "db"), []string{"mysql", "redis"})
		g.Generate()
		
		// Check MySQL init
		content, _ := os.ReadFile(filepath.Join(testDir, "db", "internal/dal/mysql/init.go"))
		assert.Contains(t, string(content), "gorm.Open")
		assert.Contains(t, string(content), "mysql.Open")
		
		// Check Redis init
		content, _ = os.ReadFile(filepath.Join(testDir, "db", "internal/dal/redis/init.go"))
		assert.Contains(t, string(content), "redis.NewClient")
		assert.Contains(t, string(content), "Ping")
	})
	
	t.Run("MiddlewareInit", func(t *testing.T) {
		g := NewMiddlewareIntegrationGenerator(filepath.Join(testDir, "mw"), []string{"jaeger", "kafka"})
		g.Generate()
		
		// Check Jaeger init
		content, _ := os.ReadFile(filepath.Join(testDir, "mw", "internal/middleware/jaeger/init.go"))
		assert.Contains(t, string(content), "jaeger")
		assert.Contains(t, string(content), "Tracer")
		
		// Check Kafka init
		content, _ = os.ReadFile(filepath.Join(testDir, "mw", "internal/middleware/kafka/init.go"))
		assert.Contains(t, string(content), "kafka-go")
		assert.Contains(t, string(content), "Producer")
	})
	
	t.Run("DTMPatterns", func(t *testing.T) {
		g := NewDTMIntegrationGenerator("test-dtm", filepath.Join(testDir, "dtm"))
		g.Generate()
		
		// Check SAGA
		content, _ := os.ReadFile(filepath.Join(testDir, "dtm", "internal/dtm/saga/order.go"))
		assert.Contains(t, string(content), "Saga")
		assert.Contains(t, string(content), "Add")
		
		// Check TCC
		content, _ = os.ReadFile(filepath.Join(testDir, "dtm", "internal/dtm/tcc/order.go"))
		assert.Contains(t, string(content), "TCC")
		assert.Contains(t, string(content), "CallBranch")
	})
	
	t.Run("TestStructure", func(t *testing.T) {
		g := NewTestIntegrationGenerator("test-svc", filepath.Join(testDir, "test"))
		g.Generate()
		
		// Check unit test
		content, _ := os.ReadFile(filepath.Join(testDir, "test", "tests/unit/example_test.go"))
		assert.Contains(t, string(content), "func Test")
		assert.Contains(t, string(content), "assert")
		
		// Check integration test
		content, _ = os.ReadFile(filepath.Join(testDir, "test", "tests/integration/example_test.go"))
		assert.Contains(t, string(content), "suite.Suite")
		assert.Contains(t, string(content), "SetupSuite")
	})
	
	t.Run("Documentation", func(t *testing.T) {
		g := NewDocsReleaseGenerator("test-docs", filepath.Join(testDir, "docs"))
		g.Generate()
		
		// Check README
		content, _ := os.ReadFile(filepath.Join(testDir, "docs", "README.md"))
		assert.Contains(t, string(content), "# test-docs")
		assert.Contains(t, string(content), "## 特性")
		assert.Contains(t, string(content), "## 快速开始")
		
		// Check LICENSE
		content, _ = os.ReadFile(filepath.Join(testDir, "docs", "LICENSE"))
		assert.Contains(t, strings.ToLower(string(content)), "apache")
	})
}
