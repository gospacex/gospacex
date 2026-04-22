package generator

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

// TemplateESCRUDGenerator 基于模板的 Elasticsearch CRUD 生成器
type TemplateESCRUDGenerator struct {
	outputDir   string
	entity      string
	indexName   string
	projectName string // 模块名称
	fields      []ESField // 从 MySQL 提取的字段
}

// ESField ES CRUD 字段结构
type ESField struct {
	Name   string
	Type   string
	GoType string
}

// NewTemplateESCRUDGenerator creates new template-based ES CRUD generator
func NewTemplateESCRUDGenerator(outputDir, entity, indexName string) *TemplateESCRUDGenerator {
	return &TemplateESCRUDGenerator{
		outputDir:   outputDir,
		entity:      entity,
		indexName:   indexName,
		projectName: "github.com/gospacex/gpx-scripts", // 默认模块名
	}
}

// NewTemplateESCRUDGeneratorWithMySQL 从 MySQL 提取表结构创建 ES CRUD 生成器
func NewTemplateESCRUDGeneratorWithMySQL(outputDir, entity, indexName, mysqlHost, mysqlPort, mysqlUser, mysqlPassword, mysqlDB, mysqlTable string) *TemplateESCRUDGenerator {
	gen := &TemplateESCRUDGenerator{
		outputDir:   outputDir,
		entity:      entity,
		indexName:   indexName,
		projectName: "github.com/gospacex/gpx-scripts", // 默认模块名
	}

	// 从 MySQL 读取表结构
	if err := gen.readMySQLFields(mysqlHost, mysqlPort, mysqlUser, mysqlPassword, mysqlDB, mysqlTable); err != nil {
		fmt.Printf("Warning: cannot read MySQL table structure: %v, using default fields\n", err)
		gen.defaultFields()
	}

	return gen
}

// SetProjectName 设置模块名称
func (tg *TemplateESCRUDGenerator) SetProjectName(name string) {
	tg.projectName = name
}

