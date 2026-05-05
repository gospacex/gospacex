package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// RedisCRUDGenerator Redis CRUD 代码生成器
type RedisCRUDGenerator struct {
	outputDir string
}

// NewRedisCRUDGenerator creates new Redis CRUD generator
func NewRedisCRUDGenerator(outputDir string) *RedisCRUDGenerator {
	return &RedisCRUDGenerator{
		outputDir: outputDir,
	}
}

// RedisCRUDTemplateData 模板数据
type RedisCRUDTemplateData struct {
	StructName string
	LowerName  string
}

// getTemplateDir 获取模板目录
func (rg *RedisCRUDGenerator) getTemplateDir() string {
	return filepath.Join("/Users/hyx/work/gowork/src/gospacex", "templates")
}

// renderTemplate 渲染模板
func (rg *RedisCRUDGenerator) renderTemplate(templatePath string, data RedisCRUDTemplateData) (string, error) {
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

// Generate 生成 Redis CRUD 代码
func (rg *RedisCRUDGenerator) Generate(entityName string) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Join(rg.outputDir, "internal/repository"), 0o755); err != nil {
		return err
	}

	// 生成 Redis Repository
	if err := rg.generateRedisRepository(entityName); err != nil {
		return err
	}

	return nil
}

// generateRedisRepository 生成 Redis Repository 代码
func (rg *RedisCRUDGenerator) generateRedisRepository(entityName string) error {
	structName := rg.toPascalCase(entityName)
	lowerName := strings.ToLower(structName)

	data := RedisCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
	}

	templatePath := filepath.Join(rg.getTemplateDir(), "crud", "redis_repository.go.tmpl")
	content, err := rg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(rg.outputDir, "internal/repository", lowerName+"_redis_repository.go"),
		[]byte(content),
		0o644,
	)
}

// toPascalCase 转换为 PascalCase
func (rg *RedisCRUDGenerator) toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// 移除 t_ 前缀
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
