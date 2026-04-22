package microservice

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type SimpleGenerator struct {
	config *config.ProjectConfig
}

func NewSimpleGenerator(cfg *config.ProjectConfig) *SimpleGenerator {
	return &SimpleGenerator{config: cfg}
}

func (g *SimpleGenerator) Generate() error {
	templateDir := "templates/microservice/simple"

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
