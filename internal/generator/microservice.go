package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type MicroserviceGenerator struct {
	config *ProjectConfig
}

type ProjectConfig struct {
	ProjectType    string
	Style          string
	ServiceName    string
	ModuleName     string
	OutputDir      string
	DB             []string
	DTMEnabled     bool
	DTMServer      string
	DTMMode        string
	IstioEnabled   bool
	IstioNamespace string
	IstioReplicas  int
	IstioImage     string
	Env            string
	Registry       string
	Config         string
	WithLayers     []string
	// Feature flags for template rendering
	NacosEnabled     bool
	ConsulEnabled    bool
	EtcdEnabled      bool
	SwaggerEnabled   bool
}

// GoModuleName returns module name for template rendering
func (g *ProjectConfig) GoModuleName() string {
	if g.ModuleName != "" {
		return g.ModuleName
	}
	if g.ServiceName != "" {
		return "github.com/gospacex/" + g.ServiceName
	}
	return "github.com/gospacex/" + filepath.Base(g.OutputDir)
}

// HasDB checks if database is enabled
func (g *ProjectConfig) HasDB(db string) bool {
	for _, d := range g.DB {
		if strings.EqualFold(d, db) {
			return true
		}
	}
	return false
}

// IsEnabled checks feature flag
func (g *ProjectConfig) IsEnabled(registry string) bool {
	if registry == "" {
		return false
	}
	return registry == g.Registry
}

func NewMicroserviceGenerator(cfg *ProjectConfig) *MicroserviceGenerator {
	return &MicroserviceGenerator{config: cfg}
}

func (g *MicroserviceGenerator) Generate() error {
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return err
	}

	serviceName := g.config.ServiceName
	if serviceName == "" {
		serviceName = strings.ToLower(filepath.Base(g.config.OutputDir))
	}

	switch strings.ToLower(g.config.Style) {
	case "simple", "standard":
		gen := NewMicroserviceStandardGenerator(serviceName, g.config.OutputDir)
		if err := gen.Generate(); err != nil {
			return err
		}
	case "ddd":
		gen := NewMicroserviceDDDGenerator(serviceName, g.config.OutputDir)
		if err := gen.Generate(); err != nil {
			return err
		}
	case "full":
		gen := NewMicroserviceThriftGenerator(serviceName, g.config.OutputDir)
		if err := gen.Generate(); err != nil {
			return err
		}
	case "istio":
		gen := NewMicroserviceIstioGenerator(serviceName, g.config.OutputDir)
		if err := gen.Generate(); err != nil {
			return err
		}
	}

	// Generate enhanced features based on config
	if err := g.GenerateEnv(serviceName); err != nil {
		return fmt.Errorf("GenerateEnv: %w", err)
	}
	if err := g.GenerateRegistry(serviceName); err != nil {
		return fmt.Errorf("GenerateRegistry: %w", err)
	}
	if err := g.GenerateConfig(serviceName); err != nil {
		return fmt.Errorf("GenerateConfig: %w", err)
	}

	// Generate priority config system when config center is enabled
	if g.config.Config != "" {
		// Copy all files from templates/pkg/config to output/pkg/config
		baseDir := "templates/pkg/config"
		if err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".tmpl") {
				return nil
			}
			
			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}
			
			destPath := filepath.Join(g.config.OutputDir, "pkg", "config", relPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return g.renderTemplate(destPath, string(content))
		}); err != nil {
			return err
		}
	}

	// Generate extra layers if requested
	if len(g.config.WithLayers) > 0 {
		if err := g.GenerateLayers(); err != nil {
			return fmt.Errorf("GenerateLayers: %w", err)
		}
	}

	return nil
}

func generateBasicStructure(outputDir, moduleName string, databases []string) error {
	dirs := []string{"cmd/api", "cmd/service", "internal/handler", "internal/service", "internal/model", "pkg/config", "pkg/logger", "pkg/database", "configs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(outputDir, dir), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (g *MicroserviceGenerator) getTemplateDir() string {
	base := "templates/microservice"
	switch g.config.Style {
	case "ddd":
		return filepath.Join(base, "ddd")
	case "istio-standard", "istio":
		return filepath.Join(base, "istio", "standard")
	case "istio-ddd":
		return filepath.Join(base, "istio", "ddd")
	default:
		return filepath.Join(base, "standard")
	}
}

func (g *MicroserviceGenerator) copyTemplates(templateDir string) error {
	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(g.config.OutputDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".tmpl") {
			return g.renderTemplate(destPath, string(content))
		}
		return os.WriteFile(destPath, content, info.Mode())
	})
}

