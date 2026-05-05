package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateRedisCRUDGenerator 基于模板的 Redis CRUD 生成器
type TemplateRedisCRUDGenerator struct {
	outputDir string
	entity    string
}

// NewTemplateRedisCRUDGenerator creates new template-based Redis CRUD generator
func NewTemplateRedisCRUDGenerator(outputDir, entity string) *TemplateRedisCRUDGenerator {
	return &TemplateRedisCRUDGenerator{
		outputDir: outputDir,
		entity:    entity,
	}
}

// RedisFullCRUDTemplateData 模板数据
type RedisFullCRUDTemplateData struct {
	StructName string
	LowerName  string
	OutputDir  string
}

// getTemplateDir 获取模板目录
func (tg *TemplateRedisCRUDGenerator) getTemplateDir() string {
	return filepath.Join("/Users/hyx/work/gowork/src/gospacex", "templates")
}

// renderTemplate 渲染模板
func (tg *TemplateRedisCRUDGenerator) renderTemplate(templatePath string, data RedisFullCRUDTemplateData) (string, error) {
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

// Generate 生成完整分层 CRUD 代码
func (tg *TemplateRedisCRUDGenerator) Generate() error {
	// 创建目录结构
	dirs := []string{
		"internal/service",
		"internal/repository",
		"internal/model",
		"internal/dal/redis",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tg.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	// 生成 Model
	if err := tg.generateModel(); err != nil {
		return err
	}

	// 生成 Repository 接口
	if err := tg.generateRepositoryInterface(); err != nil {
		return err
	}

	// 生成 Repository 实现
	if err := tg.generateRepositoryImpl(); err != nil {
		return err
	}

	// 生成 Service
	if err := tg.generateService(); err != nil {
		return err
	}

	// 生成 DAL init
	if err := tg.generateDALInit(); err != nil {
		return err
	}

	return nil
}

// generateModel 生成 Model
func (tg *TemplateRedisCRUDGenerator) generateModel() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)

	data := RedisFullCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "redis_model.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/model", lowerName+".go"),
		[]byte(content),
		0o644,
	)
}

// generateRepositoryInterface 生成 Repository 接口
func (tg *TemplateRedisCRUDGenerator) generateRepositoryInterface() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)

	data := RedisFullCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "redis_repository_interface.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/repository", lowerName+"_repository.go"),
		[]byte(content),
		0o644,
	)
}

// generateRepositoryImpl 生成 Repository 实现
func (tg *TemplateRedisCRUDGenerator) generateRepositoryImpl() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)

	data := RedisFullCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "redis_repository_impl.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/repository", lowerName+"_repo.go"),
		[]byte(content),
		0o644,
	)
}

// generateService 生成 Service
func (tg *TemplateRedisCRUDGenerator) generateService() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)

	data := RedisFullCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "redis_service.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/service", lowerName+"_service.go"),
		[]byte(content),
		0o644,
	)
}

// generateDALInit 生成 DAL 初始化
func (tg *TemplateRedisCRUDGenerator) generateDALInit() error {
	data := RedisFullCRUDTemplateData{
		OutputDir: tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "redis_dal_init.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/dal/redis/init.go"),
		[]byte(content),
		0o644,
	)
}
