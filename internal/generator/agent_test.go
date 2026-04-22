package generator

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentGenerator(t *testing.T) {
	g := NewAgentGenerator("/tmp/test", "test-module")
	assert.NotNil(t, g)
}

func TestAgentGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	g := NewAgentGenerator("test-agent", tmpDir)
	
	err := g.Generate()
	assert.NoError(t, err)
	
	files := []string{"go.mod", "readme.md"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		_, err := os.Stat(path)
		assert.False(t, os.IsNotExist(err), "File %s should exist", f)
	}
}

func TestAgentContent(t *testing.T) {
	g := NewAgentGenerator("test", "/tmp")
	content := g.agentContent()
	assert.Contains(t, content, "package agent")
}

func TestLLMClientContent(t *testing.T) {
	g := NewAgentGenerator("test", "/tmp")
	content := g.llmClientContent()
	assert.Contains(t, content, "package llm")
}

func TestPromptContent(t *testing.T) {
	g := NewAgentGenerator("test", "/tmp")
	content := g.promptContent()
	assert.Contains(t, content, "assistant")
}

func TestConfigContent(t *testing.T) {
	g := NewAgentGenerator("test", "/tmp")
	content := g.configContent()
	assert.Contains(t, content, "server")
}

func TestGoModContent(t *testing.T) {
	g := NewAgentGenerator("test-module", "/tmp")
	content := g.goModContent()
	assert.Contains(t, content, "module test-module")
}
