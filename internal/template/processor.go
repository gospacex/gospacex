package template

import (
	"fmt"
	"os"
	"path/filepath"
)

// Processor 模板处理器 - 处理整个目录的模板渲染
type Processor struct {
	loader   *Loader
	renderer *Renderer
}

// NewProcessor 创建新的模板处理器
func NewProcessor(loader *Loader) *Processor {
	return &Processor{
		loader:   loader,
		renderer: NewRenderer(loader),
	}
}

// ProcessDirectory 处理整个模板目录，渲染所有 .tmpl 文件到输出目录
// 模板文件路径保持相对结构，输出时移除 .tmpl 扩展名
func (p *Processor) ProcessDirectory(templateDir string, outputDir string, data interface{}) error {
	absTemplateDir, err := filepath.Abs(templateDir)
	if err != nil {
		return fmt.Errorf("get absolute path for %s: %w", templateDir, err)
	}

	err = filepath.Walk(absTemplateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".tmpl" {
			return nil
		}

		relPath, err := filepath.Rel(absTemplateDir, path)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}

		tmplName := filepath.ToSlash(relPath)

		if err := p.renderer.Render(tmplName, outputDir, data); err != nil {
			return fmt.Errorf("render %s: %w", tmplName, err)
		}

		return nil
	})

	return err
}

// RenderFile 渲染单个模板文件到输出路径
func (p *Processor) RenderFile(templatePath, outputPath string, data interface{}) error {
	dir := filepath.Dir(templatePath)
	filename := filepath.Base(templatePath)

	if filepath.Ext(filename) != ".tmpl" {
		return fmt.Errorf("template file must have .tmpl extension: %s", templatePath)
	}

	loader := NewLoader(dir)
	if err := loader.Load(""); err != nil {
		return fmt.Errorf("load template: %w", err)
	}

	renderer := NewRenderer(loader)
	tmplName := filename
	outputDir := filepath.Dir(outputPath)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	return renderer.Render(tmplName, outputDir, data)
}
