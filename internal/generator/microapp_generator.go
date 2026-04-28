package generator

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

// MicroAppConfig 微服务应用配置
type MicroAppConfig struct {
	ProjectName  string // 项目名 (如 myShop)
	BFFName      string // BFF 名称 (如 h5, web, app)
	Modules      []ModuleConfig
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	Protocol     string // 通信协议: grpc (默认), kitex
	HTTP         string // BFF HTTP 框架: gin (默认), hertz
}

// ModuleConfig 模块配置
type ModuleConfig struct {
	Name         string // 模块名 (如 user, article)
	UpperName    string // 首字母大写
	ServiceName  string // 服务名
	TableName    string // 数据库表名
	Port         int    // gRPC 端口
	Fields       []ProtoField
}

// MicroAppGenerator 微服务项目生成器
type MicroAppGenerator struct {
	config       *MicroAppConfig
	outputDir    string
	templateDir  string
	db           *sql.DB
}

// NewMicroAppGenerator 创建微服务项目生成器
func NewMicroAppGenerator(config *MicroAppConfig, outputDir string) (*MicroAppGenerator, error) {
	// 连接数据库获取表结构
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 获取模板目录
	execPath, _ := os.Executable()
	templateDir := filepath.Join(filepath.Dir(execPath), "templates", "microapp")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		templateDir = filepath.Join(".", "templates", "microapp")
	}

	return &MicroAppGenerator{
		config:       config,
		outputDir:    outputDir,
		templateDir:  templateDir,
		db:           db,
	}, nil
}

// Close 关闭数据库连接
func (g *MicroAppGenerator) Close() {
	if g.db != nil {
		g.db.Close()
	}
}

// Generate 生成完整的微服务项目
func (g *MicroAppGenerator) Generate() error {
	// 1. 准备模块配置
	if err := g.prepareModules(); err != nil {
		return fmt.Errorf("prepare modules: %w", err)
	}

	// 2. 创建目录结构
	if err := g.createDirectories(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	// 3. 生成公共模块
	if err := g.generateCommon(); err != nil {
		return fmt.Errorf("generate common: %w", err)
	}

	// 4. 生成 pkg 模块
	if err := g.generatePkg(); err != nil {
		return fmt.Errorf("generate pkg: %w", err)
	}

	// 5. 生成 BFF 层
	if err := g.generateBFF(); err != nil {
		return fmt.Errorf("generate BFF: %w", err)
	}

	// 6. 生成微服务
	for _, mod := range g.config.Modules {
		if err := g.generateMicroService(&mod); err != nil {
			return fmt.Errorf("generate microservice %s: %w", mod.Name, err)
		}
	}

	// 7. 生成脚本和 README
	if err := g.generateScripts(); err != nil {
		return fmt.Errorf("generate scripts: %w", err)
	}

	return nil
}

func (g *MicroAppGenerator) prepareModules() error {
	protoGen := NewProtoGenerator(g.db, "", g.config.ProjectName)

	for i, mod := range g.config.Modules {
		// 生成服务名和首字母大写
		mod.UpperName = strings.ToUpper(mod.Name[:1]) + mod.Name[1:]
		mod.ServiceName = mod.UpperName

		// 分配端口 (从 8001 开始)
		mod.Port = 8001 + i

		// 如果没有指定表名，使用模块名
		if mod.TableName == "" {
			mod.TableName = "t_" + mod.Name
		}

		// 从数据库获取字段信息
		info, err := protoGen.GenerateFromTable(mod.TableName)
		if err != nil {
			// 表不存在时使用默认值
			fmt.Printf("Warning: table %s not found, using default fields\n", mod.TableName)
			mod.Fields = []ProtoField{
				{Name: "id", ProtoType: "int64", IsPrimary: true},
				{Name: "name", ProtoType: "string"},
				{Name: "status", ProtoType: "int32"},
			}
		} else {
			mod.Fields = info.Fields
		}

		g.config.Modules[i] = mod
	}

	return nil
}

func (g *MicroAppGenerator) createDirectories() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	dirs := []string{
		filepath.Join(projectDir, "common", "idl"),
		filepath.Join(projectDir, "common", "kitex_gen"),
		filepath.Join(projectDir, "common", "errors"),
		filepath.Join(projectDir, "common", "constants"),
		filepath.Join(projectDir, "pkg", "config"),
		filepath.Join(projectDir, "pkg", "database"),
		filepath.Join(projectDir, "pkg", "logger"),
		filepath.Join(projectDir, "pkg", "registry"),
		filepath.Join(projectDir, "pkg", "utils"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "cmd"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "configs"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "internal", "handler"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "internal", "middleware"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "internal", "rpc_client"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "internal", "router"),
		filepath.Join(projectDir, fmt.Sprintf("bff_%s", g.config.BFFName), "internal", "service"),
		filepath.Join(projectDir, "scripts"),
		filepath.Join(projectDir, "deploy"),
		filepath.Join(projectDir, "tests", "integration"),
		filepath.Join(projectDir, "tests", "e2e"),
		filepath.Join(projectDir, "logs"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// 微服务目录 - 仅在有数据库配置时创建 internal 目录和文件
	if g.config.DBName != "" {
		for _, mod := range g.config.Modules {
srvDir := filepath.Join(projectDir, toCamelCaseDir(mod.Name))
			srvDirs := []string{
				filepath.Join(srvDir, "cmd"),
				filepath.Join(srvDir, "configs"),
				filepath.Join(srvDir, "internal", "handler"),
				filepath.Join(srvDir, "internal", "model"),
				filepath.Join(srvDir, "internal", "repository"),
				filepath.Join(srvDir, "internal", "service"),
			}
			for _, d := range srvDirs {
				if err := os.MkdirAll(d, 0755); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (g *MicroAppGenerator) generateCommon() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// 生成 errors
	errorsContent := `package errors

import "fmt"

type BusinessError struct {
	Code    int
	Message string
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

var (
	ErrNotFound     = NewBusinessError(404, "资源不存在")
	ErrInternal     = NewBusinessError(500, "服务器内部错误")
	ErrInvalidParam = NewBusinessError(400, "参数错误")
)
`
	if err := os.WriteFile(filepath.Join(projectDir, "common", "errors", "error_code.go"), []byte(errorsContent), 0644); err != nil {
		return err
	}

	// 生成 constants
	constantsContent := `package constants

const (
	StatusNormal  = 0
	StatusDeleted = 1
)
`
	if err := os.WriteFile(filepath.Join(projectDir, "common", "constants", "constants.go"), []byte(constantsContent), 0644); err != nil {
		return err
	}

	// 生成 Proto 文件并编译为 pb.go（仅在有数据库配置时）
	if g.config.DBName != "" {
		// 验证 proto 工具是否已安装
		if !checkProtoTools() {
			return fmt.Errorf("protoc 工具未安装，请先安装以下工具：\n" +
				"1. protoc (protobuf 编译器):\n" +
				"   macOS: brew install protobuf\n" +
				"   Linux: sudo apt install protobuf-compiler\n" +
				"   Windows: 下载 https://github.com/protocolbuffers/protobuf/releases\n\n" +
				"2. protoc-gen-go (Go protobuf 插件):\n" +
				"   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest\n\n" +
				"3. protoc-gen-go-grpc (Go gRPC 插件):\n" +
				"   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest\n\n" +
				"安装完成后重新运行此命令")
		}

		idlDir := filepath.Join(projectDir, "common", "idl")
		genDir := filepath.Join(projectDir, "common", "kitex_gen")

		// 确保输出目录存在
		if err := os.MkdirAll(genDir, 0755); err != nil {
			return fmt.Errorf("create kitex_gen dir: %w", err)
		}

		for _, mod := range g.config.Modules {
			protoContent := g.generateProtoContent(&mod)
			// 使用表名转换作为 proto 文件名（去掉前缀，小驼峰）
			protoFileName := toCamelCaseFile(mod.TableName) + ".proto"
			protoPath := filepath.Join(idlDir, protoFileName)
			if err := os.WriteFile(protoPath, []byte(protoContent), 0644); err != nil {
				return fmt.Errorf("write proto file: %w", err)
			}

			// 调用 protoc 生成 pb.go 文件
			cmd := exec.Command("sh", "-c",
				fmt.Sprintf("cd %s && protoc -I . --go_out=../kitex_gen --go-grpc_out=../kitex_gen %s",
					idlDir, protoFileName))
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("protoc output: %%s", string(output))
				log.Printf("protoc error (ignored): %%v", err)
			}
		}
	}

	return nil
}

func (g *MicroAppGenerator) generateProtoContent(mod *ModuleConfig) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`syntax = "proto3";

package %s;

option go_package = "%s/common/kitex_gen/%s";

service %%sService {
  rpc Create(Create%sReq) returns (Create%sResp);
  rpc Get(Get%sReq) returns (Get%sResp);
  rpc List(List%sReq) returns (List%sResp);
  rpc Update(Update%sReq) returns (Update%sResp);
  rpc Delete(Delete%sReq) returns (Delete%%sResp);
}

`, mod.Name, g.config.ProjectName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName))

	// 请求消息
	sb.WriteString(fmt.Sprintf(`message Create%sReq {
  string name = 1;
}

message Get%sReq {
  int64 id = 1;
}

message List%sReq {
  int32 page = 1;
  int32 page_size = 10;
}

message Update%sReq {
  int64 id = 1;
  string name = 2;
}

message Delete%sReq {
  int64 id = 1;
}

`, mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName))

	// 响应消息
	sb.WriteString(fmt.Sprintf(`message Create%sResp {
  int64 id = 1;
  bool success = 2;
}

message Get%sResp {
  int64 id = 1;
  string name = 2;
  int32 status = 3;
}

message %sItem {
  int64 id = 1;
  string name = 2;
  int32 status = 3;
}

message List%sResp {
  repeated %sItem items = 1;
  int64 total = 2;
}

message Update%sResp {
  bool success = 1;
}

message Delete%sResp {
  bool success = 1;
}
`, mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName, mod.ServiceName))

	return sb.String()
}

func (g *MicroAppGenerator) generatePkg() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// config
	configContent := fmt.Sprintf(`package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   ` + "`yaml:\"server\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\"`" + `
	Registry RegistryConfig ` + "`yaml:\"registry\"`" + `
	Log      LogConfig      ` + "`yaml:\"log\"`" + `
}

type ServerConfig struct {
	Host string ` + "`yaml:\"host\"`" + `
	Port int    ` + "`yaml:\"port\"`" + `
}

type DatabaseConfig struct {
	Host     string ` + "`yaml:\"host\"`" + `
	Port     string ` + "`yaml:\"port\"`" + `
	User     string ` + "`yaml:\"user\"`" + `
	Password string ` + "`yaml:\"password\"`" + `
	Database string ` + "`yaml:\"database\"`" + `
}

type RegistryConfig struct {
	Type string ` + "`yaml:\"type\"`" + `
	Addr string ` + "`yaml:\"addr\"`" + `
}

type LogConfig struct {
	Level  string ` + "`yaml:\"level\"`" + `
	Format string ` + "`yaml:\"format\"`" + `
}

func (c *Config) GetAddr() string {
	return fmt.Sprintf("%%s:%%d", c.Server.Host, c.Server.Port)
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
`)
	if err := os.WriteFile(filepath.Join(projectDir, "pkg", "config", "config.go"), []byte(configContent), 0644); err != nil {
		return err
	}

	// database
	dbContent := fmt.Sprintf(`package database

import (
	"fmt"

	"%s/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%%s:%%s@tcp(%%s:%%s)/%%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
`, g.config.ProjectName)
	if err := os.WriteFile(filepath.Join(projectDir, "pkg", "database", "database.go"), []byte(dbContent), 0644); err != nil {
		return err
	}

	// logger
	loggerContent := `package logger

import (
	"log"
	"os"
)

var (
	Info  = log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile)
	Error = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile)
	Debug = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
)
`
	if err := os.WriteFile(filepath.Join(projectDir, "pkg", "logger", "logger.go"), []byte(loggerContent), 0644); err != nil {
		return err
	}

	// utils
	utilsContent := `package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "pkg", "utils", "crypto.go"), []byte(utilsContent), 0644); err != nil {
		return err
	}

	// registry - 服务注册与发现
	if err := g.generatePkgRegistry(); err != nil {
		return err
	}

	// utils - JWT 和 validator
	if err := g.generateUtilsJWT(); err != nil {
		return err
	}

	return nil
}

// generatePkgRegistry 生成 pkg/registry/etcd.go
func (g *MicroAppGenerator) generatePkgRegistry() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	registryContent := `package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/etcd/client/v3"
	"github.com/coreos/etcd/client/v3/naming/endpoints"
)

