package generator

import (
	"os"
	"path/filepath"
	"fmt"
)

type MicroserviceStandardGenerator struct {
	serviceName string
	outputDir   string
}

func NewMicroserviceStandardGenerator(serviceName, outputDir string) *MicroserviceStandardGenerator {
	return &MicroserviceStandardGenerator{serviceName: serviceName, outputDir: outputDir}
}

func (g *MicroserviceStandardGenerator) Generate() error {
	dirs := []string{"app/" + g.serviceName + "/biz/dal/mysql", "app/" + g.serviceName + "/biz/dal/redis", "app/" + g.serviceName + "/biz/model", "app/" + g.serviceName + "/biz/repository", "app/" + g.serviceName + "/biz/service", "app/" + g.serviceName + "/conf", "app/" + g.serviceName + "/handler", "idl", "kitex_gen", "conf/dev"}
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755)
	}
	files := map[string]string{
		"app/" + g.serviceName + "/main.go": g.main(),
		"app/" + g.serviceName + "/handler/handler.go": g.handler(),
		"app/" + g.serviceName + "/biz/model/base.go": g.baseModel(),
		"app/" + g.serviceName + "/biz/model/example.go": g.exampleModel(),
		"app/" + g.serviceName + "/biz/dal/mysql/init.go": g.mysqlInit(),
		"app/" + g.serviceName + "/biz/dal/redis/init.go": g.redisInit(),
		"app/" + g.serviceName + "/biz/repository/example_repo.go": g.repo(),
		"app/" + g.serviceName + "/biz/service/example_service.go": g.service(),
		"app/" + g.serviceName + "/conf/conf.go": g.conf(),
		"conf/dev/conf.yaml": g.devConfig(),
		"idl/example.proto": g.proto(),
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

func (g *MicroserviceStandardGenerator) main() string {
	return fmt.Sprintf(`package main
import (
	"log"
	"github.com/cloudwego/kitex/server"
	"%s/kitex_gen/example/v1/exampleservice"
	"%s/app/%s/handler"
)
func main() {
	h := handler.NewExampleHandler()
	svr := exampleservice.NewServer(h, server.WithServiceAddr(":8888"))
	log.Printf("Starting %s on :8888", "%s")
	svr.Run()
}
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceStandardGenerator) handler() string {
	return fmt.Sprintf(`package handler
import (
	"context"
	"%s/biz/service"
	"%s/kitex_gen/example/v1"
)
type ExampleHandler struct{ svc *service.ExampleService }
func NewExampleHandler() *ExampleHandler { return &ExampleHandler{svc: service.NewExampleService()} }
func (h *ExampleHandler) GetExample(ctx context.Context, req *example.GetExampleReq) (*example.GetExampleResp, error) {
	e, err := h.svc.GetByID(ctx, req.Id)
	if err != nil { return &example.GetExampleResp{}, err }
	return &example.GetExampleResp{Data: &example.Example{Id: e.ID, Name: e.Name, Data: e.Data}}, nil
}
func (h *ExampleHandler) CreateExample(ctx context.Context, req *example.CreateExampleReq) (*example.CreateExampleResp, error) {
	e, _ := h.svc.Create(ctx, &model.Example{Name: req.Name, Data: req.Data})
	return &example.CreateExampleResp{Id: e.ID}, nil
}
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceStandardGenerator) baseModel() string {
	return `package model
import ("time"; "gorm.io/gorm")
// BaseModel 基础模型，包含 gorm.Model 的所有字段
type BaseModel struct {
	gorm.Model
	ID int64 gorm:"primaryKey;autoIncrement"
}
`
}

func (g *MicroserviceStandardGenerator) exampleModel() string {
	return `package model
type Example struct {
	BaseModel
	Name string gorm:"size:255;not null"
	Data string gorm:"type:text"
}
func (Example) TableName() string { return "examples" }
`
}

func (g *MicroserviceStandardGenerator) mysqlInit() string {
	return fmt.Sprintf(`package mysql
import (
	"fmt"
	"os"
	"%s/biz/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)
var DB *gorm.DB
func Init() {
	dsn := fmt.Sprintf("%%s:%%s@tcp(%%s:3306)/%%s?parseTime=true", getEnv("MYSQL_USER","root"), getEnv("MYSQL_PASSWORD",""), getEnv("MYSQL_HOST","localhost"), getEnv("MYSQL_DATABASE","%s"))
	DB, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if os.Getenv("GO_ENV") != "online" { DB.AutoMigrate(&model.Example{}) }
}
func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceStandardGenerator) redisInit() string {
	return `package redis
import (
	"context"
	"os"
	"time"
	"github.com/redis/go-redis/v9"
)
var Client *redis.Client
func Init() {
	Client = redis.NewClient(&redis.Options{Addr: getEnv("REDIS_ADDR","localhost:6379"), Password: os.Getenv("REDIS_PASSWORD"), DB: 0})
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	Client.Ping(ctx)
}
func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
func Close() { if Client != nil { Client.Close() } }
`
}

func (g *MicroserviceStandardGenerator) repo() string {
	return fmt.Sprintf(`package repository
import (
	"context"
	"fmt"
	"%s/biz/dal/mysql"
	"%s/biz/model"
	"gorm.io/gorm"
)
type ExampleRepository struct{ db *gorm.DB }
func NewExampleRepository() *ExampleRepository { return &ExampleRepository{db: mysql.DB} }
func (r *ExampleRepository) Create(ctx context.Context, e *model.Example) error { return r.db.Create(e).Error }
func (r *ExampleRepository) GetByID(ctx context.Context, id int64) (*model.Example, error) {
	var e model.Example
	if err := r.db.First(&e, id).Error; err != nil { if err == gorm.ErrRecordNotFound { return nil, fmt.Errorf("not found") }; return nil, err }
	return &e, nil
}
func (r *ExampleRepository) List(ctx context.Context, offset, limit int) ([]*model.Example, error) {
	var es []*model.Example
	return es, r.db.Offset(offset).Limit(limit).Find(&es).Error
}
func (r *ExampleRepository) Update(ctx context.Context, e *model.Example) error { return r.db.Save(e).Error }
func (r *ExampleRepository) Delete(ctx context.Context, id int64) error { return r.db.Delete(&model.Example{}, id).Error }
func (r *ExampleRepository) Count(ctx context.Context) (int64, error) { var c int64; return c, r.db.Model(&model.Example{}).Count(&c).Error }
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceStandardGenerator) service() string {
	return fmt.Sprintf(`package service
import (
	"context"
	"%s/biz/model"
	"%s/biz/repository"
)
type ExampleService struct{ repo *repository.ExampleRepository }
func NewExampleService() *ExampleService { return &ExampleService{repo: repository.NewExampleRepository()} }
func (s *ExampleService) Create(ctx context.Context, e *model.Example) error { return s.repo.Create(ctx, e) }
func (s *ExampleService) GetByID(ctx context.Context, id int64) (*model.Example, error) { return s.repo.GetByID(ctx, id) }
func (s *ExampleService) List(ctx context.Context, page, size int) ([]*model.Example, int64, error) {
	es, _ := s.repo.List(ctx, (page-1)*size, size); c, _ := s.repo.Count(ctx); return es, c, nil
}
func (s *ExampleService) Update(ctx context.Context, e *model.Example) error { return s.repo.Update(ctx, e) }
func (s *ExampleService) Delete(ctx context.Context, id int64) error { return s.repo.Delete(ctx, id) }
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceStandardGenerator) conf() string {
	return `package conf
import ("os"; "gopkg.in/yaml.v3")
var Conf *Config
type Config struct {
	Server ServerConfig yaml:"server"
	MySQL MySQLConfig yaml:"mysql"
	Redis RedisConfig yaml:"redis"
}
type ServerConfig struct{ ServiceName, Address string }
type MySQLConfig struct{ User, Password, Host, Database string }
type RedisConfig struct{ Addr, Password string; DB int }
func Init() { p := os.Getenv("CONFIG_PATH"); if p == "" { p = "conf/dev/conf.yaml" }; d, _ := os.ReadFile(p); yaml.Unmarshal(d, &Conf) }
func Get() *Config { return Conf }
`
}

func (g *MicroserviceStandardGenerator) devConfig() string {
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
redis:
  addr: localhost:6379
  password: ""
  db: 0
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceStandardGenerator) proto() string {
	return fmt.Sprintf(`syntax = "proto3";
package example.v1;
option go_package = "%s/kitex_gen/example/v1";
message Example { int64 id=1; string name=2; string data=3; }
message GetExampleReq { int64 id=1; }
message GetExampleResp { Example data=1; }
message CreateExampleReq { string name=1; string data=2; }
message CreateExampleResp { int64 id=1; }
service ExampleService { rpc GetExample(GetExampleReq) returns (GetExampleResp); rpc CreateExample(CreateExampleReq) returns (CreateExampleResp); }
`, g.serviceName)
}

func (g *MicroserviceStandardGenerator) goMod() string {
	return fmt.Sprintf(`module %s
go %s
require (
	github.com/cloudwego/kitex v0.9.0
	github.com/redis/go-redis/v9 v9.3.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.5.2
	gorm.io/gorm v1.25.5
)
`, g.serviceName, GetGoVersion())
}

func (g *MicroserviceStandardGenerator) readme() string {
	return fmt.Sprintf(`# %s - Standard Microservice
## Run
kitex -module %s -service %s idl/example.proto
go mod tidy
go run app/%s/main.go
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}
