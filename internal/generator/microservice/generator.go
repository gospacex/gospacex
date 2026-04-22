package microservice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

// Generator 微服务生成器
type Generator struct {
	config *config.ProjectConfig
}

// NewGenerator 创建新的微服务生成器
func NewGenerator(cfg *config.ProjectConfig) *Generator {
	return &Generator{
		config: cfg,
	}
}

// Generate 生成微服务项目
func (g *Generator) Generate() error {
	templateDir := filepath.Join("templates/microservice", g.config.Style)

	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("template directory not found: %s", templateDir)
	}

	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return err
	}

	loader := template.NewLoader(templateDir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	processor := template.NewProcessor(loader)
	if err := processor.ProcessDirectory(templateDir, g.config.OutputDir, g.config); err != nil {
		return fmt.Errorf("process directory: %w", err)
	}

	return nil
}

// GenerateDocker 生成 Docker 相关文件
func (g *Generator) GenerateDocker(outputDir string) error {
	srcFiles := []string{
		"templates/microservice/docker-compose.yaml.tmpl",
		"templates/microservice/Dockerfile.tmpl",
	}

	for _, src := range srcFiles {
		dstName := strings.TrimSuffix(filepath.Base(src), ".tmpl")
		dst := filepath.Join(outputDir, dstName)
		loader := template.NewLoader(filepath.Dir(src))
		if err := loader.Load(""); err != nil {
			return err
		}
		processor := template.NewProcessor(loader)
		if err := processor.RenderFile(src, dst, g.config); err != nil {
			return err
		}
	}

	return nil
}

// GenerateLayers 生成额外指定的层级
func (g *Generator) GenerateLayers() error {
	baseTemplateDir := filepath.Join("templates/microservice", g.getTemplateStyle())

	loader := template.NewLoader(baseTemplateDir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	processor := template.NewProcessor(loader)
	if err := processor.ProcessDirectory(baseTemplateDir, g.config.OutputDir, g.config); err != nil {
		return fmt.Errorf("process directory: %w", err)
	}

	return nil
}

func (g *Generator) getTemplateStyle() string {
	switch g.config.Style {
	case "ddd":
		return "ddd"
	case "clean-arch":
		return "clean-arch"
	case "service-mesh":
		return "service-mesh"
	case "istio":
		return "istio/standard"
	case "full":
		return "standard"
	default:
		return "standard"
	}
}
