package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_New(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{"current dir", "."},
		{"templates dir", "templates"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader(tt.dir)
			if loader == nil {
				t.Error("NewLoader() returned nil loader")
			}
			if loader.baseDir != tt.dir {
				t.Errorf("baseDir = %v, want %v", loader.baseDir, tt.dir)
			}
		})
	}
}

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()

	validTemplate := filepath.Join(tmpDir, "valid.tmpl")
	if err := os.WriteFile(validTemplate, []byte("Hello {{.Name}}"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)

	t.Run("load all templates in directory", func(t *testing.T) {
		err := loader.Load("*.tmpl")
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		tmpl := loader.templates.Lookup("valid.tmpl")
		if tmpl == nil {
			t.Error("Load() did not load template")
		}
	})

	t.Run("load non-existent directory", func(t *testing.T) {
		loader := NewLoader("/nonexistent/path/12345")
		err := loader.Load("*.tmpl")
		if err == nil {
			t.Error("Load() expected error for non-existent directory")
		}
	})
}

func TestLoader_Get(t *testing.T) {
	tmpDir := t.TempDir()
	validTemplate := filepath.Join(tmpDir, "valid.tmpl")
	if err := os.WriteFile(validTemplate, []byte("Hello {{.Name}}"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	if err := loader.Load("*.tmpl"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		tmpl    string
		wantErr bool
	}{
		{"existing template", "valid.tmpl", false},
		{"non-existent template", "nonexistent.tmpl", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := loader.Get(tt.tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tmpl == nil {
				t.Error("Get() returned nil template without error")
			}
		})
	}
}

func TestLoader_List(t *testing.T) {
	tmpDir := t.TempDir()

	templates := []string{"template1.tmpl", "template2.tmpl", "template3.tmpl"}
	for _, name := range templates {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	loader := NewLoader(tmpDir)
	if err := loader.Load("*.tmpl"); err != nil {
		t.Fatal(err)
	}

	names := loader.List()
	if len(names) != len(templates) {
		t.Errorf("List() returned %d templates, want %d", len(names), len(templates))
	}
}
