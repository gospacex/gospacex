package scriptcenter

import (
	"fmt"
	"os"
	"path/filepath"

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
	// 如果没有启用 MQ，跳过模板渲染（避免加载不需要的模板）
	if g.config.MQ == "" {
		return nil
	}

	baseTemplateDir := "templates/scriptcenter"

	// 根据配置选择模板目录
	var templateDirs []string

	// 基础脚本中心模板（如果有的话）
	if hasTemplates(baseTemplateDir) {
		templateDirs = append(templateDirs, baseTemplateDir)
	}

	// 根据启用的 MQ 类型追加对应模板目录
	if g.config.MQ != "" {
		mqTemplateDir := baseTemplateDir + "/" + g.config.MQ
		if hasTemplates(mqTemplateDir) {
			templateDirs = append(templateDirs, mqTemplateDir)
		}
	}

	// 如果没有指定目录，退回 script（旧模板位置，根目录有 .tmpl）
	if len(templateDirs) == 0 {
		templateDirs = append(templateDirs, "templates/script")
	}

	loader := template.NewLoader("")
	for _, dir := range templateDirs {
		subLoader := template.NewLoader(dir)
		if err := subLoader.Load(""); err != nil {
			return fmt.Errorf("load templates from %s: %w", dir, err)
		}
		// 合并模板
		loader.Merge(subLoader)
	}

	processor := template.NewProcessor(loader)
	for _, dir := range templateDirs {
		if err := processor.ProcessDirectory(dir, g.config.OutputDir, g.config); err != nil {
			return fmt.Errorf("process directory %s: %w", dir, err)
		}
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
