package generator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

// CRUDGenerator MySQL 数据库 CRUD 代码生成器
type CRUDGenerator struct {
	host        string
	port        string
	user        string
	password    string
	database    string
	table       string
	output      string
	module      string
	handlerType string // "http" or "cli"
}

// CRUDTableInfo 表信息
type CRUDTableInfo struct {
	Name       string
	GoName     string // 大驼峰: EbArticle
	CamelName  string // 小驼峰: ebArticle
	Comment    string
	Columns    []CRUDColumnInfo
	PrimaryKey *CRUDColumnInfo
}

// CRUDColumnInfo 列信息
type CRUDColumnInfo struct {
	Name       string
	GoName     string
	GoType     string
	SQLType    string
	IsPrimary  bool
	IsNullable bool
	Comment    string
}

// NewCRUDGenerator 创建默认 CRUD 生成器 (HTTP handler)
func NewCRUDGenerator(host, port, user, password, database, table, output, module string) *CRUDGenerator {
	return NewCRUDGeneratorWithHandlerType(host, port, user, password, database, table, output, module, "http")
}

// NewCRUDGeneratorWithHandlerType 创建指定 handler 类型的 CRUD 生成器
func NewCRUDGeneratorWithHandlerType(host, port, user, password, database, table, output, module, handlerType string) *CRUDGenerator {
	return &CRUDGenerator{
		host:        host,
		port:        port,
		user:        user,
		password:    password,
		database:    database,
		table:       table,
		output:      output,
		module:      module,
		handlerType: handlerType,
	}
}

// Generate 生成 CRUD 代码
func (g *CRUDGenerator) Generate() error {
	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		g.user, g.password, g.host, g.port, g.database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	defer db.Close()

	// 获取表信息
	tableInfo, err := g.getTableInfo(db, g.table)
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}

	// 创建输出目录（生成到 internal 目录下）
	outputDir := filepath.Join(g.output, "internal")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// 创建子目录
	subDirs := []string{"model", "repository", "service", "handler"}
	for _, subDir := range subDirs {
		if err := os.MkdirAll(filepath.Join(outputDir, subDir), 0755); err != nil {
			return err
		}
	}

	// 检查是否找到主键
	if tableInfo.PrimaryKey == nil {
		return fmt.Errorf("table %s has no primary key or table not found", g.table)
	}

	// 生成 model
	if err := g.generateModel(tableInfo, outputDir); err != nil {
		return fmt.Errorf("failed to generate model: %w", err)
	}

	// 生成 repository (dao)
	if err := g.generateRepository(tableInfo, outputDir); err != nil {
		return fmt.Errorf("failed to generate repository: %w", err)
	}

	// 生成 service
	if err := g.generateService(tableInfo, outputDir); err != nil {
		return fmt.Errorf("failed to generate service: %w", err)
	}

	if g.handlerType == "cli" {
		if err := g.generateCLIHandler(tableInfo, outputDir); err != nil {
			return fmt.Errorf("failed to generate CLI handler: %w", err)
		}
	} else {
		if err := g.generateHandler(tableInfo, outputDir); err != nil {
			return fmt.Errorf("failed to generate handler: %w", err)
		}
	}

	return nil
}