func (g *MicroserviceGenerator) renderTemplate(destPath string, content string) error {
	destPath = strings.TrimSuffix(destPath, ".tmpl")
	tmpl, err := template.New("template").Funcs(template.FuncMap{
		"camelcase": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(string(s[0])) + s[1:]
		},
		"lowercase": strings.ToLower,
		"default": func(val, def string) string {
			if val == "" {
				return def
			}
			return val
		},
	}).Parse(content)
	if err != nil {
		return err
	}
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, g.config)
}

func (g *MicroserviceGenerator) copyDTMConfig() error {
	return nil
}

// GenerateEnv generates environment-specific configuration files
func (g *MicroserviceGenerator) GenerateEnv(serviceName string) error {
	if g.config.Env == "" || g.config.Env == "dev" {
		return nil
	}
	if g.config.Env == "prod" {
		svcName := serviceName
		if svcName == "" {
			svcName = g.config.ServiceName
		}
		return copyTemplate(
			"templates/microservice/conf/prod/conf.yaml.tmpl",
			filepath.Join(g.config.OutputDir, "conf/prod/conf.yaml"),
			svcName,
		)
	}
	return nil
}

// GenerateRegistry generates service registry integration code
func (g *MicroserviceGenerator) GenerateRegistry(serviceName string) error {
	if g.config.Registry == "" {
		return nil
	}
	registryFile := ""
	switch g.config.Registry {
	case "etcd":
		registryFile = "templates/simple/pkg/registry/etcd.go.tmpl"
	case "nacos":
		registryFile = "templates/simple/pkg/registry/nacos.go.tmpl"
	default:
		return nil
	}
	return copyTemplate(registryFile, filepath.Join(g.config.OutputDir, "pkg/registry/registry.go"), g.config.ServiceName)
}

// GenerateConfig generates config center integration code
func (g *MicroserviceGenerator) GenerateConfig(serviceName string) error {
	if g.config.Config == "" {
		return nil
	}
	configFile := ""
	switch g.config.Config {
	case "nacos":
		configFile = "templates/microservice/pkg/config/nacos.go.tmpl"
	case "apollo":
		configFile = "templates/microservice/pkg/config/apollo.go.tmpl"
	default:
		return nil
	}
	// We already copied all priority config files in the main flow, just copy the center integration
	return copyTemplate(configFile, filepath.Join(g.config.OutputDir, "pkg/config/center.go"), g.config.ServiceName)
}

// GenerateDocker generates Docker-related files
func (g *MicroserviceGenerator) GenerateDocker(serviceName string) error {
	if err := copyTemplate("templates/microservice/docker-compose.yaml.tmpl", filepath.Join(g.config.OutputDir, "docker-compose.yaml"), serviceName); err != nil {
		return err
	}
	return copyTemplate("templates/microservice/Dockerfile.tmpl", filepath.Join(g.config.OutputDir, "Dockerfile"), serviceName)
}

// copyTemplate is a helper function to copy template files
func copyTemplate(src, dst, serviceName string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmpl, err := template.New(src).Parse(string(content))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, map[string]string{"ServiceName": serviceName})
}

// GenerateLayers generates API/BFF/RPC layers based on flags
func (g *MicroserviceGenerator) GenerateLayers() error {
	templateDir := filepath.Join("templates/microservice", g.getTemplateStyle())
	
	// Walk the template directory and copy all .tmpl files for requested layers
	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		
		// Check if this layer is requested
		include := false
		for _, layer := range g.config.WithLayers {
			layer = strings.TrimSpace(layer)
			// Simple check: if the path contains the layer name anywhere
			if strings.Contains(relPath, layer) {
				include = true
				break
			}
		}
		if !include {
			return nil
		}
		
		// Remove .tmpl suffix for destination
		ext := filepath.Ext(relPath)
		var destRelPath string
		if ext == ".tmpl" {
			destRelPath = strings.TrimSuffix(relPath, ".tmpl")
		} else {
			destRelPath = relPath
		}
		destPath := filepath.Join(g.config.OutputDir, destRelPath)
		
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		return g.renderTemplate(destPath, string(content))
	})
}

// getTemplateStyle returns the template style directory
func (g *MicroserviceGenerator) getTemplateStyle() string {
	switch g.config.Style {
	case "ddd":
		return "ddd"
	case "istio":
		return "istio/standard"
	default:
		return "standard"
	}
}