// EtcdRegistry Etcd 服务注册与发现
type EtcdRegistry struct {
	client *clientv3.Client
	opts   []clientv3.OpOption
}

// ServiceInstance 服务实例
type ServiceInstance struct {
	Name string
	Addr string
	Port int
}

// NewEtcdRegistry 创建 Etcd 注册中心客户端
func NewEtcdRegistry(addr string) (*EtcdRegistry, error) {
	cli, err := clientv3.NewFromURL(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect etcd: %w", err)
	}
	return &EtcdRegistry{client: cli}, nil
}

// Close 关闭连接
func (r *EtcdRegistry) Close() error {
	return r.client.Close()
}

// Register 注册服务
func (r *EtcdRegistry) Register(ctx context.Context, service, addr string, port int, ttl int64) error {
	key := fmt.Sprintf("/%s/%s:%d", service, addr, port)
	leaseID, err := r.registerLease(ctx, ttl)
	if err != nil {
		return err
	}
	_, err = r.client.Put(ctx, key, addr, clientv3.WithLease(leaseID))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}
	return nil
}

// Unregister 注销服务
func (r *EtcdRegistry) Unregister(ctx context.Context, service, addr string, port int) error {
	key := fmt.Sprintf("/%s/%s:%d", service, addr, port)
	_, err := r.client.Delete(ctx, key)
	return err
}

// Discover 发现服务
func (r *EtcdRegistry) Discover(ctx context.Context, service string) ([]string, error) {
	key := fmt.Sprintf("/%s/", service)
	resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	var addrs []string
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}
	return addrs, nil
}

// Watch 监听服务变更
func (r *EtcdRegistry) Watch(ctx context.Context, service string) (<-chan []string, error) {
	key := fmt.Sprintf("/%s/", service)
	rch := r.client.Watch(ctx, key, clientv3.WithPrefix())

	ch := make(chan []string, 1)
	go func() {
		for wresp := range rch {
			var addrs []string
			for _, ev := range wresp.Events {
				if ev.Type == endpoints.EventTypeDelete {
					continue
				}
				addrs = append(addrs, string(ev.Kv.Value))
			}
			if len(addrs) > 0 {
				ch <- addrs
			}
		}
	}()
	return ch, nil
}

func (r *EtcdRegistry) registerLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error) {
	resp, err := r.client.Grant(ctx, ttl)
	if err != nil {
		return 0, err
	}
	return resp.ID, nil
}

// ServiceDiscovery 服务发现（简化版）
type ServiceDiscovery struct {
	mu      sync.RWMutex
	servers map[string][]string
}

// NewServiceDiscovery 创建服务发现
func NewServiceDiscovery() *ServiceDiscovery {
	return &ServiceDiscovery{
		servers: make(map[string][]string),
	}
}

// Register 注册服务地址
func (sd *ServiceDiscovery) Register(service string, addr string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.servers[service] = append(sd.servers[service], addr)
}

// Unregister 注销服务地址
func (sd *ServiceDiscovery) Unregister(service string, addr string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	addrs := sd.servers[service]
	for i, a := range addrs {
		if a == addr {
			sd.servers[service] = append(addrs[:i], addrs[i+1:]...)
			break
		}
	}
}

// GetServices 获取服务地址列表
func (sd *ServiceDiscovery) GetServices(service string) []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.servers[service]
}

// RoundRobin 轮询获取服务地址
func (sd *ServiceDiscovery) RoundRobin(service string) string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	addrs := sd.servers[service]
	if len(addrs) == 0 {
		return ""
	}
	return addrs[time.Now().UnixNano()%int64(len(addrs))]
}
`
	return os.WriteFile(filepath.Join(projectDir, "pkg", "registry", "etcd.go"), []byte(registryContent), 0644)
}

// generateUtilsJWT 生成 pkg/utils/jwt.go 和 validator.go
func (g *MicroAppGenerator) generateUtilsJWT() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// jwt.go
	jwtContent := `package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken 无效的 Token
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken 过期的 Token
	ErrExpiredToken = errors.New("token has expired")
)

// JWTClaims JWT Claims
type JWTClaims struct {
	UserID   int64    ` + "`" + `json:"user_id"` + "`" + `
	Username string   ` + "`" + `json:"username"` + "`" + `
	Roles    []string ` + "`" + `json:"roles"` + "`" + `
	jwt.RegisteredClaims
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret      string
	ExpireHours int
	Issuer      string
}

// JWT JWT 工具
type JWT struct {
	secret      []byte
	expireHours int
	issuer      string
}

// NewJWT 创建 JWT 实例
func NewJWT(secret string, expireHours int, issuer string) *JWT {
	return &JWT{
		secret:      []byte(secret),
		expireHours: expireHours,
		issuer:      issuer,
	}
}

// DefaultJWT 默认 JWT 实例
var DefaultJWT = NewJWT("your-secret-key", 24, "gospacex")

// GenerateToken 生成 Token
func GenerateToken(userID int64, username string, roles []string) (string, error) {
	return DefaultJWT.GenerateToken(userID, username, roles)
}

// GenerateTokenWithConfig 使用自定义配置生成 Token
func GenerateTokenWithConfig(cfg *JWTConfig, userID int64, username string, roles []string) (string, error) {
	jwtInstance := NewJWT(cfg.Secret, cfg.ExpireHours, cfg.Issuer)
	return jwtInstance.GenerateToken(userID, username, roles)
}

// GenerateToken 生成 Token
func (j *JWT) GenerateToken(userID int64, username string, roles []string) (string, error) {
	now := time.Now()
	expireTime := now.Add(time.Duration(j.expireHours) * time.Hour)

	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expireTime),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

// ParseJWT 解析 Token
func ParseJWT(tokenString string) (*JWTClaims, error) {
	return DefaultJWT.ParseJWT(tokenString)
}

// ParseJWTWithConfig 使用自定义配置解析 Token
func ParseJWTWithConfig(cfg *JWTConfig, tokenString string) (*JWTClaims, error) {
	jwtInstance := NewJWT(cfg.Secret, cfg.ExpireHours, cfg.Issuer)
	return jwtInstance.ParseJWT(tokenString)
}

// ParseJWT 解析 Token
func (j *JWT) ParseJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return j.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken 刷新 Token
func RefreshToken(tokenString string) (string, error) {
	claims, err := ParseJWT(tokenString)
	if err != nil && !errors.Is(err, ErrExpiredToken) {
		return "", err
	}
	return GenerateToken(claims.UserID, claims.Username, claims.Roles)
}

