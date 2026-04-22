package generator

import (
	"os"
	"path/filepath"
	"fmt"
)

type MicroserviceDDDGenerator struct {
	serviceName string
	outputDir   string
}

func NewMicroserviceDDDGenerator(serviceName, outputDir string) *MicroserviceDDDGenerator {
	return &MicroserviceDDDGenerator{serviceName: serviceName, outputDir: outputDir}
}

func (g *MicroserviceDDDGenerator) Generate() error {
	dirs := []string{
		"app/" + g.serviceName + "/domain/entity",
		"app/" + g.serviceName + "/domain/repository",
		"app/" + g.serviceName + "/domain/service",
		"app/" + g.serviceName + "/application/service",
		"app/" + g.serviceName + "/application/dto",
		"app/" + g.serviceName + "/infrastructure/persistence/mysql",
		"app/" + g.serviceName + "/infrastructure/persistence/redis",
		"app/" + g.serviceName + "/infrastructure/logger",
		"app/" + g.serviceName + "/interfaces/rpc",
		"app/" + g.serviceName + "/interfaces/http",
		"idl", "kitex_gen", "conf/dev",
	}
	for _, dir := range dirs { os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755) }
	
	files := map[string]string{
		"app/" + g.serviceName + "/domain/entity/example.go": g.entity(),
		"app/" + g.serviceName + "/domain/repository/example.go": g.repoInterface(),
		"app/" + g.serviceName + "/domain/service/example.go": g.domainService(),
		"app/" + g.serviceName + "/application/service/example.go": g.appService(),
		"app/" + g.serviceName + "/infrastructure/persistence/mysql/example.go": g.mysqlRepo(),
		"app/" + g.serviceName + "/infrastructure/persistence/mysql/init.go": g.mysqlInit(),
		"app/" + g.serviceName + "/interfaces/rpc/handler.go": g.handler(),
		"app/" + g.serviceName + "/main.go": g.main(),
		"idl/example.proto": g.proto(),
		"conf/dev/conf.yaml": g.devConfig(),
		"go.mod": g.goMod(),
		"readme.md": g.readme(),
	}
	for path, content := range files {
		fullPath := filepath.Join(g.outputDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}
	return nil
}

func (g *MicroserviceDDDGenerator) entity() string {
	return `package entity

import "time"

// Example 领域实体
type Example struct {
	ID        int64
	Name      string
	Data      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewExample 创建新的 Example 实体
func NewExample(name, data string) *Example {
	return &Example{
		Name:      name,
		Data:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证实体有效性
func (e *Example) Validate() error {
	if e.Name == "" {
		return ErrInvalidName
	}
	return nil
}
`
}

func (g *MicroserviceDDDGenerator) repoInterface() string {
	return fmt.Sprintf(`package repository

import (
	"context"
	"%s/domain/entity"
)

// ExampleRepository 领域仓储接口
type ExampleRepository interface {
	Create(ctx context.Context, e *entity.Example) error
	GetByID(ctx context.Context, id int64) (*entity.Example, error)
	Update(ctx context.Context, e *entity.Example) error
	Delete(ctx context.Context, id int64) error
	FindByName(ctx context.Context, name string) (*entity.Example, error)
}
`, g.serviceName)
}

func (g *MicroserviceDDDGenerator) domainService() string {
	return `package service

import (
	"errors"
	"%s/domain/entity"
)

// 领域错误定义
var (
	ErrInvalidName   = errors.New("invalid name")
	ErrExampleExists = errors.New("example already exists")
)

// ExampleDomainService 领域服务
type ExampleDomainService struct{}

// NewExampleDomainService 创建领域服务
func NewExampleDomainService() *ExampleDomainService {
	return &ExampleDomainService{}
}

// ValidateExample 验证实体
func (s *ExampleDomainService) ValidateExample(e *entity.Example) error {
	return e.Validate()
}
`
}

func (g *MicroserviceDDDGenerator) appService() string {
	return fmt.Sprintf(`package service

import (
	"context"
	"errors"
	"%s/domain/entity"
	"%s/domain/repository"
)

// ExampleAppService 应用服务
type ExampleAppService struct {
	repo repository.ExampleRepository
}

// NewExampleAppService 创建应用服务
func NewExampleAppService(repo repository.ExampleRepository) *ExampleAppService {
	return &ExampleAppService{repo: repo}
}

// CreateExample 创建示例
func (s *ExampleAppService) CreateExample(ctx context.Context, name, data string) (*entity.Example, error) {
	e := entity.NewExample(name, data)
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// GetExample 获取示例
func (s *ExampleAppService) GetExample(ctx context.Context, id int64) (*entity.Example, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateExample 更新示例
func (s *ExampleAppService) UpdateExample(ctx context.Context, id int64, name, data string) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	e.Name = name
	e.Data = data
	return s.repo.Update(ctx, e)
}

// DeleteExample 删除示例
func (s *ExampleAppService) DeleteExample(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceDDDGenerator) mysqlRepo() string {
	return fmt.Sprintf(`package mysql

import (
	"context"
	"fmt"
	"%s/domain/entity"
	"%s/domain/repository"
	"gorm.io/gorm"
)

// ExampleRepositoryImpl 仓储实现
type ExampleRepositoryImpl struct {
	db *gorm.DB
}

// NewExampleRepository 创建仓储实例
func NewExampleRepository(db *gorm.DB) repository.ExampleRepository {
	return &ExampleRepositoryImpl{db: db}
}

// Create 创建实体
func (r *ExampleRepositoryImpl) Create(ctx context.Context, e *entity.Example) error {
	return r.db.Create(e).Error
}

// GetByID 根据 ID 查询
func (r *ExampleRepositoryImpl) GetByID(ctx context.Context, id int64) (*entity.Example, error) {
	var e entity.Example
	if err := r.db.First(&e, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("example not found")
		}
		return nil, err
	}
	return &e, nil
}

// Update 更新实体
func (r *ExampleRepositoryImpl) Update(ctx context.Context, e *entity.Example) error {
	return r.db.Save(e).Error
}

// Delete 删除实体
func (r *ExampleRepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.Delete(&entity.Example{}, id).Error
}

// FindByName 根据名称查询
func (r *ExampleRepositoryImpl) FindByName(ctx context.Context, name string) (*entity.Example, error) {
	var e entity.Example
	if err := r.db.Where("name = ?", name).First(&e).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("example not found")
		}
		return nil, err
	}
	return &e, nil
}
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceDDDGenerator) mysqlInit() string {
	return fmt.Sprintf(`package mysql

import (
	"fmt"
	"os"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化 MySQL 连接
func Init() {
	dsn := fmt.Sprintf("%%s:%%s@tcp(%%s:3306)/%%s?parseTime=true&loc=Local",
		getEnv("MYSQL_USER", "root"),
		getEnv("MYSQL_PASSWORD", ""),
		getEnv("MYSQL_HOST", "localhost"),
		getEnv("MYSQL_DATABASE", "%s"))
	
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MySQL: %%v", err))
	}
}

// Close 关闭数据库连接
func Close() {
	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
`, g.serviceName)
}

