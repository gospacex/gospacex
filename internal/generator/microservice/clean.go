package microservice

import (
	"fmt"
	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type CleanGenerator struct {
	config *config.ProjectConfig
}

func NewCleanGenerator(cfg *config.ProjectConfig) *CleanGenerator {
	return &CleanGenerator{config: cfg}
}

func (g *CleanGenerator) Generate() error {
	templateDir := "templates/microservice/clean"
	loader := template.NewLoader(templateDir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}
	processor := template.NewProcessor(loader)
	return processor.ProcessDirectory(templateDir, g.config.OutputDir, g.config)
}
