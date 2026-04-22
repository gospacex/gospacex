package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// TestIntegrationGenerator 测试集成生成器
type TestIntegrationGenerator struct {
	serviceName string
	outputDir   string
}

// NewTestIntegrationGenerator creates new test integration generator
func NewTestIntegrationGenerator(serviceName, outputDir string) *TestIntegrationGenerator {
	return &TestIntegrationGenerator{
		serviceName: serviceName,
		outputDir:   outputDir,
	}
}

// Generate generates test integration code
func (g *TestIntegrationGenerator) Generate() error {
	dirs := []string{
		"tests",
		"tests/unit",
		"tests/integration",
		"tests/e2e",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"tests/unit/example_test.go":        g.unitTestContent(),
		"tests/integration/example_test.go": g.integrationTestContent(),
		"tests/e2e/example_test.go":         g.e2eTestContent(),
		"scripts/test.sh":                   g.testScriptContent(),
		".github/workflows/test.yml":        g.githubActionsContent(),
		"Makefile":                          g.makefileContent(),
	}

	for path, content := range files {
		fullPath := filepath.Join(g.outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (g *TestIntegrationGenerator) unitTestContent() string {
	return fmt.Sprintf(`package unit

import (
	"context"
	"testing"
	"%s/internal/service"
	"github.com/stretchr/testify/assert"
)

// TestExampleService_Create tests Create method
func TestExampleService_Create(t *testing.T) {
	svc := service.NewExampleService()
	
	entity, err := svc.Create(context.Background(), "test", "data")
	
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, "test", entity.Name)
}

// TestExampleService_GetByID tests GetByID method
func TestExampleService_GetByID(t *testing.T) {
	svc := service.NewExampleService()
	
	// Create first
	created, _ := svc.Create(context.Background(), "test", "data")
	
	// Get by ID
	entity, err := svc.GetByID(context.Background(), created.ID)
	
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, created.ID, entity.ID)
}

// TestExampleService_List tests List method
func TestExampleService_List(t *testing.T) {
	svc := service.NewExampleService()
	
	// Create some data
	svc.Create(context.Background(), "test1", "data1")
	svc.Create(context.Background(), "test2", "data2")
	
	// List
	entities, count, err := svc.List(context.Background(), 1, 10)
	
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(2))
	assert.Len(t, entities, 2)
}
`, g.serviceName)
}

func (g *TestIntegrationGenerator) integrationTestContent() string {
	return fmt.Sprintf(`package integration

import (
	"context"
	"os"
	"testing"
	"%s/internal/dal/mysql"
	"%s/internal/dal/redis"
	"%s/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ExampleTestSuite test suite
type ExampleTestSuite struct {
	suite.Suite
	svc *service.ExampleService
}

// SetupSuite runs once before all tests
func (s *ExampleTestSuite) SetupSuite() {
	// Initialize test database
	mysql.Init()
	redis.Init()
	
	// Create service
	s.svc = service.NewExampleService()
}

// TearDownSuite runs once after all tests
func (s *ExampleTestSuite) TearDownSuite() {
	// Clean up
	redis.Close()
}

// SetupTest runs before each test
func (s *ExampleTestSuite) SetupTest() {
	// Clean database before each test
}

// TearDownTest runs after each test
func (s *ExampleTestSuite) TearDownTest() {
	// Clean database after each test
}

// TestCreate tests create operation
func (s *ExampleTestSuite) TestCreate() {
	entity, err := s.svc.Create(context.Background(), "integration-test", "data")
	
	s.NoError(err)
	s.NotNil(entity)
	s.Greater(entity.ID, int64(0))
}

// TestGetByID tests get by ID operation
func (s *ExampleTestSuite) TestGetByID() {
	// Create first
	created, _ := s.svc.Create(context.Background(), "get-test", "data")
	
	// Get by ID
	entity, err := s.svc.GetByID(context.Background(), created.ID)
	
	s.NoError(err)
	s.Equal(created.ID, entity.ID)
}

// TestList tests list operation
func (s *ExampleTestSuite) TestList() {
	entities, count, err := s.svc.List(context.Background(), 1, 10)
	
	s.NoError(err)
	s.GreaterOrEqual(count, int64(0))
	s.NotNil(entities)
}

// TestMain runs all tests
func TestMain(m *testing.M) {
	if os.Getenv("TEST_ENV") != "integration" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
`, g.serviceName, g.serviceName, g.serviceName)
}

func (g *TestIntegrationGenerator) e2eTestContent() string {
	return fmt.Sprintf(`package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"github.com/stretchr/testify/assert"
)

// ExampleE2ETest e2e test
type ExampleE2ETest struct {
	baseURL string
}

// CreateRequest create request body
type CreateRequest struct {
	Name string json:"name"
	Data string json:"data"
}

// CreateResponse create response body
type CreateResponse struct {
	ID int64 json:"id"
}

// ListResponse list response body
type ListResponse struct {
	Data []interface{} json:"data"
	Total int64 json:"total"
}

// Setup sets up e2e test
func (e *ExampleE2ETest) Setup() {
	e.baseURL = os.Getenv("API_BASE_URL")
	if e.baseURL == "" {
		e.baseURL = "http://localhost:8080"
	}
}

// TestCreateAndList tests create and list operations
func TestCreateAndList(t *testing.T) {
	e := &ExampleE2ETest{}
	e.Setup()
	
	// Create
	createReq := CreateRequest{Name: "e2e-test", Data: "data"}
	createBody, _ := json.Marshal(createReq)
	
	resp, err := http.Post(e.baseURL+"/examples", "application/json", bytes.NewBuffer(createBody))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	
	var createResp CreateResponse
	json.NewDecoder(resp.Body).Decode(&createResp)
	assert.Greater(t, createResp.ID, int64(0))
	
	// List
	resp, err = http.Get(e.baseURL + "/examples")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var listResp ListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)
	assert.GreaterOrEqual(t, listResp.Total, int64(1))
}
`)
}

func (g *TestIntegrationGenerator) testScriptContent() string {
	return `#!/bin/bash
set -e

echo "Running tests..."

# Unit tests
echo "Running unit tests..."
go test ./... -v -cover -coverprofile=coverage.out

# Integration tests
echo "Running integration tests..."
TEST_ENV=integration go test ./tests/integration/... -v

# Coverage report
echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Tests completed!"
`
}

func (g *TestIntegrationGenerator) githubActionsContent() string {
	return `name: Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: password
          MYSQL_DATABASE: testdb
        ports:
          - 3306:3306
      redis:
        image: redis:alpine
        ports:
          - 6379:6379
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.26.2
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run unit tests
      run: go test ./... -v -cover -coverprofile=coverage.out
    
    - name: Run integration tests
      env:
        TEST_ENV: integration
        MYSQL_HOST: localhost
        REDIS_ADDR: localhost:6379
      run: go test ./tests/integration/... -v
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
`
}

func (g *TestIntegrationGenerator) makefileContent() string {
	return `.PHONY: test test-unit test-integration test-e2e coverage

# Run all tests
test:
	./scripts/test.sh

# Run unit tests
test-unit:
	go test ./... -v -short

# Run integration tests
test-integration:
	TEST_ENV=integration go test ./tests/integration/... -v

# Run e2e tests
test-e2e:
	TEST_ENV=e2e go test ./tests/e2e/... -v

# Generate coverage report
coverage:
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
`
}
