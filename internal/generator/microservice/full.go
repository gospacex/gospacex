package microservice

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type FullGenerator struct {
	config *config.ProjectConfig
}

func NewFullGenerator(cfg *config.ProjectConfig) *FullGenerator {
	return &FullGenerator{config: cfg}
}

func (g *FullGenerator) Generate() error {
	templateDir := "templates/microservice/full"

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
