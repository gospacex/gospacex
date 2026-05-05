package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"gopkg.in/yaml.v3"
)

// TemplateEngine 模板引擎
type TemplateEngine struct {
	config     *GeneratorConfig
	templates  map[string]*template.Template
	outputDir  string
}

// NewTemplateEngine creates new template engine
func NewTemplateEngine(outputDir string) *TemplateEngine {
	return &TemplateEngine{
		templates: make(map[string]*template.Template),
		outputDir: outputDir,
	}
}

// LoadConfig loads generator configuration
func (e *TemplateEngine) LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	e.config = &GeneratorConfig{}
	if err := yaml.Unmarshal(data, e.config); err != nil {
		return err
	}

	if e.config.EntityNameLower == "" {
		e.config.EntityNameLower = ToLowerCamelCase(e.config.EntityName)
	}

	return nil
}

// LoadTemplate loads a template file
func (e *TemplateEngine) LoadTemplate(name, path string) error {
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	e.templates[name] = tmpl
	return nil
}

// LoadTemplates loads all templates
func (e *TemplateEngine) LoadTemplates(templateDir string) error {
	absDir, err := filepath.Abs(templateDir)
	if err != nil {
		return err
	}
	
	dirs := []string{
		"pkg/config",
		"pkg/database",
		"pkg/logger",
		"internal/model",
		"internal/service",
		"internal/handler",
		"cmd/commands",
		"cmd",
	}
	
	for _, dir := range dirs {
		fullDir := filepath.Join(absDir, dir)
		if err := e.loadFromDir(fullDir, dir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	
	return nil
}

// loadFromDir loads templates from a directory
func (e *TemplateEngine) loadFromDir(dir string, prefix string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".go.tmpl") {
			name = name[:len(name)-len(".go.tmpl")]
		} else if strings.HasSuffix(name, ".tmpl") {
			name = name[:len(name)-len(".tmpl")]
		} else {
			continue
		}

		if prefix != "" {
			name = prefix + "/" + name
		}

		path := filepath.Join(dir, entry.Name())
		if err := e.LoadTemplate(name, path); err != nil {
			return err
		}
	}
	
	return nil
}

// Render renders a template to file
func (e *TemplateEngine) Render(tmplName, outputPath string) error {
	tmpl, ok := e.templates[tmplName]
	if !ok {
		return fmt.Errorf("template not found: %s", tmplName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, e.config); err != nil {
		return err
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// Generate generates all files for a project
func (e *TemplateEngine) Generate() error {
	// Generate pkg files
	pkgDirs := []string{"config", "database", "logger"}
	for _, dir := range pkgDirs {
		outputDir := filepath.Join(e.outputDir, "pkg", dir)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}
		
		prefix := "pkg/" + dir
		for name, tmpl := range e.templates {
			if !strings.HasPrefix(name, prefix + "/") {
				continue
			}
			
			fileName := strings.TrimPrefix(name, prefix + "/") + ".go"
			outputPath := filepath.Join(outputDir, fileName)
			
			if err := e.renderTemplate(tmpl, outputPath); err != nil {
				return err
			}
		}
	}
	
	// Generate internal files
	internalDirs := []string{"model", "service", "handler"}
	for _, dir := range internalDirs {
		outputDir := filepath.Join(e.outputDir, "internal", dir)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}
		
		prefix := "internal/" + dir
		for name, tmpl := range e.templates {
			if !strings.HasPrefix(name, prefix + "/") {
				continue
			}
			
			var fileName string
			if dir == "model" {
				fileName = toSnakeCase(e.config.EntityName) + ".go"
			} else if dir == "service" {
				fileName = toSnakeCase(e.config.EntityName) + "_service.go"
			} else if dir == "handler" {
				fileName = toSnakeCase(e.config.EntityName) + "_handler.go"
			}
			
			outputPath := filepath.Join(outputDir, fileName)
			if err := e.renderTemplate(tmpl, outputPath); err != nil {
				return err
			}
		}
	}
	
	// Generate cmd files
	cmdDirs := []string{"commands", ""}
	for _, dir := range cmdDirs {
		var outputDir string
		if dir == "" {
			outputDir = filepath.Join(e.outputDir, "cmd")
		} else {
			outputDir = filepath.Join(e.outputDir, "cmd", dir)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}
		
		var prefix string
		if dir == "" {
			prefix = "cmd"
		} else {
			prefix = "cmd/" + dir
		}
		
		for name, tmpl := range e.templates {
			if !strings.HasPrefix(name, prefix + "/") && !(prefix == "cmd" && name == "cmd/main") {
				continue
			}
			
			fileName := strings.TrimPrefix(name, prefix + "/") + ".go"
			outputPath := filepath.Join(outputDir, fileName)
			
			if err := e.renderTemplate(tmpl, outputPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// renderTemplate renders a single template
func (e *TemplateEngine) renderTemplate(tmpl *template.Template, outputPath string) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, e.config); err != nil {
		return err
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}