// ValidateToken 验证 Token 有效性
func ValidateToken(tokenString string) bool {
	_, err := ParseJWT(tokenString)
	return err == nil
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "pkg", "utils", "jwt.go"), []byte(jwtContent), 0644); err != nil {
		return err
	}

	// validator.go
	validatorContent := `package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrValidationFailed 验证失败
	ErrValidationFailed = errors.New("validation failed")
	// ErrInvalidFormat 格式错误
	ErrInvalidFormat = errors.New("invalid format")
)

// Validator 参数校验器
type Validator struct{}

// NewValidator 创建校验器
func NewValidator() *Validator {
	return &Validator{}
}

// DefaultValidator 默认校验器
var DefaultValidator = NewValidator()

// Required 必填校验
func Required(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: field is required", ErrValidationFailed)
	}
	return nil
}

// MinLength 最小长度校验
func MinLength(value string, min int) error {
	if len(value) < min {
		return fmt.Errorf("%w: minimum length is %d", ErrValidationFailed, min)
	}
	return nil
}

// MaxLength 最大长度校验
func MaxLength(value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%w: maximum length is %d", ErrValidationFailed, max)
	}
	return nil
}

// Email 邮箱格式校验
func Email(email string) error {
	pattern := "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
	if !regexp.MustCompile(pattern).MatchString(email) {
		return fmt.Errorf("%w: invalid email format", ErrInvalidFormat)
	}
	return nil
}

// Phone 手机号格式校验（中国大陆）
func Phone(phone string) error {
	pattern := "^1[3-9]\\d{9}$"
	if !regexp.MustCompile(pattern).MatchString(phone) {
		return fmt.Errorf("%w: invalid phone format", ErrInvalidFormat)
	}
	return nil
}

// IDCard 身份证格式校验（中国大陆）
func IDCard(idCard string) error {
	pattern := "^[1-9]\\d{5}(18|19|20)\\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\\d|3[01])\\d{3}[\\dXx]$"
	if !regexp.MustCompile(pattern).MatchString(idCard) {
		return fmt.Errorf("%w: invalid ID card format", ErrInvalidFormat)
	}
	return nil
}

// URL URL 格式校验
func URL(url string) error {
	pattern := "^https?://[^\\s/$.?#].[^\\s]*$"
	if !regexp.MustCompile(pattern).MatchString(url) {
		return fmt.Errorf("%w: invalid URL format", ErrInvalidFormat)
	}
	return nil
}

// Numeric 纯数字校验
func Numeric(value string) error {
	pattern := "^\\d+$"
	if !regexp.MustCompile(pattern).MatchString(value) {
		return fmt.Errorf("%w: must be numeric", ErrInvalidFormat)
	}
	return nil
}

// Alpha 英文字母校验
func Alpha(value string) error {
	pattern := "^[a-zA-Z]+$"
	if !regexp.MustCompile(pattern).MatchString(value) {
		return fmt.Errorf("%w: must be alphabetic", ErrInvalidFormat)
	}
	return nil
}

// AlphaNumeric 英文字母和数字校验
func AlphaNumeric(value string) error {
	pattern := "^[a-zA-Z0-9]+$"
	if !regexp.MustCompile(pattern).MatchString(value) {
		return fmt.Errorf("%w: must be alphanumeric", ErrInvalidFormat)
	}
	return nil
}

// Range 范围校验
func Range(value, min, max int) error {
	if value < min || value > max {
		return fmt.Errorf("%w: value must be between %d and %d", ErrValidationFailed, min, max)
	}
	return nil
}

// Validate 通用校验函数
func (v *Validator) Validate(value string, rules ...func(string) error) error {
	for _, rule := range rules {
		if err := rule(value); err != nil {
			return err
		}
	}
	return nil
}

// ValidateStruct 结构体校验（简单实现）
func ValidateStruct(data map[string]interface{}, rules map[string][]func(string) error) error {
	for field, valueRules := range rules {
		value, ok := data[field].(string)
		if !ok {
			return fmt.Errorf("field %s is not a string", field)
		}
		for _, rule := range valueRules {
			if err := rule(value); err != nil {
				return fmt.Errorf("field %s: %w", field, err)
			}
		}
	}
	return nil
}
`
	return os.WriteFile(filepath.Join(projectDir, "pkg", "utils", "validator.go"), []byte(validatorContent), 0644)
}

func (g *MicroAppGenerator) generateBFF() error {
	// 根据 HTTP 框架选择不同的模板
	if g.config.HTTP == "hertz" {
		return g.generateBFFHertz()
	}
	// 默认使用 Gin
	return g.generateBFFGin()
}

// generateBFFGin 生成 Gin BFF 层
func (g *MicroAppGenerator) generateBFFGin() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	bffName := g.config.BFFName

	// main.go
	mainContent := fmt.Sprintf(`package main

import (
	"flag"
	"fmt"
	"log"

	"%s/bff_%s/internal/router"
	"%s/pkg/config"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file")
}

func main() {
	flag.Parse()
	cfg, err := config.Load(confPath)
	if err != nil {
		log.Fatal(err)
	}
	addr := cfg.GetAddr()
	log.Printf("BFF starting on %%s", addr)
	router.NewRouter().Run(addr)
}
`, g.config.ProjectName, bffName, g.config.ProjectName)
	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "cmd", "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// config.yaml
	configYaml := `server:
  host: 0.0.0.0
  port: 8080

registry:
  type: direct
  addr: localhost:2379

log:
  level: info
  format: json
`
	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "configs", "config.yaml"), []byte(configYaml), 0644); err != nil {
		return err
	}

	// router.go
	var routerContent strings.Builder
	routerContent.WriteString(fmt.Sprintf(`package router

import (
	"%s/bff_%s/internal/handler"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

`, g.config.ProjectName, bffName))

	for _, mod := range g.config.Modules {
		routerContent.WriteString(fmt.Sprintf(`	// %s 模块
	%sHandler := handler.New%sHandler()
	r.GET("/api/v1/%ss", %%sHandler.List)
	r.GET("/api/v1/%ss/:id", %%sHandler.Get)
	r.POST("/api/v1/%ss", %%sHandler.Create)
	r.PUT("/api/v1/%ss/:id", %%sHandler.Update)
r.DELETE("/api/v1/%%ss/:id", %%sHandler.Delete)

`, mod.UpperName, mod.Name,
		mod.Name, mod.Name, mod.Name, mod.Name, mod.Name))
	}
	routerContent.WriteString("\treturn r\n}\n")

	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "internal", "router", "router.go"), []byte(routerContent.String()), 0644); err != nil {
		return err
	}

	// middleware - 完整版本
	if err := g.generateBFFMiddleware(bffName); err != nil {
		return err
	}

	// 生成各模块的 handler, rpc_client
	for _, mod := range g.config.Modules {
		if err := g.generateBFFModule(&mod, bffName); err != nil {
			return err
		}
	}

	return nil
}

func (g *MicroAppGenerator) generateBFFModule(mod *ModuleConfig, bffName string) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// rpc_client
	clientContent := fmt.Sprintf(`package rpc_client

import (
	"context"
	"fmt"
	"time"

	"%s/common/kitex_gen/%s"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type %sClient struct {
	conn   *grpc.ClientConn
	client %s.%sServiceClient
}

func New%sClient(addr string) (*%sClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %%w", err)
	}

	return &%sClient{
		conn:   conn,
		client: %s.New%sServiceClient(conn),
	}, nil
}

func (c *%sClient) Close() error {
	return c.conn.Close()
}

func (c *%sClient) Create(ctx context.Context, name string) (*%s.Create%sResp, error) {
	return c.client.Create%s(ctx, &%s.Create%sReq{Name: name})
}

func (c *%sClient) Get(ctx context.Context, id int64) (*%s.Get%sResp, error) {
	return c.client.Get%s(ctx, &%s.Get%sReq{Id: id})
}

func (c *%sClient) List(ctx context.Context) (*%s.List%sResp, error) {
	return c.client.List%s(ctx, &%s.List%sReq{})
}

func (c *%sClient) Update(ctx context.Context, id int64, name string) (*%s.Update%sResp, error) {
	return c.client.Update%s(ctx, &%s.Update%sReq{Id: id, Name: name})
}

func (c *%sClient) Delete(ctx context.Context, id int64) (*%s.Delete%sResp, error) {
	return c.client.Delete%s(ctx, &%s.Delete%sReq{Id: id})
}
`, g.config.ProjectName, mod.Name,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName,
		mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName)

	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "internal", "rpc_client", toCamelCaseFile(mod.Name)+"Client.go"), []byte(clientContent), 0644); err != nil {
		return err
	}

	// handler
	handlerContent := fmt.Sprintf(`package handler

import (
	"net/http"
	"strconv"

	"%s/bff_%s/internal/rpc_client"
	"%s/pkg/logger"
	"github.com/gin-gonic/gin"
)

type %sHandler struct {
	client *rpc_client.%sClient
}

func New%sHandler() *%sHandler {
	client, _ := rpc_client.New%sClient("127.0.0.1:%s")
	return &%sHandler{client: client}
}

func (h *%sHandler) Create(c *gin.Context) {
	var req struct {
		Name string `+"`json:\"name\" binding:\"required\"`"+`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.client.Create(c.Request.Context(), req.Name)
	if err != nil {
		logger.Error.Printf("Create failed: %%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": resp.Id, "success": resp.Success})
}

func (h *%sHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	resp, err := h.client.Get(c.Request.Context(), id)
	if err != nil {
		logger.Error.Printf("Get failed: %%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": resp.Id, "name": resp.Name, "status": resp.Status})
}

func (h *%sHandler) List(c *gin.Context) {
	resp, err := h.client.List(c.Request.Context())
	if err != nil {
		logger.Error.Printf("List failed: %%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, gin.H{"id": item.Id, "name": item.Name, "status": item.Status})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": resp.Total})
}

func (h *%sHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name string `+"`json:\"name\"`"+`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.client.Update(c.Request.Context(), id, req.Name)
	if err != nil {
		logger.Error.Printf("Update failed: %%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}

func (h *%sHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	resp, err := h.client.Delete(c.Request.Context(), id)
	if err != nil {
		logger.Error.Printf("Delete failed: %%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}
`, g.config.ProjectName, bffName, g.config.ProjectName, g.config.ProjectName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, fmt.Sprintf("%d", mod.Port))

	return os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "internal", "handler", toCamelCaseFile(mod.Name)+"Handler.go"), []byte(handlerContent), 0644)
}

