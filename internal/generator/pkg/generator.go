package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

// Generator pkg 组件生成器
// 负责在现有项目的 pkg 目录下添加各类通用组件（snowflake、config、database 等）
type Generator struct {
	cfg *config.ProjectConfig
}

// NewGenerator 创建 pkg 生成器
func NewGenerator(cfg *config.ProjectConfig) *Generator {
	return &Generator{cfg: cfg}
}

// Generate 根据配置生成所有启用的 pkg 组件
func (g *Generator) Generate() error {
	if g.cfg.PkgSnowflake {
		if err := g.generateSnowflake(); err != nil {
			return fmt.Errorf("generate snowflake: %w", err)
		}
	}
	return nil
}

// generateSnowflake 在目标项目的 pkg/snowflake 目录中生成 Snowflake ID 生成器
func (g *Generator) generateSnowflake() error {
	templates := map[string]string{
		"pkg/snowflake/snowflake.go":      "templates/pkg/snowflake/snowflake.go.tmpl",
		"pkg/snowflake/snowflake_test.go": "templates/pkg/snowflake/snowflake_test.go.tmpl",
	}
	return g.renderTemplates(templates)
}

// renderTemplates 渲染并写出模板文件到目标目录
func (g *Generator) renderTemplates(templates map[string]string) error {
	for dst, src := range templates {
		if _, err := os.Stat(src); err != nil {
			// 模板文件不存在时跳过
			fmt.Printf("Warning: template not found, skipping: %s\n", src)
			continue
		}

		dstPath := filepath.Join(g.cfg.OutputDir, dst)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dstPath), err)
		}

		loader := template.NewLoader(filepath.Dir(src))
		if err := loader.Load(""); err != nil {
			return fmt.Errorf("load template %s: %w", src, err)
		}

		processor := template.NewProcessor(loader)
		if err := processor.RenderFile(src, dstPath, g.cfg); err != nil {
			return fmt.Errorf("render %s -> %s: %w", src, dstPath, err)
		}
	}
	return nil
}