func (g *MicroserviceDDDGenerator) handler() string {
	return fmt.Sprintf(`package rpc
import (
	"context"
	"%s/application/service"
	"%s/app/%s/kitex_gen/example/v1"
)
type ExampleHandler struct{ svc *service.ExampleService }
func NewExampleHandler(svc *service.ExampleService) *ExampleHandler { return &ExampleHandler{svc: svc} }
func (h *ExampleHandler) GetExample(ctx context.Context, req *example.GetExampleReq) (*example.GetExampleResp, error) {
	e, err := h.svc.GetExample(ctx, req.Id)
	if err != nil { return &example.GetExampleResp{}, err }
	return &example.GetExampleResp{Data: &example.Example{Id: e.ID, Name: e.Name, Data: e.Data}}, nil
}
func (h *ExampleHandler) CreateExample(ctx context.Context, req *example.CreateExampleReq) (*example.CreateExampleResp, error) {
	e, err := h.svc.CreateExample(ctx, req.Name, req.Data)
	if err != nil { return &example.CreateExampleResp{}, err }
	return &example.CreateExampleResp{Id: e.ID}, nil
}
`, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceDDDGenerator) main() string {
	return fmt.Sprintf(`package main
import (
	"github.com/cloudwego/kitex/server"
	"%s/app/%s/kitex_gen/example/v1/exampleservice"
	"%s/app/%s/interfaces/rpc"
	"%s/app/%s/application/service"
	"%s/app/%s/infrastructure/persistence/mysql"
)
func main() {
	mysql.Init()
	repo := mysql.NewExampleRepository(mysql.DB)
	svc := service.NewExampleService(repo)
	handler := rpc.NewExampleHandler(svc)
	svr := exampleservice.NewServer(handler, server.WithServiceAddr(":8888"))
	svr.Run()
}
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceDDDGenerator) proto() string {
	return fmt.Sprintf(`syntax = "proto3";
package example.v1;
option go_package = "%s/kitex_gen/example/v1";

// Example 示例实体
message Example {
	int64 id = 1;
	string name = 2;
	string data = 3;
}

// GetExampleReq 获取示例请求
message GetExampleReq {
	int64 id = 1;
}

// GetExampleResp 获取示例响应
message GetExampleResp {
	Example data = 1;
}

// CreateExampleReq 创建示例请求
message CreateExampleReq {
	string name = 1;
	string data = 2;
}

// CreateExampleResp 创建示例响应
message CreateExampleResp {
	int64 id = 1;
}

// ExampleService 示例服务
service ExampleService {
	rpc GetExample(GetExampleReq) returns (GetExampleResp);
	rpc CreateExample(CreateExampleReq) returns (CreateExampleResp);
}
`, g.serviceName)
}

func (g *MicroserviceDDDGenerator) devConfig() string {
	return fmt.Sprintf(`server:
  service_name: %s
  address: ":8888"
registry:
  type: etcd
  addresses:
    - localhost:2379
mysql:
  user: root
  password: ""
  host: localhost
  database: %s
  max_open_conns: 100
  max_idle_conns: 10
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceDDDGenerator) goMod() string {
	return fmt.Sprintf(`module %s

go %s

require (
	github.com/cloudwego/kitex v0.9.0
	gorm.io/driver/mysql v1.5.2
	gorm.io/gorm v1.25.5
)
`, g.serviceName, GetGoVersion())
}

func (g *MicroserviceDDDGenerator) readme() string {
	return fmt.Sprintf(`# %s - DDD Microservice

基于 DDD (领域驱动设计) 的微服务项目

## DDD 分层架构

- domain/ - 领域层 (Entity/Repository/Domain Service)
- application/ - 应用层 (Application Service/DTO)
- infrastructure/ - 基础设施层 (Persistence/RPC)
- interfaces/ - 接口层 (RPC/HTTP Handler)

## 快速开始

1. 生成 Kitex 代码:
   kitex -module %s -service %s idl/example.proto

2. 安装依赖:
   go mod tidy

3. 运行服务:
   go run app/%s/main.go

## 设计原则

- 领域驱动：业务逻辑集中在 domain 层
- 依赖倒置：应用层依赖仓储接口，而非实现
- 单一职责：每层有明确的职责边界
- 可测试性：接口隔离，便于单元测试
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}