func (g *MicroAppGenerator) generateMicroService(mod *ModuleConfig) error {
	// 仅在有数据库配置时生成 srv 内部文件
	if g.config.DBName == "" {
		return nil
	}

	// 根据协议选择不同的模板
	if g.config.Protocol == "kitex" {
		return g.generateKitexMicroService(mod)
	}
	// 默认使用 gRPC
	return g.generateGRPCMicroService(mod)
}

// generateGRPCMicroService 生成 gRPC 微服务
func (g *MicroAppGenerator) generateGRPCMicroService(mod *ModuleConfig) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	srvDir := filepath.Join(projectDir, toCamelCaseDir(mod.Name))

	// main.go
	mainContent := fmt.Sprintf(`package main

import (
	"flag"
	"fmt"
	"log"

	"%s/srv_%s/internal/handler"
	"%s/pkg/config"
	"%s/pkg/logger"
	"%s/pkg/database"
	"%s/common/errors"
	"%s/common/kitex_gen/%s"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file path")
}

func main() {
	flag.Parse()
	cfg, err := config.Load(confPath)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化数据库
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		logger.Error.Fatalf("Failed to connect database: %%v", err)
	}

	// 创建 gRPC 服务
	grpcServer := grpc.NewServer()
	
	// 注册服务
	%sServiceServer := handler.New%sHandler(db)
	%s.Register%sServiceServer(grpcServer, %sServiceServer)
	
	// 注册反射服务（用于调试）
	reflection.Register(grpcServer)

	addr := cfg.GetAddr()
	logger.Info.Printf("%%s service starting on %%s", "%s", addr)
	if err := grpcServer.Serve(generateListener(addr)); err != nil {
		logger.Error.Fatalf("Failed to serve: %%v", err)
	}
}

func generateListener(addr string) net.Listener {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %%v", err)
	}
	return lis
}
`, g.config.ProjectName, mod.Name,
		g.config.ProjectName, g.config.ProjectName,
		g.config.ProjectName, g.config.ProjectName,
		g.config.ProjectName, mod.Name,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.UpperName)
	if err := os.WriteFile(filepath.Join(srvDir, "cmd", "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// config.yaml
	configYaml := fmt.Sprintf(`server:
  host: 0.0.0.0
  port: %d

database:
  host: %%s
  port: %%s
  user: %%s
  password: %%s
  database: %%s

registry:
  type: direct
  addr: localhost:2379

log:
  level: info
  format: json
`, mod.Port)
	if err := os.WriteFile(filepath.Join(srvDir, "configs", "config.yaml"), []byte(configYaml), 0644); err != nil {
		return err
	}

	// handler
	handlerContent := fmt.Sprintf(`package handler

import (
	"context"

	"%s/srv_%s/internal/model"
	"%s/srv_%s/internal/repository"
	"%s/srv_%s/internal/service"
	"%s/common/kitex_gen/%s"
)

type %sHandler struct {
	repo *repository.%sRepository
	svc  *service.%sService
}

func New%sHandler(db interface{}) *%sHandler {
	repo := repository.New%sRepository(db)
	svc := service.New%sService(repo)
	return &%sHandler{
		repo: repo,
		svc:  svc,
	}
}

func (h *%sHandler) Create(ctx context.Context, req *%s.Create%sReq) (*%s.Create%sResp, error) {
	id, err := h.svc.Create(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &%s.Create%sResp{Id: id, Success: true}, nil
}

func (h *%sHandler) Get(ctx context.Context, req *%s.Get%sReq) (*%s.Get%sResp, error) {
	item, err := h.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &%s.Get%sResp{Id: item.Id, Name: item.Name, Status: item.Status}, nil
}

func (h *%sHandler) List(ctx context.Context, req *%s.List%sReq) (*%s.List%sResp, error) {
	items, total, err := h.svc.List(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	
	respItems := make([]*%s.%sItem, len(items))
	for i, item := range items {
		respItems[i] = &%s.%sItem{Id: item.Id, Name: item.Name, Status: item.Status}
	}
	return &%s.List%sResp{Items: respItems, Total: total}, nil
}

func (h *%sHandler) Update(ctx context.Context, req *%s.Update%sReq) (*%s.Update%sResp, error) {
	err := h.svc.Update(ctx, req.Id, req.Name)
	if err != nil {
		return &%s.Update%sResp{Success: false}, err
	}
	return &%s.Update%sResp{Success: true}, nil
}

func (h *%sHandler) Delete(ctx context.Context, req *%s.Delete%sReq) (*%s.Delete%sResp, error) {
	err := h.svc.Delete(ctx, req.Id)
	if err != nil {
		return &%s.Delete%sResp{Success: false}, err
	}
	return &%s.Delete%sResp{Success: true}, nil
}
`, g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name,
		mod.ServiceName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName)
	if err := os.WriteFile(filepath.Join(srvDir, "internal", "handler", toCamelCaseFile(mod.Name)+"Handler.go"), []byte(handlerContent), 0644); err != nil {
		return err
	}

	// model
	modelContent := fmt.Sprintf(`package model

type %s struct {
	Id        int64  `+"`gorm:\"primaryKey;autoIncrement\" json:\"id\"`"+`
	Name      string `+"`gorm:\"size:255;not null\" json:\"name\"`"+`
	Status    int32  `+"`gorm:\"default:0\" json:\"status\"`"+`
}

func (%s) TableName() string {
	return "%s"
}
`, mod.UpperName, mod.UpperName, mod.TableName)
	if err := os.WriteFile(filepath.Join(srvDir, "internal", "model", "model.go"), []byte(modelContent), 0644); err != nil {
		return err
	}

	// repository
	repoContent := fmt.Sprintf(`package repository

import (
	"context"
	"errors"

	"%s/srv_%s/internal/model"
	"gorm.io/gorm"
)

type %sRepository struct {
	db *gorm.DB
}

func New%sRepository(db interface{}) *%sRepository {
	return &%sRepository{db: db.(*gorm.DB)}
}

func (r *%sRepository) Create(ctx context.Context, name string) (int64, error) {
	item := &model.%s{Name: name, Status: 0}
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return 0, err
	}
	return item.Id, nil
}

func (r *%sRepository) GetById(ctx context.Context, id int64) (*model.%s, error) {
	var item model.%s
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &item, nil
}

func (r *%sRepository) List(ctx context.Context, page, pageSize int) ([]*model.%s, int64, error) {
	var items []*model.%s
	var total int64
	
	r.db.WithContext(ctx).Model(&model.%s{}).Count(&total)
	
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *%sRepository) Update(ctx context.Context, id int64, name string) error {
	return r.db.WithContext(ctx).Model(&model.%s{}).Where("id = ?", id).Update("name", name).Error
}

func (r *%sRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.%s{}, id).Error
}
`, g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName)
	if err := os.WriteFile(filepath.Join(srvDir, "internal", "repository", toCamelCaseFile(mod.Name)+"Repo.go"), []byte(repoContent), 0644); err != nil {
		return err
	}

	// service
	svcContent := fmt.Sprintf(`package service

import (
	"context"

	"%s/srv_%s/internal/model"
	"%s/srv_%s/internal/repository"
	"%s/common/errors"
)

type %sService struct {
	repo *repository.%sRepository
}

func New%sService(repo *repository.%sRepository) *%sService {
	return &%sService{repo: repo}
}

func (s *%sService) Create(ctx context.Context, name string) (int64, error) {
	if name == "" {
		return 0, errors.ErrInvalidParam
	}
	return s.repo.Create(ctx, name)
}

func (s *%sService) Get(ctx context.Context, id int64) (*model.%s, error) {
	return s.repo.GetById(ctx, id)
}

func (s *%sService) List(ctx context.Context, page, pageSize int) ([]*model.%s, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.List(ctx, page, pageSize)
}

func (s *%sService) Update(ctx context.Context, id int64, name string) error {
	if id == 0 || name == "" {
		return errors.ErrInvalidParam
	}
	return s.repo.Update(ctx, id, name)
}

func (s *%sService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.ErrInvalidParam
	}
	return s.repo.Delete(ctx, id)
}
`, g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		g.config.ProjectName,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName)
	return os.WriteFile(filepath.Join(srvDir, "internal", "service", toCamelCaseFile(mod.Name)+"Service.go"), []byte(svcContent), 0644)
}

func (g *MicroAppGenerator) generateScripts() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// gen_proto.sh
	var genProtoContent strings.Builder
genProtoContent.WriteString(`#!/bin/bash
set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
IDL_DIR="$PROJECT_DIR/common/idl"
GEN_DIR="$PROJECT_DIR/common/kitex_gen"

echo "Generating proto files..."
echo "IDL dir: $IDL_DIR"
echo "Output dir: $GEN_DIR"

# 创建输出目录
mkdir -p "$GEN_DIR"

# 切换到 IDL 目录
cd "$IDL_DIR"

# 为每个模块生成 proto 代码
`)
	for _, mod := range g.config.Modules {
		genProtoContent.WriteString(fmt.Sprintf(`echo "Generating %s..."
protoc --go_out=../../common/kitex_gen --go-grpc_out=../../common/kitex_gen %s.proto
`, mod.Name, mod.Name))
	}
	genProtoContent.WriteString(`
echo ""
echo "✓ Proto files generated successfully!"
echo "Output: $GEN_DIR"
echo ""
echo "Next steps:"
echo "  1. cd $PROJECT_DIR"
echo "  2. go mod tidy"
echo "  3. go run cmd/main.go"
`)

	if err := os.WriteFile(filepath.Join(projectDir, "scripts", "gen_proto.sh"), []byte(genProtoContent.String()), 0755); err != nil {
		return err
	}

	// build.sh
	var buildContent strings.Builder
	buildContent.WriteString(`#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$PROJECT_DIR/bin"
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "========================================="
echo "Building microservices project"
echo "========================================="
echo "Project: ${PROJECT_DIR##*/}"
echo "Build time: $BUILD_TIME"
echo "Git commit: $GIT_COMMIT"
echo ""

# 创建 bin 目录
mkdir -p "$BIN_DIR"

# 清理旧的二进制文件
rm -f "$BIN_DIR"/*

echo "Building BFF layer..."
`)
	buildContent.WriteString(fmt.Sprintf(`cd "$PROJECT_DIR/bff_%s/cmd"
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" -o "$BIN_DIR/bff_%s" main.go
echo "✓ BFF built: $BIN_DIR/bff_%s"
`, g.config.BFFName, g.config.BFFName, g.config.BFFName))

	for _, mod := range g.config.Modules {
		buildContent.WriteString(fmt.Sprintf(`
echo "Building %s microservice..."
cd "$PROJECT_DIR/srv_%s/cmd"
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.BuildTime=%s -X main.GitCommit=$GIT_COMMIT" -o "$BIN_DIR/srv_%s" main.go
echo "✓ %s built: $BIN_DIR/srv_%s"
`, mod.Name, mod.Name, "$BUILD_TIME", mod.Name, mod.Name, mod.Name))
	}

	buildContent.WriteString(`
echo ""
echo "========================================="
echo "Build complete!"
echo "========================================="
echo "Binary files:"
ls -lh "$BIN_DIR"
`)

	if err := os.WriteFile(filepath.Join(projectDir, "scripts", "build.sh"), []byte(buildContent.String()), 0755); err != nil {
		return err
	}

	// init_db.sql
	var dbContent strings.Builder
	dbContent.WriteString(`-- 微服务项目数据库初始化脚本
-- Created by gospacex

-- 创建数据库
CREATE DATABASE IF NOT EXISTS gospacex DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE gospacex;

`)
	for _, mod := range g.config.Modules {
		dbContent.WriteString(fmt.Sprintf(`-- %s 表
CREATE TABLE IF NOT EXISTS %s (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL COMMENT '名称',
    status INT DEFAULT 0 COMMENT '状态: 0=正常, 1=禁用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    INDEX idx_name (name),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='%s表';

`, mod.Name, mod.TableName, mod.Name))
	}

	dbContent.WriteString(`
-- 用户表（示例）
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE COMMENT '用户名',
    password VARCHAR(255) NOT NULL COMMENT '密码(加密)',
    email VARCHAR(128) COMMENT '邮箱',
    phone VARCHAR(32) COMMENT '手机号',
    status INT DEFAULT 0 COMMENT '状态: 0=正常, 1=禁用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    INDEX idx_username (username),
    INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 订单表（示例）
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL UNIQUE COMMENT '订单号',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '总金额',
    status INT DEFAULT 0 COMMENT '状态: 0=待支付, 1=已支付, 2=已取消',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_order_no (order_no),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';

-- 订单项表（示例）
CREATE TABLE IF NOT EXISTS order_items (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL COMMENT '订单ID',
    product_id BIGINT NOT NULL COMMENT '商品ID',
    product_name VARCHAR(255) NOT NULL COMMENT '商品名称',
    price DECIMAL(10,2) NOT NULL COMMENT '单价',
    quantity INT NOT NULL DEFAULT 1 COMMENT '数量',
    subtotal DECIMAL(10,2) NOT NULL COMMENT '小计',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单项表';

SELECT 'Database initialization complete!' AS result;
`)

	if err := os.WriteFile(filepath.Join(projectDir, "scripts", "init_db.sql"), []byte(dbContent.String()), 0644); err != nil {
		return err
	}

	// 生成 deploy 文件
	if err := g.generateDeploy(); err != nil {
		return err
	}

	// README
	readmeContent := g.generateReadme()
	return os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(readmeContent), 0644)
}

