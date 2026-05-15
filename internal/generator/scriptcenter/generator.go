package scriptcenter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

// Generator 脚本中心生成器
type Generator struct {
	config *config.ProjectConfig
}

// NewGenerator 创建新的脚本中心生成器
func NewGenerator(cfg *config.ProjectConfig) *Generator {
	return &Generator{
		config: cfg,
	}
}

// Generate 生成脚本中心项目
func (g *Generator) Generate() error {
	if g.config == nil {
		return fmt.Errorf("project config is nil")
	}

	if len(normalizeMQTypes(g.config.MQ)) == 0 {
		return NewScriptCenterGenerator(g.config).Generate(g.config.OutputDir)
	}

	if err := g.createDirectoryStructure(); err != nil {
		return fmt.Errorf("create directory structure: %w", err)
	}

	if err := g.renderTemplates(); err != nil {
		return fmt.Errorf("render templates: %w", err)
	}

	return nil
}

func (g *Generator) createDirectoryStructure() error {
	dirs := []string{
		"cmd/commands",
		"internal/handler",
		"internal/service",
		"internal/model",
		"internal/repository",
		"internal/mq",
		"pkg/config",
		"pkg/database",
		"pkg/logger",
		"configs",
		"deploy",
		"scripts",
		"logs",
	}

	for _, dir := range dirs {
		fullDir := filepath.Join(g.config.OutputDir, dir)
		if err := os.MkdirAll(fullDir, 0o755); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) renderTemplates() error {
	mqTypes := normalizeMQTypes(g.config.MQ)
	if len(mqTypes) == 0 {
		return nil
	}
	if len(mqTypes) > 1 {
		return fmt.Errorf("scriptcenter template generator supports one integration type at a time: %s", strings.Join(mqTypes, ", "))
	}

	templateDir := filepath.Join("templates", "scriptcenter", mqTypes[0])
	if !hasTemplates(templateDir) {
		return fmt.Errorf("scriptcenter templates not found for %q in %s", mqTypes[0], templateDir)
	}

	loader := template.NewLoader(templateDir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load templates from %s: %w", templateDir, err)
	}

	processor := template.NewProcessor(loader)
	if err := processor.ProcessDirectory(templateDir, g.config.OutputDir, g.config); err != nil {
		return fmt.Errorf("process directory %s: %w", templateDir, err)
	}

	return nil
}

// hasTemplates 检查目录是否存在且包含 .tmpl 文件（递归子目录）
func hasTemplates(dir string) bool {
	found := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过无法访问的路径，不中断遍历
		}
		if !info.IsDir() && len(info.Name()) > 5 && info.Name()[len(info.Name())-5:] == ".tmpl" {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}
