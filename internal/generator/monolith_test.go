package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewMonolithGenerator tests constructor
func TestNewMonolithGenerator(t *testing.T) {
	projectName := "admin-app"
	outputDir := "/tmp/test"

	gen := NewMonolithGenerator(projectName, outputDir)

	assert.Equal(t, projectName, gen.projectName)
	assert.Equal(t, outputDir, gen.outputDir)
}

// TestMonolithGenerator_Generate tests generation
func TestMonolithGenerator_Generate(t *testing.T) {
	testDir := "/tmp/gpx-test-monolith"
	os.RemoveAll(testDir)

	projectName := "test-admin"
	gen := NewMonolithGenerator(projectName, testDir)

	err := gen.Generate()
	assert.NoError(t, err)

	expectedFiles := []string{
		"main.go",
		"go.mod",
		"readme.md",
		"internal/handler/handler.go",
		"internal/service/service.go",
		"internal/repository/repository.go",
		"internal/model/model.go",
		"configs/config.yaml",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(testDir, file)
		_, err := os.Stat(fullPath)
		assert.NoError(t, err, "File %s should exist", file)
	}

	os.RemoveAll(testDir)
}

// TestMonolithGenerator_MainContent tests main.go generation
func TestMonolithGenerator_MainContent(t *testing.T) {
	gen := NewMonolithGenerator("test-app", "/tmp/test")
	content := gen.mainContent()

	checks := []string{
		"package main",
		"github.com/cloudwego/hertz/pkg/app/server",
		"test-app/internal/handler",
		"handler.Register",
		"h.Spin()",
	}

	for _, check := range checks {
		if !containsMonolith(content, check) {
			t.Errorf("mainContent should contain: %s", check)
		}
	}
}

// TestMonolithGenerator_HandlerContent tests handler generation
func TestMonolithGenerator_HandlerContent(t *testing.T) {
	gen := NewMonolithGenerator("test-app", "/tmp/test")
	content := gen.handlerContent()

	checks := []string{
		"package handler",
		"func Register",
		"h.GET",
		"h.POST",
	}

	for _, check := range checks {
		if !containsMonolith(content, check) {
			t.Errorf("handlerContent should contain: %s", check)
		}
	}
}

// TestMonolithGenerator_ServiceContent tests service generation
func TestMonolithGenerator_ServiceContent(t *testing.T) {
	gen := NewMonolithGenerator("test-app", "/tmp/test")
	content := gen.serviceContent()

	checks := []string{
		"package service",
		"type ExampleService struct",
	}

	for _, check := range checks {
		if !containsMonolith(content, check) {
			t.Errorf("serviceContent should contain: %s", check)
		}
	}
}

// TestMonolithGenerator_ModelContent tests model generation
func TestMonolithGenerator_ModelContent(t *testing.T) {
	gen := NewMonolithGenerator("test-app", "/tmp/test")
	content := gen.modelContent()

	checks := []string{
		"package model",
		"type Example struct",
		"gorm:\"primaryKey;autoIncrement\"",
	}

	for _, check := range checks {
		if !containsMonolith(content, check) {
			t.Errorf("modelContent should contain: %s", check)
		}
	}
}

// TestMonolithGenerator_RepoContent tests repository generation
func TestMonolithGenerator_RepoContent(t *testing.T) {
	gen := NewMonolithGenerator("test-app", "/tmp/test")
	content := gen.repoContent()

	checks := []string{
		"package repository",
		"type ExampleRepository struct",
		"db *gorm.DB",
	}

	for _, check := range checks {
		if !containsMonolith(content, check) {
			t.Errorf("repoContent should contain: %s", check)
		}
	}
}

func containsMonolith(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsMonolithHelper(s, substr, 0))
}

func containsMonolithHelper(s, substr string, start int) bool {
	if start > len(s)-len(substr) {
		return false
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return containsMonolithHelper(s, substr, start+1)
}