// generateDeploy 生成部署文件
func (g *MicroAppGenerator) generateDeploy() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// docker-compose.yaml
	dockerComposeContent := `version: '3.8'

services:
  # etcd 服务注册中心
  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: gospacex-etcd
    environment:
      - ETCD_AUTH_TOKEN=simple
      - ETCD_NAME=etcd
      - ETCD_ADVERTISE_CLIENT_URLS=http://etcd:2379
      - ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379
    ports:
      - "2379:2379"
    volumes:
      - etcd_data:/etcd
    command: etcd -data-dir=/etcd
    networks:
      - gospacex-net

  # MySQL 数据库
  mysql:
    image: mysql:8.0
    container_name: gospacex-mysql
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: gospacex
      TZ: Asia/Shanghai
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./scripts/init_db.sql:/docker-entrypoint-initdb.d/init.sql
    command: --default-authentication-plugin=mysql_native_password --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    networks:
      - gospacex-net

  # BFF 层
  bff_` + g.config.BFFName + `:
    build:
      context: .
      dockerfile: deploy/Dockerfile.bff
    container_name: gospacex-bff-` + g.config.BFFName + `
    environment:
      - TZ=Asia/Shanghai
    ports:
      - "8080:8080"
    depends_on:
      - etcd
      - mysql
    networks:
      - gospacex-net

`
	for _, mod := range g.config.Modules {
		dockerComposeContent += fmt.Sprintf(`  # %s 微服务
  srv_%s:
    build:
      context: .
      dockerfile: deploy/Dockerfile.%s
    container_name: gospacex-srv-%s
    environment:
      - TZ=Asia/Shanghai
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=root
      - DB_PASSWORD=root123
      - DB_NAME=gospacex
      - ETCD_ADDR=etcd:2379
    depends_on:
      - etcd
      - mysql
    networks:
      - gospacex-net

`, mod.Name, mod.Name, mod.Name, mod.Name)
	}

	dockerComposeContent += `networks:
  gospacex-net:
    driver: bridge

volumes:
  etcd_data:
  mysql_data:
`

	if err := os.WriteFile(filepath.Join(projectDir, "deploy", "docker-compose.yaml"), []byte(dockerComposeContent), 0644); err != nil {
		return err
	}

	// Dockerfile.bff
	bffDockerfile := `FROM golang:1.22-alpine AS builder

WORKDIR /build

# 安装 protoc 和 grpc 插件
RUN apk add --no-cache protobuf bash git make

# 复制源码
COPY . .

# 设置 Go 镜像代理
RUN go env -w GOPROXY=https://goproxy.cn,direct

# 编译 BFF
WORKDIR /build/bff_` + g.config.BFFName + `/cmd
RUN go build -ldflags="-s -w" -o /app/bff main.go

# 运行阶段
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 复制二进制文件
COPY --from=builder /app/bff /app/bff
COPY --from=builder /build/bff_` + g.config.BFFName + `/configs /app/configs

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8080

ENTRYPOINT ["/app/bff"]
CMD ["-config", "/app/configs/config.yaml"]
`

	if err := os.WriteFile(filepath.Join(projectDir, "deploy", "Dockerfile.bff"), []byte(bffDockerfile), 0644); err != nil {
		return err
	}

	// 微服务 Dockerfile
	for _, mod := range g.config.Modules {
		srvDockerfile := fmt.Sprintf(`FROM golang:1.22-alpine AS builder

WORKDIR /build

# 安装 protoc 和 grpc 插件
RUN apk add --no-cache protobuf bash git make

# 复制源码
COPY . .

# 设置 Go 镜像代理
RUN go env -w GOPROXY=https://goproxy.cn,direct

# 编译微服务
WORKDIR /build/srv_%s/cmd
RUN go build -ldflags="-s -w" -o /app/srv main.go

# 运行阶段
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 复制二进制文件
COPY --from=builder /app/srv /app/srv
COPY --from=builder /build/srv_%s/configs /app/configs

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE %d

ENTRYPOINT ["/app/srv"]
CMD ["-config", "/app/configs/config.yaml"]
`, mod.Name, mod.Name, mod.Port)

		if err := os.WriteFile(filepath.Join(projectDir, "deploy", fmt.Sprintf("Dockerfile.%s", mod.Name)), []byte(srvDockerfile), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (g *MicroAppGenerator) generateReadme() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n微服务架构项目\n\n## 项目结构\n\n```\n%s/\n├── common/\n│   ├── idl/\n", g.config.ProjectName, g.config.ProjectName))

	for _, mod := range g.config.Modules {
		sb.WriteString(fmt.Sprintf("│   │   └── %s.proto\n", mod.Name))
	}
	sb.WriteString(`│   ├── kitex_gen/
│   ├── errors/
│   └── constants/
├── pkg/
│   ├── config/
│   ├── database/
│   ├── logger/
│   └── utils/
`)

	sb.WriteString(fmt.Sprintf("├── bff_%s/\n", g.config.BFFName))
	sb.WriteString(`│   ├── cmd/
│   ├── configs/
│   └── internal/
│       ├── handler/
│       ├── middleware/
│       ├── rpc_client/
│       └── router/
`)

	for _, mod := range g.config.Modules {
		sb.WriteString(fmt.Sprintf("├── srv_%s/\n", mod.Name))
	}
	sb.WriteString("└── scripts/\n")
	sb.WriteString("```\n\n")
	sb.WriteString("## 快速开始\n\n")
	sb.WriteString("### 1. 初始化\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("go mod init github.com/yourorg/%s\n", g.config.ProjectName))
	sb.WriteString("go mod tidy\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### 2. 生成 Proto 代码\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("./scripts/gen_proto.sh\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### 3. 启动服务\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# BFF 层\n")
	sb.WriteString(fmt.Sprintf("cd bff_%s/cmd && go run main.go\n\n", g.config.BFFName))
	sb.WriteString("# 微服务\n")
	for _, mod := range g.config.Modules {
		sb.WriteString(fmt.Sprintf("cd srv_%s/cmd && go run main.go &\n", mod.Name))
	}

	sb.WriteString("```\n\n## API 接口\n\n| 方法 | 路径 | 描述 |\n|------|------|------|\n")
	for _, mod := range g.config.Modules {
		sb.WriteString(fmt.Sprintf("| GET | /api/v1/%ss | 列表 |\n", mod.Name))
		sb.WriteString(fmt.Sprintf("| GET | /api/v1/%ss/:id | 详情 |\n", mod.Name))
		sb.WriteString(fmt.Sprintf("| POST | /api/v1/%ss | 创建 |\n", mod.Name))
		sb.WriteString(fmt.Sprintf("| PUT | /api/v1/%ss/:id | 更新 |\n", mod.Name))
		sb.WriteString(fmt.Sprintf("| DELETE | /api/v1/%ss/:id | 删除 |\n\n", mod.Name))
	}

	sb.WriteString("## 服务端口\n\n")
	sb.WriteString(fmt.Sprintf("| 服务 | 端口 |\n|------|------|\n| bff_%s | 8080 |\n", g.config.BFFName))
	for _, mod := range g.config.Modules {
		sb.WriteString(fmt.Sprintf("| srv_%s | %d |\n", mod.Name, mod.Port))
	}

	return sb.String()
}

// Helper function for template execution
func executeTemplate(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CopyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// generateBFFMiddleware 生成 BFF 中间件
func (g *MicroAppGenerator) generateBFFMiddleware(bffName string) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	bffPath := filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName))

	// CORS 中间件
	corsContent := `package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
`
	if err := os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "cors.go"), []byte(corsContent), 0644); err != nil {
		return err
	}

	// Logger 中间件
	loggerContent := `package middleware

import (
	"fmt"
	"time"

	"` + g.config.ProjectName + `/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		logger.Info.Printf("[HTTP] %s %s %d %v %s",
			method, path, status, latency, clientIP)

		// 记录错误
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				logger.Error.Printf("[HTTP Error] %s", e.Error())
			}
		}
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}
`
	if err := os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "logger.go"), []byte(loggerContent), 0644); err != nil {
		return err
	}

	// Recovery 中间件
	recoveryContent := `package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"` + g.config.ProjectName + `/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Recovery Panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录堆栈信息
				stack := debug.Stack()
				logger.Error.Printf("[PANIC] %v\n%s", err, stack)

				// 返回错误响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Internal server error",
					"error":   fmt.Sprintf("%v", err),
				})
			}
		}()
		c.Next()
	}
}

// Timeout 超时中间件
func Timeout(timeoutSeconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里可以添加超时控制逻辑
		c.Next()
	}
}
`
	if err := os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "recovery.go"), []byte(recoveryContent), 0644); err != nil {
		return err
	}

	// Auth 中间件
	authContent := `package middleware

import (
	"net/http"
	"strings"

	"` + g.config.ProjectName + `/common/errors"
	"` + g.config.ProjectName + `/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Auth JWT 鉴权中间件
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Missing authorization header",
			})
			return
		}

		// 检查 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization format",
			})
			return
		}

		tokenString := parts[1]

		// 解析 JWT token
		claims, err := utils.ParseJWT(tokenString)
		if err != nil {
			logger.Error.Printf("[Auth] Parse JWT failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token",
				"error":   err.Error(),
			})
			return
		}

		// 将用户信息存入 Context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// GetUserID 从 Context 获取用户ID
func GetUserID(c *gin.Context) int64 {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(int64)
	}
	return 0
}

// GetUsername 从 Context 获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

// RequireRoles 角色权限检查中间件
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Access denied: no roles found",
			})
			return
		}

		userRolesList := userRoles.([]string)
		for _, role := range roles {
			for _, userRole := range userRolesList {
				if role == userRole {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Access denied: insufficient permissions",
		})
	}
}
`
	return os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "auth.go"), []byte(authContent), 0644)
}

