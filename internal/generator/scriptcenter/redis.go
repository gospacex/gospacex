package scriptcenter

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

type RedisGenerator struct {
	*Generator
}

func NewRedisGenerator(cfg *config.ProjectConfig) *RedisGenerator {
	return &RedisGenerator{
		Generator: NewGenerator(cfg),
	}
}

func (g *RedisGenerator) Generate() error {
	g.config.MQ = "redis"
	return g.Generator.Generate()
}

func (g *RedisGenerator) GenerateWithTemplates() error {
	templateDir := "templates/scriptcenter/redis"

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
