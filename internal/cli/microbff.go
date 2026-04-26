package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// micro-bff 命令参数
var (
	microBffName       string
	microBffOutputDir  string
	microBffModules    []string
	microBffMiddleware string
	microBffHTTP       string
	microBffDBHost     string
	microBffDBPort     string
	microBffDBUser     string
	microBffDBPassword string
	microBffDBName     string
)

var newMicroBffCmd = &cobra.Command{
	Use:   "micro-bff",
	Short: "添加 BFF 到已有微服务项目",
	Long: `向已有微服务项目添加新的 BFF 层

示例：
  # 添加 BFF
  gpx new micro-bff \
    --name h5 \
    --output ./myShop \
    --modules article,user

  # 指定数据库
  gpx new micro-bff \
    --name web \
    --output ./myShop \
    --modules article,user \
    --db-host 127.0.0.1 \
    --db-port 3306`,
	RunE: runNewMicroBff,
}

func runNewMicroBff(cmd *cobra.Command, args []string) error {
	// 参数验证
	if microBffOutputDir == "" {
		return fmt.Errorf("--output is required")
	}
	if microBffMiddleware == "" {
		return fmt.Errorf("--middleware is required")
	}

	// 如果没有指定 --name，从输出目录推断
	if microBffName == "" {
		microBffName = filepath.Base(microBffOutputDir)
	}

	// --modules 在没有指定 --middleware 时是必需的
	if len(microBffModules) == 0 && microBffMiddleware == "" {
		return fmt.Errorf("--modules is required (or use --middleware to generate only middleware)")
	}

	// 如果目录不存在且没有指定 middleware，则报错
	// 如果只生成 middleware，目录不存在时可以创建
	if _, err := os.Stat(microBffOutputDir); os.IsNotExist(err) {
		if microBffMiddleware == "" {
			return fmt.Errorf("project directory not found: %s", microBffOutputDir)
		}
		// 只生成 middleware 时创建目录
		if err := os.MkdirAll(microBffOutputDir, 0755); err != nil {
			return err
		}
	}

	// 获取项目名（从输出目录推断）
	projectName := filepath.Base(microBffOutputDir)

	// 生成中间件模式（只需要 middleware 和基础结构）
	if microBffMiddleware != "" && len(microBffModules) == 0 {
		// 如果没有指定 --name，从 output 路径推断
		if microBffName == "" {
			baseName := filepath.Base(microBffOutputDir)
			// 如果目录名已经是 bff_xxx 格式，提取 xxx 作为 bffName
			if strings.HasPrefix(baseName, "bff_") {
				microBffName = baseName[4:] // 去掉 "bff_" 前缀
			} else {
				microBffName = baseName
			}
		}

		// 如果 output 路径包含 bff_ 前缀，说明用户指定的是 BFF 目录
		// 否则 output 是项目根目录，BFF 在 bff_{name} 子目录
		var bffDir, projectRoot string
		baseName := filepath.Base(microBffOutputDir)
		if strings.HasPrefix(baseName, "bff_") {
			// output 是 BFF 目录
			bffDir = microBffOutputDir
			projectRoot = filepath.Dir(microBffOutputDir)
		} else if baseName == microBffName {
			// output 目录名等于 bffName，说明用户想直接输出到 BFF 目录
			bffDir = microBffOutputDir
			projectRoot = filepath.Dir(microBffOutputDir)
		} else {
			// output 是项目根目录
			projectRoot = microBffOutputDir
			bffDir = filepath.Join(microBffOutputDir, "bff_"+microBffName)
		}

		projectName := filepath.Base(projectRoot)

		fmt.Printf("🎯 Generating BFF %s with middleware only...\n", microBffName)
		fmt.Printf("   HTTP: %s\n", microBffHTTP)
		fmt.Printf("   Middleware: %s\n", microBffMiddleware)
		fmt.Printf("   Project: %s\n", projectName)
		fmt.Printf("   BFF: %s\n", bffDir)

		// 创建目录
		bffMiddlewareDir := filepath.Join(bffDir, "internal", "middleware")
		cmdDir := filepath.Join(bffDir, "cmd")
		if err := os.MkdirAll(bffMiddlewareDir, 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(cmdDir, 0755); err != nil {
			return err
		}

		// 生成 go.mod（在项目根目录）
		goModContent := fmt.Sprintf("module %s\n\ngo 1.26\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.9.1\n\tgo.uber.org/zap v1.26.0\n)\n", projectName)
		if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte(goModContent), 0644); err != nil {
			fmt.Printf("WARNING: generate go.mod failed: %v\n", err)
		} else {
			fmt.Printf("  Generated go.mod\n")
		}

		// 生成 pkg 层（在项目根目录）
		genBffPkgMiddleware(projectRoot)

		// 生成中间件包（在 BFF 的 internal/middleware 下）
		genBffMiddlewarePackages(bffDir)

		// 生成 cmd/main.go
		mainContent := fmt.Sprintf(`package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"%s/pkg/config"
	"%s/pkg/logger"
	"%s/internal/router"
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
	if cfg == nil {
		log.Fatal("Config is nil - check if config file exists: " + confPath)
	}

	logCfg, err := logger.LoadConfig(confPath)
	if err != nil {
		log.Fatal(err)
	}
	l, err := logger.NewLogger(logCfg)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Sync()

	logger.Business.Infow("BFF 服务启动", "host", cfg.Server.Host, "port", cfg.Server.Port)

	addr := fmt.Sprintf("%%s:%%d", cfg.Server.Host, cfg.Server.Port)
	logger.Business.Infow("BFF listening", "addr", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router.NewRouter(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error.Errorw("listen error", "error", err.Error())
		}
	}()

	select {}
}
`, projectName, projectName, "bff_"+microBffName)
		if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainContent), 0644); err != nil {
			fmt.Printf("WARNING: generate main.go failed: %v\n", err)
		} else {
			fmt.Printf("  Generated cmd/main.go\n")
		}

		// 生成中间件
		if err := genBffMiddleware(bffDir, microBffName, projectName); err != nil {
			fmt.Printf("WARNING: generate middleware failed: %v\n", err)
		}

		fmt.Printf("\n✅ BFF %s middleware generated!\n\n", microBffName)
		fmt.Printf("📁 Project: %s\n", projectRoot)
		fmt.Printf("📁 BFF: %s\n", bffDir)
		fmt.Println("\n📝 Next steps:")
		fmt.Printf("   1. cd %s\n", projectRoot)
		fmt.Println("   2. go mod tidy")
		fmt.Printf("   3. Add route handlers in %s/internal/router/\n", filepath.Base(bffDir))
		fmt.Printf("   4. go run %s/cmd/main.go\n", filepath.Base(bffDir))
		return nil
	}

	// 完整 BFF 模式
	bffDir := filepath.Join(microBffOutputDir, "bff_"+microBffName)
	fmt.Printf("🎯 Adding BFF %s to project...\n", microBffName)
	fmt.Printf("   Modules: %v\n", microBffModules)

	// 创建 BFF 目录
	dirs := []string{
		filepath.Join(bffDir, "cmd"),
		filepath.Join(bffDir, "configs"),
		filepath.Join(bffDir, "internal", "dto"),
		filepath.Join(bffDir, "internal", "handler"),
		filepath.Join(bffDir, "internal", "middleware"),
		filepath.Join(bffDir, "internal", "rpc_client"),
		filepath.Join(bffDir, "internal", "service"),
		filepath.Join(bffDir, "internal", "router"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// 创建 BFF 文件
	if err := createBFFInExistingProject(bffDir, microBffName, projectName, microBffModules); err != nil {
		return fmt.Errorf("create BFF files failed: %w", err)
	}

	// 生成中间件（如果指定了 --middleware）
	if microBffMiddleware != "" {
		if err := genBffMiddleware(bffDir, microBffName, projectName); err != nil {
			fmt.Printf("WARNING: generate middleware failed: %v\n", err)
		}
	}

	fmt.Printf("\n✅ BFF %s added successfully!\n\n", microBffName)
	fmt.Printf("📁 BFF directory: %s\n", bffDir)
	fmt.Println("\n📝 Next steps:")
	fmt.Printf("   1. cd %s\n", microBffOutputDir)
	fmt.Println("   2. go mod tidy")
	fmt.Println("   3. Update router/router.go to register routes")
	fmt.Println("   4. go run bff_" + microBffName + "/cmd/main.go")

	return nil
}

// createBFFInExistingProject 在已有项目中创建 BFF
func createBFFInExistingProject(bffDir string, bffName string, projectName string, modules []string) error {
	// main.go
	mainContent := fmt.Sprintf(`package main

import (
	"flag"
	"fmt"
	"log"

	"%s/pkg/config"
	"%s/bff_%s/internal/router"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file")
}

func main() {
	flag.Parse()
	cfg, err := config.Load(confPath)
	if err != nil {
		log.Fatalf("Load config failed: %%v", err)
	}

	addr := fmt.Sprintf("%%s:%%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("BFF %s starting on %%s", addr)

	r := router.NewRouter()
	if err := r.Run(addr); err != nil {
		log.Fatalf("Start server failed: %%v", err)
	}
}
`, projectName, projectName, bffName, bffName)
	if err := os.WriteFile(filepath.Join(bffDir, "cmd", "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// config.yaml
	configContent := `server:
  host: 0.0.0.0
  port: 8080

registry:
  type: direct
  addr: localhost:2379

log:
  level: info
  format: json
`
	if err := os.WriteFile(filepath.Join(bffDir, "configs", "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	// router.go
	routerContent := fmt.Sprintf(`package router

import (
	"%s/bff_%s/internal/handler"

	"github.com/gin-gonic/gin"
)

// Router 路由管理器
type Router struct {
	engine *gin.Engine
}

func NewRouter() *Router {
	r := &Router{
		engine: gin.Default(),
	}
	r.registerRoutes()
	return r
}

func (r *Router) registerRoutes() {
`, projectName, bffName)
	for _, module := range modules {
		moduleUpper := strings.ToUpper(module[:1]) + module[1:]
		routerContent += fmt.Sprintf("\th%[1]s := handler.New%[1]sHandler()\n", moduleUpper)
		routerContent += fmt.Sprintf("\tv1 := r.engine.Group(\"/api/v1\")\n")
		routerContent += fmt.Sprintf("\t{\n")
		routerContent += fmt.Sprintf("\t\tv1.POST(\"/%[1]ss\", h%[2]s.Create)\n", module, moduleUpper)
		routerContent += fmt.Sprintf("\t\tv1.GET(\"/%[1]ss\", h%[2]s.List)\n", module, moduleUpper)
		routerContent += fmt.Sprintf("\t\tv1.GET(\"/%[1]ss/:id\", h%[2]s.Get)\n", module, moduleUpper)
		routerContent += fmt.Sprintf("\t\tv1.PUT(\"/%[1]ss/:id\", h%[2]s.Update)\n", module, moduleUpper)
		routerContent += fmt.Sprintf("\t\tv1.DELETE(\"/%[1]ss/:id\", h%[2]s.Delete)\n", module, moduleUpper)
		routerContent += fmt.Sprintf("\t}\n\n")
	}
	routerContent += `}

// Run 启动服务器
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
`
	if err := os.WriteFile(filepath.Join(bffDir, "internal", "router", "router.go"), []byte(routerContent), 0644); err != nil {
		return err
	}

	// middleware
	middleware := `package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		log.Printf("[%s] %s %d %v", method, path, status, latency)
	}
}

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}
`
	if err := os.WriteFile(filepath.Join(bffDir, "internal", "middleware", "middleware.go"), []byte(middleware), 0644); err != nil {
		return err
	}

	// 为每个模块创建 handler, rpc_client, dto
	for _, module := range modules {
		moduleUpper := strings.ToUpper(module[:1]) + module[1:]

		// gRPC client
		grpcClient := fmt.Sprintf(`package rpc_client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"%s/common/kitex_gen/%s"
)

var (
	client *%sClient
	once   sync.Once
)

// %sClient %s 服务 RPC 客户端
type %sClient struct {
	conn   *grpc.ClientConn
	client %s.%sServiceClient
}

// New%sClient 创建 %s RPC 客户端
func New%sClient(addr string) (*%sClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %%s failed: %%w", addr, err)
	}

	return &%sClient{
		conn:   conn,
		client: %s.New%sServiceClient(conn),
	}, nil
}

// Close 关闭连接
func (c *%sClient) Close() error {
	return c.conn.Close()
}

// GetClient 获取单例客户端
func GetClient() *%sClient {
	return client
}

// InitClient 初始化单例客户端
func InitClient(addr string) error {
	var err error
	once.Do(func() {
		client, err = New%sClient(addr)
	})
	return err
}
`,
			projectName, module,     // import: %s/common/kitex_gen/%s
			moduleUpper,             // var: *%sClient
			moduleUpper, module,     // comment: %sClient %s
			moduleUpper,             // type: %sClient struct
			module, moduleUpper,     // client: %s.%sServiceClient
			moduleUpper, module,     // New%sClient 创建 %s
			moduleUpper, moduleUpper, // func New%sClient ... *%sClient
			moduleUpper,             // &%sClient
			module, moduleUpper,     // %s.New%sServiceClient
			moduleUpper,             // %sClient Close
			moduleUpper,             // %sClient GetClient
			moduleUpper,             // New%sClient in InitClient
		)
		if err := os.WriteFile(filepath.Join(bffDir, "internal", "rpc_client", module+"_client.go"), []byte(grpcClient), 0644); err != nil {
			return err
		}

		// Handler
		handler := fmt.Sprintf(`package handler

import (
	"net/http"

	"%s/bff_%s/internal/rpc_client"

	"github.com/gin-gonic/gin"
)

// %sHandler %s HTTP 处理器
type %sHandler struct{}

// New%sHandler 创建 Handler
func New%sHandler() *%sHandler {
	return &%sHandler{}
}

// Create 创建
func (h *%sHandler) Create(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "create %s"})
}

// List 获取列表
func (h *%sHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "list %ss"})
}

// Get 获取单个
func (h *%sHandler) Get(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// Update 更新
func (h *%sHandler) Update(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// Delete 删除
func (h *%sHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}
`,
			projectName, bffName,   // import: %s/bff_%s/internal/rpc_client
			moduleUpper, module,    // comment: %sHandler %s
			moduleUpper,            // type: %sHandler struct
			moduleUpper,            // New%sHandler
			moduleUpper, moduleUpper, // func New%sHandler *%sHandler
			moduleUpper,            // &%sHandler{}
			moduleUpper,            // (h *%sHandler) Create
			module,                 // "create %s"
			moduleUpper,            // (h *%sHandler) List
			module,                 // "list %ss"
			moduleUpper,            // (h *%sHandler) Get
			moduleUpper,            // (h *%sHandler) Update
			moduleUpper,            // (h *%sHandler) Delete
		)
		if err := os.WriteFile(filepath.Join(bffDir, "internal", "handler", module+"_handler.go"), []byte(handler), 0644); err != nil {
			return err
		}

		// DTO
		dto := fmt.Sprintf(`package dto

// %sCreateReq 创建请求
type %sCreateReq struct {
	Name string `+"`"+`json:"name" binding:"required"`+"`"+` 
}

// %sUpdateReq 更新请求
type %sUpdateReq struct {
	Name string `+"`"+`json:"name"`+"`"+`
}

// %sResp 响应
type %sResp struct {
	ID   int64  `+"`"+`json:"id"`+"`"+`
	Name string `+"`"+`json:"name"`+"`"+`
}
`, moduleUpper, moduleUpper, moduleUpper, moduleUpper, moduleUpper, moduleUpper)
		if err := os.WriteFile(filepath.Join(bffDir, "internal", "dto", module+"_dto.go"), []byte(dto), 0644); err != nil {
			return err
		}
	}

	return nil
}

// genBffMiddleware 为已有项目的 BFF 生成中间件
func genBffMiddleware(bffDir, bffName, projectName string) error {
	fmt.Println("Generating BFF middleware...")

	middlewareList := strings.Split(microBffMiddleware, ",")
	for i := range middlewareList {
		middlewareList[i] = strings.TrimSpace(middlewareList[i])
	}

	bffMiddlewareDir := filepath.Join(bffDir, "internal", "middleware")
	os.MkdirAll(bffMiddlewareDir, 0755)

	// 生成 middleware.go (Builder 入口)
	middlewareBuilderPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "middleware.go.tmpl")
	if _, err := os.Stat(middlewareBuilderPath); err == nil {
		middlewareBuilderStr, err := os.ReadFile(middlewareBuilderPath)
		if err != nil {
			fmt.Printf("ERROR reading middleware builder template: %v\n", err)
		} else {
			middlewareGo, err := executeTemplate(string(middlewareBuilderStr), map[string]interface{}{
				"AppName": projectName,
				"BFFName": bffName,
			})
			if err != nil {
				fmt.Printf("ERROR executing middleware builder template: %v\n", err)
			} else {
				os.WriteFile(filepath.Join(bffMiddlewareDir, "middleware.go"), []byte(middlewareGo), 0644)
				fmt.Printf("  Generated BFF middleware builder: %s/internal/middleware/middleware.go\n", filepath.Base(bffDir))
			}
		}
	}

	// 根据 --middleware 生成对应的适配器
	for _, m := range middlewareList {
		var tmplPath string
		var outputFile string

		switch m {
		case "jwt":
			if microBffHTTP == "hertz" {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "hertz_jwt.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "hertz_jwt.go")
			} else {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "gin_jwt.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "gin_jwt.go")
			}
		case "ratelimit":
			if microBffHTTP == "hertz" {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "hertz_ratelimit.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "hertz_ratelimit.go")
			} else {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "gin_ratelimit.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "gin_ratelimit.go")
			}
		case "blacklist":
			if microBffHTTP == "hertz" {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "hertz_blacklist.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "hertz_blacklist.go")
			} else {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "gin_blacklist.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "gin_blacklist.go")
			}
		default:
			fmt.Printf("  Unknown middleware: %s\n", m)
			continue
		}

		tmplStr, err := os.ReadFile(tmplPath)
		if err != nil {
			fmt.Printf("ERROR reading BFF middleware template %s: %v\n", tmplPath, err)
			continue
		}

		middlewareGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
			"AppName": projectName,
			"BFFName": bffName,
		})
		if err != nil {
			fmt.Printf("ERROR executing BFF middleware template: %v\n", err)
			continue
		}
		os.WriteFile(outputFile, []byte(middlewareGo), 0644)
		fmt.Printf("  Generated BFF %s middleware: %s\n", m, outputFile)
	}

	// 生成 middleware.yaml 配置文件
	configTmplPath := filepath.Join(getTemplatesDir(), "configs", "middleware.yaml.tmpl")
	if _, err := os.Stat(configTmplPath); err == nil {
		configStr, err := os.ReadFile(configTmplPath)
		if err == nil {
			configContent, err := executeTemplate(string(configStr), map[string]interface{}{
				"AppName": projectName,
			})
			if err == nil {
				configDir := filepath.Join(bffDir, "configs")
				os.MkdirAll(configDir, 0755)
				os.WriteFile(filepath.Join(configDir, "middleware.yaml"), []byte(configContent), 0644)
				fmt.Printf("  Generated BFF middleware config: %s/configs/middleware.yaml\n", filepath.Base(bffDir))
			}
		}
	}

	return nil
}

// genBffMiddlewarePackages 生成 BFF 依赖的中间件包（放在 internal/middleware 下）
func genBffMiddlewarePackages(bffDir string) {
	middlewareDir := filepath.Join(bffDir, "internal", "middleware")

	for _, m := range strings.Split(microBffMiddleware, ",") {
		m = strings.TrimSpace(m)
		var pkgDir string
		switch m {
		case "jwt":
			pkgDir = filepath.Join(middlewareDir, "jwt")
		case "ratelimit":
			pkgDir = filepath.Join(middlewareDir, "ratelimit")
		case "blacklist":
			pkgDir = filepath.Join(middlewareDir, "blacklist")
		default:
			continue
		}
		os.MkdirAll(pkgDir, 0755)

		srcDir := filepath.Join(getTemplatesDir(), "pkg", m)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}

		entries, _ := os.ReadDir(srcDir)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(srcDir, entry.Name())
			dst := filepath.Join(pkgDir, strings.TrimSuffix(entry.Name(), ".tmpl"))
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			os.WriteFile(dst, data, 0644)
		}
		fmt.Printf("  Generated internal/middleware/%s\n", m)
	}
}

