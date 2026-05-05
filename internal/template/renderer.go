package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Renderer 模板渲染器
type Renderer struct {
	loader *Loader
	funcs  template.FuncMap
}

// NewRenderer 创建模板渲染器
func NewRenderer(loader *Loader) *Renderer {
	return &Renderer{
		loader: loader,
		funcs:  DefaultFuncs(),
	}
}

// Render 渲染模板到文件
func (r *Renderer) Render(tmplName string, outputDir string, data interface{}) error {
	tmpl, err := r.loader.Get(tmplName)
	if err != nil {
		return err
	}

	// 生成输出文件路径（移除 .tmpl 扩展名）
	outputPath := strings.TrimSuffix(tmplName, ".tmpl")
	outputPath = filepath.Join(outputDir, outputPath)

	// 创建输出目录
	outputDir = filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", outputDir, err)
	}

	// 渲染模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", outputPath, err)
	}

	return nil
}

// RenderString 渲染模板到字符串
func (r *Renderer) RenderString(tmplName string, data interface{}) (string, error) {
	tmpl, err := r.loader.Get(tmplName)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