// getTableInfo 获取表信息
func (g *CRUDGenerator) getTableInfo(db *sql.DB, table string) (*CRUDTableInfo, error) {
	// 查询列信息
	query := `
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_COMMENT, COLUMN_KEY
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := db.Query(query, g.database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 去除表名前缀（eb_store_product → store_product）
	strippedTable := strings.TrimPrefix(table, "eb_")
	ti := &CRUDTableInfo{
		Name:      table,
		GoName:    snakeToCamel(strippedTable),
		CamelName: toLowerCamelCase(strippedTable),
		Columns:   make([]CRUDColumnInfo, 0),
	}

	for rows.Next() {
		var colName, dataType, comment, colKey string
		err := rows.Scan(&colName, &dataType, &comment, &colKey)
		if err != nil {
			return nil, err
		}

		col := CRUDColumnInfo{
			Name:      colName,
			GoName:    snakeToCamel(strings.TrimPrefix(colName, "is_")), // 去除 is_ 前缀
			GoType:    sqlTypeToGoType(dataType),
			SQLType:   dataType,
			Comment:   comment,
			IsPrimary: colKey == "PRI",
		}
		// is_ 前缀的列映射为 bool 类型
		if strings.HasPrefix(colName, "is_") {
			col.GoType = "bool"
		}

		if col.IsPrimary {
			ti.PrimaryKey = &col
		}

		ti.Columns = append(ti.Columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ti, nil
}

// snakeToCamel snake_case 转 CamelCase
// 首字母也大写，确保结构体和字段可以被导出
func snakeToCamel(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		w = strings.ToLower(w)
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, "")
}

func toLowerCamelCase(s string) string {
	result := snakeToCamel(s)
	if result == "" {
		return result
	}
	return strings.ToLower(result[:1]) + result[1:]
}

// sqlTypeToGoType SQL 类型转 Go 类型
func sqlTypeToGoType(sqlType string) string {
	sqlType = strings.ToLower(sqlType)
	switch {
	case strings.Contains(sqlType, "int") || sqlType == "integer":
		return "int"
	case strings.Contains(sqlType, "bigint"):
		return "int64"
	case strings.Contains(sqlType, "float") || strings.Contains(sqlType, "double") || strings.Contains(sqlType, "decimal"):
		return "float64"
	case strings.Contains(sqlType, "bool") || strings.Contains(sqlType, "bit") || strings.Contains(sqlType, "tinyint(1)"):
		return "bool"
	case strings.Contains(sqlType, "datetime") || strings.Contains(sqlType, "timestamp") || strings.Contains(sqlType, "date"):
		return "time.Time"
	case strings.Contains(sqlType, "json"):
		return "json.RawMessage"
	case strings.Contains(sqlType, "text"), strings.Contains(sqlType, "varchar"), strings.Contains(sqlType, "char"):
		return "string"
	case strings.Contains(sqlType, "blob"), strings.Contains(sqlType, "binary"):
		return "[]byte"
	default:
		return "interface{}"
	}
}

// generateModel 生成 model 代码
func (g *CRUDGenerator) generateModel(ti *CRUDTableInfo, outputDir string) error {
	hasTimeType := false
	for _, col := range ti.Columns {
		sqlType := strings.ToLower(col.SQLType)
		if strings.Contains(sqlType, "datetime") || strings.Contains(sqlType, "timestamp") || strings.Contains(sqlType, "date") {
			hasTimeType = true
			break
		}
	}

	var tmpl string
	if hasTimeType {
		tmpl = `package model

import (
	"time"
)

// {{.GoName}} {{.Comment}} model
type {{.GoName}} struct {
{{range .Columns}}
	// {{.Comment}}
	{{.GoName}} {{.GoType}} ` + "`gorm:\"column:{{.Name}}\"`" + `
{{end}}
}

// TableName returns table name
func ({{.GoName}}) TableName() string {
	return "{{.Name}}"
}
`
	} else {
		tmpl = `package model

// {{.GoName}} {{.Comment}} model
type {{.GoName}} struct {
{{range .Columns}}
	// {{.Comment}}
	{{.GoName}} {{.GoType}} ` + "`gorm:\"column:{{.Name}}\"`" + `
{{end}}
}

// TableName returns table name
func ({{.GoName}}) TableName() string {
	return "{{.Name}}"
}
`
	}

	return g.render(tmpl, ti, filepath.Join(outputDir, "model", ti.CamelName+"Model.go"))
}

// generateRepository 生成 repository / dao 层
func (g *CRUDGenerator) generateRepository(ti *CRUDTableInfo, outputDir string) error {
	tmpl := `package repository

import (
	"context"

	"gorm.io/gorm"
	"{{.Module}}/internal/model"
)

// Repository {{.GoName}} repository
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new {{.GoName}} repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetByID get {{.GoName}} by ID
func (r *Repository) GetByID(ctx context.Context, id {{.Primary.GoType}}) (*model.{{.GoName}}, error) {
	var m *model.{{.GoName}}
	err := r.db.WithContext(ctx).Where("{{.Primary.Name}} = ?", id).First(&m).Error
	if err != nil {
		return nil, err
	}
	return m, nil
}

// List returns a page of {{.GoName}}
func (r *Repository) List(ctx context.Context, offset, limit int) ([]*model.{{.GoName}}, int64, error) {
	var list []*model.{{.GoName}}
	var count int64

	err := r.db.WithContext(ctx).Model(&model.{{.GoName}}{}).Unscoped().Count(&count).Offset(offset).Limit(limit).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, count, nil
}

// Create creates a new {{.GoName}}
func (r *Repository) Create(ctx context.Context, m *model.{{.GoName}}) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update updates an existing {{.GoName}}
func (r *Repository) Update(ctx context.Context, m *model.{{.GoName}}) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Delete deletes an existing {{.GoName}} (hard delete)
func (r *Repository) Delete(ctx context.Context, id {{.Primary.GoType}}) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.{{.GoName}}{}, "{{.Primary.Name}} = ?", id).Error
}

// FindBy find by where condition
func (r *Repository) FindBy(ctx context.Context, where string, args ...interface{}) (*model.{{.GoName}}, error) {
	var m model.{{.GoName}}
	err := r.db.WithContext(ctx).Where(where, args...).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}
`

	return g.render(tmpl, map[string]interface{}{
		"GoName":  ti.GoName,
		"Name":    ti.Name,
		"Primary": *ti.PrimaryKey,
		"Module":  g.moduleImport(),
	}, filepath.Join(outputDir, "repository", ti.CamelName+"Repository.go"))
}