// genBffPkgMiddleware 生成 BFF 依赖的 pkg 层（logger 在根目录）
func genBffPkgMiddleware(projectRoot string) {
	// 生成 logger 包（main.go 依赖）
	genBffPkgLogger(projectRoot)
}

// genBffPkgLogger 生成 pkg/logger 包
func genBffPkgLogger(projectRoot string) {
	pkgLoggerDir := filepath.Join(projectRoot, "pkg", "logger")
	os.MkdirAll(pkgLoggerDir, 0755)

	srcDir := filepath.Join(getTemplatesDir(), "pkg", "logger")
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		fmt.Printf("Warning: logger templates not found\n")
		return
	}

	tmplFiles := []string{
		"config.go.tmpl", "logger.go.tmpl", "rotation.go.tmpl", "cleaner.go.tmpl",
		"sampler.go.tmpl", "metrics.go.tmpl", "context.go.tmpl", "business.go.tmpl",
		"access.go.tmpl", "audit.go.tmpl", "error.go.tmpl", "formatter.go.tmpl",
		"mq.go.tmpl", "mq_kafka.go.tmpl",
	}
	goFiles := []string{
		"config.go", "logger.go", "rotation.go", "cleaner.go",
		"sampler.go", "metrics.go", "context.go", "business.go",
		"access.go", "audit.go", "error.go", "formatter.go",
		"mq.go", "mq_kafka.go",
	}

	for i, tmpl := range tmplFiles {
		src := filepath.Join(srcDir, tmpl)
		dst := filepath.Join(pkgLoggerDir, goFiles[i])
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		os.WriteFile(dst, data, 0644)
	}
	fmt.Printf("  Generated pkg/logger\n")
}

