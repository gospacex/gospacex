package scriptcenter

import (
	"fmt"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

// ElasticsearchGenerator Elasticsearch 脚本中心生成器
type ElasticsearchGenerator struct {
	*Generator
}

// NewElasticsearchGenerator 创建 Elasticsearch 生成器
func NewElasticsearchGenerator(cfg *config.ProjectConfig) *ElasticsearchGenerator {
	return &ElasticsearchGenerator{
		Generator: NewGenerator(cfg),
	}
}

// Generate 生成 Elasticsearch 脚本中心项目
func (g *ElasticsearchGenerator) Generate() error {
	// 设置 MQ 类型为 elasticsearch
	g.config.MQ = "elasticsearch"

	return g.Generator.Generate()
}

// GenerateWithTemplates 生成带有自定义模板的项目
func (g *ElasticsearchGenerator) GenerateWithTemplates() error {
	templateDir := "templates/scriptcenter/elasticsearch"

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
