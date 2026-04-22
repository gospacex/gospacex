package microservice

import (
	"fmt"
	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type MeshGenerator struct {
	config *config.ProjectConfig
}

func NewMeshGenerator(cfg *config.ProjectConfig) *MeshGenerator {
	return &MeshGenerator{config: cfg}
}

func (g *MeshGenerator) Generate() error {
	templateDir := "templates/microservice/mesh"
	loader := template.NewLoader(templateDir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}
	processor := template.NewProcessor(loader)
	return processor.ProcessDirectory(templateDir, g.config.OutputDir, g.config)
}