// generateTests 生成测试文件
func (g *MicroAppGenerator) generateTests(mod *ModuleConfig, bffName string) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// 集成测试 - 服务端
	srvTestContent := fmt.Sprintf(`package integration

import (
	"context"
	"testing"

	"%s/srv_%s/internal/service"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	testDB  *gorm.DB
	testSvc *service.%sService
)

func init() {
	// 初始化测试数据库连接
	dsn := "root:123456@tcp(127.0.0.1:3306)/gospacex?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	testDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database: " + err.Error())
	}
}

func Test%sService_CRUD(t *testing.T) {
	ctx := context.Background()
	svc := service.New%sService(testDB)

	// Create
	t.Run("Create", func(t *testing.T) {
		resp, err := svc.Create(ctx, &%s.Create%sReq{Name: "Test Item"})
		if err != nil {
			t.Errorf("Create() error = %%v", err)
			return
		}
		if resp.Id == 0 {
			t.Errorf("Create() expected non-zero ID")
		}
		t.Logf("Created item with ID: %%d", resp.Id)
	})

	// Get
	t.Run("Get", func(t *testing.T) {
		resp, err := svc.Get(ctx, &%s.Get%sReq{Id: 1})
		if err != nil {
			t.Errorf("Get() error = %%v", err)
			return
		}
		t.Logf("Got item: ID=%%d, Name=%%s", resp.Id, resp.Name)
	})

	// List
	t.Run("List", func(t *testing.T) {
		resp, err := svc.List(ctx, &%s.List%sReq{Page: 1, PageSize: 10})
		if err != nil {
			t.Errorf("List() error = %%v", err)
			return
		}
		t.Logf("Listed %%d items, total: %%d", len(resp.Items), resp.Total)
	})
}
`, g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.Name, mod.ServiceName,
		mod.Name, mod.ServiceName)

	if err := os.WriteFile(filepath.Join(projectDir, "tests", "integration", mod.Name+"_test.go"), []byte(srvTestContent), 0644); err != nil {
		return err
	}

	// E2E 测试
	e2eTestContent := fmt.Sprintf(`package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"%s/bff_%s/internal/router"
)

// TestBFFAPI E2E 测试
func TestBFFAPI_%s(t *testing.T) {
	r := router.NewRouter()
	server := httptest.NewServer(r)
	defer server.Close()

	client := &http.Client{}

	// Test List
	t.Run("List", func(t *testing.T) {
		url := fmt.Sprintf("%%s/api/v1/%ss", server.URL)
		req, _ := http.NewRequest("GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %%v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %%d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		t.Logf("List response: %%v", result)
	})

	// Test Create
	t.Run("Create", func(t *testing.T) {
		url := fmt.Sprintf("%%s/api/v1/%ss", server.URL)
		body := strings.NewReader("{\"name\":\"E2E Test\"}")
		req, _ := http.NewRequest("POST", url, body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %%v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %%d", resp.StatusCode)
		}
		t.Log("Create successful")
	})
}
`, g.config.ProjectName, bffName,
		mod.Name, mod.Name, mod.Name)

	return os.WriteFile(filepath.Join(projectDir, "tests", "e2e", "api_test.go"), []byte(e2eTestContent), 0644)
}

// generateBFFHertz 生成 Hertz BFF 层
func (g *MicroAppGenerator) generateBFFHertz() error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	bffName := g.config.BFFName

	// main.go
	mainContent := fmt.Sprintf(`package main

import (
	"context"
	"flag"
	"log"

	"%s/bff_%s/internal/router"
	"%s/pkg/config"

	"github.com/cloudwego/hertz/pkg/app/server"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file")
}

func main() {
	flag.Parse()
	cfg, err := config.Load(confPath)
	if err != nil {
		log.Fatal(err)
	}

	h := server.Default(server.WithHostPorts(cfg.GetAddr()))
	router.Register(h)

	log.Printf("BFF starting on %%s", cfg.GetAddr())
	h.Run()
}
`, g.config.ProjectName, bffName, g.config.ProjectName)
	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "cmd", "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// config.yaml
	configYaml := `server:
  host: 0.0.0.0
  port: 8080

registry:
  type: direct
  addr: localhost:2379

log:
  level: info
  format: json
`
	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "configs", "config.yaml"), []byte(configYaml), 0644); err != nil {
		return err
	}

	// router.go - Hertz 路由
	var routerContent strings.Builder
	routerContent.WriteString(fmt.Sprintf(`package router

import (
	"%s/bff_%s/internal/handler"

	"github.com/cloudwego/hertz/pkg/app"
)

func Register(h *server.Hertz) {
`, g.config.ProjectName, bffName))

	for _, mod := range g.config.Modules {
		routerContent.WriteString(fmt.Sprintf(`
	// %s 模块
	v1 := h.Group("/api/v1")
	%sHandler := handler.New%sHandler()
	v1.GET("/%ss", %%sHandler.List)
	v1.GET("/%ss/:id", %%sHandler.Get)
	v1.POST("/%ss", %%sHandler.Create)
	v1.PUT("/%ss/:id", %%sHandler.Update)
	v1.DELETE("/%ss/:id", %%sHandler.Delete)
`, mod.UpperName, mod.Name,
			mod.Name, mod.Name, mod.Name, mod.Name, mod.Name))
	}
	routerContent.WriteString("}\n")

	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "internal", "router", "router.go"), []byte(routerContent.String()), 0644); err != nil {
		return err
	}

	// middleware - Hertz 版本
	if err := g.generateBFFMiddlewareHertz(bffName); err != nil {
		return err
	}

	// 生成各模块的 handler, rpc_client (Hertz 版本)
	for _, mod := range g.config.Modules {
		if err := g.generateBFFModuleHertz(&mod, bffName); err != nil {
			return err
		}
	}

	return nil
}