func init() {
	newMicroBffCmd.Flags().StringVar(&microBffName, "name", "", "BFF 名称（必填）")
	newMicroBffCmd.Flags().StringVarP(&microBffOutputDir, "output", "o", "", "项目目录（必填）")
	newMicroBffCmd.Flags().StringArrayVar(&microBffModules, "modules", nil, "微服务列表（必填）")
	newMicroBffCmd.Flags().StringVar(&microBffMiddleware, "middleware", "", "中间件列表（jwt,ratelimit,blacklist）")
	newMicroBffCmd.Flags().StringVar(&microBffHTTP, "http", "gin", "HTTP 框架（gin/hertz）")
	newMicroBffCmd.Flags().StringVar(&microBffDBHost, "db-host", "127.0.0.1", "数据库主机")
	newMicroBffCmd.Flags().StringVar(&microBffDBPort, "db-port", "3306", "数据库端口")
	newMicroBffCmd.Flags().StringVar(&microBffDBUser, "db-user", "root", "数据库用户")
	newMicroBffCmd.Flags().StringVar(&microBffDBPassword, "db-password", "123456", "数据库密码")
	newMicroBffCmd.Flags().StringVar(&microBffDBName, "db-name", "gospacex", "数据库名称")

	_ = newMicroBffCmd.MarkFlagRequired("output")
}

// GetMicroBffCmd 返回 micro-bff 命令
func GetMicroBffCmd() *cobra.Command {
	return newMicroBffCmd
}
