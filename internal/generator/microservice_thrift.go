package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// MicroserviceThriftGenerator Thrift 微服务生成器
type MicroserviceThriftGenerator struct {
	serviceName string
	outputDir   string
}

// NewMicroserviceThriftGenerator creates new Thrift microservice generator
func NewMicroserviceThriftGenerator(serviceName, outputDir string) *MicroserviceThriftGenerator {
	return &MicroserviceThriftGenerator{
		serviceName: serviceName,
		outputDir:   outputDir,
	}
}

// Generate generates Thrift microservice project
func (g *MicroserviceThriftGenerator) Generate() error {
	dirs := []string{
		"app",
		"idl/thrift",
		"kitex_gen",
		"handler",
		"configs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"app/main.go":              g.mainContent(),
		"idl/thrift/example.thrift": g.thriftContent(),
		"handler/handler.go":       g.handlerContent(),
		"configs/config.yaml":      g.configContent(),
		"go.mod":                   g.goModContent(),
		"readme.md":                g.readmeContent(),
		"kitex_info.yaml":          g.kitexInfoContent(),
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

func (g *MicroserviceThriftGenerator) mainContent() string {
	return fmt.Sprintf(`package main

import (
	"log"
	"github.com/cloudwego/kitex/server"
	"%s/kitex_gen/example/v1/exampleservice"
	"%s/handler"
)

func main() {
	h := handler.NewExampleHandler()
	
	opts := []server.Option{
		server.WithServiceAddr(":8888"),
	}
	
	svr := exampleservice.NewServer(h, opts...)
	
	log.Printf("Starting Thrift service on :8888")
	if err := svr.Run(); err != nil {
		log.Fatalf("server stopped with error: %%v", err)
	}
}
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceThriftGenerator) thriftContent() string {
	return `namespace go example.v1

struct Example {
  1: i64 id,
  2: string name,
  3: string data,
}

struct GetExampleReq {
  1: i64 id,
}

struct GetExampleResp {
  1: Example data,
}

struct CreateExampleReq {
  1: string name,
  2: string data,
}

struct CreateExampleResp {
  1: i64 id,
}

service ExampleService {
  Example getExample(1: GetExampleReq req),
  Example createExample(1: CreateExampleReq req),
}
`
}

func (g *MicroserviceThriftGenerator) handlerContent() string {
	return fmt.Sprintf(`package handler

import (
	"context"
	"%s/kitex_gen/example/v1"
)

// ExampleHandler implements ExampleService
type ExampleHandler struct{}

// NewExampleHandler creates handler
func NewExampleHandler() *ExampleHandler {
	return &ExampleHandler{}
}

// GetExample implements ExampleService
func (h *ExampleHandler) GetExample(ctx context.Context, req *example.GetExampleReq) (*example.GetExampleResp, error) {
	// TODO: Implement business logic
	return &example.GetExampleResp{
		Data: &example.Example{
			Id:   req.Id,
			Name: "Example",
			Data: "Data",
		},
	}, nil
}

// CreateExample implements ExampleService
func (h *ExampleHandler) CreateExample(ctx context.Context, req *example.CreateExampleReq) (*example.CreateExampleResp, error) {
	// TODO: Implement business logic
	return &example.CreateExampleResp{
		Id: 1,
	}, nil
}
`, g.serviceName)
}

func (g *MicroserviceThriftGenerator) configContent() string {
	return `server:
  address: ":8888"
  service_name: example-service

kitex:
  transport: TTHeader
  codec: Thrift

log:
  level: info
  format: json
`
}

func (g *MicroserviceThriftGenerator) kitexInfoContent() string {
	return fmt.Sprintf(`kitex_info:
  serviceName: "%s"
  kitex_version: "v0.9.0"
  thriftgo_version: "v0.3.3"
`, g.serviceName)
}

func (g *MicroserviceThriftGenerator) goModContent() string {
	return fmt.Sprintf(`module %s

go %s
require (
	github.com/cloudwego/kitex v0.9.0
	github.com/apache/thrift v0.19.0
)
`, g.serviceName, GetGoVersion())
}

func (g *MicroserviceThriftGenerator) readmeContent() string {
	return fmt.Sprintf(`# %s - Thrift Microservice

基于 Kitex + Thrift 的微服务项目

## 生成代码

kitex -module %s -service %s idl/thrift/example.thrift

## 运行

go mod tidy
go run app/main.go

## Thrift IDL

idl/thrift/example.thrift 定义了:
- Example 数据结构
- GetExample RPC 方法
- CreateExample RPC 方法

## 参考

book-shop Thrift IDL 示例
`, g.serviceName, g.serviceName, g.serviceName)
}
