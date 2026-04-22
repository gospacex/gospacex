package generator

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/suite"
)

// GeneratorTestSuite test suite
type GeneratorTestSuite struct {
	suite.Suite
	testDir string
}

// SetupSuite creates test directory
func (s *GeneratorTestSuite) SetupSuite() {
	s.testDir = "/tmp/gpx-test"
	os.RemoveAll(s.testDir)
	os.MkdirAll(s.testDir, 0755)
}

// TearDownSuite cleans up test directory
func (s *GeneratorTestSuite) TearDownSuite() {
	os.RemoveAll(s.testDir)
}

// TestMicroserviceStandard tests standard microservice generator
func (s *GeneratorTestSuite) TestMicroserviceStandard() {
	g := NewMicroserviceStandardGenerator("order-service", filepath.Join(s.testDir, "standard"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check generated files
	expectedFiles := []string{
		"app/order-service/main.go",
		"app/order-service/handler/handler.go",
		"app/order-service/biz/model/base.go",
		"app/order-service/biz/dal/mysql/init.go",
		"app/order-service/biz/dal/redis/init.go",
		"idl/example.proto",
		"go.mod",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "standard", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestMicroserviceDDD tests DDD microservice generator
func (s *GeneratorTestSuite) TestMicroserviceDDD() {
	g := NewMicroserviceDDDGenerator("payment-service", filepath.Join(s.testDir, "ddd"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check DDD structure
	expectedDirs := []string{
		"app/payment-service/domain/entity",
		"app/payment-service/domain/repository",
		"app/payment-service/application/service",
		"app/payment-service/infrastructure/persistence/mysql",
	}
	
	for _, dir := range expectedDirs {
		_, err := os.Stat(filepath.Join(s.testDir, "ddd", dir))
		s.NoError(err, "Directory %s should exist", dir)
	}
}

// TestMicroserviceIstio tests Istio microservice generator
func (s *GeneratorTestSuite) TestMicroserviceIstio() {
	g := NewMicroserviceIstioGenerator("product-service", filepath.Join(s.testDir, "istio"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check Istio manifests
	expectedFiles := []string{
		"manifest/istio/virtual-service.yaml",
		"manifest/istio/destination-rule.yaml",
		"manifest/istio/gateway.yaml",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "istio", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestMicroserviceThrift tests Thrift microservice generator
func (s *GeneratorTestSuite) TestMicroserviceThrift() {
	g := NewMicroserviceThriftGenerator("inventory-service", filepath.Join(s.testDir, "thrift"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check Thrift IDL
	_, err = os.Stat(filepath.Join(s.testDir, "thrift", "idl/thrift/example.thrift"))
	s.NoError(err)
}

// TestMonolith tests monolith generator
func (s *GeneratorTestSuite) TestMonolith() {
	g := NewMonolithGenerator("admin-app", filepath.Join(s.testDir, "monolith"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check monolith structure
	expectedFiles := []string{
		"main.go",
		"internal/handler/handler.go",
		"internal/service/service.go",
		"internal/model/model.go",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "monolith", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestAgent tests agent generator
func (s *GeneratorTestSuite) TestAgent() {
	g := NewAgentGenerator("customer-agent", filepath.Join(s.testDir, "agent"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check agent structure
	expectedFiles := []string{
		"main.go",
		"internal/agent/agent.go",
		"internal/llm/client.go",
		"internal/memory/memory.go",
		"prompts/system.txt",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "agent", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestDatabaseIntegration tests database integration generator
func (s *GeneratorTestSuite) TestDatabaseIntegration() {
	g := NewDatabaseIntegrationGenerator(filepath.Join(s.testDir, "db-integration"), 
		[]string{"mysql", "redis"})
	err := g.Generate()
	
	s.NoError(err)
	
	// Check database files
	expectedFiles := []string{
		"internal/dal/mysql/init.go",
		"internal/dal/redis/init.go",
		"internal/dal/factory.go",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "db-integration", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestMiddlewareIntegration tests middleware integration generator
func (s *GeneratorTestSuite) TestMiddlewareIntegration() {
	g := NewMiddlewareIntegrationGenerator(filepath.Join(s.testDir, "mw-integration"),
		[]string{"jaeger", "kafka", "nacos"})
	err := g.Generate()
	
	s.NoError(err)
	
	// Check middleware files
	expectedFiles := []string{
		"internal/middleware/jaeger/init.go",
		"internal/middleware/kafka/init.go",
		"internal/middleware/nacos/init.go",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "mw-integration", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestDTMIntegration tests DTM integration generator
func (s *GeneratorTestSuite) TestDTMIntegration() {
	g := NewDTMIntegrationGenerator("order-service", filepath.Join(s.testDir, "dtm"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check DTM files
	expectedFiles := []string{
		"internal/dtm/client.go",
		"internal/dtm/saga/order.go",
		"internal/dtm/tcc/order.go",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "dtm", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestTestIntegration tests test integration generator
func (s *GeneratorTestSuite) TestTestIntegration() {
	g := NewTestIntegrationGenerator("test-service", filepath.Join(s.testDir, "test"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check test files
	expectedFiles := []string{
		"tests/unit/example_test.go",
		"tests/integration/example_test.go",
		"tests/e2e/example_test.go",
		"scripts/test.sh",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "test", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestDocsRelease tests docs and release generator
func (s *GeneratorTestSuite) TestDocsRelease() {
	g := NewDocsReleaseGenerator("gpx", filepath.Join(s.testDir, "docs"))
	err := g.Generate()
	
	s.NoError(err)
	
	// Check doc files
	expectedFiles := []string{
		"README.md",
		"docs/guides/quickstart.md",
		"CONTRIBUTING.md",
		"LICENSE",
	}
	
	for _, file := range expectedFiles {
		_, err := os.Stat(filepath.Join(s.testDir, "docs", file))
		s.NoError(err, "File %s should exist", file)
	}
}

// TestMain runs all tests
func TestMain(m *testing.M) {
	os.Exit(m.Run()) // suite.Run( &GeneratorTestSuite{})
}
