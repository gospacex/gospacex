package scriptcenter

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type MongoDBGenerator struct {
	*Generator
}

func NewMongoDBGenerator(cfg *config.ProjectConfig) *MongoDBGenerator {
	return &MongoDBGenerator{
		Generator: NewGenerator(cfg),
	}
}

func (g *MongoDBGenerator) Generate() error {
	g.config.MQ = "mongodb"
	return g.Generator.Generate()
}

func (g *MongoDBGenerator) GenerateWithTemplates() error {
	templateDir := "templates/scriptcenter/mongodb"

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
