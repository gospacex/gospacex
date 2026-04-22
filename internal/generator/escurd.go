package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ESCRUDGenerator Elasticsearch CRUD 代码生成器
type ESCRUDGenerator struct {
	outputDir string
}

// NewESCRUDGenerator creates new ES CRUD generator
func NewESCRUDGenerator(outputDir string) *ESCRUDGenerator {
	return &ESCRUDGenerator{
		outputDir: outputDir,
	}
}

// ESCRUDTemplateData 模板数据
type ESCRUDTemplateData struct {
	StructName string
	LowerName  string
	IndexName  string
}

// getTemplateDir 获取模板目录
func (eg *ESCRUDGenerator) getTemplateDir() string {
	return filepath.Join("/Users/hyx/work/gowork/src/gospacex", "templates")
}

// renderTemplate 渲染模板
func (eg *ESCRUDGenerator) renderTemplate(templatePath string, data ESCRUDTemplateData) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Generate 生成 Elasticsearch CRUD 代码
func (eg *ESCRUDGenerator) Generate(entityName string, indexName string) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Join(eg.outputDir, "internal/repository"), 0o755); err != nil {
		return err
	}

	// 生成 ES Repository
	if err := eg.generateESRepository(entityName, indexName); err != nil {
		return err
	}

	return nil
}

// generateESRepository 生成 ES Repository 代码
func (eg *ESCRUDGenerator) generateESRepository(entityName string, indexName string) error {
	structName := eg.toPascalCase(entityName)
	lowerName := strings.ToLower(structName)

	if indexName == "" {
		indexName = lowerName
	}

	data := ESCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		IndexName:  indexName,
	}

	templatePath := filepath.Join(eg.getTemplateDir(), "crud", "es_repository.go.tmpl")
	content, err := eg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(eg.outputDir, "internal/repository", lowerName+"_es_repository.go"),
		[]byte(content),
		0o644,
	)
}

// toPascalCase 转换为 PascalCase
func (eg *ESCRUDGenerator) toPascalCase(s string) string {
	if s == "" {
		return s
	}

	s = strings.TrimPrefix(s, "t_")

	parts := strings.Split(s, "_")
	var result strings.Builder

	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(string(part[0])))
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	return result.String()
}