// readMySQLFields 从 MySQL 读取表字段
func (tg *TemplateESCRUDGenerator) readMySQLFields(host, port, user, password, database, table string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		user, password, host, port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	rows, err := db.Query(
		"SELECT COLUMN_NAME, DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? ORDER BY ORDINAL_POSITION",
		database, table,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	tg.fields = nil
	for rows.Next() {
		var f ESField
		if err := rows.Scan(&f.Name, &f.Type); err != nil {
			continue
		}
		f.GoType = tg.mapMySQLTypeToGo(f.Type)
		tg.fields = append(tg.fields, f)
	}

	return nil
}

// mapMySQLTypeToGo 将 MySQL 类型映射为 Go 类型
func (tg *TemplateESCRUDGenerator) mapMySQLTypeToGo(mysqlType string) string {
	switch strings.ToLower(mysqlType) {
	case "bigint", "int", "mediumint", "smallint", "tinyint":
		return "int64"
	case "varchar", "text", "char", "longtext", "mediumtext":
		return "string"
	case "datetime", "timestamp", "date":
		return "string"
	case "decimal", "float", "double":
		return "float64"
	case "bit", "tinyint(1)":
		return "bool"
	default:
		return "string"
	}
}

// defaultFields 设置默认字段
func (tg *TemplateESCRUDGenerator) defaultFields() {
	tg.fields = []ESField{
		{Name: "id", Type: "bigint", GoType: "int64"},
		{Name: "order_id", Type: "bigint", GoType: "int64"},
		{Name: "user_id", Type: "bigint", GoType: "int64"},
		{Name: "address", Type: "text", GoType: "string"},
		{Name: "product_id", Type: "bigint", GoType: "int64"},
		{Name: "stock_num", Type: "int", GoType: "int64"},
		{Name: "product_snapshot", Type: "longtext", GoType: "string"},
		{Name: "status", Type: "tinyint", GoType: "bool"},
	}
}

// Generate 生成完整分层 CRUD 代码
func (tg *TemplateESCRUDGenerator) Generate() error {
	// 创建目录结构（不包含 dal 和 repository）
	dirs := []string{
		"internal/service",
		"internal/model",
		"internal/handler",
		"cmd/commands",
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

	// 生成 Service
	if err := tg.generateService(); err != nil {
		return err
	}

	// 生成 Handler
	if err := tg.generateHandler(); err != nil {
		return err
	}

	// 生成 Command
	if err := tg.generateCommand(); err != nil {
		return err
	}

	return nil
}

// toPascalCase 将下划线命名转换为 PascalCase（去除 t_ 前缀）
func (tg *TemplateESCRUDGenerator) toPascalCase(s string) string {
	// 去除 t_ 前缀
	s = strings.TrimPrefix(s, "t_")
	
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}
	return result.String()
}

// toCamelCase 将下划线命名转换为 camelCase（去除 t_ 前缀）
func (tg *TemplateESCRUDGenerator) toCamelCase(s string) string {
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

// getTemplateData 获取模板数据
type ESTemplateData struct {
	ProjectName      string
	EntityName       string
	EsEntityName     string // ES 实体名：Es + PascalCase
	EntityNameLower  string
	ESFileName       string // ES 文件名前缀：es + PascalCase
	IndexName        string
	Fields           []TemplateField
}

type TemplateField struct {
	Name       string
	Type       string
	JsonName   string
	Column     string
	DbType     string
}

func (tg *TemplateESCRUDGenerator) getTemplateData() ESTemplateData {
	entityName := tg.toPascalCase(tg.entity)
	entityNameLower := tg.toCamelCase(tg.entity)
	if entityNameLower == "" {
		entityNameLower = strings.ToLower(tg.entity)
	}

	// ES 文件名前缀：es + PascalCase（去除 t_ 前缀）
	esFileName := "es" + entityName

	var fields []TemplateField
	for _, f := range tg.fields {
		// 跳过 id 字段
		if strings.ToLower(f.Name) == "id" {
			continue
		}
		fields = append(fields, TemplateField{
			Name:     tg.toPascalCase(f.Name),
			Type:     f.GoType,
			JsonName: f.Name,
			Column:   f.Name,
			DbType:   f.Type,
		})
	}

	indexName := tg.indexName
	if indexName == "" {
		indexName = entityNameLower
	}

	// ES 实体名：Es + PascalCase
	esEntityName := "Es" + entityName

	return ESTemplateData{
		ProjectName:      tg.projectName,
		EntityName:       entityName,
		EsEntityName:     esEntityName,
		EntityNameLower:  entityNameLower,
		ESFileName:       esFileName,
		IndexName:        indexName,
		Fields:           fields,
	}
}

// getTemplateDir 获取模板目录
func (tg *TemplateESCRUDGenerator) getTemplateDir() string {
	// 从项目根目录获取模板
	return filepath.Join("/Users/hyx/work/gowork/src/gospacex", "templates")
}

// renderTemplate 渲染模板
func (tg *TemplateESCRUDGenerator) renderTemplate(templatePath string, data ESTemplateData) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.String(), nil
}

// generateModel 生成 Model
func (tg *TemplateESCRUDGenerator) generateModel() error {
	data := tg.getTemplateData()
	templatePath := filepath.Join(tg.getTemplateDir(), "internal/model/es_entity.go.tmpl")

	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/model", data.ESFileName+".go"),
		[]byte(content),
		0o644,
	)
}

// generateService 生成 Service
func (tg *TemplateESCRUDGenerator) generateService() error {
	data := tg.getTemplateData()
	templatePath := filepath.Join(tg.getTemplateDir(), "internal/service/es_entity_service.go.tmpl")

	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/service", data.ESFileName+"Service.go"),
		[]byte(content),
		0o644,
	)
}

// generateHandler 生成 Handler
func (tg *TemplateESCRUDGenerator) generateHandler() error {
	data := tg.getTemplateData()
	templatePath := filepath.Join(tg.getTemplateDir(), "internal/handler/es_entity_handler.go.tmpl")

	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "internal/handler", data.ESFileName+"Handler.go"),
		[]byte(content),
		0o644,
	)
}

// generateCommand 生成 Command
func (tg *TemplateESCRUDGenerator) generateCommand() error {
	data := tg.getTemplateData()
	templatePath := filepath.Join(tg.getTemplateDir(), "cmd/commands/es_entity.go.tmpl")

	content, err := tg.renderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(tg.outputDir, "cmd/commands", data.ESFileName+".go"),
		[]byte(content),
		0o644,
	)
}
