package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// MonolithGenerator 单体项目生成器
type MonolithGenerator struct {
	projectName string
	outputDir   string
}

// NewMonolithGenerator creates new monolith generator
func NewMonolithGenerator(projectName, outputDir string) *MonolithGenerator {
	return &MonolithGenerator{
		projectName: projectName,
		outputDir:   outputDir,
	}
}

// Generate generates monolith project
func (g *MonolithGenerator) Generate() error {
	dirs := []string{
		"internal/handler",
		"internal/service",
		"internal/repository",
		"internal/model",
		"internal/middleware",
		"static",
		"templates",
		"configs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"main.go":                        g.mainContent(),
		"internal/handler/handler.go":    g.handlerContent(),
		"internal/service/service.go":    g.serviceContent(),
		"internal/repository/repository.go": g.repoContent(),
		"internal/model/model.go":        g.modelContent(),
		"configs/config.yaml":            g.configContent(),
		"go.mod":                         g.goModContent(),
		"readme.md":                      g.readmeContent(),
	}

	for path, content := range files {
		fullPath := filepath.Join(g.outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (g *MonolithGenerator) mainContent() string {
	return fmt.Sprintf(`package main

import (
	"log"
	"github.com/cloudwego/hertz/pkg/app/server"
	"%s/internal/handler"
)

func main() {
	h := server.Default()
	
	// Register handlers
	handler.Register(h)
	
	log.Println("Starting monolith server on :8080")
	h.Spin()
}
`, g.projectName)
}

func (g *MonolithGenerator) handlerContent() string {
	return fmt.Sprintf(`package handler

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"%s/internal/service"
)

// Register registers all handlers
func Register(h *server.Hertz) {
	svc := service.NewExampleService()
	
	h.GET("/examples", func(ctx context.Context, c *app.RequestContext) {
		es, count, err := svc.List(ctx, 1, 10)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		c.JSON(consts.StatusOK, map[string]interface{}{"data": es, "total": count})
	})
	
	h.POST("/examples", func(ctx context.Context, c *app.RequestContext) {
		var req struct {
			Name string json:"name"
			Data string json:"data"
		}
		if err := c.BindAndValidate(&req); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]interface{}{"error": err.Error()})
			return
		}
		
		e, err := svc.Create(ctx, req.Name, req.Data)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		c.JSON(consts.StatusCreated, map[string]interface{}{"data": e})
	})
}
`, g.projectName)
}

func (g *MonolithGenerator) serviceContent() string {
	return fmt.Sprintf(`package service

import (
	"context"
	"%s/internal/model"
	"%s/internal/repository"
)

// ExampleService business service
type ExampleService struct {
	repo *repository.ExampleRepository
}

// NewExampleService creates service
func NewExampleService() *ExampleService {
	return &ExampleService{repo: repository.NewExampleRepository()}
}

// Create creates example
func (s *ExampleService) Create(ctx context.Context, name, data string) (*model.Example, error) {
	e := &model.Example{Name: name, Data: data}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// List lists examples
func (s *ExampleService) List(ctx context.Context, page, size int) ([]*model.Example, int64, error) {
	es, err := s.repo.List(ctx, (page-1)*size, size)
	if err != nil {
		return nil, 0, err
	}
	c, _ := s.repo.Count(ctx)
	return es, c, nil
}
`, g.projectName, g.projectName)
}

func (g *MonolithGenerator) repoContent() string {
	return fmt.Sprintf(`package repository

import (
	"context"
	"%s/internal/model"
	"gorm.io/gorm"
)

// ExampleRepository data access
type ExampleRepository struct {
	db *gorm.DB
}

// NewExampleRepository creates repository
func NewExampleRepository() *ExampleRepository {
	// Initialize DB connection here
	return &ExampleRepository{}
}

// Create creates example
func (r *ExampleRepository) Create(ctx context.Context, e *model.Example) error {
	return r.db.Create(e).Error
}

// List lists examples
func (r *ExampleRepository) List(ctx context.Context, offset, limit int) ([]*model.Example, error) {
	var es []*model.Example
	return es, r.db.Offset(offset).Limit(limit).Find(&es).Error
}

// Count counts examples
func (r *ExampleRepository) Count(ctx context.Context) (int64, error) {
	var c int64
	return c, r.db.Model(&model.Example{}).Count(&c).Error
}
`, g.projectName)
}

func (g *MonolithGenerator) modelContent() string {
	return `package model

import (
	"time"
	"gorm.io/gorm"
)

// Example model
type Example struct {
	ID        int64          gorm:"primaryKey;autoIncrement"
	Name      string         gorm:"size:255;not null"
	Data      string         gorm:"type:text"
	CreatedAt time.Time      gorm:"autoCreateTime"
	UpdatedAt time.Time      gorm:"autoUpdateTime"
	DeletedAt gorm.DeletedAt gorm:"index"
}
`
}

func (g *MonolithGenerator) configContent() string {
	return `server:
  address: ":8080"

database:
  dsn: "root:@tcp(localhost:3306)/mydb?parseTime=true"

log:
  level: "info"
`
}

func (g *MonolithGenerator) goModContent() string {
	return fmt.Sprintf(`module %s

go %s
require (
	github.com/cloudwego/hertz v0.9.0
	gorm.io/driver/mysql v1.5.2
	gorm.io/gorm v1.25.5
)
`, g.projectName, GetGoVersion())
}

func (g *MonolithGenerator) readmeContent() string {
	return fmt.Sprintf(`# %s - Monolith Application

单体应用 - Hertz HTTP 框架

## 运行

go mod tidy
go run main.go

## API

- GET /examples - 列表查询
- POST /examples - 创建
`, g.projectName)
}

// generateMQForMonolith 为单体项目生成 MQ 相关代码
func (g *MonolithGenerator) generateMQForMonolith(outputDir string) error {
	// 复制 MQ 中间件模板
	if err := g.copyMonolithMQTemplates(outputDir); err != nil {
		return err
	}
	return nil
}

// copyMonolithMQTemplates 复制单体 MQ 模板
func (g *MonolithGenerator) copyMonolithMQTemplates(outputDir string) error {
	// TODO: 实现模板复制逻辑
	return nil
}