// generateBFFModuleHertz 生成 Hertz 版本的 BFF 模块
func (g *MicroAppGenerator) generateBFFModuleHertz(mod *ModuleConfig, bffName string) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)

	// rpc_client - 使用 gRPC 或 Kitex
	clientContent := g.generateRPCClient(mod)
	if err := os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "internal", "rpc_client", toCamelCaseFile(mod.Name)+"Client.go"), []byte(clientContent), 0644); err != nil {
		return err
	}

	// handler - Hertz 版本
	handlerContent := fmt.Sprintf(`package handler

import (
	"context"
	"strconv"

	"%s/bff_%s/internal/rpc_client"
	"%s/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type %sHandler struct {
	client *rpc_client.%sClient
}

func New%sHandler() *%sHandler {
	client, _ := rpc_client.New%sClient("localhost:%d")
	return &%sHandler{client: client}
}

func (h *%sHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Name string `+"`json:\"name\"`"+`
	}
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	resp, err := h.client.Create(ctx, req.Name)
	if err != nil {
		logger.Error.Printf("Create failed: %%v", err)
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{"id": resp.Id, "success": resp.Success})
}

func (h *%sHandler) Get(ctx context.Context, c *app.RequestContext) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	resp, err := h.client.Get(ctx, id)
	if err != nil {
		logger.Error.Printf("Get failed: %%v", err)
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{"id": resp.Id, "name": resp.Name, "status": resp.Status})
}

func (h *%sHandler) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.client.List(ctx)
	if err != nil {
		logger.Error.Printf("List failed: %%v", err)
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	items := make([]map[string]interface{}, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, map[string]interface{}{"id": item.Id, "name": item.Name, "status": item.Status})
	}
	c.JSON(consts.StatusOK, map[string]interface{}{"items": items, "total": resp.Total})
}

func (h *%sHandler) Update(ctx context.Context, c *app.RequestContext) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name string `+"`json:\"name\"`"+`
	}
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	resp, err := h.client.Update(ctx, id, req.Name)
	if err != nil {
		logger.Error.Printf("Update failed: %%v", err)
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{"success": resp.Success})
}

func (h *%sHandler) Delete(ctx context.Context, c *app.RequestContext) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	resp, err := h.client.Delete(ctx, id)
	if err != nil {
		logger.Error.Printf("Delete failed: %%v", err)
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{"success": resp.Success})
}
`, g.config.ProjectName, bffName, g.config.ProjectName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.Port, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName)

	return os.WriteFile(filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName), "internal", "handler", toCamelCaseFile(mod.Name)+"Handler.go"), []byte(handlerContent), 0644)
}

// generateRPCClient 根据 Protocol 生成 RPC 客户端代码
func (g *MicroAppGenerator) generateRPCClient(mod *ModuleConfig) string {
	if g.config.Protocol == "kitex" {
		return g.generateKitexClient(mod)
	}
	// 默认使用 gRPC
	return g.generateGRPCClient(mod)
}

// generateGRPCClient 生成 gRPC 客户端代码
func (g *MicroAppGenerator) generateGRPCClient(mod *ModuleConfig) string {
	return fmt.Sprintf(`package rpc_client

import (
	"context"
	"fmt"
	"time"

	"%s/common/kitex_gen/%s"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type %sClient struct {
	conn   *grpc.ClientConn
	client %s.%sServiceClient
}

func New%sClient(addr string) (*%sClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %%w", err)
	}

	return &%sClient{
		conn:   conn,
		client: %s.New%sServiceClient(conn),
	}, nil
}

func (c *%sClient) Close() error {
	return c.conn.Close()
}

func (c *%sClient) Create(ctx context.Context, name string) (*%s.Create%sResp, error) {
	return c.client.Create%s(ctx, &%s.Create%sReq{Name: name})
}

func (c *%sClient) Get(ctx context.Context, id int64) (*%s.Get%sResp, error) {
	return c.client.Get%s(ctx, &%s.Get%sReq{Id: id})
}

func (c *%sClient) List(ctx context.Context) (*%s.List%sResp, error) {
	return c.client.List%s(ctx, &%s.List%sReq{})
}

func (c *%sClient) Update(ctx context.Context, id int64, name string) (*%s.Update%sResp, error) {
	return c.client.Update%s(ctx, &%s.Update%sReq{Id: id, Name: name})
}

func (c *%sClient) Delete(ctx context.Context, id int64) (*%s.Delete%sResp, error) {
	return c.client.Delete%s(ctx, &%s.Delete%sReq{Id: id})
}
`, g.config.ProjectName, mod.Name,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName,
		mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName)
}

// generateKitexClient 生成 Kitex 客户端代码
func (g *MicroAppGenerator) generateKitexClient(mod *ModuleConfig) string {
	return fmt.Sprintf(`package rpc_client

import (
	"context"

	"%s/common/kitex_gen/%s"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
)

var %sClientInstance *%sClient

func init() {
	// Kitex 客户端初始化
	cli, err := New%sClient("localhost:%d",
		client.WithHostPorts("localhost:%d"),
		client.WithMiddleware(),
	)
	if err != nil {
		panic(err)
	}
	%sClientInstance = cli
}

type %sClient struct {
	kitexClient %s.%sServiceClient
}

func New%sClient(addr string, opts ...client.Option) (*%sClient, error) {
	opts = append(opts,
		client.WithTransportType(transmeta.HTTP2),
	)

	cli, err := %s.NewClient("%sService",
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return &%sClient{
		kitexClient: cli,
	}, nil
}

func (c *%sClient) Create(ctx context.Context, name string) (*%s.Create%sResp, error) {
	return c.kitexClient.Create%s(ctx, &%s.Create%sReq{Name: name})
}

func (c *%sClient) Get(ctx context.Context, id int64) (*%s.Get%sResp, error) {
	return c.kitexClient.Get%s(ctx, &%s.Get%sReq{Id: id})
}

func (c *%sClient) List(ctx context.Context) (*%s.List%sResp, error) {
	return c.kitexClient.List%s(ctx, &%s.List%sReq{})
}

func (c *%sClient) Update(ctx context.Context, id int64, name string) (*%s.Update%sResp, error) {
	return c.kitexClient.Update%s(ctx, &%s.Update%sReq{Id: id, Name: name})
}

func (c *%sClient) Delete(ctx context.Context, id int64) (*%s.Delete%sResp, error) {
	return c.kitexClient.Delete%s(ctx, &%s.Delete%sReq{Id: id})
}
`, g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Port, mod.Port,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName,
		mod.Name, mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName,
		mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName, mod.Name, mod.ServiceName)
}

// generateBFFMiddlewareHertz 生成 Hertz 中间件
func (g *MicroAppGenerator) generateBFFMiddlewareHertz(bffName string) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	bffPath := filepath.Join(projectDir, fmt.Sprintf("bff_%s", bffName))

	// CORS 中间件
	corsContent := `package middleware

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/cors"
)

// CORS 跨域中间件
func CORS() app.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
	})
}
`
	if err := os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "cors.go"), []byte(corsContent), 0644); err != nil {
		return err
	}

	// Logger 中间件
	loggerContent := `package middleware

import (
	"time"

	"` + g.config.ProjectName + `/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// Logger 请求日志中间件
func Logger() app.HandlerFunc {
	return func(ctx app.RequestContext) {
		start := time.Now()

		// 处理请求
		ctx.Next()

		// 记录日志
		latency := time.Since(start)
		status := ctx.Response.StatusCode()
		clientIP := ctx.ClientIP()
		method := string(ctx.Request.Method())
		path := string(ctx.Request.URI().Path())

		hlog.CtxInfof(ctx.Request.Context(), "[HTTP] %s %s %d %v %s",
			method, path, status, latency, clientIP)
	}
}
`
	if err := os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "logger.go"), []byte(loggerContent), 0644); err != nil {
		return err
	}

	// Recovery 中间件
	recoveryContent := `package middleware

import (
	"fmt"
	"runtime/debug"

	"` + g.config.ProjectName + `/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Recovery Panic 恢复中间件
func Recovery() app.HandlerFunc {
	return func(ctx app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				// 记录堆栈信息
				stack := debug.Stack()
				logger.Error.Printf("[PANIC] %v\n%s", err, stack)

				// 返回错误响应
				ctx.JSON(consts.StatusInternalServerError, map[string]interface{}{
					"code":    500,
					"message": "Internal server error",
					"error":   fmt.Sprintf("%v", err),
				})
			}
		}()
		ctx.Next()
	}
}
`
	return os.WriteFile(filepath.Join(bffPath, "internal", "middleware", "recovery.go"), []byte(recoveryContent), 0644)
}

