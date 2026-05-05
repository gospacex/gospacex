package microservice

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type DDDGenerator struct {
	config *config.ProjectConfig
}

func NewDDDGenerator(cfg *config.ProjectConfig) *DDDGenerator {
	return &DDDGenerator{config: cfg}
}

func (g *DDDGenerator) Generate() error {
	templateDir := "templates/microservice/ddd"
	loader := template.NewLoader(templateDir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}
	processor := template.NewProcessor(loader)
	return processor.ProcessDirectory(templateDir, g.config.OutputDir, g.config)
}