// generateService 生成 service 层
func (g *CRUDGenerator) generateService(ti *CRUDTableInfo, outputDir string) error {
	tmpl := `package service

import (
	"context"

	"{{.Module}}/internal/repository"
	"{{.Module}}/internal/model"
)

// Service {{.GoName}} service
type Service struct {
	repo *repository.Repository
}

// NewService creates a new {{.GoName}} service
func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// GetByID get {{.GoName}} by ID
func (s *Service) GetByID(ctx context.Context, id {{.Primary.GoType}}) (*model.{{.GoName}}, error) {
	return s.repo.GetByID(ctx, id)
}

// List list {{.GoName}} with pagination
func (s *Service) List(ctx context.Context, page, size int) ([]*model.{{.GoName}}, int64, error) {
	return s.repo.List(ctx, (page-1)*size, size)
}

// Create creates a new {{.GoName}}
func (s *Service) Create(ctx context.Context, m *model.{{.GoName}}) error {
	return s.repo.Create(ctx, m)
}

// Update updates an existing {{.GoName}}
func (s *Service) Update(ctx context.Context, m *model.{{.GoName}}) error {
	return s.repo.Update(ctx, m)
}

// Delete deletes a {{.GoName}} by ID
func (s *Service) Delete(ctx context.Context, id {{.Primary.GoType}}) error {
	return s.repo.Delete(ctx, id)
}
`

	return g.render(tmpl, map[string]interface{}{
		"GoName":  ti.GoName,
		"Primary": *ti.PrimaryKey,
		"Module":  g.moduleImport(),
	}, filepath.Join(outputDir, "service", ti.CamelName+"Service.go"))
}