// generateKitexMicroService 生成 Kitex 微服务
func (g *MicroAppGenerator) generateKitexMicroService(mod *ModuleConfig) error {
	projectDir := filepath.Join(g.outputDir, g.config.ProjectName)
	srvDir := filepath.Join(projectDir, toCamelCaseDir(mod.Name))

	// main.go - Kitex 服务端
	mainContent := fmt.Sprintf(`package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"%s/srv_%s/internal/handler"
	"%s/pkg/config"
	"%s/pkg/logger"
	"%s/pkg/database"

	"github.com/cloudwego/kitex/pkg/utils/kitexutil"
	"github.com/cloudwego/kitex/server"
	kitex_gen "%s/common/kitex_gen/%s"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file path")
}

func main() {
	flag.Parse()
	cfg, err := config.Load(confPath)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化数据库
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		logger.Error.Fatalf("Failed to connect database: %%v", err)
	}

	// 创建 Kitex 服务
	svr := %s.NewServer(
		new(handler.%sHandler),
		server.WithServiceAddr(&net.TCPAddr{
			IP:   net.ParseIP(cfg.Server.Host),
			Port: cfg.Server.Port,
		}),
	)

	logger.Info.Printf("%%s service starting on %%s", "%s", cfg.GetAddr())
	if err := svr.Run(); err != nil {
		log.Fatalf("%%s service run failed: %%v", err)
	}
}
`, g.config.ProjectName, mod.Name,
		g.config.ProjectName, g.config.ProjectName,
		g.config.ProjectName, g.config.ProjectName,
		g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName,
		mod.UpperName)
	if err := os.WriteFile(filepath.Join(srvDir, "cmd", "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// config.yaml
	configYaml := fmt.Sprintf(`server:
  host: 0.0.0.0
  port: %d

database:
  host: %%s
  port: %%s
  user: %%s
  password: %%s
  database: %%s

registry:
  type: direct
  addr: localhost:2379

log:
  level: info
  format: json
`, mod.Port)
	if err := os.WriteFile(filepath.Join(srvDir, "configs", "config.yaml"), []byte(configYaml), 0644); err != nil {
		return err
	}

	// handler - Kitex 版本
	handlerContent := fmt.Sprintf(`package handler

import (
	"context"

	"%s/srv_%s/internal/model"
	"%s/srv_%s/internal/repository"
	"%s/srv_%s/internal/service"
	"%s/common/kitex_gen/%s"
)

type %sHandler struct {
	repo *repository.%sRepository
	svc  *service.%sService
}

func New%sHandler() *%sHandler {
	return &%sHandler{}
}

func (h *%sHandler) Create(ctx context.Context, req *%s.Create%sReq) (*%s.Create%sResp, error) {
	id, err := h.svc.Create(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &%s.Create%sResp{Id: id, Success: true}, nil
}

func (h *%sHandler) Get(ctx context.Context, req *%s.Get%sReq) (*%s.Get%sResp, error) {
	item, err := h.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &%s.Get%sResp{Id: item.Id, Name: item.Name, Status: item.Status}, nil
}

func (h *%sHandler) List(ctx context.Context, req *%s.List%sReq) (*%s.List%sResp, error) {
	items, total, err := h.svc.List(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	
	respItems := make([]*%s.%sItem, len(items))
	for i, item := range items {
		respItems[i] = &%s.%sItem{Id: item.Id, Name: item.Name, Status: item.Status}
	}
	return &%s.List%sResp{Items: respItems, Total: total}, nil
}

func (h *%sHandler) Update(ctx context.Context, req *%s.Update%sReq) (*%s.Update%sResp, error) {
	err := h.svc.Update(ctx, req.Id, req.Name)
	if err != nil {
		return &%s.Update%sResp{Success: false}, err
	}
	return &%s.Update%sResp{Success: true}, nil
}

func (h *%sHandler) Delete(ctx context.Context, req *%s.Delete%sReq) (*%s.Delete%sResp, error) {
	err := h.svc.Delete(ctx, req.Id)
	if err != nil {
		return &%s.Delete%sResp{Success: false}, err
	}
	return &%s.Delete%sResp{Success: true}, nil
}
`, g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName,
		mod.Name,
		mod.ServiceName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName,
		mod.ServiceName, mod.Name, mod.ServiceName)
	if err := os.WriteFile(filepath.Join(srvDir, "internal", "handler", toCamelCaseFile(mod.Name)+"Handler.go"), []byte(handlerContent), 0644); err != nil {
		return err
	}

	// model
	modelContent := fmt.Sprintf(`package model

type %s struct {
	Id        int64  `+"`" + `gorm:"primaryKey;autoIncrement" json:"id"` + "`" + `
	Name      string `+"`" + `gorm:"size:255;not null" json:"name"` + "`" + `
	Status    int32  `+"`" + `gorm:"default:0" json:"status"` + "`" + `
}

func (%s) TableName() string {
	return "%s"
}
`, mod.UpperName, mod.UpperName, mod.TableName)
	if err := os.WriteFile(filepath.Join(srvDir, "internal", "model", "model.go"), []byte(modelContent), 0644); err != nil {
		return err
	}

	// repository
	repoContent := fmt.Sprintf(`package repository

import (
	"context"
	"errors"

	"%s/srv_%s/internal/model"
	"gorm.io/gorm"
)

type %sRepository struct {
	db *gorm.DB
}

func New%sRepository(db interface{}) *%sRepository {
	return &%sRepository{db: db.(*gorm.DB)}
}

func (r *%sRepository) Create(ctx context.Context, name string) (int64, error) {
	item := &model.%s{Name: name, Status: 0}
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return 0, err
	}
	return item.Id, nil
}

func (r *%sRepository) GetById(ctx context.Context, id int64) (*model.%s, error) {
	var item model.%s
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &item, nil
}

func (r *%sRepository) List(ctx context.Context, page, pageSize int) ([]*model.%s, int64, error) {
	var items []*model.%s
	var total int64
	
	r.db.WithContext(ctx).Model(&model.%s{}).Count(&total)
	
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *%sRepository) Update(ctx context.Context, id int64, name string) error {
	return r.db.WithContext(ctx).Model(&model.%s{}).Where("id = ?", id).Update("name", name).Error
}

func (r *%sRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.%s{}, id).Error
}
`, g.config.ProjectName, mod.Name,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName)
	if err := os.WriteFile(filepath.Join(srvDir, "internal", "repository", toCamelCaseFile(mod.Name)+"Repo.go"), []byte(repoContent), 0644); err != nil {
		return err
	}

	// service
	svcContent := fmt.Sprintf(`package service

import (
	"context"

	"%s/srv_%s/internal/model"
	"%s/srv_%s/internal/repository"
	"%s/common/errors"
)

type %sService struct {
	repo *repository.%sRepository
}

func New%sService(repo *repository.%sRepository) *%sService {
	return &%sService{repo: repo}
}

func (s *%sService) Create(ctx context.Context, name string) (int64, error) {
	if name == "" {
		return 0, errors.ErrInvalidParam
	}
	return s.repo.Create(ctx, name)
}

func (s *%sService) Get(ctx context.Context, id int64) (*model.%s, error) {
	return s.repo.GetById(ctx, id)
}

func (s *%sService) List(ctx context.Context, page, pageSize int) ([]*model.%s, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.List(ctx, page, pageSize)
}

func (s *%sService) Update(ctx context.Context, id int64, name string) error {
	if id == 0 || name == "" {
		return errors.ErrInvalidParam
	}
	return s.repo.Update(ctx, id, name)
}

func (s *%sService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.ErrInvalidParam
	}
	return s.repo.Delete(ctx, id)
}
`, g.config.ProjectName, mod.Name,
		g.config.ProjectName, mod.Name,
		g.config.ProjectName,
		mod.ServiceName,
		mod.ServiceName, mod.ServiceName, mod.ServiceName,
		mod.ServiceName,
		mod.ServiceName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName,
		mod.UpperName)
	return os.WriteFile(filepath.Join(srvDir, "internal", "service", toCamelCaseFile(mod.Name)+"Service.go"), []byte(svcContent), 0644)
}

// checkProtoTools 检查 protoc 和相关插件是否已安装
func checkProtoTools() bool {
	tools := []struct {
		name    string
		checkCmd string
	}{
		{"protoc", "which protoc"},
		{"protoc-gen-go", "which protoc-gen-go"},
		{"protoc-gen-go-grpc", "which protoc-gen-go-grpc"},
	}

	allInstalled := true
	for _, tool := range tools {
		cmd := exec.Command("sh", "-c", tool.checkCmd)
		if err := cmd.Run(); err != nil {
			log.Printf("⚠️ %s not found, please install it", tool.name)
			allInstalled = false
		}
	}
	return allInstalled
}

func toCamelCaseDir(tableName string) string {
	prefixes := []string{"eb_", "t_", "sys_", "tb_", "bc_"}
	name := tableName
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	parts := strings.Split(name, "_")
	result := "srv"
	for _, part := range parts {
		if part == "" {
			continue
		}
		result += strings.ToUpper(part[:1]) + part[1:]
	}
	return result
}

func toCamelCaseFile(tableName string) string {
	prefixes := []string{"eb_", "t_", "sys_", "tb_", "bc_"}
	name := tableName
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	parts := strings.Split(name, "_")
	result := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if result == "" {
			result += strings.ToLower(part[:1]) + part[1:]
		} else {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}

func toUpperCamelCase(tableName string) string {
	parts := strings.Split(tableName, "_")
	result := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		result += strings.ToUpper(part[:1]) + part[1:]
	}
	return result
}
