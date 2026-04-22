package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospacex/gpx/internal/config"
)

// ScriptCenterGenerator 脚本中心生成器
type ScriptCenterGenerator struct {
	config *config.ProjectConfig
}

// NewScriptCenterGenerator 创建脚本中心生成器
func NewScriptCenterGenerator(cfg *config.ProjectConfig) *ScriptCenterGenerator {
	return &ScriptCenterGenerator{
		config: cfg,
	}
}

// Generate 生成脚本中心项目（支持动态数据库选择）
func (g *ScriptCenterGenerator) Generate(outputDir string) error {
	// 创建项目目录结构（参考目标项目 my-scripts）
	dirs := []string{
		"cmd/commands",
		"internal/handler",
		"internal/service",
		"internal/model",
		"pkg/config",
		"pkg/database",
		"pkg/logger",
		"configs",
		"deploy/supervisor",
		"deploy/systemd",
		"scripts",
		"logs",
		"tests",
		"tmp/nacos/cache",
		"tmp/nacos/log",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(outputDir, dir), 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// 生成基础文件（参考目标项目 my-scripts 结构）
	files := map[string]func() (string, error){
		"cmd/main.go":           g.generateMain,
		"cmd/commands/root.go":  g.generateRootCmd,
		"cmd/commands/start.go": func() (string, error) { return g.generateStartCmdNew() },
		"pkg/config/config.go":  g.generatePkgConfig,
		"pkg/config/types.go":   g.generateConfigTypes,
		"pkg/logger/logger.go":  g.generatePkgLogger,
		"configs/config.yaml":   g.generateConfig,

		"deploy/supervisor/app.conf":     g.generateSupervisorConfig,
		"deploy/systemd/service.service": g.generateSystemdService,
		"scripts/run.sh":                 g.generateRunScript,
		"scripts/build.sh":               g.generateBuildScript,
		"Makefile":                       g.generateMakefile,
		"readme.md":                      g.generateREADME,
		".gitignore":                     g.generateGitignore,
	}

	// 写入基础文件
	for path, generator := range files {
		content, err := generator()
		if err != nil {
			return fmt.Errorf("generate %s: %w", path, err)
		}

		// 动态替换项目名称
		projectName := g.getProjectName()
		content = strings.ReplaceAll(content, "github.com/gospacex/gpx-scripts", projectName)

		fullPath := filepath.Join(outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	// 生成数据库文件（根据 config.DB 动态选择）
	if err := g.generateDBFiles(outputDir); err != nil {
		return fmt.Errorf("generate database files: %w", err)
	}

	// 生成中间件文件（根据 config.MQ 动态选择）
	// 生成中间件文件（根据 config.MQ 动态选择）
	if err := g.generateMQFiles(outputDir); err != nil {
		return fmt.Errorf("generate middleware files: %w", err)
	}

	// 生成 internal 下的 producer/consumer（根据 config.MQ）
	if err := g.generateMQInternalFiles(outputDir); err != nil {
		return fmt.Errorf("generate mq internal files: %w", err)
	}

	// 生成 MQ CLI 命令（根据 config.MQ）
	if err := g.generateMQCommands(outputDir); err != nil {
		return fmt.Errorf("generate mq commands: %w", err)
	}

	// 生成 go.mod（根据 config.DB 更新依赖）
	goModContent := g.updateGoModForDB("")
	goModContent = g.updateGoModForMQ(goModContent)
	fullPath := filepath.Join(outputDir, "go.mod")
	if err := os.WriteFile(fullPath, []byte(goModContent), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	// 如果指定了 MySQL 表，生成 CRUD 代码和对应的 CLI 子命令
	if g.config.MySQLTable != "" {
		// 使用 CRUD 生成器从 MySQL 表结构生成分层 CRUD 代码
		// 脚本中心只需要 CLI handler，不生成 HTTP handler
		crudGen := NewCRUDGeneratorWithHandlerType(
			g.config.MySQLHost,
			g.config.MySQLPort,
			g.config.MySQLUser,
			g.config.MySQLPassword,
			g.config.MySQLDatabase,
			g.config.MySQLTable,
			outputDir,
			g.getProjectName(),
			"cli", // 脚本中心只需要 CLI handler
		)
		if err := crudGen.Generate(); err != nil {
			return fmt.Errorf("generate CRUD code: %w", err)
		}
		fmt.Printf("✓ Generated CRUD code for table %s\n", g.config.MySQLTable)

		// 生成对应的 CLI 子命令 (create/get/list/update/delete)
		// 任务调度由外部 gocron 平台管理，这里只提供 CLI 命令
		if err := g.generateCRUDCommands(outputDir); err != nil {
			return fmt.Errorf("generate CRUD CLI commands: %w", err)
		}
		fmt.Printf("✓ Generated CRUD CLI commands for table %s\n", g.config.MySQLTable)
	}

	return nil
}

// generateMain 生成 Main 函数
func (g *ScriptCenterGenerator) generateMain() (string, error) {
	return `package main

import (
	"log"
	"os"

	"github.com/gospacex/gpx-scripts/cmd/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	os.Exit(0)
}
`, nil
}

// generateRootCmd 生成 Root 命令
func (g *ScriptCenterGenerator) generateRootCmd() (string, error) {
	return `package commands

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "gpx-scripts",
		Short: "脚本中心 - 基于 Cobra CLI 的任务调度平台",
		Long: ` + "`" + `脚本中心是一个基于 Cobra CLI 的任务调度平台。

任务调度由外部 gocron 平台 (https://github.com/ouqiang/gocron) 管理。
本脚手架提供:
- 数据库连接管理 (MySQL, Redis)
- Repository/Service 分层架构
- Cobra CLI 命令框架
- supervisor/systemd 部署配置` + "`" + `,
	}
)

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认：configs/config.yaml)")
}

func initConfig() {
	// 初始化配置逻辑
	_ = cfgFile
}
`, nil
}

// generateStartCmd 生成 Start 命令
func (g *ScriptCenterGenerator) generateStartCmdOld() (string, error) {
	return `package commands

import (
	"strings"
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/gospacex/gpx-scripts/internal/dal/mysql"
	"github.com/gospacex/gpx-scripts/internal/dal/redis"
	"github.com/spf13/cobra"
)

// startCmd 启动命令
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动任务调度服务",
	Long:  ` + "`" + `启动脚本中心服务，初始化数据库连接并加载任务` + "`" + `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 初始化数据库连接
		mysql.Init()
		redis.Init()

		log.Println("✓ Services initialized")

		// 等待退出信号
		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		<-ctx.Done()
		log.Println("Shutting down...")

		// 关闭连接
		redis.Close()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
`, nil
}

// generateExampleCmd 生成 Example Cobra 命令 (Handler 层)
func (g *ScriptCenterGenerator) generateExampleCmd() (string, error) {
	return `package commands

import (
	"strings"
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/gospacex/gpx-scripts/internal/model"
	"github.com/gospacex/gpx-scripts/internal/service"
	"github.com/spf13/cobra"
)

var (
	exampleName string
	exampleData string
)

// exampleCmd Example 相关命令
var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Example 管理命令",
	Long:  ` + "`" + `Example 管理相关命令集合` + "`" + `,
}

// exampleCreateCmd 创建 Example
var exampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建 Example",
	Long:  ` + "`" + `创建一个新的 Example 记录` + "`" + `,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		svc := service.NewExampleService()

		entity := &model.Example{
			Name: exampleName,
			Data: exampleData,
		}

		if err := svc.Create(ctx, entity); err != nil {
			return fmt.Errorf("create example failed: %w", err)
		}

		log.Printf("✓ Example created with ID: %d", entity.ID)
		return nil
	},
}

// exampleGetCmd 查询 Example
var exampleGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "查询 Example",
	Long:  ` + "`" + `根据 ID 查询 Example 记录` + "`" + `,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		svc := service.NewExampleService()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		// 使用缓存查询
		entity, err := svc.GetByIDWithCache(ctx, id)
		if err != nil {
			return fmt.Errorf("get example failed: %w", err)
		}

		log.Printf("Example: ID=%d, Name=%s, Data=%s", entity.ID, entity.Name, entity.Data)
		return nil
	},
}

// exampleListCmd 列表查询 Example
var exampleListCmd = &cobra.Command{
	Use:   "list",
	Short: "列表查询 Example",
	Long:  ` + "`" + `分页查询 Example 记录` + "`" + `,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		svc := service.NewExampleService()

		page := 1
		pageSize := 10

		entities, total, err := svc.List(ctx, page, pageSize)
		if err != nil {
			return fmt.Errorf("list examples failed: %w", err)
		}

		log.Printf("Total: %d, Page: %d, PageSize: %d", total, page, pageSize)
		for _, e := range entities {
			log.Printf("  - ID=%d, Name=%s", e.ID, e.Name)
		}
		return nil
	},
}

// exampleDeleteCmd 删除 Example
var exampleDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "删除 Example",
	Long:  ` + "`" + `根据 ID 删除 Example 记录` + "`" + `,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		svc := service.NewExampleService()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		if err := svc.Delete(ctx, id); err != nil {
			return fmt.Errorf("delete example failed: %w", err)
		}

		log.Printf("✓ Example deleted: %d", id)
		return nil
	},
}

func init() {
	// 注册子命令
	exampleCmd.AddCommand(exampleCreateCmd)
	exampleCmd.AddCommand(exampleGetCmd)
	exampleCmd.AddCommand(exampleListCmd)
	exampleCmd.AddCommand(exampleDeleteCmd)

	// 添加标志
	exampleCreateCmd.Flags().StringVar(&exampleName, "name", "", "Example 名称 (required)")
	exampleCreateCmd.Flags().StringVar(&exampleData, "data", "", "Example 数据")
	exampleCreateCmd.MarkFlagRequired("name")
}
`, nil
}

// generateExampleModel 生成 Example Model
func (g *ScriptCenterGenerator) generateExampleModel() (string, error) {
	return `package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Example 示例模型
type Example struct {
	ID        int64          ` + "`" + `gorm:"primaryKey;autoIncrement"` + "`" + `
	CreatedAt time.Time      ` + "`" + `gorm:"autoCreateTime"` + "`" + `
	UpdatedAt time.Time      ` + "`" + `gorm:"autoUpdateTime"` + "`" + `
	DeletedAt gorm.DeletedAt ` + "`" + `gorm:"index"` + "`" + `
	Name      string         ` + "`" + `gorm:"size:255"` + "`" + `
	Data      string         ` + "`" + `gorm:"type:text"` + "`" + `
}

// TableName specifies table name
func (Example) TableName() string {
	return "examples"
}
`, nil
}

// generateMySQLInit 生成 MySQL 初始化
func (g *ScriptCenterGenerator) generateMySQLInit() (string, error) {
	return `package mysql

import (
	"strings"
	"fmt"
	"os"

	"github.com/gospacex/gpx-scripts/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB  *gorm.DB
	err error
)

// Init initializes MySQL connection
func Init() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true&loc=Local",
			getEnv("MYSQL_USER", "root"),
			getEnv("MYSQL_PASSWORD", "password"),
			getEnv("MYSQL_HOST", "localhost"),
			getEnv("MYSQL_DATABASE", "gpx_scripts"),
		)
	}

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MySQL: %v", err))
	}

	// Auto migrate models
	if os.Getenv("GO_ENV") != "online" {
		DB.AutoMigrate(&model.Example{})
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
`, nil
}

// generateRedisInit 生成 Redis 初始化
func (g *ScriptCenterGenerator) generateRedisInit() (string, error) {
	return `package redis

import (
	"strings"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
)

// Init initializes Redis connection
func Init() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	password := os.Getenv("REDIS_PASSWORD")
	db := 0

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
}

// Close closes Redis connection
func Close() {
	if RedisClient != nil {
		RedisClient.Close()
	}
}
`, nil
}

// generateExampleRepo 生成 Example Repository
func (g *ScriptCenterGenerator) generateExampleRepo() (string, error) {
	return `package repository

import (
	"strings"
	"context"
	"fmt"

	"github.com/gospacex/gpx-scripts/internal/dal/mysql"
	"github.com/gospacex/gpx-scripts/internal/dal/redis"
	"github.com/gospacex/gpx-scripts/internal/model"
	"gorm.io/gorm"
)

// ExampleRepository Example 仓储
type ExampleRepository struct {
	db *gorm.DB
}

// NewExampleRepository creates new repository
func NewExampleRepository() *ExampleRepository {
	return &ExampleRepository{
		db: mysql.DB,
	}
}

// Create creates a new example
func (r *ExampleRepository) Create(ctx context.Context, entity *model.Example) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// GetByID gets example by ID
func (r *ExampleRepository) GetByID(ctx context.Context, id int64) (*model.Example, error) {
	var entity model.Example
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("example not found")
		}
		return nil, err
	}
	return &entity, nil
}

// GetByIDWithCache gets example by ID with Redis cache
func (r *ExampleRepository) GetByIDWithCache(ctx context.Context, id int64) (*model.Example, error) {
	// Try Redis cache first
	cacheKey := fmt.Sprintf("example:%d", id)
	_, err := redis.RedisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		// Cache hit - return from DB (simplified, should parse from cache)
		return r.GetByID(ctx, id)
	}

	// Cache miss - get from DB
	entity, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Write to cache (simplified - should use proper serialization)
	_ = cacheKey // placeholder for cache write

	return entity, nil
}

// List lists examples
func (r *ExampleRepository) List(ctx context.Context, offset, limit int) ([]*model.Example, error) {
	var entities []*model.Example
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&entities).Error
	return entities, err
}

// Update updates an example
func (r *ExampleRepository) Update(ctx context.Context, entity *model.Example) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete deletes an example
func (r *ExampleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Example{}, id).Error
}

// Count counts examples
func (r *ExampleRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Example{}).Count(&count).Error
	return count, err
}
`, nil
}

// generateExampleService 生成 Example Service
func (g *ScriptCenterGenerator) generateExampleService() (string, error) {
	return `package service

import (
	"strings"
	"context"

	"github.com/gospacex/gpx-scripts/internal/model"
	"github.com/gospacex/gpx-scripts/internal/repository"
)

// ExampleService Example 服务
type ExampleService struct {
	repo *repository.ExampleRepository
}

// NewExampleService creates new service
func NewExampleService() *ExampleService {
	return &ExampleService{
		repo: repository.NewExampleRepository(),
	}
}

// Create creates a new example
func (s *ExampleService) Create(ctx context.Context, entity *model.Example) error {
	return s.repo.Create(ctx, entity)
}

// GetByID gets example by ID
func (s *ExampleService) GetByID(ctx context.Context, id int64) (*model.Example, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByIDWithCache gets example by ID with cache
func (s *ExampleService) GetByIDWithCache(ctx context.Context, id int64) (*model.Example, error) {
	return s.repo.GetByIDWithCache(ctx, id)
}

// List lists examples
func (s *ExampleService) List(ctx context.Context, page, pageSize int) ([]*model.Example, int64, error) {
	offset := (page - 1) * pageSize
	entities, err := s.repo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return entities, count, nil
}

// Update updates an example
func (s *ExampleService) Update(ctx context.Context, entity *model.Example) error {
	return s.repo.Update(ctx, entity)
}

// Delete deletes an example
func (s *ExampleService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
`, nil
}

// generateInternalConfig 生成内部配置
func (g *ScriptCenterGenerator) generateInternalConfig() (string, error) {
	return `package config

// Config 配置结构
type Config struct {
	App      AppConfig     ` + "`" + `yaml:"app"` + "`" + `
	MySQL    MySQLConfig   ` + "`" + `yaml:"mysql"` + "`" + `
	Redis    RedisConfig   ` + "`" + `yaml:"redis"` + "`" + `
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string ` + "`" + `yaml:"name"` + "`" + `
	Environment string ` + "`" + `yaml:"environment"` + "`" + `
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	User     string ` + "`" + `yaml:"user"` + "`" + `
	Password string ` + "`" + `yaml:"password"` + "`" + `
	Host     string ` + "`" + `yaml:"host"` + "`" + `
	Database string ` + "`" + `yaml:"database"` + "`" + `
	Port     int    ` + "`" + `yaml:"port"` + "`" + `
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string ` + "`" + `yaml:"addr"` + "`" + `
	Password string ` + "`" + `yaml:"password"` + "`" + `
	DB       int    ` + "`" + `yaml:"db"` + "`" + `
}
`, nil
}

// generateLogger 生成 Logger
func (g *ScriptCenterGenerator) generateLogger() (string, error) {
	return `package logger

import (
	"strings"
	"log"
	"os"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "[SCRIPT] ", log.LstdFlags|log.Lshortfile)
}

// Info 输出信息日志
func Info(format string, v ...interface{}) {
	logger.Printf(format, v...)
}

// Error 输出错误日志
func Error(format string, v ...interface{}) {
	logger.Printf("ERROR: "+format, v...)
}
`, nil
}

// generateConfig 生成配置文件
func (g *ScriptCenterGenerator) generateConfig() (string, error) {
	mysqlHost := "127.0.0.1"
	mysqlPort := "3306"
	mysqlUser := "root"
	mysqlPassword := "your_password"
	mysqlDatabase := "gpx_scripts"

	if g.config.MySQLHost != "" {
		mysqlHost = g.config.MySQLHost
	}
	if g.config.MySQLPort != "" {
		mysqlPort = g.config.MySQLPort
	}
	if g.config.MySQLUser != "" {
		mysqlUser = g.config.MySQLUser
	}
	if g.config.MySQLPassword != "" {
		mysqlPassword = g.config.MySQLPassword
	}
	if g.config.MySQLDatabase != "" {
		mysqlDatabase = g.config.MySQLDatabase
	}

	return `# 脚本中心配置文件

app:
  name: gpx-scripts
  environment: development

# 数据库配置 - 支持多数据库同时连接
database:
  # 启用的数据库类型
  types: [mysql, redis]
  
  # MySQL 配置
  mysql:
    host: ` + mysqlHost + `
    port: ` + mysqlPort + `
    user: ` + mysqlUser + `
    password: ` + mysqlPassword + `
    database: ` + mysqlDatabase + `
    max_open_conns: 100
    max_idle_conns: 10
    conn_max_lifetime: 3600
    table: ""  # CRUD 生成的表名
  
  # PostgreSQL 配置（可选）
  # postgresql:
  #   host: 127.0.0.1
  #   port: 5432
  #   user: postgres
  #   password: your_password
  #   database: gpx_scripts
  #   schema: public
  #   max_open_conns: 100
  #   max_idle_conns: 10
  
  # Redis 配置
  redis:
    addr: localhost:6379
    password: ""
    db: 0
    pool_size: 100
    min_idle_conns: 5
  
  # MongoDB 配置（可选）
  # mongodb:
  #   uri: mongodb://localhost:27017
  #   database: gpx_scripts
  #   max_pool_size: 100
  #   min_pool_size: 10
  
  # Elasticsearch 配置（可选）
  # elasticsearch:
  #   addresses: ["http://localhost:9200"]
  #   username: ""
  #   password: ""
  #   index: gpx_scripts
  #   sniff: false
  #   # MySQL to ES sync configuration
  #   sync_from_mysql: false
  #   mysql_host: ` + mysqlHost + `
  #   mysql_port: ` + mysqlPort + `
  #   mysql_user: ` + mysqlUser + `
  #   mysql_password: ` + mysqlPassword + `
  #   mysql_database: ` + mysqlDatabase + `
  #   mysql_table: ""

# 兼容旧格式（从 nacos 读取时使用）
mysql:
  host: ` + mysqlHost + `
  port: ` + mysqlPort + `
  user: ` + mysqlUser + `
  password: ` + mysqlPassword + `
  database: ` + mysqlDatabase + `

# Redis 配置（兼容旧格式）
redis:
  addr: localhost:6379
  password: ""
  db: 0

# Elasticsearch 配置（兼容旧格式）
elasticsearch:
  addr: http://localhost:9200
  username: ""
  password: ""

# MongoDB 配置（兼容旧格式）
mongodb:
  uri: mongodb://localhost:27017
  database: gpx_scripts

# Kafka 配置（消息队列）
kafka:
  enabled: true
  brokers:
    - localhost:9092
  topic: order-events
  consumer_group: gpx-scripts-group

# RabbitMQ 配置（可选）
rabbitmq:
  enabled: false
  addr: localhost:5672
  vhost: /
  username: guest
  password: guest
  exchange: order-exchange
  queue: order-queue

# RocketMQ 配置（可选）
rocketmq:
  enabled: false
  name_server: localhost:9876
  topic: order-topic
  consumer_group: order-consumer-group
  producer_group: order-producer-group

# 日志配置
log:
  level: info
  format: json
  output: stdout
  path: logs/app.log
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true
`, nil
}

// generateSupervisorConfig 生成 supervisor 配置
func (g *ScriptCenterGenerator) generateSupervisorConfig() (string, error) {
	return `[program:gpx-scripts]
command=/opt/gpx-scripts/gpx-scripts start
directory=/opt/gpx-scripts
autostart=true
autorestart=true
stderr_logfile=/var/log/gpx-scripts.err.log
stdout_logfile=/var/log/gpx-scripts.out.log
user=www-data
`, nil
}

// generateSystemdService 生成 systemd 服务配置
func (g *ScriptCenterGenerator) generateSystemdService() (string, error) {
	return `[Unit]
Description=GPX Scripts - Task Scheduler
After=network.target mysql.service redis.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/gpx-scripts
ExecStart=/opt/gpx-scripts/gpx-scripts start
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, nil
}

// generateRunScript 生成运行脚本
func (g *ScriptCenterGenerator) generateRunScript() (string, error) {
	return `#!/bin/bash
set -e
echo "Starting gpx-scripts..."
go run cmd/main.go start
`, nil
}

// generateBuildScript 生成构建脚本
func (g *ScriptCenterGenerator) generateBuildScript() (string, error) {
	return `#!/bin/bash
set -e
echo "Building gpx-scripts..."
go build -o bin/gpx-scripts ./cmd/main.go
echo "✓ Build complete: bin/gpx-scripts"
`, nil
}

// generateMakefile 生成 Makefile
func (g *ScriptCenterGenerator) generateMakefile() (string, error) {
	return `.PHONY: build run test clean

build:
	go build -o bin/gpx-scripts ./cmd/main.go

run:
	go run ./cmd/main.go start

test:
	go test ./... -cover

clean:
	rm -rf bin/
	go clean
`, nil
}

// generateREADME 生成 README
func (g *ScriptCenterGenerator) generateREADME() (string, error) {
	return `# gpx-scripts

基于 Cobra CLI 的脚本中心，使用外部 gocron 平台进行任务调度。

## 特性

- ✅ Cobra CLI 命令框架
- ✅ Repository/Service 分层架构
- ✅ MySQL/Redis 连接管理
- ✅ 缓存支持 (Redis)
- ✅ supervisor/systemd 部署配置

## 快速开始

### 安装依赖

` + "```bash" + `
go mod tidy
` + "```" + `

### 配置环境变量

` + "```bash" + `
export MYSQL_USER=root
export MYSQL_PASSWORD=password
export MYSQL_HOST=localhost
export MYSQL_DATABASE=gpx_scripts
export REDIS_ADDR=localhost:6379
` + "```" + `

### 运行

` + "```bash" + `
./gpx-scripts start
` + "```" + `

### 使用 Example 命令

` + "```bash" + `
./gpx-scripts example create --name test --data "test"
./gpx-scripts example get 1
./gpx-scripts example list
./gpx-scripts example delete 1
` + "```" + `

## 许可证

Apache 2.0
`, nil
}

// generateGitignore 生成 .gitignore
func (g *ScriptCenterGenerator) generateGitignore() (string, error) {
	return `# Go
bin/
*.exe
*.sum
vendor/

# IDE
.idea/
.vscode/

# Local config
*.local.yaml
.env

# Logs
logs/
*.log
`, nil
}

// generateGoMod 生成 go.mod
func (g *ScriptCenterGenerator) generateGoMod() (string, error) {
	version := GetGoVersion()
	return `module ` + g.getProjectName() + `

go ` + version + `

require (
	github.com/redis/go-redis/v9 v9.3.0
	github.com/spf13/cobra v1.10.2
	gorm.io/driver/mysql v1.5.2
	gorm.io/gorm v1.25.5
)
`, nil
}

// generatePGInitPkg 生成 PostgreSQL 初始化（GORM）
func (g *ScriptCenterGenerator) generatePGInitPkg() (string, error) {
	content := `package database

import (
	"strings"
	"fmt"
	"time"
	
	CONFIG_IMPORT
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitPostgreSQL initializes PostgreSQL connection
func InitPostgreSQL() {
	cfg := config.Get()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Database)
	
	var gormLogger logger.Interface
	if cfg.App.Environment == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}
	
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil { panic(err) }
	
	sqlDB, err := DB.DB()
	if err != nil { panic(err) }
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	fmt.Println("PostgreSQL connection initialized")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateSQLiteInitPkg 生成 SQLite 初始化 (GORM)
func (g *ScriptCenterGenerator) generateSQLiteInitPkg() (string, error) {
	content := `package database

import (
	"strings"
	"fmt"
	
	CONFIG_IMPORT
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitSQLite initializes SQLite connection
func InitSQLite() {
	cfg := config.Get()
	dsn := "./data/app.db"
	
	var gormLogger logger.Interface
	if cfg.App.Environment == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}
	
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil { panic(err) }
	
	sqlDB, err := DB.DB()
	if err != nil { panic(err) }
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	fmt.Println("SQLite connection initialized")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateESInitPkg 生成 Elasticsearch 初始化
func (g *ScriptCenterGenerator) generateESInitPkg() (string, error) {
	content := `package database

import (
	"fmt"
	
	"github.com/elastic/go-elasticsearch/v8"
	CONFIG_IMPORT
)

// ESClient Elasticsearch client
var ESClient *elasticsearch.Client

// InitElasticsearch initializes Elasticsearch connection
func InitElasticsearch() {
	cfg := config.Get()
	
	addr := "http://localhost:9200"
	if cfg.Elasticsearch.Addr != "" {
		addr = cfg.Elasticsearch.Addr
	}
	
	esCfg := elasticsearch.Config{
		Addresses: []string{addr},
		Username:  cfg.Elasticsearch.Username,
		Password:  cfg.Elasticsearch.Password,
	}
	
	fmt.Printf("Connecting to Elasticsearch at %s\n", addr)
	
	var err error
	ESClient, err = elasticsearch.NewClient(esCfg)
	if err != nil {
		panic(fmt.Sprintf("Error creating Elasticsearch client: %s", err))
	}
	
	res, err := ESClient.Info()
	if err != nil {
		panic(fmt.Sprintf("Error pinging Elasticsearch: %s", err))
	}
	defer res.Body.Close()
	
	fmt.Println("Elasticsearch connection initialized")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateMongoInitPkg 生成 MongoDB 初始化
func (g *ScriptCenterGenerator) generateMongoInitPkg() (string, error) {
	content := `package database

import (
	"context"
	"fmt"
	"time"
	
	CONFIG_IMPORT
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoClient    *mongo.Client
	MongoDatabase  *mongo.Database
)

// InitMongoDB initializes MongoDB connection
func InitMongoDB() {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	uri := cfg.MongoDB.URI
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	
	clientOptions := options.Client().ApplyURI(uri)
	var err error
	MongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil { panic(err) }
	
	if err := MongoClient.Ping(ctx, nil); err != nil { panic(err) }
	
	dbName := cfg.MongoDB.Database
	if dbName == "" {
		dbName = "app"
	}
	MongoDatabase = MongoClient.Database(dbName)
	
	fmt.Println("MongoDB connection initialized")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateDBFiles generates database files based on config.DB
func (g *ScriptCenterGenerator) generateDBFiles(outputDir string) error {
	files := make(map[string]func() (string, error))

	for _, db := range g.config.DB {
		switch strings.ToLower(db) {
		case "mysql":
			files["pkg/database/mysql_init.go"] = g.generateMySQLInitPkg
		case "postgresql", "postgres", "pg":
			files["pkg/database/postgres_init.go"] = g.generatePGInitPkg
		case "sqlite", "sqlite3":
			files["pkg/database/sqlite_init.go"] = g.generateSQLiteInitPkg
		case "redis":
			files["pkg/database/redis_init.go"] = g.generateRedisInitPkg
		case "elasticsearch", "es":
			files["pkg/database/es_init.go"] = g.generateESInitPkg
		case "mongodb", "mongo":
			files["pkg/database/mongo_init.go"] = g.generateMongoInitPkg
		}
	}

	// Always generate database.go
	files["pkg/database/database.go"] = g.generateDatabasePkg

	for path, gen := range files {
		content, err := gen()
		if err != nil {
			return err
		}
		fullPath := filepath.Join(outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// updateGoModForDB returns updated go.mod content based on config.DB
func (g *ScriptCenterGenerator) updateGoModForDB(baseContent string) string {
	requires := ""
	for _, db := range g.config.DB {
		switch strings.ToLower(db) {
		case "mysql":
			requires += "\tgorm.io/driver/mysql v1.5.2\n"
		case "postgresql", "postgres", "pg":
			requires += "\tgorm.io/driver/postgres v1.5.4\n"
		case "sqlite", "sqlite3":
			requires += "\tgorm.io/driver/sqlite v1.5.4\n"
		case "redis":
			requires += "\tgithub.com/redis/go-redis/v9 v9.3.0\n"
		case "elasticsearch", "es":
			requires += "\tgithub.com/elastic/go-elasticsearch/v8 v8.11.0\n"
		case "mongodb", "mongo":
			requires += "\tgo.mongodb.org/mongo-driver v1.17.9\n"
		}
	}

	content := `module ` + g.getProjectName() + `

go ` + GetGoVersion() + `

require (
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.18.2
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gorm.io/gorm v1.25.2
`
	if requires != "" {
		content += strings.TrimSpace(requires) + "\n"
	}
	content += ")\n"

	return content
}

// generateMQInternalFiles 生成 internal 下的 producer/consumer 文件
func (g *ScriptCenterGenerator) generateMQInternalFiles(outputDir string) error {
	if g.config.MQ == "" {
		return nil
	}

	mqTypes := strings.Split(g.config.MQ, ",")
	projectName := g.getProjectName()

	for _, mqType := range mqTypes {
		mqType = strings.TrimSpace(mqType)
		if mqType == "" {
			continue
		}

		switch mqType {
		case "kafka":
			// 创建 producer 目录
			producerDir := filepath.Join(outputDir, "internal", "producer")
			if err := os.MkdirAll(producerDir, 0o755); err != nil {
				return err
			}

			// 生成 producer 文件
			producerTmplPaths := []string{
				filepath.Join("templates", "script", "internal", "producer", "kafka_producer.go.tmpl"),
				filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "internal", "producer", "kafka_producer.go.tmpl"),
			}
			for _, producerTmpl := range producerTmplPaths {
				if content, err := os.ReadFile(producerTmpl); err == nil {
					contentStr := strings.ReplaceAll(string(content), "{{.ProjectName}}", projectName)
					dstFile := filepath.Join(producerDir, "kafka_producer.go")
					os.WriteFile(dstFile, []byte(contentStr), 0o644)
					fmt.Printf("✓ Generated kafka producer\n")
					break
				}
			}

			// 创建 consumer 目录
			consumerDir := filepath.Join(outputDir, "internal", "consumer")
			if err := os.MkdirAll(consumerDir, 0o755); err != nil {
				return err
			}

			// 生成 consumer 文件
			consumerTmplPaths := []string{
				filepath.Join("templates", "script", "internal", "consumer", "kafka_consumer.go.tmpl"),
				filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "internal", "consumer", "kafka_consumer.go.tmpl"),
			}
			for _, consumerTmpl := range consumerTmplPaths {
				if content, err := os.ReadFile(consumerTmpl); err == nil {
					contentStr := strings.ReplaceAll(string(content), "{{.ProjectName}}", projectName)
					dstFile := filepath.Join(consumerDir, "kafka_consumer.go")
					os.WriteFile(dstFile, []byte(contentStr), 0o644)
					fmt.Printf("✓ Generated kafka consumer\n")
					break
				}
			}

		case "rabbitmq", "rabbit":
			// TODO: RabbitMQ producer/consumer
			fmt.Printf("⚠ RabbitMQ internal files not implemented yet\n")

		case "rocketmq":
			// 创建 producer 目录
			producerDir := filepath.Join(outputDir, "internal", "producer")
			if err := os.MkdirAll(producerDir, 0o755); err != nil {
				return err
			}

			// 生成 producer 文件
			producerTmplPaths := []string{
				filepath.Join("templates", "script", "internal", "producer", "rocketmq_producer.go.tmpl"),
				filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "internal", "producer", "rocketmq_producer.go.tmpl"),
			}
			for _, producerTmpl := range producerTmplPaths {
				if content, err := os.ReadFile(producerTmpl); err == nil {
					contentStr := strings.ReplaceAll(string(content), "{{.ProjectName}}", projectName)
					dstFile := filepath.Join(producerDir, "rocketmq_producer.go")
					os.WriteFile(dstFile, []byte(contentStr), 0o644)
					fmt.Printf("✓ Generated rocketmq producer\n")
					break
				}
			}

			// 创建 consumer 目录
			consumerDir := filepath.Join(outputDir, "internal", "consumer")
			if err := os.MkdirAll(consumerDir, 0o755); err != nil {
				return err
			}

			// 生成 consumer 文件
			consumerTmplPaths := []string{
				filepath.Join("templates", "script", "internal", "consumer", "rocketmq_consumer.go.tmpl"),
				filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "internal", "consumer", "rocketmq_consumer.go.tmpl"),
			}
			for _, consumerTmpl := range consumerTmplPaths {
				if content, err := os.ReadFile(consumerTmpl); err == nil {
					contentStr := strings.ReplaceAll(string(content), "{{.ProjectName}}", projectName)
					dstFile := filepath.Join(consumerDir, "rocketmq_consumer.go")
					os.WriteFile(dstFile, []byte(contentStr), 0o644)
					fmt.Printf("✓ Generated rocketmq consumer\n")
					break
				}
			}
		}
	}

	return nil
}

// generateMQCommands 生成 MQ CLI 命令文件
func (g *ScriptCenterGenerator) generateMQCommands(outputDir string) error {
	if g.config.MQ == "" {
		return nil
	}

	mqTypes := strings.Split(g.config.MQ, ",")
	projectName := g.getProjectName()

	for _, mqType := range mqTypes {
		mqType = strings.TrimSpace(mqType)
		if mqType == "" {
			continue
		}

		switch mqType {
		case "kafka":
			// 生成 kafka.go 命令文件
			templatePaths := []string{
				filepath.Join("templates", "script", "cmd", "commands", "kafka.go.tmpl"),
				filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "cmd", "commands", "kafka.go.tmpl"),
			}

			var tmplContent []byte
			var err error
			for _, srcFile := range templatePaths {
				tmplContent, err = os.ReadFile(srcFile)
				if err == nil {
					break
				}
			}

			if err != nil {
				fmt.Printf("⚠ Kafka command template not found, skipping...\n")
				continue
			}

			// 替换模板变量 - 提取项目名称的最后部分
			// 例如：从 "github.com/gospacex/my-scripts-2" 提取 "my-scripts-2"
			projectSimpleName := projectName
			if parts := strings.Split(projectName, "/"); len(parts) > 0 {
				projectSimpleName = parts[len(parts)-1]
			}
			contentStr := strings.ReplaceAll(string(tmplContent), "{{.ProjectName}}", projectSimpleName)
			dstFile := filepath.Join(outputDir, "cmd", "commands", "kafka.go")
			os.WriteFile(dstFile, []byte(contentStr), 0o644)
			fmt.Printf("✓ Generated kafka command\n")

		case "rabbitmq", "rabbit":
			// TODO: RabbitMQ commands
			fmt.Printf("⚠ RabbitMQ commands not implemented yet\n")

		case "rocketmq":
			// TODO: RocketMQ commands
			fmt.Printf("⚠ RocketMQ commands not implemented yet\n")
		}
	}

	return nil
}

// getProjectName 获取项目名称
func (g *ScriptCenterGenerator) getProjectName() string {
	if g.config.ProjectName != "" {
		return g.config.ProjectName
	}
	return "github.com/gospacex/gpx-scripts"
}

// generateDatabasePkg 生成 pkg/database/database.go
func (g *ScriptCenterGenerator) generateDatabasePkg() (string, error) {
	return `package database

import (
	"context"
	"fmt"
	
	"gorm.io/gorm"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/elastic/go-elasticsearch/v8"
)

var (
	// MySQL/PostgreSQL
	DB *gorm.DB
	
	// Redis
	RedisClient *redis.Client
	
	// MongoDB
	MongoClient *mongo.Client
	MongoDatabase *mongo.Database
	
	// Elasticsearch
	ESClient *elasticsearch.Client
)

func SetDB(db *gorm.DB) {
	DB = db
}

func GetDB() *gorm.DB {
	return DB
}

// Close closes all database connections
func Close() {
	fmt.Println("Closing database connections...")
	
	// Close MySQL/PostgreSQL
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
			fmt.Println("MySQL/PostgreSQL connection closed")
		}
	}
	
	// Close Redis
	if RedisClient != nil {
		RedisClient.Close()
		fmt.Println("Redis connection closed")
	}
	
	// Close MongoDB
	if MongoClient != nil {
		MongoClient.Disconnect(context.Background())
		fmt.Println("MongoDB connection closed")
	}
	
	// Elasticsearch doesn't require explicit closing
	fmt.Println("All database connections closed")
}
`, nil
}

// generatePkgConfig 生成 pkg/config/config.go
func (g *ScriptCenterGenerator) generatePkgConfig() (string, error) {
	return `package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var globalConfig *Config

func Load(path string) (*Config, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("configs")
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.SetConfigType("yaml")
	}

	v.SetDefault("app.name", "gpx-scripts")
	v.SetDefault("app.environment", "development")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "3306")
	v.SetDefault("database.user", "gorm")
	v.SetDefault("database.password", "gorm")
	v.SetDefault("database.database", "gorm")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("log.path", "./logs/app.log")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 尝试读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

func Get() *Config {
	if globalConfig == nil {
		cfg, err := Load("")
		if err != nil {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
		globalConfig = cfg
	}
	return globalConfig
}
`, nil
}

// generateConfigTypes 生成 pkg/config/types.go
func (g *ScriptCenterGenerator) generateConfigTypes() (string, error) {
	return `package config

import (
	"fmt"
)

// Config 主配置结构
type Config struct {
	App         AppConfig         ` + "`mapstructure:\"app\" yaml:\"app\"`" + `
	Database    DatabaseConfig    ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
	MySQL       MySQLConfig       ` + "`mapstructure:\"mysql\" yaml:\"mysql\"`" + `
	PostgreSQL  PostgreSQLConfig  ` + "`mapstructure:\"postgresql\" yaml:\"postgresql\"`" + `
	Redis       RedisConfig       ` + "`mapstructure:\"redis\" yaml:\"redis\"`" + `
	MongoDB     MongoDBConfig     ` + "`mapstructure:\"mongodb\" yaml:\"mongodb\"`" + `
	Elasticsearch ElasticsearchConfig ` + "`mapstructure:\"elasticsearch\" yaml:\"elasticsearch\"`" + `
	Kafka       KafkaConfig       ` + "`mapstructure:\"kafka\" yaml:\"kafka\"`" + `
	RabbitMQ    RabbitMQConfig    ` + "`mapstructure:\"rabbitmq\" yaml:\"rabbitmq\"`" + `
	RocketMQ    RocketMQConfig    ` + "`mapstructure:\"rocketmq\" yaml:\"rocketmq\"`" + `
	Log         LogConfig         ` + "`mapstructure:\"log\" yaml:\"log\"`" + `
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string ` + "`mapstructure:\"name\" yaml:\"name\" default:\"gpx-scripts\"`" + `
	Environment string ` + "`mapstructure:\"environment\" yaml:\"environment\" default:\"development\"`" + `
}

// DatabaseConfig 数据库主配置（支持多数据库）
type DatabaseConfig struct {
	Types       []string          ` + "`mapstructure:\"types\" yaml:\"types\"`" + `
	MySQL       MySQLDBConfig     ` + "`mapstructure:\"mysql\" yaml:\"mysql\"`" + `
	PostgreSQL  PostgreSQLDBConfig ` + "`mapstructure:\"postgresql\" yaml:\"postgresql\"`" + `
	Redis       RedisDBConfig     ` + "`mapstructure:\"redis\" yaml:\"redis\"`" + `
	MongoDB     MongoDBDBConfig   ` + "`mapstructure:\"mongodb\" yaml:\"mongodb\"`" + `
	Elasticsearch ElasticDBConfig ` + "`mapstructure:\"elasticsearch\" yaml:\"elasticsearch\"`" + `
	Table       string            ` + "`mapstructure:\"table\" yaml:\"table\"`" + `
}

// MySQLDBConfig MySQL 数据库配置
type MySQLDBConfig struct {
	Host            string ` + "`mapstructure:\"host\" yaml:\"host\" default:\"127.0.0.1\"`" + `
	Port            int    ` + "`mapstructure:\"port\" yaml:\"port\" default:\"3306\"`" + `
	User            string ` + "`mapstructure:\"user\" yaml:\"user\"`" + `
	Password        string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	Database        string ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
	MaxOpenConns    int    ` + "`mapstructure:\"max_open_conns\" yaml:\"max_open_conns\" default:\"100\"`" + `
	MaxIdleConns    int    ` + "`mapstructure:\"max_idle_conns\" yaml:\"max_idle_conns\" default:\"10\"`" + `
	ConnMaxLifetime int    ` + "`mapstructure:\"conn_max_lifetime\" yaml:\"conn_max_lifetime\" default:\"3600\"`" + `
	Table           string ` + "`mapstructure:\"table\" yaml:\"table\"`" + `
}

// GetDSN returns MySQL DSN connection string
func (c *MySQLDBConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", c.User, c.Password, c.Host, c.Port, c.Database)
}

// PostgreSQLDBConfig PostgreSQL 数据库配置
type PostgreSQLDBConfig struct {
	Host            string ` + "`mapstructure:\"host\" yaml:\"host\" default:\"127.0.0.1\"`" + `
	Port            int    ` + "`mapstructure:\"port\" yaml:\"port\" default:\"5432\"`" + `
	User            string ` + "`mapstructure:\"user\" yaml:\"user\"`" + `
	Password        string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	Database        string ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
	Schema          string ` + "`mapstructure:\"schema\" yaml:\"schema\" default:\"public\"`" + `
	MaxOpenConns    int    ` + "`mapstructure:\"max_open_conns\" yaml:\"max_open_conns\" default:\"100\"`" + `
	MaxIdleConns    int    ` + "`mapstructure:\"max_idle_conns\" yaml:\"max_idle_conns\" default:\"10\"`" + `
	ConnMaxLifetime int    ` + "`mapstructure:\"conn_max_lifetime\" yaml:\"conn_max_lifetime\" default:\"3600\"`" + `
	Table           string ` + "`mapstructure:\"table\" yaml:\"table\"`" + `
}

// GetDSN returns PostgreSQL DSN connection string
func (c *PostgreSQLDBConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", c.Host, c.Port, c.User, c.Password, c.Database)
}

// RedisDBConfig Redis 配置
type RedisDBConfig struct {
	Addr         string ` + "`mapstructure:\"addr\" yaml:\"addr\" default:\"localhost:6379\"`" + `
	Password     string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	DB           int    ` + "`mapstructure:\"db\" yaml:\"db\" default:\"0\"`" + `
	PoolSize     int    ` + "`mapstructure:\"pool_size\" yaml:\"pool_size\" default:\"100\"`" + `
	MinIdleConns int    ` + "`mapstructure:\"min_idle_conns\" yaml:\"min_idle_conns\" default:\"5\"`" + `
}

// MongoDBDBConfig MongoDB 配置
type MongoDBDBConfig struct {
	URI         string ` + "`mapstructure:\"uri\" yaml:\"uri\" default:\"mongodb://localhost:27017\"`" + `
	Database    string ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
	Collection  string ` + "`mapstructure:\"collection\" yaml:\"collection\"`" + `
	MaxPoolSize int    ` + "`mapstructure:\"max_pool_size\" yaml:\"max_pool_size\" default:\"100\"`" + `
	MinPoolSize int    ` + "`mapstructure:\"min_pool_size\" yaml:\"min_pool_size\" default:\"10\"`" + `
}

// ElasticDBConfig Elasticsearch 配置
type ElasticDBConfig struct {
	Addresses     []string ` + "`mapstructure:\"addresses\" yaml:\"addresses\"`" + `
	Username      string   ` + "`mapstructure:\"username\" yaml:\"username\"`" + `
	Password      string   ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	Index         string   ` + "`mapstructure:\"index\" yaml:\"index\"`" + `
	Sniff         bool     ` + "`mapstructure:\"sniff\" yaml:\"sniff\" default:\"false\"`" + `
	HealthcheckInterval int ` + "`mapstructure:\"healthcheck_interval\" yaml:\"healthcheck_interval\" default:\"60\"`" + `
	// MySQL to ES sync configuration
	SyncFromMySQL bool   ` + "`mapstructure:\"sync_from_mysql\" yaml:\"sync_from_mysql\"`" + `
	MySQLHost     string ` + "`mapstructure:\"mysql_host\" yaml:\"mysql_host\"`" + `
	MySQLPort     int    ` + "`mapstructure:\"mysql_port\" yaml:\"mysql_port\"`" + `
	MySQLUser     string ` + "`mapstructure:\"mysql_user\" yaml:\"mysql_user\"`" + `
	MySQLPassword string ` + "`mapstructure:\"mysql_password\" yaml:\"mysql_password\"`" + `
	MySQLDatabase string ` + "`mapstructure:\"mysql_database\" yaml:\"mysql_database\"`" + `
	MySQLTable    string ` + "`mapstructure:\"mysql_table\" yaml:\"mysql_table\"`" + `
}

// MySQLConfig 兼容旧配置（从 nacos 读取时使用）
type MySQLConfig struct {
	Host     string ` + "`mapstructure:\"host\" yaml:\"host\"`" + `
	Port     string ` + "`mapstructure:\"port\" yaml:\"port\"`" + `
	User     string ` + "`mapstructure:\"user\" yaml:\"user\"`" + `
	Password string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	Database string ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
}

// PostgreSQLConfig 兼容旧配置
type PostgreSQLConfig struct {
	Host     string ` + "`mapstructure:\"host\" yaml:\"host\"`" + `
	Port     string ` + "`mapstructure:\"port\" yaml:\"port\"`" + `
	User     string ` + "`mapstructure:\"user\" yaml:\"user\"`" + `
	Password string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	Database string ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
	Schema   string ` + "`mapstructure:\"schema\" yaml:\"schema\"`" + `
}

// RedisConfig 兼容旧配置
type RedisConfig struct {
	Addr     string ` + "`mapstructure:\"addr\" yaml:\"addr\"`" + `
	Password string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	DB       int    ` + "`mapstructure:\"db\" yaml:\"db\"`" + `
}

// MongoDBConfig 兼容旧配置
type MongoDBConfig struct {
	URI      string ` + "`mapstructure:\"uri\" yaml:\"uri\"`" + `
	Database string ` + "`mapstructure:\"database\" yaml:\"database\"`" + `
}

// ElasticsearchConfig 兼容旧配置
type ElasticsearchConfig struct {
	Addr     string ` + "`mapstructure:\"addr\" yaml:\"addr\"`" + `
	Username string ` + "`mapstructure:\"username\" yaml:\"username\"`" + `
	Password string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string ` + "`mapstructure:\"level\" yaml:\"level\" default:\"info\"`" + `
	Format     string ` + "`mapstructure:\"format\" yaml:\"format\" default:\"json\"`" + `
	Output     string ` + "`mapstructure:\"output\" yaml:\"output\" default:\"stdout\"`" + `
	Path       string ` + "`mapstructure:\"path\" yaml:\"path\" default:\"./logs/app.log\"`" + `
	MaxSize    int    ` + "`mapstructure:\"max_size\" yaml:\"max_size\" default:\"100\"`" + `
	MaxBackups int    ` + "`mapstructure:\"max_backups\" yaml:\"max_backups\" default:\"3\"`" + `
	MaxAge     int    ` + "`mapstructure:\"max_age\" yaml:\"max_age\" default:\"28\"`" + `
	Compress   bool   ` + "`mapstructure:\"compress\" yaml:\"compress\" default:\"true\"`" + `
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Enabled       bool     ` + "`mapstructure:\"enabled\" yaml:\"enabled\"`" + `
	Brokers       []string ` + "`mapstructure:\"brokers\" yaml:\"brokers\"`" + `
	Topic         string   ` + "`mapstructure:\"topic\" yaml:\"topic\"`" + `
	ConsumerGroup string   ` + "`mapstructure:\"consumer_group\" yaml:\"consumer_group\"`" + `
}

// RabbitMQConfig RabbitMQ 配置
type RabbitMQConfig struct {
	Enabled    bool   ` + "`mapstructure:\"enabled\" yaml:\"enabled\"`" + `
	Addr       string ` + "`mapstructure:\"addr\" yaml:\"addr\"`" + `
	Vhost      string ` + "`mapstructure:\"vhost\" yaml:\"vhost\"`" + `
	Username   string ` + "`mapstructure:\"username\" yaml:\"username\"`" + `
	Password   string ` + "`mapstructure:\"password\" yaml:\"password\"`" + `
	Exchange   string ` + "`mapstructure:\"exchange\" yaml:\"exchange\"`" + `
	Queue      string ` + "`mapstructure:\"queue\" yaml:\"queue\"`" + `
}

// RocketMQConfig RocketMQ 配置
type RocketMQConfig struct {
	Enabled       bool   ` + "`mapstructure:\"enabled\" yaml:\"enabled\"`" + `
	NameServer    string ` + "`mapstructure:\"name_server\" yaml:\"name_server\"`" + `
	Topic         string ` + "`mapstructure:\"topic\" yaml:\"topic\"`" + `
	ConsumerGroup string ` + "`mapstructure:\"consumer_group\" yaml:\"consumer_group\"`" + `
	ProducerGroup string ` + "`mapstructure:\"producer_group\" yaml:\"producer_group\"`" + `
}
`, nil
}

// generatePkgLogger 生成 pkg/logger/logger.go
func (g *ScriptCenterGenerator) generatePkgLogger() (string, error) {
	return `package logger

import (
	"fmt"
	"os"

	"github.com/gospacex/gpx-scripts/pkg/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logFile *lumberjack.Logger

func Init() {
	cfg := config.Get()
	
	logFile = &lumberjack.Logger{
		Filename:   cfg.Log.Path,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}
	
	fmt.Printf("Logger initialized, output: %s\n", cfg.Log.Path)
}

func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	if logFile != nil {
		_, _ = logFile.Write([]byte(msg + "\n"))
	}
}

func Log(format string, args ...interface{}) {
	Info(format, args...)
}

func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf("[ERROR] "+format, args...)
	fmt.Fprintln(os.Stderr, msg)
	if logFile != nil {
		_, _ = logFile.Write([]byte(msg + "\n"))
	}
}

func Debug(format string, args ...interface{}) {
	msg := fmt.Sprintf("[DEBUG] "+format, args...)
	fmt.Println(msg)
	if logFile != nil {
		_, _ = logFile.Write([]byte(msg + "\n"))
	}
}
`, nil
}

// generateMySQLInitPkg 生成 pkg/database/mysql_init.go（GORM）
func (g *ScriptCenterGenerator) generateMySQLInitPkg() (string, error) {
	content := `package database

import (
	"fmt"
	"time"
	
	CONFIG_IMPORT
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitMySQL initializes MySQL connection
func InitMySQL() {
	cfg := config.Get()
	
	// Use nested database.mysql config if available, otherwise fall back to legacy mysql config
	var dsn string
	if cfg.Database.MySQL.Host != "" {
		fmt.Printf("Using MySQL config from database.mysql: host=%s, port=%d, database=%s\n", 
			cfg.Database.MySQL.Host, cfg.Database.MySQL.Port, cfg.Database.MySQL.Database)
		dsn = cfg.Database.MySQL.GetDSN()
	} else if cfg.MySQL.Host != "" {
		fmt.Printf("Using MySQL config from legacy mysql: host=%s, port=%s, database=%s\n", 
			cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", 
			cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	} else {
		panic("MySQL configuration not found")
	}
	
	var gormLogger logger.Interface
	if cfg.App.Environment == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
		fmt.Println("Development environment: SQL logging enabled")
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
		fmt.Println("Production environment: SQL logging disabled")
	}
	
	var err error
	var db *gorm.DB
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MySQL: %v", err))
	}
	SetDB(db)
	
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	
	// Use config values or defaults
	maxOpenConns := cfg.Database.MySQL.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	maxIdleConns := cfg.Database.MySQL.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	connMaxLifetime := cfg.Database.MySQL.ConnMaxLifetime
	if connMaxLifetime == 0 {
		connMaxLifetime = 3600
	}
	
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	
	fmt.Println("MySQL connection initialized successfully")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateRedisInitPkg 生成 pkg/database/redis_init.go
func (g *ScriptCenterGenerator) generateRedisInitPkg() (string, error) {
	content := `package database

import (
	"context"
	"fmt"
	"time"
	
	CONFIG_IMPORT
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedis initializes Redis connection
func InitRedis() {
	cfg := config.Get()
	ctx := context.Background()
	
	// Use nested database.redis config if available, otherwise fall back to legacy redis config
	var addr, password string
	var db int
	
	if cfg.Database.Redis.Addr != "" {
		fmt.Printf("Using Redis config from database.redis: addr=%s, db=%d\n", 
			cfg.Database.Redis.Addr, cfg.Database.Redis.DB)
		addr = cfg.Database.Redis.Addr
		password = cfg.Database.Redis.Password
		db = cfg.Database.Redis.DB
	} else if cfg.Redis.Addr != "" {
		fmt.Printf("Using Redis config from legacy redis: addr=%s, db=%d\n", 
			cfg.Redis.Addr, cfg.Redis.DB)
		addr = cfg.Redis.Addr
		password = cfg.Redis.Password
		db = cfg.Redis.DB
	} else {
		panic("Redis configuration not found")
	}
	
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := RedisClient.Ping(pingCtx).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
	
	fmt.Println("Redis connection initialized successfully")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generatePostgreSQLInitPkg 生成 pkg/database/postgres_init.go
func (g *ScriptCenterGenerator) generatePostgreSQLInitPkg() (string, error) {
	content := `package database

import (
	"fmt"
	"time"
	
	CONFIG_IMPORT
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitPostgreSQL initializes PostgreSQL connection
func InitPostgreSQL() {
	cfg := config.Get()
	
	// Use nested database.postgresql config
	var dsn string
	if cfg.Database.PostgreSQL.Host != "" {
		fmt.Printf("Using PostgreSQL config: host=%s, port=%d, database=%s\n", 
			cfg.Database.PostgreSQL.Host, cfg.Database.PostgreSQL.Port, cfg.Database.PostgreSQL.Database)
		dsn = cfg.Database.PostgreSQL.GetDSN()
	} else {
		panic("PostgreSQL configuration not found")
	}
	
	var gormLogger logger.Interface
	if cfg.App.Environment == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}
	
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to PostgreSQL: %v", err))
	}
	
	sqlDB, err := DB.DB()
	if err != nil {
		panic(err)
	}
	
	maxOpenConns := cfg.Database.PostgreSQL.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	maxIdleConns := cfg.Database.PostgreSQL.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	connMaxLifetime := cfg.Database.PostgreSQL.ConnMaxLifetime
	if connMaxLifetime == 0 {
		connMaxLifetime = 3600
	}
	
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	
	fmt.Println("PostgreSQL connection initialized successfully")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateMongoDBInitPkg 生成 pkg/database/mongo_init.go
func (g *ScriptCenterGenerator) generateMongoDBInitPkg() (string, error) {
	content := `package database

import (
	"context"
	"fmt"
	"time"
	
	CONFIG_IMPORT
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var MongoDatabase *mongo.Database

// InitMongoDB initializes MongoDB connection
func InitMongoDB() {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Use nested database.mongodb config
	var uri, dbName string
	if cfg.Database.MongoDB.URI != "" {
		fmt.Printf("Using MongoDB config: uri=%s, database=%s\n", 
			cfg.Database.MongoDB.URI, cfg.Database.MongoDB.Database)
		uri = cfg.Database.MongoDB.URI
		dbName = cfg.Database.MongoDB.Database
	} else {
		panic("MongoDB configuration not found")
	}
	
	if dbName == "" {
		dbName = "gpx_scripts"
	}
	
	clientOptions := options.Client().ApplyURI(uri)
	
	var err error
	MongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MongoDB: %v", err))
	}
	
	// Ping to verify connection
	if err := MongoClient.Ping(ctx, nil); err != nil {
		panic(fmt.Sprintf("MongoDB ping failed: %v", err))
	}
	
	MongoDatabase = MongoClient.Database(dbName)
	
	fmt.Println("MongoDB connection initialized successfully")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateElasticsearchInitPkg 生成 pkg/database/es_init.go
func (g *ScriptCenterGenerator) generateElasticsearchInitPkg() (string, error) {
	content := `package database

import (
	"fmt"
	
	CONFIG_IMPORT
	"github.com/elastic/go-elasticsearch/v8"
)

var ESClient *elasticsearch.Client

// InitElasticsearch initializes Elasticsearch connection
func InitElasticsearch() {
	cfg := config.Get()
	
	// Use nested database.elasticsearch config
	var addresses []string
	var username, password string
	
	if len(cfg.Database.Elasticsearch.Addresses) > 0 {
		fmt.Printf("Using Elasticsearch config: addresses=%v\n", 
			cfg.Database.Elasticsearch.Addresses)
		addresses = cfg.Database.Elasticsearch.Addresses
		username = cfg.Database.Elasticsearch.Username
		password = cfg.Database.Elasticsearch.Password
	} else {
		panic("Elasticsearch configuration not found")
	}
	
	esConfig := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	}
	
	var err error
	ESClient, err = elasticsearch.NewClient(esConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Elasticsearch client: %v", err))
	}
	
	// Ping to verify connection
	res, err := ESClient.Info()
	if err != nil {
		panic(fmt.Sprintf("Elasticsearch info request failed: %v", err))
	}
	defer res.Body.Close()
	
	fmt.Println("Elasticsearch connection initialized successfully")
}
`
	content = strings.ReplaceAll(content, "CONFIG_IMPORT", "\""+g.getProjectName()+"/pkg/config\"")
	return content, nil
}

// generateStartCmd 生成 Start 命令（使用新路径）
func (g *ScriptCenterGenerator) generateStartCmdNew() (string, error) {
	// 动态生成只包含用户选择的数据库 case
	cases := ""
	for _, db := range g.config.DB {
		switch strings.ToLower(db) {
		case "mysql":
			cases += `		case "mysql":
				log.Println("Initializing MySQL...")
				database.InitMySQL()
`
		case "postgresql", "postgres", "pg":
			cases += `		case "postgresql", "postgres", "pg":
				log.Println("Initializing PostgreSQL...")
				database.InitPostgreSQL()
`
		case "redis":
			cases += `		case "redis":
				log.Println("Initializing Redis...")
				database.InitRedis()
`
		case "mongodb", "mongo":
			cases += `		case "mongodb", "mongo":
				log.Println("Initializing MongoDB...")
				database.InitMongoDB()
`
		case "elasticsearch", "es":
			cases += `		case "elasticsearch", "es":
				log.Println("Initializing Elasticsearch...")
				database.InitElasticsearch()
`
		}
	}

	// 如果没有任何 case，至少保留 default
	if cases == "" {
		cases = `		default:
`
	}

	// 获取项目名称
	projectName := g.getProjectName()

	// 不需要额外导入 - CRUD CLI 命令已经通过 init() 自动注册到 rootCmd
	extraImports := ""
	// 不需要 job 注册 - 任务调度由外部 gocron 平台管理，通过 CLI 命令调用
	jobRegistration := ""

	template := `package commands

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"` + projectName + `/pkg/database"
	"` + projectName + `/pkg/config"
` + extraImports + `	"github.com/spf13/cobra"
)

// startCmd 启动命令
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动任务调度服务",
	Long:  ` + "`" + `启动脚本中心服务，初始化数据库连接并加载任务` + "`" + `,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.Load("")
		if err != nil {
			return err
		}
		cfg := config.Get()
		
		// 根据配置动态初始化数据库连接
		for _, dbType := range cfg.Database.Types {
			switch dbType {
` + cases + `		default:
			log.Printf("Unknown database type: %s, skipping", dbType)
			}
		}

		log.Println("✓ All database connections initialized")
` + jobRegistration + `
		// 等待退出信号
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		<-ctx.Done()
		log.Println("Shutting down...")
		
		// 关闭连接
		database.Close()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
`
	// 替换模板变量
	template = strings.ReplaceAll(template, "github.com/gospacex/gpx-scripts", projectName)
	return template, nil
}

// generateMQFiles 根据配置.MQ 生成中间件文件
func (g *ScriptCenterGenerator) generateMQFiles(outputDir string) error {
	if g.config.MQ == "" {
		return nil
	}

	mqDir := filepath.Join(outputDir, "pkg", "middleware")
	if err := os.MkdirAll(mqDir, 0o755); err != nil {
		return err
	}

	mqTypes := strings.Split(g.config.MQ, ",")
	for _, mqType := range mqTypes {
		mqType = strings.TrimSpace(mqType)
		if mqType == "" {
			continue
		}

		// 尝试多个可能的模板路径
		templatePaths := []string{
			filepath.Join("templates", "script", "pkg", "middleware", mqType+".go.tmpl"),
			filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "pkg", "middleware", mqType+".go.tmpl"),
		}

		var tmplContent []byte
		var err error
		for _, srcFile := range templatePaths {
			tmplContent, err = os.ReadFile(srcFile)
			if err == nil {
				break
			}
		}

		if err != nil {
			fmt.Printf("⚠ Template not found for %s, skipping...\n", mqType)
			continue
		}

		dstFile := filepath.Join(mqDir, mqType+".go")
		if err := os.WriteFile(dstFile, tmplContent, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", mqType, err)
		}
		fmt.Printf("✓ Generated %s middleware\n", mqType)
	}

	// 生成 types.go
	typesPaths := []string{
		filepath.Join("templates", "script", "pkg", "middleware", "types.go.tmpl"),
		filepath.Join("/Users/hyx/work/gowork/src/gospacex/templates", "script", "pkg", "middleware", "types.go.tmpl"),
	}
	typesDst := filepath.Join(mqDir, "types.go")
	for _, typesSrc := range typesPaths {
		if content, err := os.ReadFile(typesSrc); err == nil {
			os.WriteFile(typesDst, content, 0o644)
			break
		}
	}

	return nil
}

// updateGoModForMQ 更新 go.mod 添加 MQ 依赖
func (g *ScriptCenterGenerator) updateGoModForMQ(content string) string {
	if g.config.MQ == "" {
		return content
	}

	mqTypes := strings.Split(g.config.MQ, ",")
	for _, mqType := range mqTypes {
		mqType = strings.TrimSpace(mqType)
		switch mqType {
		case "kafka":
			content += "\nrequire github.com/IBM/sarama v1.43.3"
		case "rabbitmq", "rabbit":
			content += "\nrequire github.com/rabbitmq/amqp091-go v1.9.0"
		case "rocketmq":
			content += "\nrequire github.com/apache/rocketmq-client-go/v2 v2.1.2"
		}
	}

	return content
}

// generateCRUDCommands creates CLI commands for CRUD operations based on table name
// table: eb_article → file: cmd/commands/eb_article.go
func (g *ScriptCenterGenerator) generateCRUDCommands(outputDir string) error {
	tableName := g.config.MySQLTable
	entityName := toCamelCaseUpper(tableName)
	entityLower := strings.ToLower(entityName)
	cmdDir := filepath.Join(outputDir, "cmd", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		return err
	}

	// Use strings.Builder to construct the content properly
	var b strings.Builder
	projectName := g.getProjectName()

	b.WriteString("package commands\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"context\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"log\"\n")
	b.WriteString("\t\"strconv\"\n\n")
	b.WriteString("\t\"" + projectName + "/internal/handler\"\n")
	b.WriteString("\t\"" + projectName + "/internal/model\"\n")
	b.WriteString("\t\"" + projectName + "/internal/repository\"\n")
	b.WriteString("\t\"" + projectName + "/internal/service\"\n")
	b.WriteString("\t\"" + projectName + "/pkg/config\"\n")
	b.WriteString("\t\"" + projectName + "/pkg/database\"\n")
	b.WriteString("\t\"github.com/spf13/cobra\"\n")
	b.WriteString(")\n\n")

	b.WriteString("func init() {\n")
	b.WriteString("\t// 确保每次命令执行前初始化数据库\n")
	b.WriteString("\trootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\t_, err := config.Load(\"\")\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn err\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\t// 只初始化 MySQL，其他数据库按需初始化\n")
	b.WriteString("\t\tif database.DB == nil {\n")
	b.WriteString("\t\t\tdatabase.InitMySQL()\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	b.WriteString("var (\n")
	b.WriteString("\t// " + entityName + " 命令标志\n")
	b.WriteString("\t" + entityLower + "Title string\n")
	b.WriteString("\t" + entityLower + "Author string\n")
	b.WriteString("\t" + entityLower + "Visit string\n\n")
	b.WriteString("\t" + entityName + "Cmd = &cobra.Command{\n")
	b.WriteString("\tUse:   \"" + entityLower + "\",\n")
	b.WriteString("\tShort: \"" + entityName + " 管理命令\",\n")
	b.WriteString("\tLong:  \"" + entityName + " 相关 CRUD 操作\",\n")
	b.WriteString("}\n")
	b.WriteString(")\n\n")

	b.WriteString("// " + entityName + "CreateCmd 创建 " + entityName + "\n")
	b.WriteString("var " + entityName + "CreateCmd = &cobra.Command{\n")
	b.WriteString("\tUse:   \"create\",\n")
	b.WriteString("\tShort: \"创建 " + entityName + "\",\n")
	b.WriteString("\tLong:  \"创建一条新的 " + entityName + " 记录\",\n")
	b.WriteString("\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\tctx := context.Background()\n")
	b.WriteString("\t\trepo := repository.NewRepository(database.DB)\n")
	b.WriteString("\t\tsvc := service.NewService(repo)\n")
	b.WriteString("\t\th := handler.NewCLIHandler(svc)\n\n")
	b.WriteString("\t\tentity := &model." + entityName + "{\n")
	b.WriteString("\t\t\tTitle:  " + entityLower + "Title,\n")
	b.WriteString("\t\t\tAuthor: " + entityLower + "Author,\n")
	b.WriteString("\t\t\tVisit:  " + entityLower + "Visit,\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tif err := h.CreateCLI(ctx, entity); err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"create " + entityName + " failed: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tlog.Printf(\"✓ " + entityName + " created with ID: %d\", entity.Id)\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("// " + entityName + "GetCmd 查询 " + entityName + "\n")
	b.WriteString("var " + entityName + "GetCmd = &cobra.Command{\n")
	b.WriteString("\tUse:   \"get [id]\",\n")
	b.WriteString("\tShort: \"查询 " + entityName + "\",\n")
	b.WriteString("\tLong:  \"根据 ID 查询 " + entityName + " 记录\",\n")
	b.WriteString("\tArgs:  cobra.ExactArgs(1),\n")
	b.WriteString("\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\tctx := context.Background()\n")
	b.WriteString("\t\trepo := repository.NewRepository(database.DB)\n")
	b.WriteString("\t\tsvc := service.NewService(repo)\n")
	b.WriteString("\t\th := handler.NewCLIHandler(svc)\n\n")
	b.WriteString("\t\tid, err := strconv.ParseInt(args[0], 10, 64)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"invalid ID: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tentity, err := h.GetByIDCLI(ctx, int(id))\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"get " + entityName + " failed: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tlog.Printf(\"" + entityName + ": Id=%d\\n\", entity.Id)\n")
	b.WriteString("\t\tlog.Printf(\"  Title: %s\\n\", entity.Title)\n")
	b.WriteString("\t\tlog.Printf(\"  Author: %s\\n\", entity.Author)\n")
	b.WriteString("\t\tlog.Printf(\"  Visit: %s\\n\", entity.Visit)\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("// " + entityName + "ListCmd 列表查询 " + entityName + "\n")
	b.WriteString("var " + entityName + "ListCmd = &cobra.Command{\n")
	b.WriteString("\tUse:   \"list [page] [page_size]\",\n")
	b.WriteString("\tShort: \"列表查询 " + entityName + "\",\n")
	b.WriteString("\tLong:  \"分页查询 " + entityName + " 记录\",\n")
	b.WriteString("\tArgs:  cobra.RangeArgs(0, 2),\n")
	b.WriteString("\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\tctx := context.Background()\n")
	b.WriteString("\t\trepo := repository.NewRepository(database.DB)\n")
	b.WriteString("\t\tsvc := service.NewService(repo)\n")
	b.WriteString("\t\th := handler.NewCLIHandler(svc)\n\n")
	b.WriteString("\t\tpage := 1\n")
	b.WriteString("\t\tpageSize := 10\n\n")
	b.WriteString("\t\tif len(args) >= 1 {\n")
	b.WriteString("\t\t\tif p, err := strconv.Atoi(args[0]); err == nil {\n")
	b.WriteString("\t\t\t\tpage = p\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif len(args) >= 2 {\n")
	b.WriteString("\t\t\tif ps, err := strconv.Atoi(args[1]); err == nil {\n")
	b.WriteString("\t\t\t\tpageSize = ps\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tentities, total, err := h.ListCLI(ctx, page, pageSize)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"list " + entityName + " failed: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tlog.Printf(\"Total: %d, Page: %d, PageSize: %d\\n\", total, page, pageSize)\n")
	b.WriteString("\t\tfor i, e := range entities {\n")
	b.WriteString("\t\t\tlog.Printf(\"  %d. Id=%d Title=%s Author=%s Visit=%s\", i+1, e.Id, e.Title, e.Author, e.Visit)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("// " + entityName + "UpdateCmd 更新 " + entityName + "\n")
	b.WriteString("var " + entityName + "UpdateCmd = &cobra.Command{\n")
	b.WriteString("\tUse:   \"update [id]\",\n")
	b.WriteString("\tShort: \"更新 " + entityName + "\",\n")
	b.WriteString("\tLong:  \"根据 ID 更新 " + entityName + " 记录\",\n")
	b.WriteString("\tArgs:  cobra.ExactArgs(1),\n")
	b.WriteString("\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\tctx := context.Background()\n")
	b.WriteString("\t\trepo := repository.NewRepository(database.DB)\n")
	b.WriteString("\t\tsvc := service.NewService(repo)\n")
	b.WriteString("\t\th := handler.NewCLIHandler(svc)\n\n")
	b.WriteString("\t\tid, err := strconv.ParseInt(args[0], 10, 64)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"invalid ID: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tentity, err := h.GetByIDCLI(ctx, int(id))\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"get " + entityName + " failed: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Only update fields that have flags set\n")
	b.WriteString("\t\tif " + entityLower + "Title != \"\" {\n")
	b.WriteString("\t\t\tentity.Title = " + entityLower + "Title\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif " + entityLower + "Author != \"\" {\n")
	b.WriteString("\t\t\tentity.Author = " + entityLower + "Author\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif " + entityLower + "Visit != \"\" {\n")
	b.WriteString("\t\t\tentity.Visit = " + entityLower + "Visit\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tif err := h.UpdateCLI(ctx, entity); err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"update " + entityName + " failed: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tlog.Printf(\"✓ " + entityName + " updated: ID=%d\", id)\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("// " + entityName + "DeleteCmd 删除 " + entityName + "\n")
	b.WriteString("var " + entityName + "DeleteCmd = &cobra.Command{\n")
	b.WriteString("\tUse:   \"delete [id]\",\n")
	b.WriteString("\tShort: \"删除 " + entityName + "\",\n")
	b.WriteString("\tLong:  \"根据 ID 删除 " + entityName + " 记录\",\n")
	b.WriteString("\tArgs:  cobra.ExactArgs(1),\n")
	b.WriteString("\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\tctx := context.Background()\n")
	b.WriteString("\t\trepo := repository.NewRepository(database.DB)\n")
	b.WriteString("\t\tsvc := service.NewService(repo)\n")
	b.WriteString("\t\th := handler.NewCLIHandler(svc)\n\n")
	b.WriteString("\t\tid, err := strconv.ParseInt(args[0], 10, 64)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"invalid ID: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tif err := h.DeleteCLI(ctx, int(id)); err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"delete " + entityName + " failed: %w\", err)\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tlog.Printf(\"✓ " + entityName + " deleted: %d\", id)\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("func init() {\n")
	b.WriteString("\t// 注册子命令\n")
	b.WriteString("\t" + entityName + "Cmd.AddCommand(" + entityName + "CreateCmd)\n")
	b.WriteString("\t" + entityName + "Cmd.AddCommand(" + entityName + "GetCmd)\n")
	b.WriteString("\t" + entityName + "Cmd.AddCommand(" + entityName + "ListCmd)\n")
	b.WriteString("\t" + entityName + "Cmd.AddCommand(" + entityName + "UpdateCmd)\n")
	b.WriteString("\t" + entityName + "Cmd.AddCommand(" + entityName + "DeleteCmd)\n\n")
	b.WriteString("\t// 添加标志给 create\n")
	b.WriteString("\t" + entityName + "CreateCmd.Flags().StringVar(&" + entityLower + "Title, \"title\", \"\", \"标题 (required)\")\n")
	b.WriteString("\t" + entityName + "CreateCmd.Flags().StringVar(&" + entityLower + "Author, \"author\", \"\", \"作者\")\n")
	b.WriteString("\t" + entityName + "CreateCmd.Flags().StringVar(&" + entityLower + "Visit, \"visit\", \"\", \"访问次数\")\n")
	b.WriteString("\t" + entityName + "CreateCmd.MarkFlagRequired(\"title\")\n\n")
	b.WriteString("\t// 添加标志给 update\n")
	b.WriteString("\t" + entityName + "UpdateCmd.Flags().StringVar(&" + entityLower + "Title, \"title\", \"\", \"新标题\")\n")
	b.WriteString("\t" + entityName + "UpdateCmd.Flags().StringVar(&" + entityLower + "Author, \"author\", \"\", \"新作者\")\n")
	b.WriteString("\t" + entityName + "UpdateCmd.Flags().StringVar(&" + entityLower + "Visit, \"visit\", \"\", \"新访问次数\")\n\n")
	b.WriteString("\t// 将此命令添加到 root 命令\n")
	b.WriteString("\trootCmd.AddCommand(" + entityName + "Cmd)\n")
	b.WriteString("}\n")

	content := b.String()

	// 写入文件
	dstFile := filepath.Join(cmdDir, entityLower+"Cmd.go")
	if err := os.WriteFile(dstFile, []byte(content), 0o644); err != nil {
		return err
	}

	return nil
}

// toCamelCaseUpper 将下划线分隔的表名转换为大驼峰（用于 Go 结构体名称）
// 例如: eb_article -> EbArticle
func toCamelCaseUpper(s string) string {
	parts := strings.Split(s, "_")
	result := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		// 首字母大写
		result += strings.ToUpper(part[:1]) + part[1:]
	}
	return result
}
