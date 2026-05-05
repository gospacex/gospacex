package generator

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/generator/microservice"
	"github.com/gospacex/gpx/internal/generator/monolith"
	"github.com/gospacex/gpx/internal/generator/scriptcenter"
)

// ProjectGenerator 项目生成器接口
type ProjectGenerator interface {
	Generate() error
}

// NewProjectGenerator creates a new generator based on project type
func NewProjectGenerator(cfg *config.ProjectConfig) ProjectGenerator {
	switch cfg.ProjectType {
	case "script":
		return scriptcenter.NewGenerator(cfg)
	case "microservice":
		return microservice.NewGenerator(cfg)
	case "monolith":
		return monolith.NewGenerator(cfg)
	default:
		return nil
	}
}

// GenerateProject dispatches generation based on project type
func GenerateProject(cfg *config.ProjectConfig) error {
	gen := NewProjectGenerator(cfg)
	if gen == nil {
		return fmt.Errorf("unknown project type: %s", cfg.ProjectType)
	}
	return gen.Generate()
}

// New creates a new generator based on project type
func New(cfg *config.ProjectConfig) interface{} {
	switch cfg.ProjectType {
	case "script":
		return scriptcenter.NewGenerator(cfg)
	case "microservice":
		return microservice.NewGenerator(cfg)
	case "monolith":
		return monolith.NewGenerator(cfg)
	default:
		return nil
	}
}

// Generate dispatches generation based on project type
func Generate(cfg *config.ProjectConfig) error {
	switch cfg.ProjectType {
	case "script":
		gen := scriptcenter.NewGenerator(cfg)
		return gen.Generate()
	case "microservice":
		gen := microservice.NewGenerator(cfg)
		return gen.Generate()
	case "monolith":
		gen := monolith.NewGenerator(cfg)
		return gen.Generate()
	default:
		return fmt.Errorf("unknown project type: %s", cfg.ProjectType)
	}
}
