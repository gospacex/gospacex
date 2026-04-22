package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewDTMIntegrationGenerator tests constructor
func TestNewDTMIntegrationGenerator(t *testing.T) {
	serviceName := "order-service"
	outputDir := "/tmp/test"

	gen := NewDTMIntegrationGenerator(serviceName, outputDir)

	assert.Equal(t, serviceName, gen.serviceName)
	assert.Equal(t, outputDir, gen.outputDir)
}

// TestDTMIntegrationGenerator_Generate tests generation
func TestDTMIntegrationGenerator_Generate(t *testing.T) {
	testDir := "/tmp/gpx-test-dtm"
	os.RemoveAll(testDir)

	serviceName := "test-order"
	gen := NewDTMIntegrationGenerator(serviceName, testDir)

	err := gen.Generate()
	assert.NoError(t, err)

	expectedFiles := []string{
		"internal/dtm/client.go",
		"internal/dtm/saga/order.go",
		"internal/dtm/tcc/order.go",
		"internal/dtm/workflow/order.go",
		"configs/dtm.yaml",
		"readme_dtm.md",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(testDir, file)
		_, err := os.Stat(fullPath)
		assert.NoError(t, err, "File %s should exist", file)
	}

	os.RemoveAll(testDir)
}

// TestDTMIntegrationGenerator_ClientContent tests client generation
func TestDTMIntegrationGenerator_ClientContent(t *testing.T) {
	gen := NewDTMIntegrationGenerator("test-svc", "/tmp/test")
	content := gen.clientContent()

	checks := []string{
		"package dtm",
		"dtmcli.NewDtmClient",
		"Init()",
		"DTM_SERVER",
	}

	for _, check := range checks {
		if !hasSubstring(content, check) {
			t.Errorf("clientContent should contain: %s", check)
		}
	}
}

// TestDTMIntegrationGenerator_SagaOrderContent tests SAGA order generation
func TestDTMIntegrationGenerator_SagaOrderContent(t *testing.T) {
	gen := NewDTMIntegrationGenerator("test-svc", "/tmp/test")
	content := gen.sagaOrderContent()

	checks := []string{
		"package saga",
		"CreateOrderSaga",
		"NewSaga",
		"Add(dtmcli.GenActionURL",
		" saga.Submit()",
	}

	for _, check := range checks {
		if !hasSubstring(content, check) {
			t.Errorf("sagaOrderContent should contain: %s", check)
		}
	}
}

// TestDTMIntegrationGenerator_TDTMOrderContent tests TCC order generation
func TestDTMIntegrationGenerator_TDTMOrderContent(t *testing.T) {
	gen := NewDTMIntegrationGenerator("test-svc", "/tmp/test")
	content := gen.tccOrderContent()

	checks := []string{
		"package tcc",
		"CreateOrderTCC",
		"NewTCC(",
		" tcc.Call",
	}

	for _, check := range checks {
		if !hasSubstring(content, check) {
			t.Errorf("tccOrderContent should contain: %s", check)
		}
	}
}

// TestDTMIntegrationGenerator_WorkflowOrderContent tests workflow order generation
func TestDTMIntegrationGenerator_WorkflowOrderContent(t *testing.T) {
	gen := NewDTMIntegrationGenerator("test-svc", "/tmp/test")
	content := gen.workflowOrderContent()

	checks := []string{
		"package workflow",
		"CreateOrderWorkflow",
		"NewWorkflow(",
	}

	for _, check := range checks {
		if !hasSubstring(content, check) {
			t.Errorf("workflowOrderContent should contain: %s", check)
		}
	}
}

// Helper function for substring checking
func hasSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || hasSubstringHelper(s, substr, 0))
}

func hasSubstringHelper(s, substr string, start int) bool {
	if start > len(s)-len(substr) {
		return false
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return hasSubstringHelper(s, substr, start+1)
}
