package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

// Loader 模板加载器
type Loader struct {
	baseDir   string
	templates *template.Template
}

// NewLoader 创建模板加载器
func NewLoader(baseDir string) *Loader {
	l := &Loader{
		baseDir:   baseDir,
		templates: template.New("gpx"),
	}
	l.templates.Funcs(DefaultFuncs())
	return l
}

// Load 加载模板
func (l *Loader) Load(pattern string) error {
	// 遍历模板目录
	err := filepath.WalkDir(l.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// 只加载 .tmpl 文件
		if filepath.Ext(path) == ".tmpl" {
			relPath, err := filepath.Rel(l.baseDir, path)
			if err != nil {
				return err
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			tmplName := filepath.ToSlash(relPath)
			_, err = l.templates.New(tmplName).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse template %s: %w", tmplName, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	return nil
}

// Get 获取模板
func (l *Loader) Get(name string) (*template.Template, error) {
	tmpl := l.templates.Lookup(name)
	if tmpl == nil {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

// List 列出所有模板
func (l *Loader) List() []string {
	var names []string
	for _, tmpl := range l.templates.Templates() {
		names = append(names, tmpl.Name())
	}
	return names
}

// Merge 合并另一个 Loader 的模板
func (l *Loader) Merge(other *Loader) {
	if other == nil || other.templates == nil {
		return
	}
	for _, tmpl := range other.templates.Templates() {
		if l.templates.Lookup(tmpl.Name()) == nil {
			l.templates.AddParseTree(tmpl.Name(), tmpl.Tree)
		}
	}
}
