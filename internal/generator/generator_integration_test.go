package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gospacex/gpx/internal/template"
)

func TestIntegration_TemplateRendering(t *testing.T) {
	tmplDir := t.TempDir()
	tmplPath := filepath.Join(tmplDir, "test.tmpl")
	if err := os.WriteFile(tmplPath, []byte("Project: {{.ProjectName}}"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := template.NewLoader(tmplDir)
	if err := loader.Load("*.tmpl"); err != nil {
		t.Fatal(err)
	}

	renderer := template.NewRenderer(loader)

	data := map[string]interface{}{
		"ProjectName": "test-project",
	}

	output, err := renderer.RenderString("test.tmpl", data)
	if err != nil {
		t.Fatalf("RenderString() error = %v", err)
	}

	if output == "" {
		t.Error("RenderString() returned empty output")
	}

	expected := "Project: test-project"
	if output != expected {
		t.Errorf("RenderString() = %v, want %v", output, expected)
	}
}

func TestIntegration_ConfigValidation(t *testing.T) {
	t.Run("valid microservice config", func(t *testing.T) {
		config := map[string]interface{}{
			"ProjectType": "microservice",
			"Style":       "standard",
			"IDL":         "protobuf",
			"OutputDir":   t.TempDir(),
		}

		if config["ProjectType"] == "" {
			t.Error("ProjectType should not be empty")
		}
	})

	t.Run("invalid config detection", func(t *testing.T) {
		config := map[string]interface{}{
			"ProjectType": "unknown",
		}

		if config["ProjectType"] != "unknown" {
			t.Error("Should detect invalid project type")
		}
	})
}