// generateHandler 生成 handler / API handler
func (g *CRUDGenerator) generateHandler(ti *CRUDTableInfo, outputDir string) error {
	tmpl := `package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"{{.Module}}/internal/model"
	"{{.Module}}/internal/service"
)

// Handler {{.GoName}} HTTP handler (for Gin HTTP server)
type Handler struct {
	service *service.Service
}

// NewHandler creates a new {{.GoName}} handler
func NewHandler(s *service.Service) *Handler {
	return &Handler{service: s}
}

// GetByID godoc
// @Summary Get {{.GoName}} by ID
// @Description get {{.GoName}} by primary key
// @Tags {{.GoName}}
// @Accept json
// @Produce json
// @Param id path {{.Primary.GoType}} true "{{.Primary.Name}}"
// @Success 200 {object} model.{{.GoName}}
// @Router /{{.Name}}/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
{{- if eq .Primary.GoType "int"}}
	id, err := strconv.Atoi(idStr)
{{- else if eq .Primary.GoType "int64"}}
	id, err := strconv.ParseInt(idStr, 10, 64)
{{- else if eq .Primary.GoType "float64"}}
	id, err := strconv.ParseFloat(idStr, 64)
{{- else}}
	id, err := strconv.Parse{{.Primary.GoType}}(idStr)
{{- end}}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result, err := h.service.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// List godoc
// @Summary list {{.GoName}} with pagination
// @Description list {{.GoName}} with page and size
// @Tags {{.GoName}}
// @Accept json
// @Produce json
// @Param page query int false "page number" default(1)
// @Param size query int false "page size" default(10)
// @Success 200 {object} gin.H{data=[]model.{{.GoName}}, count=int64}
// @Router /{{.Name}} [get]
func (h *Handler) List(c *gin.Context) {
	pageStr := c.Query("page")
	sizeStr := c.Query("size")
	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	list, count, err := h.service.List(context.Background(), page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  list,
		"count": count,
	})
}

// Create godoc
// @Summary create new {{.GoName}}
// @Description create new {{.GoName}}
// @Tags {{.GoName}}
// @Accept json
// @Produce json
// @Param {{.GoName}} body model.{{.GoName}} true "{{.GoName}} entity"
// @Success 200 {object} gin.H{"id":{{.Primary.GoType}}}
// @Router /{{.Name}} [post]
func (h *Handler) Create(c *gin.Context) {
	var m model.{{.GoName}}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Create(context.Background(), &m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": m.{{.Primary.GoName}}})
}

// Update godoc
// @Summary update existing {{.GoName}}
// @Description update existing {{.GoName}}
// @Tags {{.GoName}}
// @Accept json
// @Produce json
// @Param id path {{.Primary.GoType}} true "{{.Primary.Name}}"
// @Param {{.GoName}} body model.{{.GoName}} true "{{.GoName}} entity"
// @Success 200 {object} gin.H{"ok":bool}
// @Router /{{.Name}}/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
{{- if eq .Primary.GoType "int"}}
	id, err := strconv.Atoi(idStr)
{{- else if eq .Primary.GoType "int64"}}
	id, err := strconv.ParseInt(idStr, 10, 64)
{{- else if eq .Primary.GoType "float64"}}
	id, err := strconv.ParseFloat(idStr, 64)
{{- else}}
	id, err := strconv.Parse{{.Primary.GoType}}(idStr)
{{- end}}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var m model.{{.GoName}}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m.{{.Primary.GoName}} = id

	if err := h.service.Update(context.Background(), &m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Delete godoc
// @Summary delete {{.GoName}} by ID
// @Description delete {{.GoName}} by primary key
// @Tags {{.GoName}}
// @Produce json
// @Param id path {{.Primary.GoType}} true "{{.Primary.Name}}"
// @Success 200 {object} gin.H{"ok":bool}
// @Router /{{.Name}}/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
{{- if eq .Primary.GoType "int"}}
	id, err := strconv.Atoi(idStr)
{{- else if eq .Primary.GoType "int64"}}
	id, err := strconv.ParseInt(idStr, 10, 64)
{{- else if eq .Primary.GoType "float64"}}
	id, err := strconv.ParseFloat(idStr, 64)
{{- else}}
	id, err := strconv.Parse{{.Primary.GoType}}(idStr)
{{- end}}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
`

	return g.render(tmpl, map[string]interface{}{
		"GoName":  ti.GoName,
		"Name":    ti.Name,
		"Primary": *ti.PrimaryKey,
		"Module":  g.moduleImport(),
	}, filepath.Join(outputDir, "handler", ti.CamelName+"Handler.go"))
}

func (g *CRUDGenerator) generateCLIHandler(ti *CRUDTableInfo, outputDir string) error {
	tmpl := `package handler

import (
	"context"

	"{{.Module}}/internal/model"
	"{{.Module}}/internal/service"
)

type CLIHandler struct {
	service *service.Service
}

func NewCLIHandler(s *service.Service) *CLIHandler {
	return &CLIHandler{service: s}
}

func (h *CLIHandler) CreateCLI(ctx context.Context, m *model.{{.GoName}}) error {
	return h.service.Create(ctx, m)
}

func (h *CLIHandler) GetByIDCLI(ctx context.Context, id int) (*model.{{.GoName}}, error) {
	return h.service.GetByID(ctx, id)
}

func (h *CLIHandler) ListCLI(ctx context.Context, page, size int) ([]*model.{{.GoName}}, int64, error) {
	return h.service.List(ctx, page, size)
}

func (h *CLIHandler) UpdateCLI(ctx context.Context, m *model.{{.GoName}}) error {
	return h.service.Update(ctx, m)
}

func (h *CLIHandler) DeleteCLI(ctx context.Context, id int) error {
	return h.service.Delete(ctx, id)
}
`

	return g.render(tmpl, map[string]interface{}{
		"GoName":  ti.GoName,
		"Name":    ti.Name,
		"Primary": *ti.PrimaryKey,
		"Module":  g.moduleImport(),
	}, filepath.Join(outputDir, "handler", ti.CamelName+"CliHandler.go"))
}

// render 渲染模板写入文件
func (g *CRUDGenerator) render(tmpl string, data interface{}, outputPath string) error {
	t, err := template.New("crud").Parse(tmpl)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// moduleImport 返回模块导入路径
func (g *CRUDGenerator) moduleImport() string {
	if g.module == "" {
		return ""
	}
	return g.module
}
