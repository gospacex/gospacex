package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// MongoCRUDGenerator MongoDB CRUD 代码生成器
type MongoCRUDGenerator struct {
	outputDir string
}

// NewMongoCRUDGenerator creates new MongoDB CRUD generator
func NewMongoCRUDGenerator(outputDir string) *MongoCRUDGenerator {
	return &MongoCRUDGenerator{
		outputDir: outputDir,
	}
}

// MongoCRUDTemplateData 模板数据
type MongoCRUDTemplateData struct {
	StructName     string
	LowerName      string
	CollectionName string
}

// getTemplateDir 获取模板目录
func (mg *MongoCRUDGenerator) getTemplateDir() string {
	return filepath.Join("/Users/hyx/work/gowork/src/gospacex", "templates")
}

// renderTemplate 渲染模板
func (mg *MongoCRUDGenerator) renderTemplate(templatePath string, data MongoCRUDTemplateData) (string, error) {
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

// Generate 生成 MongoDB CRUD 代码
func (mg *MongoCRUDGenerator) Generate(entityName string, collectionName string) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Join(mg.outputDir, "internal/repository"), 0o755); err != nil {
		return err
	}

	// 生成 MongoDB Repository
	if err := mg.generateMongoRepository(entityName, collectionName); err != nil {
		return err
	}

	return nil
}

// generateMongoRepository 生成 MongoDB Repository 代码
func (mg *MongoCRUDGenerator) generateMongoRepository(entityName string, collectionName string) error {
	structName := mg.toPascalCase(entityName)
	lowerName := strings.ToLower(structName)

	if collectionName == "" {
		collectionName = lowerName
	}

	data := MongoCRUDTemplateData{
		StructName:     structName,
		LowerName:      lowerName,
		CollectionName: collectionName,
	}

	templatePath := filepath.Join(mg.getTemplateDir(), "crud", "mongo_repository.go.tmpl")
	content, err := mg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(mg.outputDir, "internal/repository", lowerName+"_mongo_repository.go"),
		[]byte(content),
		0o644,
	)
}

// toPascalCase 转换为 PascalCase
func (mg *MongoCRUDGenerator) toPascalCase(s string) string {
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
