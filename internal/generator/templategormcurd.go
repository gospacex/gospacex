package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateCRUDGenerator 基于模板的 CRUD 生成器
type TemplateCRUDGenerator struct {
	outputDir string
	entity    string
	tableName string
}

// NewTemplateCRUDGenerator creates new template-based CRUD generator
func NewTemplateCRUDGenerator(outputDir, entity, tableName string) *TemplateCRUDGenerator {
	return &TemplateCRUDGenerator{
		outputDir: outputDir,
		entity:    entity,
		tableName: tableName,
	}
}

// GormCRUDTemplateData 模板数据
type GormCRUDTemplateData struct {
	StructName   string
	LowerName    string
	CamelName    string // 驼峰命名（用于文件名）
	TableName    string
	OutputDir    string
}

// getTemplateDir 获取模板目录
func (tg *TemplateCRUDGenerator) getTemplateDir() string {
	return filepath.Join("/Users/hyx/work/gowork/src/gospacex", "templates")
}

// toCamelName 将下划线命名转换为 camelCase（用于文件名）
func (tg *TemplateCRUDGenerator) toCamelName(s string) string {
	// 去除 t_ 前缀
	s = strings.TrimPrefix(s, "t_")
	
	parts := strings.Split(s, "_")
	var result strings.Builder
	for i, part := range parts {
		if len(part) > 0 {
			if i == 0 {
				result.WriteString(part)
			} else {
				result.WriteString(strings.ToUpper(part[:1]))
				if len(part) > 1 {
					result.WriteString(part[1:])
				}
			}
		}
	}
	return result.String()
}

// renderTemplate 渲染模板
func (tg *TemplateCRUDGenerator) renderTemplate(templatePath string, data GormCRUDTemplateData) (string, error) {
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
func (tg *TemplateCRUDGenerator) Generate() error {
	// 创建目录结构 (参考 book-shop)
	dirs := []string{
		"internal/service",
		"internal/repository",
		"internal/model",
		"internal/dal/mysql",
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
func (tg *TemplateCRUDGenerator) generateModel() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)
	camelName := tg.toCamelName(tg.entity)

	data := GormCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		CamelName:  camelName,
		TableName:  tg.tableName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "gorm_model.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/model", camelName+".go"),
		[]byte(content),
		0o644,
	)
}

// generateRepositoryInterface 生成 Repository 接口
func (tg *TemplateCRUDGenerator) generateRepositoryInterface() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)
	camelName := tg.toCamelName(tg.entity)

	data := GormCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		CamelName:  camelName,
		TableName:  tg.tableName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "gorm_repository_interface.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/repository", camelName+"Repository.go"),
		[]byte(content),
		0o644,
	)
}

// generateRepositoryImpl 生成 Repository 实现
func (tg *TemplateCRUDGenerator) generateRepositoryImpl() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)
	camelName := tg.toCamelName(tg.entity)

	data := GormCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		CamelName:  camelName,
		TableName:  tg.tableName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "gorm_repository_impl.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/repository", camelName+"Repo.go"),
		[]byte(content),
		0o644,
	)
}

// generateService 生成 Service
func (tg *TemplateCRUDGenerator) generateService() error {
	structName := strings.Title(tg.entity)
	lowerName := strings.ToLower(tg.entity)
	camelName := tg.toCamelName(tg.entity)

	data := GormCRUDTemplateData{
		StructName: structName,
		LowerName:  lowerName,
		CamelName:  camelName,
		TableName:  tg.tableName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "gorm_service.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/service", camelName+"Service.go"),
		[]byte(content),
		0o644,
	)
}

// generateDALInit 生成 DAL 初始化
func (tg *TemplateCRUDGenerator) generateDALInit() error {
	structName := strings.Title(tg.entity)

	data := GormCRUDTemplateData{
		StructName: structName,
		LowerName:  strings.ToLower(tg.entity),
		CamelName:  tg.toCamelName(tg.entity),
		TableName:  tg.tableName,
		OutputDir:  tg.outputDir,
	}

	templatePath := filepath.Join(tg.getTemplateDir(), "crud", "gorm_dal_init.go.tmpl")
	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/dal/mysql/init.go"),
		[]byte(content),
		0o644,
	)
}
