package cli

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os/exec"
	"runtime"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"github.com/pelletier/go-toml/v2"
)

var (
	microAppName       string
	microAppOutputDir  string
	microAppBFFName    string
	microAppModules    []string
	microAppStyle      string
	microAppIDL        string
	microAppProtocol   string
	microAppHTTP       string
	microAppDBHost     string
	microAppDBPort     string
	microAppDBUser     string
	microAppDBPassword string
	microAppDBName     string
	microAppDBTable    string
	microAppTest       bool
	microAppOtel       bool
	microAppRegister   string
	microAppSrvs       []string
	microAppJoinKey       []string // 联表条件: table1.field1=table2.field2
	microAppJoinStyle     []string // 联表关系: table1:table2=1t1|1tn|nt1
	microAppMiddleware    string   // 中间件列表: jwt,ratelimit,blacklist
	microAppConfig        string   // 配置中心: nacos|viper(默认)
	microAppConfigFile    string   // 配置文件路径: yaml/json/toml
)

// toCamelCaseFile converts table name to camelCase file name
// e.g., "eb_store_product" -> "storeProduct"
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

var newMicroAppCmd = &cobra.Command{
	Use:     "micro-app",
	Aliases: []string{"micro"},
	Short:   "Create micro-app project (BFF + Microservices)",
	Long:    "Create a complete micro-app project with BFF layer and multiple microservices.",
	RunE:    runNewMicroApp,
}

func runNewMicroApp(cmd *cobra.Command, args []string) error {
	// 检测是否进入交互式模式
	if detectInteractiveMode() {
		cfg, err := runInteractiveMode()
		if err != nil {
			return err
		}
		// 将交互式配置转换为命令行变量
		if err := applyInteractiveConfig(cfg); err != nil {
			return err
		}
	}
	// 执行原有逻辑
	return runNewMicroAppWithFlags()
}

// ColumnInfo 表字段信息
type ColumnInfo struct {
	Name       string // 字段名
	Type       string // MySQL 类型
	ProtoType  string // Proto 类型
	ProtoIndex int    // Proto 字段编号
	IsPrimary  bool   // 是否主键
	IsNullable bool   // 是否可空
	Comment    string // 字段注释
}

// TableInfo 表信息，包含表名和字段
type TableInfo struct {
	TableName string       // 表名
	Columns   []ColumnInfo // 字段列表
}

// ModuleTables module -> 多张表的映射
type ModuleTables map[string][]TableInfo

// JoinConfig 联表查询配置
type JoinConfig struct {
	LeftTable  string // 左表名
	LeftField  string // 左表字段
	RightTable string // 右表名
	RightField string // 右表字段
	Style      string // 1t1, 1tn, nt1
}

// ConfigFileConfig 微应用配置文件的结构，支持 YAML/JSON/TOML 三种格式
type ConfigFileConfig struct {
	Name     string   `yaml:"name" json:"name" toml:"name"`
	Output   string   `yaml:"output" json:"output" toml:"output"`
	BFF      string   `yaml:"bff" json:"bff" toml:"bff"`
	Modules  []string `yaml:"modules" json:"modules" toml:"modules"`
	Database struct {
		Host     string   `yaml:"host" json:"host" toml:"host"`
		Port     int      `yaml:"port" json:"port" toml:"port"`
		User     string   `yaml:"user" json:"user" toml:"user"`
		Password string   `yaml:"password" json:"password" toml:"password"`
		Name     string   `yaml:"name" json:"name" toml:"name"`
		Tables   []string `yaml:"tables" json:"tables" toml:"tables"`
	} `yaml:"database" json:"database" toml:"database"`
}

// InteractiveConfig 交互式配置结果
type InteractiveConfig struct {
	Mode           string   // "default" or "diy"
	ProjectName     string   // 项目名称
	BFFName         string   // BFF 名称
	Modules         []string // 模块列表
	Style           string   // standard, ddd, serviceMesh
	IDLType         string   // proto, thrift
	StorageTypes    []string // sql, cache, nosql, es, mq
	RegistryEnabled bool     // 是否启用注册中心
	RegistryType    string   // nacos, consul, etcd, zookeeper, polaris
	RegistryAddr    string   // 注册中心地址
	ConfigEnabled   bool     // 是否启用配置中心
	ConfigType      string   // nacos, apollo, consul, etcd, zookeeper
	ConfigAddr      string   // 配置中心地址
	ConfigFormat    string   // yaml, json, properties
	SQLType         string   // mysql, pg
	SQLHost         string
	SQLPort         string
	SQLUser         string
	SQLPassword     string
	SQLDatabase     string
	SQLTables       []string
	CacheType       string   // redis, memcached, dragonfly, keydb
	CacheHost       string
	CachePort       string
	CachePassword   string
	CacheDB         string
	MQTypes         []string // rabbitmq, rocketmq, kafka, pulsar, redis-stream
	DTMType         string   // XA, TCC, saga, msg
	EnableTracing   bool
	EnableTest      bool
}

// parseConfigFile 解析配置文件，支持 yaml/json/toml 三种格式
func parseConfigFile(configPath string) (*ConfigFileConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	var cfg ConfigFileConfig
	ext := strings.ToLower(filepath.Ext(configPath))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse yaml config failed: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse json config failed: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse toml config failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (supported: yaml, json, toml)", ext)
	}

	return &cfg, nil
}

// readTableSchema 读取数据库表结构
func readTableSchema(host, port, user, password, dbName, tableName string) ([]ColumnInfo, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	// 查询表结构（同时取 DATA_TYPE 和 COLUMN_TYPE，COLUMN_TYPE 含精度如 tinyint(1)）
	query := `
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, COLUMN_KEY, IS_NULLABLE, COLUMN_COMMENT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	rows, err := db.Query(query, dbName, tableName)
	if err != nil {
		return nil, fmt.Errorf("查询表结构失败: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	protoIndex := 1
	for rows.Next() {
		var col ColumnInfo
		var colType, colTypeFull, colKey, nullable, comment sql.NullString
		if err := rows.Scan(&col.Name, &colType, &colTypeFull, &colKey, &nullable, &comment); err != nil {
			return nil, err
		}
		col.Type = colType.String
		col.IsPrimary = colKey.String == "PRI"
		col.IsNullable = nullable.String == "YES"
		col.Comment = comment.String
		col.ProtoType = mysqlTypeToProto(colType.String, colTypeFull.String, col.Name)
		col.ProtoIndex = protoIndex
		protoIndex++
		columns = append(columns, col)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("表 %s 不存在或没有字段", tableName)
	}

	return columns, nil
}

// mysqlTypeToProto 将 MySQL 类型转换为 Proto 类型
// colName: 字段名，用于检测 is_ 前缀的 bool 类字段（仅对 tinyint 类型有效）
func mysqlTypeToProto(mysqlType, colTypeFull, colName string) string {
	// 仅当列类型是 tinyint 且字段名为 bool 类命名时，才映射为 bool
	// 注意：不要只看命名，如 is_gift 列是 int 类型，不能转 bool
	if strings.ToLower(mysqlType) == "tinyint" && isBoolField(colName) {
		return "bool"
	}
	switch strings.ToLower(mysqlType) {
	case "int", "smallint", "mediumint", "bigint", "tinyint":
		return "int64"
	case "float", "double", "decimal":
		return "double"
	case "char", "varchar", "text", "longtext", "mediumtext", "tinytext":
		return "string"
	case "blob", "binary", "varbinary":
		return "bytes"
	case "datetime", "timestamp", "date", "time":
		return "string"
	case "bool", "boolean":
		return "bool"
	default:
		return "string"
	}
}

// detectInteractiveMode 检测是否进入交互式模式
// 无参数或参数只有 "micro" 时进入交互式模式
func detectInteractiveMode() bool {
	if len(os.Args) <= 2 {
		return true
	}
	return false
}

// runInteractiveMode 启动交互式配置模式
func runInteractiveMode() (*InteractiveConfig, error) {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    欢迎使用微应用生成器                              ║")
	fmt.Println("║              GoSpaceX Micro-App Generator v1.0                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	for {
		fmt.Println("请选择生成模式：")
		fmt.Println("  A. 默认标准微服（快速生成，使用默认配置）")
		fmt.Println("  B. 自定义配置（DIY 配置看板）")
		fmt.Println()
		fmt.Print("请输入选项 [A/B]: ")

		var choice string
		fmt.Scanln(&choice)
		choice = strings.TrimSpace(strings.ToUpper(choice))

		switch choice {
		case "A":
			return handleOptionA()
		case "B":
			return handleOptionB()
		default:
			fmt.Println("无效选项，请输入 A 或 B")
			fmt.Println()
		}
	}
}

// handleOptionA 默认模式处理
func handleOptionA() (*InteractiveConfig, error) {
	fmt.Println()
	fmt.Println("正在使用默认配置生成...")
	return &InteractiveConfig{
		Mode:            "default",
		ProjectName:    "myapp",
		BFFName:        "bff",
		Modules:        []string{"srv"},
		Style:          "standard",
		IDLType:        "proto",
		StorageTypes:   []string{"sql", "cache"},
		RegistryEnabled: false,
		ConfigEnabled:  false,
		CacheType:       "redis",
		EnableTest:     false,
	}, nil
}

// handleOptionB DIY 配置看板
func handleOptionB() (*InteractiveConfig, error) {
	cfg := &InteractiveConfig{
		Mode:           "diy",
		StorageTypes:  []string{},
		MQTypes:        []string{},
		SQLTables:      []string{},
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        DIY 微服务配置看板                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")

	// 阶段 1: 基础信息
	fmt.Println()
	fmt.Println("【阶段 1】基础信息")
	cfg.Style = selectOption("  微服务类型", []string{"standard", "ddd", "serviceMesh"}, "standard")
	cfg.IDLType = selectOption("  IDL 类型", []string{"proto", "thrift"}, "proto")

	// 阶段 2: 服务命名
	fmt.Println()
	fmt.Println("【阶段 2】服务命名")
	fmt.Print("  项目名称: ")
	fmt.Scanln(&cfg.ProjectName)
	if cfg.ProjectName == "" {
		cfg.ProjectName = "myapp"
	}
	fmt.Print("  BFF 名称 (默认: bff): ")
	var bffName string
	fmt.Scanln(&bffName)
	if bffName == "" {
		bffName = "bff"
	}
	cfg.BFFName = bffName
	cfg.Modules = inputModules()

	// 阶段 3: 数据存储选型
	fmt.Println()
	fmt.Println("【阶段 3】数据存储选型")
	fmt.Println("  可选: sql, cache, nosql, es, mq (多个用逗号分隔)")
	fmt.Print("  请选择: ")
	var storageInput string
	fmt.Scanln(&storageInput)
	if storageInput == "" {
		cfg.StorageTypes = []string{"sql", "cache"}
	} else {
		cfg.StorageTypes = strings.Split(strings.ToLower(storageInput), ",")
		for i := range cfg.StorageTypes {
			cfg.StorageTypes[i] = strings.TrimSpace(cfg.StorageTypes[i])
		}
	}

	// 阶段 4: 注册中心
	fmt.Println()
	fmt.Println("【阶段 4】注册中心")
	cfg.RegistryEnabled = selectYesNo("  启用注册中心")
	if cfg.RegistryEnabled {
		cfg.RegistryType = selectOption("  类型", []string{"nacos", "consul", "etcd", "zookeeper", "polaris"}, "nacos")
		fmt.Print("  连接地址 (默认: 127.0.0.1:8848): ")
		fmt.Scanln(&cfg.RegistryAddr)
		if cfg.RegistryAddr == "" {
			cfg.RegistryAddr = "127.0.0.1:8848"
		}
	}

	// 阶段 5: 配置中心
	fmt.Println()
	fmt.Println("【阶段 5】配置中心")
	cfg.ConfigEnabled = selectYesNo("  启用配置中心")
	if cfg.ConfigEnabled {
		cfg.ConfigType = selectOption("  类型", []string{"nacos", "apollo", "consul", "etcd", "zookeeper"}, "nacos")
		fmt.Print("  连接地址 (默认: 127.0.0.1:8848): ")
		fmt.Scanln(&cfg.ConfigAddr)
		if cfg.ConfigAddr == "" {
			cfg.ConfigAddr = "127.0.0.1:8848"
		}
		cfg.ConfigFormat = selectOption("  配置格式", []string{"yaml", "json", "properties"}, "yaml")
	}

	// 阶段 6: SQL 配置 (条件显示)
	if contains(cfg.StorageTypes, "sql") {
		fmt.Println()
		fmt.Println("【阶段 6】SQL 配置")
		cfg.SQLType = selectOption("  SQL 类型", []string{"mysql", "pg"}, "mysql")
		fmt.Print("  主机: ")
		fmt.Scanln(&cfg.SQLHost)
		if cfg.SQLHost == "" {
			cfg.SQLHost = "127.0.0.1"
		}
		fmt.Print("  端口 (默认: 3306): ")
		fmt.Scanln(&cfg.SQLPort)
		if cfg.SQLPort == "" {
			cfg.SQLPort = "3306"
		}
		fmt.Print("  用户 (默认: root): ")
		fmt.Scanln(&cfg.SQLUser)
		if cfg.SQLUser == "" {
			cfg.SQLUser = "root"
		}
		fmt.Print("  密码: ")
		fmt.Scanln(&cfg.SQLPassword)
		fmt.Print("  数据库名: ")
		fmt.Scanln(&cfg.SQLDatabase)
		fmt.Print("  表名 (多个用逗号分隔): ")
		var tablesInput string
		fmt.Scanln(&tablesInput)
		if tablesInput != "" {
			cfg.SQLTables = strings.Split(tablesInput, ",")
		}
	}

	// 阶段 7: Cache 配置 (条件显示)
	if contains(cfg.StorageTypes, "cache") {
		fmt.Println()
		fmt.Println("【阶段 7】Cache 配置")
		cfg.CacheType = selectOption("  Cache 类型", []string{"redis", "memcached", "dragonfly", "keydb"}, "redis")
		fmt.Print("  主机 (默认: 127.0.0.1): ")
		fmt.Scanln(&cfg.CacheHost)
		if cfg.CacheHost == "" {
			cfg.CacheHost = "127.0.0.1"
		}
		fmt.Print("  端口 (默认: 6379): ")
		fmt.Scanln(&cfg.CachePort)
		if cfg.CachePort == "" {
			cfg.CachePort = "6379"
		}
		fmt.Print("  密码: ")
		fmt.Scanln(&cfg.CachePassword)
		fmt.Print("  数据库编号 (默认: 0): ")
		var dbStr string
		fmt.Scanln(&dbStr)
		if dbStr == "" {
			dbStr = "0"
		}
		cfg.CacheDB = dbStr
	}

	// 阶段 8: MQ 配置 (条件显示)
	if contains(cfg.StorageTypes, "mq") {
		fmt.Println()
		fmt.Println("【阶段 8】MQ 配置")
		fmt.Println("  可选: rabbitmq, rocketmq, kafka, pulsar, redis-stream")
		fmt.Print("  请选择: ")
		var mqInput string
		fmt.Scanln(&mqInput)
		if mqInput != "" {
			cfg.MQTypes = strings.Split(strings.ToLower(mqInput), ",")
		}
	}

	// 阶段 9: 进阶特性
	fmt.Println()
	fmt.Println("【阶段 9】进阶特性")
	cfg.DTMType = selectOption("  DTM 分布式事务模式", []string{"XA", "TCC", "saga", "msg", "none"}, "none")
	cfg.EnableTracing = selectYesNo("  启用调用链")
	cfg.EnableTest = selectYesNo("  生成测试代码")

	// 确认
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Print("  确认生成？[Y/n]: ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) == "n" {
		fmt.Println("  重新配置...")
		return handleOptionB()
	}

	return cfg, nil
}

// selectOption 选择选项
func selectOption(prompt string, options []string, defaultVal string) string {
	fmt.Printf("  %s:\n", prompt)
	for i, opt := range options {
		fmt.Printf("    %d. %s\n", i+1, opt)
	}
	fmt.Printf("  请选择 [默认: %s]: ", defaultVal)
	var idxStr string
	fmt.Scanln(&idxStr)
	idxStr = strings.TrimSpace(idxStr)
	if idxStr == "" {
		return defaultVal
	}
	idx := 0
	fmt.Sscanf(idxStr, "%d", &idx)
	if idx > 0 && idx <= len(options) {
		return options[idx-1]
	}
	return defaultVal
}

// selectYesNo 是/否选择
func selectYesNo(prompt string) bool {
	fmt.Printf("  %s [y/N]: ", prompt)
	var choice string
	fmt.Scanln(&choice)
	return strings.ToLower(choice) == "y"
}

// inputModules 输入模块列表
func inputModules() []string {
	fmt.Print("  模块列表 (多个用逗号分隔): ")
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return []string{"product"}
	}
	modules := strings.Split(input, ",")
	for i := range modules {
		modules[i] = strings.TrimSpace(modules[i])
	}
	return modules
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// applyInteractiveConfig 将交互式配置转换为命令行变量
func applyInteractiveConfig(cfg *InteractiveConfig) error {
	// 设置命令行变量
	if cfg.ProjectName != "" {
		microAppName = cfg.ProjectName
	}
	if cfg.BFFName != "" {
		microAppBFFName = cfg.BFFName
	}
	if len(cfg.Modules) > 0 {
		microAppModules = cfg.Modules
	}
	microAppOutputDir = "output"
	if cfg.IDLType != "" {
		microAppIDL = cfg.IDLType
	}
	if cfg.Style != "" {
		microAppStyle = cfg.Style
	}
	microAppTest = cfg.EnableTest

	// SQL 配置
	microAppDBHost = cfg.SQLHost
	if cfg.SQLHost == "" {
		microAppDBHost = "127.0.0.1"
	}
	microAppDBPort = cfg.SQLPort
	if cfg.SQLPort == "" {
		microAppDBPort = "3306"
	}
	microAppDBUser = cfg.SQLUser
	if cfg.SQLUser == "" {
		microAppDBUser = "root"
	}
	microAppDBPassword = cfg.SQLPassword
	microAppDBName = cfg.SQLDatabase
	if len(cfg.SQLTables) > 0 {
		microAppDBTable = strings.Join(cfg.SQLTables, ",")
	}

	// 注册中心配置
	if cfg.RegistryEnabled && cfg.RegistryType != "" {
		microAppRegister = cfg.RegistryType
	}

	// 处理 modules 中英文逗号
	if len(microAppModules) > 0 {
		var normalizedModules []string
		for _, m := range microAppModules {
			m = strings.ReplaceAll(m, "，", ",")
			for _, part := range strings.Split(m, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					normalizedModules = append(normalizedModules, part)
				}
			}
		}
		microAppModules = normalizedModules
	}

	return nil
}

// runNewMicroAppWithFlags 在设置好变量后执行原有逻辑
func runNewMicroAppWithFlags() error {
	// 设置默认值
	if microAppName == "" {
		microAppName = "myapp"
	}
	if microAppOutputDir == "" {
		microAppOutputDir = "output"
	}
	if microAppBFFName == "" {
		microAppBFFName = "bff"
	}
	if len(microAppModules) == 0 {
		microAppModules = []string{"srv"}
	}

	// 如果指定了 --config-file，从配置文件加载参数
	if microAppConfigFile != "" {
		cfg, err := parseConfigFile(microAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}
		// 覆盖命令行参数
		if cfg.Name != "" {
			microAppName = cfg.Name
		}
		if cfg.Output != "" {
			microAppOutputDir = cfg.Output
		}
		if cfg.BFF != "" {
			microAppBFFName = cfg.BFF
		}
		if len(cfg.Modules) > 0 {
			microAppModules = cfg.Modules
		}
		if cfg.Database.Host != "" {
			microAppDBHost = cfg.Database.Host
		}
		if cfg.Database.Port > 0 {
			microAppDBPort = fmt.Sprintf("%d", cfg.Database.Port)
		}
		if cfg.Database.User != "" {
			microAppDBUser = cfg.Database.User
		}
		if cfg.Database.Password != "" {
			microAppDBPassword = cfg.Database.Password
		}
		if cfg.Database.Name != "" {
			microAppDBName = cfg.Database.Name
		}
		if len(cfg.Database.Tables) > 0 {
			// tables 可以有多个，用逗号分隔的字符串
			microAppDBTable = strings.Join(cfg.Database.Tables, ",")
		}
	}

	// 合并 --srvs 到 --modules（只有非默认值时才合并）
	// 如果 microAppModules 是默认值 ["srv"] 且用户指定了 --srvs，则用用户值替换
	if len(microAppSrvs) > 0 {
		// 检查 microAppModules 是否是默认值（只有 "srv" 一个元素的情况视为默认值）
		if len(microAppModules) == 1 && microAppModules[0] == "srv" {
			microAppModules = microAppSrvs
		} else {
			microAppModules = append(microAppModules, microAppSrvs...)
		}
	}

	// 支持中英文逗号分隔的 modules 参数
	if len(microAppModules) > 0 {
		var normalizedModules []string
		for _, m := range microAppModules {
			// 替换中文逗号为英文逗号
			m = strings.ReplaceAll(m, "，", ",")
			// 按英文逗号分割
			for _, part := range strings.Split(m, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					normalizedModules = append(normalizedModules, part)
				}
			}
		}
		microAppModules = normalizedModules
	}

	// 验证协议和 HTTP 框架
	if microAppProtocol != "grpc" && microAppProtocol != "kitex" {
		fmt.Printf("Warning: unknown protocol '%s', using 'grpc' as default\n", microAppProtocol)
		microAppProtocol = "grpc"
	}
	if microAppHTTP != "gin" && microAppHTTP != "hertz" {
		fmt.Printf("Warning: unknown HTTP framework '%s', using 'gin' as default\n", microAppHTTP)
		microAppHTTP = "gin"
	}

	// 验证服务注册中心参数
	if microAppRegister != "" && microAppRegister != "consul" && microAppRegister != "etcd" {
		return fmt.Errorf("invalid --register value '%s', must be consul or etcd", microAppRegister)
	}

	// 验证 IDL 参数
	if microAppIDL != "proto" && microAppIDL != "thrift" {
		fmt.Printf("Warning: unknown IDL type '%s', using 'proto' as default\n", microAppIDL)
		microAppIDL = "proto"
	}

	// 验证配置中心参数
	if microAppConfig != "" && microAppConfig != "nacos" && microAppConfig != "viper" {
		return fmt.Errorf("invalid --config value '%s', must be nacos or viper", microAppConfig)
	}

	// 验证联表查询参数（支持多组）
	var joinConfigs []*JoinConfig
	if len(microAppJoinKey) > 0 || len(microAppJoinStyle) > 0 {
		if len(microAppJoinKey) != len(microAppJoinStyle) {
			return fmt.Errorf("--db-join-condition 和 --db-join-style 数量不匹配（%d vs %d），必须一一对应", len(microAppJoinKey), len(microAppJoinStyle))
		}
		for i := range microAppJoinKey {
			// 解析 --db-join-style: "table1:table2=1t1"
			js := strings.TrimSpace(microAppJoinStyle[i])
			styleParts := strings.SplitN(js, "=", 2)
			if len(styleParts) != 2 {
				return fmt.Errorf("invalid --db-join-style format '%s', expected: table1:table2=1t1|1tn|nt1|ntn", js)
			}
			tables := strings.TrimSpace(styleParts[0])
			style := strings.TrimSpace(styleParts[1])
			if style != "1t1" && style != "1tn" && style != "nt1" && style != "ntn" {
				return fmt.Errorf("invalid join style '%s', must be 1t1（一对一）, 1tn（一对多）, nt1（多对一）, ntn（多对多）", style)
			}
			tableParts := strings.Split(tables, ":")
			if len(tableParts) != 2 {
				return fmt.Errorf("invalid --db-join-style table format '%s', expected: table1:table2", tables)
			}
			leftTable := strings.TrimSpace(tableParts[0])
			rightTable := strings.TrimSpace(tableParts[1])

			// 解析 --db-join-condition: "table1.field1=table2.field2"
			jk := strings.TrimSpace(microAppJoinKey[i])
			eqParts := strings.SplitN(jk, "=", 2)
			if len(eqParts) != 2 {
				return fmt.Errorf("invalid --db-join-condition format '%s', expected: table1.field1=table2.field2", jk)
			}
			left := strings.TrimSpace(eqParts[0])
			right := strings.TrimSpace(eqParts[1])
			leftFP := strings.Split(left, ".")
			rightFP := strings.Split(right, ".")
			if len(leftFP) != 2 || len(rightFP) != 2 {
				return fmt.Errorf("invalid --db-join-condition format '%s', expected: table1.field1=table2.field2", jk)
			}

			cfg := &JoinConfig{
				LeftTable:  leftTable,
				LeftField:  leftFP[1],
				RightTable: rightTable,
				RightField: rightFP[1],
				Style:      style,
			}
			joinConfigs = append(joinConfigs, cfg)
			fmt.Printf("Join config: %s.%s = %s.%s (%s)\n", cfg.LeftTable, cfg.LeftField, cfg.RightTable, cfg.RightField, cfg.Style)
		}
	}

	fmt.Printf("Creating micro-app: %s (protocol: %s, http: %s, idl: %s", microAppName, microAppProtocol, microAppHTTP, microAppIDL)
	if microAppRegister != "" {
		fmt.Printf(", register: %s", microAppRegister)
	}
	if microAppMiddleware != "" {
		fmt.Printf(", middleware: %s", microAppMiddleware)
	}
	if microAppConfig != "" {
		fmt.Printf(", config: %s", microAppConfig)
	}
	fmt.Println(")")

	// 如果指定了 --db-table，读取表结构
	hasDBTable := microAppDBTable != ""
	moduleTables := make(ModuleTables)           // module -> 多张表
	allTableColumns := make(map[string][]ColumnInfo) // tableName -> columns（包含所有表）
	var tableNames []string // 所有表名（提升到外层，供后续 aux model 和 join 使用）

	if hasDBTable {
		// 支持中英文逗号分隔的多表名
		microAppDBTable = strings.ReplaceAll(microAppDBTable, "，", ",")
		tableNames = strings.Split(microAppDBTable, ",")
		for i := range tableNames {
			tableNames[i] = strings.TrimSpace(tableNames[i])
		}
		fmt.Printf("Reading table schema from DB: %s.%v\n", microAppDBName, tableNames)
		for _, tableName := range tableNames {
			if tableName == "" {
				continue
			}
			columns, err := readTableSchema(microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName, tableName)
			if err != nil {
				return fmt.Errorf("读取表结构失败: %w", err)
			}
			fmt.Printf("Table %s has %d columns\n", tableName, len(columns))
			// 打印列信息
			for _, col := range columns {
				pk := ""
				if col.IsPrimary {
					pk = " (PK)"
				}
				fmt.Printf("  - %s: %s -> %s%s\n", col.Name, col.Type, col.ProtoType, pk)
			}
			// 所有表都存入 allTableColumns（以表名为 key）
			allTableColumns[tableName] = columns

			// 确定 module 名称
			var moduleName string
			if len(tableNames) == len(microAppModules) {
				// 一对一：按顺序对应
				moduleName = microAppModules[len(moduleTables)]
			} else if len(tableNames) > len(microAppModules) {
				// 多表归一：所有表都归入第一个 module
				moduleName = microAppModules[0]
			} else {
				// 表数少于 module 数：归入第一个 module
				moduleName = microAppModules[0]
			}

			// 添加到 moduleTables（每张表都添加，不跳过）
			ti := TableInfo{TableName: tableName, Columns: columns}
			moduleTables[moduleName] = append(moduleTables[moduleName], ti)
		}
	}

	projectDir := filepath.Join(microAppOutputDir, microAppName)

	// Create directories - BFF internal 不创建 dto 目录
	dirs := []string{
		filepath.Join(projectDir, toBffDirName(microAppBFFName), "cmd"),
		filepath.Join(projectDir, toBffDirName(microAppBFFName), "configs"),
		// NOTE: BFF internal 禁止创建 dto 目录
		filepath.Join(projectDir, toBffDirName(microAppBFFName), "internal", "handler"),
		filepath.Join(projectDir, toBffDirName(microAppBFFName), "internal", "middleware"),
		filepath.Join(projectDir, toBffDirName(microAppBFFName), "internal", "rpcClient"),
		filepath.Join(projectDir, toBffDirName(microAppBFFName), "internal", "router"),
		filepath.Join(projectDir, "common", "idl"),
		filepath.Join(projectDir, "common", "kitexGen"),
		filepath.Join(projectDir, "common", "errors"),
		filepath.Join(projectDir, "common", "constants"),
		filepath.Join(projectDir, "pkg", "config"),
		filepath.Join(projectDir, "pkg", "database"),
		filepath.Join(projectDir, "pkg", "logger"),
		filepath.Join(projectDir, "pkg", "utils"),
		filepath.Join(projectDir, "scripts"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}
	for _, m := range microAppModules {
		for _, d := range []string{
			filepath.Join(projectDir, toSrvDirName(m), "cmd"),
			filepath.Join(projectDir, toSrvDirName(m), "configs"),
			filepath.Join(projectDir, toSrvDirName(m), "internal", "handler"),
			filepath.Join(projectDir, toSrvDirName(m), "internal", "model"),
			filepath.Join(projectDir, toSrvDirName(m), "internal", "repository"),
			filepath.Join(projectDir, toSrvDirName(m), "internal", "service"),
		} {
			os.MkdirAll(d, 0755)
		}
	}

	// Generate files - 根据是否有表结构选择生成方式
	if hasDBTable {
		// 基于表结构生成
		if microAppIDL == "thrift" {
			genThriftFilesFromSchema(projectDir, moduleTables)
		} else {
			genProtoFilesFromSchema(projectDir, moduleTables)
		}
		genErrors(projectDir)
		genConstants(projectDir)
		genConfig(projectDir)
		genDatabase(projectDir)
		genLogger(projectDir)
		genUtils(projectDir)

		// 根据 HTTP 框架选择 BFF 生成函数
		if microAppHTTP == "hertz" {
			genBFFHertzFromSchema(projectDir, microAppBFFName, moduleTables, microAppProtocol)
		} else {
			genBFFFromSchema(projectDir, microAppBFFName, moduleTables, microAppProtocol)
		}

		// Step 1: 为每个 module 生成一次 main.go 和 config.yaml
		for i, m := range microAppModules {
			tables := moduleTables[m]
			port := 8001 + i
			genSrvMainAndConfig(projectDir, m, port, tables, tableNames)
		}

		// Step 2: 为每个 module 下的每张表生成 model/repo/service/handler 文件
		for _, m := range microAppModules {
			tables := moduleTables[m]
			for _, tbl := range tables {
				entityName := tableToEntityName(tbl.TableName, tableNames)
				port := 8001
				genSrvTableFiles(projectDir, m, tbl.Columns, tbl.TableName, entityName, tableNames, port)
			}
		}

		// 如果有联表配置，生成联表查询代码
		if len(joinConfigs) > 0 {
			for _, jc := range joinConfigs {
				genJoinCode(projectDir, microAppModules, jc, allTableColumns)
			}
		}
	} else {
		// 使用默认生成
		if microAppIDL == "thrift" {
			genThriftFiles(projectDir)
		} else {
			genProtoFiles(projectDir)
		}
		genErrors(projectDir)
		genConstants(projectDir)
		genConfig(projectDir)
		genDatabase(projectDir)
		genLogger(projectDir)
		genUtils(projectDir)

		// 根据 HTTP 框架选择 BFF 生成函数
		if microAppHTTP == "hertz" {
			genBFFHertz(projectDir, microAppBFFName, microAppModules, microAppProtocol)
		} else {
			genBFF(projectDir, microAppBFFName, microAppModules, microAppProtocol)
		}

		for i, m := range microAppModules {
			genMicroservice(projectDir, m, 8000+i+1, microAppProtocol)
		}

		// 如果有联表配置，生成联表查询代码
		if len(joinConfigs) > 0 {
			for _, jc := range joinConfigs {
				genJoinCode(projectDir, microAppModules, jc, nil)
			}
		}
	}
	// 生成中间件/拦截器（如果 --middleware 启用）
	if microAppMiddleware != "" {
		genMiddleware(projectDir, microAppBFFName, microAppModules)
	}

	// 生成配置中心支持（如果 --config=nacos 启用）
	if microAppConfig == "nacos" {
		genNacosConfig(projectDir, microAppBFFName, microAppModules)
	}

	// 生成测试代码（如果 --test 启用）
	if microAppTest {
		bffPort := 8080
		srvPort := 8001
		// 构建 tableColumns 和 moduleTableName（供测试代码使用）
		tableColumns := make(map[string][]ColumnInfo)
		moduleTableName := make(map[string]string)
		for m, tables := range moduleTables {
			if len(tables) > 0 {
				tableColumns[m] = tables[0].Columns
				moduleTableName[m] = tables[0].TableName
			}
		}
		genTestDirs(projectDir, microAppBFFName, microAppModules, hasDBTable, tableColumns, moduleTableName)
		genShellScripts(projectDir, microAppBFFName, microAppModules, bffPort, srvPort, moduleTableName)
	}

	genScripts(projectDir, moduleTables)
	genMakefile(projectDir, microAppModules)
	genReadme(projectDir, microAppBFFName, microAppModules)
	genGoMod(projectDir)

	runGenProtoScript(projectDir)

	genGitignore(projectDir)
	cleanHiddenFiles(projectDir)

	fmt.Printf("\nDone! Project at: %s\n", projectDir)
	return nil
}

func genProtoFiles(projectDir string) {
	for _, m := range microAppModules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "proto", "proto_header.go.tmpl")
		tmplBytes, err := os.ReadFile(tmplPath)
		if err != nil {
			fmt.Printf("Error reading proto template: %v\n", err)
			continue
		}
		result, err := executeTemplate(string(tmplBytes), map[string]interface{}{
			"Module":    m,
			"AppName":   microAppName,
			"UpperName": upper,
		})
		if err != nil {
			fmt.Printf("Error executing proto template: %v\n", err)
			continue
		}
		os.WriteFile(filepath.Join(projectDir, "common", "idl", m+".proto"), []byte(result), 0644)
	}
}

// =============================================================================
// R4: --idl thrift Thrift IDL 生成
// =============================================================================

// genThriftFiles 生成默认 Thrift IDL 文件（不含表结构）
func genThriftFiles(projectDir string) {
	for _, m := range microAppModules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		content := fmt.Sprintf(`namespace go %s.%s

struct %s {
    1: required i64 id,
    2: required string name,
    3: optional i32 status,
}

struct Create%sReq {
    1: required string name,
}

struct Create%sResp {
    1: required i64 id,
}

struct Get%sReq {
    1: required i64 id,
}

struct Get%sResp {
    1: required i64 id,
    2: required string name,
    3: optional i32 status,
}

struct List%sReq {}

struct List%sResp {
    1: required list<%s> items,
}

struct Update%sReq {
    1: required i64 id,
    2: optional string name,
}

struct Update%sResp {
    1: required bool success,
}

struct Delete%sReq {
    1: required i64 id,
}

struct Delete%sResp {
    1: required bool success,
}
`, microAppName, m, upper, upper, upper, upper, upper, upper, upper, upper, upper, upper, upper, upper)
		os.WriteFile(filepath.Join(projectDir, "common", "idl", m+".thrift"), []byte(content), 0644)
	}
}

// genThriftFilesFromSchema 基于表结构生成 Thrift IDL 文件
func genThriftFilesFromSchema(projectDir string, moduleTables ModuleTables) {
	for moduleName, tables := range moduleTables {
		for _, table := range tables {
			columns := table.Columns
			upper := strings.ToUpper(moduleName[:1]) + moduleName[1:]
			var buf strings.Builder

			thriftFileName := table.TableName
			if thriftFileName == "" {
				thriftFileName = moduleName
			}

			buf.WriteString(fmt.Sprintf("namespace go %s.%s\n\n", microAppName, thriftFileName))

			// 主结构体
			buf.WriteString(fmt.Sprintf("struct %s {\n", upper))
			idx := 1
			for _, col := range columns {
				thriftType := mysqlTypeToThrift(col.Type)
				required := "optional"
				if col.IsPrimary {
					required = "required"
				}
				comment := ""
				if col.Comment != "" {
					comment = " // " + col.Comment
				}
				buf.WriteString(fmt.Sprintf("    %d: %s %s %s,%s\n", idx, required, thriftType, getLowerFirst(col.Name), comment))
				idx++
			}
			buf.WriteString("}\n\n")

			pkField := getPrimaryKeyField(columns)
			pkThriftType := mysqlTypeToThrift(pkField.Type)
			pkLowerName := getLowerFirst(pkField.Name)

			// Create
			buf.WriteString(fmt.Sprintf("struct Create%sReq {\n", upper))
			cidx := 1
			for _, col := range columns {
				if col.IsPrimary && strings.Contains(strings.ToLower(col.Type), "int") {
					continue
				}
				thriftType := mysqlTypeToThrift(col.Type)
				buf.WriteString(fmt.Sprintf("    %d: required %s %s,\n", cidx, thriftType, getLowerFirst(col.Name)))
				cidx++
			}
			buf.WriteString("}\n\n")

			buf.WriteString(fmt.Sprintf("struct Create%sResp {\n    1: required %s %s,\n}\n\n", upper, pkThriftType, pkLowerName))

			// Get
			buf.WriteString(fmt.Sprintf("struct Get%sReq {\n    1: required %s %s,\n}\n\n", upper, pkThriftType, pkLowerName))
			buf.WriteString(fmt.Sprintf("struct Get%sResp {\n", upper))
			gidx := 1
			for _, col := range columns {
				thriftType := mysqlTypeToThrift(col.Type)
				required := "optional"
				if col.IsPrimary {
					required = "required"
				}
				buf.WriteString(fmt.Sprintf("    %d: %s %s %s,\n", gidx, required, thriftType, getLowerFirst(col.Name)))
				gidx++
			}
			buf.WriteString("}\n\n")

			// List
			buf.WriteString(fmt.Sprintf("struct List%sReq {}\n\n", upper))
			buf.WriteString(fmt.Sprintf("struct List%sResp {\n    1: required list<%s> items,\n}\n\n", upper, upper))

			// Update
			buf.WriteString(fmt.Sprintf("struct Update%sReq {\n    1: required %s %s,\n", upper, pkThriftType, pkLowerName))
			uidx := 2
			for _, col := range columns {
				if col.IsPrimary {
					continue
				}
				thriftType := mysqlTypeToThrift(col.Type)
				buf.WriteString(fmt.Sprintf("    %d: optional %s %s,\n", uidx, thriftType, getLowerFirst(col.Name)))
				uidx++
			}
			buf.WriteString("}\n\n")

			buf.WriteString(fmt.Sprintf("struct Update%sResp {\n    1: required bool success,\n}\n\n", upper))

			// Delete
			buf.WriteString(fmt.Sprintf("struct Delete%sReq {\n    1: required %s %s,\n}\n\n", upper, pkThriftType, pkLowerName))
			buf.WriteString(fmt.Sprintf("struct Delete%sResp {\n    1: required bool success,\n}\n\n", upper))

			// Service
			buf.WriteString(fmt.Sprintf(`service %sService {
    Create%sResp Create(1: Create%sReq req),
    Get%sResp Get(1: Get%sReq req),
    List%sResp List(1: List%sReq req),
    Update%sResp Update(1: Update%sReq req),
    Delete%sResp Delete(1: Delete%sReq req),
}
`, upper, upper, upper, upper, upper, upper, upper, upper, upper, upper, upper))

			os.WriteFile(filepath.Join(projectDir, "common", "idl", thriftFileName+".thrift"), []byte(buf.String()), 0644)
			fmt.Printf("  Generated Thrift IDL: common/idl/%s.thrift\n", thriftFileName)
		}
	}
}

// mysqlTypeToThrift 将 MySQL 类型转换为 Thrift 类型
func mysqlTypeToThrift(mysqlType string) string {
	switch strings.ToLower(mysqlType) {
	case "int", "tinyint", "smallint", "mediumint":
		return "i32"
	case "bigint":
		return "i64"
	case "float":
		return "float"
	case "double", "decimal":
		return "double"
	case "bool", "boolean":
		return "bool"
	case "blob", "binary", "varbinary":
		return "binary"
	default:
		return "string"
	}
}

func genErrors(projectDir string) {
	content := `package errors

import "fmt"

const (
	ErrCodeSuccess = 0
	ErrCodeNotFound = 1002
)

type BusinessError struct {
	Code int ` + "`" + `json:"code"` + "`" + `
	Message string ` + "`" + `json:"message"` + "`" + `
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func NewBusinessError(code int, msg string) *BusinessError {
	return &BusinessError{Code: code, Message: msg}
}
`
	os.WriteFile(filepath.Join(projectDir, "common", "errors", "error_code.go"), []byte(content), 0644)
}

func genConstants(projectDir string) {
	content := `package constants

const (
	StatusNormal = 0
	StatusDeleted = 1
)
`
	os.WriteFile(filepath.Join(projectDir, "common", "constants", "constants.go"), []byte(content), 0644)
}

func genConfig(projectDir string) {
	content := `package config

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig ` + "`" + `yaml:"server"` + "`" + `
	Database DatabaseConfig ` + "`" + `yaml:"database"` + "`" + `
	Registry RegistryConfig ` + "`" + `yaml:"registry"` + "`" + `
}

type ServerConfig struct {
	Host string ` + "`" + `yaml:"host"` + "`" + `
	Port int ` + "`" + `yaml:"port"` + "`" + `
}

type DatabaseConfig struct {
	Host string ` + "`" + `yaml:"host"` + "`" + `
	Port string ` + "`" + `yaml:"port"` + "`" + `
	User string ` + "`" + `yaml:"user"` + "`" + `
	Password string ` + "`" + `yaml:"password"` + "`" + `
	Database string ` + "`" + `yaml:"database"` + "`" + `
}

type RegistryConfig struct {
	Type    string ` + "`" + `yaml:"type"` + "`" + `
	Address string ` + "`" + `yaml:"address"` + "`" + `
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file not found: %s (error: %v)", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config file parse error: %v", err)
	}
	return &cfg, nil
}
`
	os.WriteFile(filepath.Join(projectDir, "pkg", "config", "config.go"), []byte(content), 0644)
}

func genDatabase(projectDir string) {
	content := `package database

import (
	"fmt"
	"` + microAppName + `/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
}
`
	os.WriteFile(filepath.Join(projectDir, "pkg", "database", "database.go"), []byte(content), 0644)
}

func genLogger(projectDir string) {
	// 确定模板目录（兼容 go run 和编译后的二进制）
	srcDir := "templates/pkg/logger"
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		execPath, _ := os.Executable()
		srcDir = filepath.Join(filepath.Dir(execPath), "templates", "pkg", "logger")
	}
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		fmt.Printf("Warning: log templates not found in %s\n", srcDir)
		return
	}
	dstDir := filepath.Join(projectDir, "pkg", "logger")
	os.MkdirAll(dstDir, 0755)

	tmplFiles := []string{
		"config.go.tmpl",
		"logger.go.tmpl",
		"rotation.go.tmpl",
		"cleaner.go.tmpl",
		"sampler.go.tmpl",
		"metrics.go.tmpl",
		"context.go.tmpl",
		"business.go.tmpl",
		"access.go.tmpl",
		"audit.go.tmpl",
		"error.go.tmpl",
		"mq.go.tmpl",
		"mq_kafka.go.tmpl",
	}

	goFiles := []string{
		"config.go",
		"logger.go",
		"rotation.go",
		"cleaner.go",
		"sampler.go",
		"metrics.go",
		"context.go",
		"business.go",
		"access.go",
		"audit.go",
		"error.go",
		"mq.go",
		"mq_kafka.go",
	}

	for i, tmpl := range tmplFiles {
		src := filepath.Join(srcDir, tmpl)
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Printf("Warning: skip log template %s: %v\n", tmpl, err)
			continue
		}
		os.WriteFile(filepath.Join(dstDir, goFiles[i]), data, 0644)
	}

	// 生成 log.yaml 配置文件
	configDir := filepath.Join(projectDir, "configs")
	os.MkdirAll(configDir, 0755)
	logYaml := `env: dev
level: info
sampling:
  initial: 100
  thereafter: 200
  tick: 1s
rotation:
  enabled: true
  max_age_days: 7
output:
  file: ./logs/app.log
  stdout: true
prometheus:
  enabled: false
  namespace: app
  subsystem: log
mq:
  enabled: false
  type: kafka
  brokers:
    - localhost:9092
  topic: app-logs
  async: true
  batch_size: 100
  flush_interval: 3s
`
	os.WriteFile(filepath.Join(configDir, "log.yaml"), []byte(logYaml), 0644)


}

func genUtils(projectDir string) {
	content := `package utils

import "crypto/md5"

func MD5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
`
	// Fix missing import
	content = strings.Replace(content, "return fmt.Sprintf", "import \"encoding/hex\"\n\nfunc MD5(data string) string { h := md5.New(); h.Write([]byte(data)); b := h.Sum(nil); return hex.EncodeToString(b) }", 1)
	content = `package utils

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
	os.WriteFile(filepath.Join(projectDir, "pkg", "utils", "crypto.go"), []byte(content), 0644)
}

func genBFF(projectDir, bffName string, modules []string, protocol string) {
	// main.go
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "main", "gin_main.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFF main template: %v\n", err)
		return
	}
	mainGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":      microAppName,
		"BFFName":      bffName,
		"BffDirName":   toBffDirName(bffName),
		"NacosEnabled": microAppConfig == "nacos",
		"Otel":        microAppOtel,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFF main template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "cmd", "main.go"), []byte(mainGo), 0644)

	// router.go
	type routeModule struct {
		Name      string
		UpperName string
	}
	var routeModules []routeModule
	for _, m := range modules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		routeModules = append(routeModules, routeModule{Name: m, UpperName: upper})
	}
	tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "router", "gin_router.go.tmpl")
	tmplStr, err = os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFF router template: %v\n", err)
		return
	}
	routerGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName": microAppName,
		"BFFName": bffName,
		"BffDirName": toBffDirName(bffName),
		"Modules": routeModules,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFF router template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "router", "router.go"), []byte(routerGo), 0644)

	// config.yaml
	var bffCfg string
	if microAppRegister == "consul" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: consul\n  address: 127.0.0.1:8500\n"
	} else if microAppRegister == "etcd" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: etcd\n  address: 127.0.0.1:2379\n"
	} else {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n"
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "configs", "config.yaml"), []byte(bffCfg), 0644)

	// middleware
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "middleware", "middleware.go"), []byte(`package middleware

import "github.com/gin-gonic/gin"

func Logger() gin.HandlerFunc {
	return gin.Logger()
}
`), 0644)

	// generate handlers and rpc clients - BFF internal 不生成 dto 文件
	for _, m := range modules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		// rpc client - 根据协议选择
		if protocol == "kitex" {
			genKitexClient(projectDir, bffName, m, upper)
		} else {
			genGRPCClient(projectDir, bffName, m, upper)
		}

		// handler - BFF internal 不使用 dto，直接在 handler 中处理
		tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "handler", "handler.go.tmpl")
		tmplStr, err := os.ReadFile(tmplPath)
		if err != nil {
			fmt.Printf("ERROR reading handler template: %v\n", err)
			continue
		}
		handler, err := executeTemplate(string(tmplStr), map[string]interface{}{
			"UpperName": upper,
			"AppName":   microAppName,
			"LowerName": m,
		})
		if err != nil {
			fmt.Printf("ERROR executing handler template: %v\n", err)
			continue
		}
		os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "handler", toCamelFileName(m, "Handler.go")), []byte(handler), 0644)
	}
}

// genGRPCClient 生成 gRPC 客户端
func genGRPCClient(projectDir, bffName, m, upper string) {
	var tmplFile string
	if microAppRegister == "consul" {
		tmplFile = "grpc_client_consul.go.tmpl"
	} else if microAppRegister == "etcd" {
		tmplFile = "grpc_client_etcd.go.tmpl"
	} else {
		tmplFile = "grpc_client_direct.go.tmpl"
	}

	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "client", tmplFile)
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading gRPC client template: %v\n", err)
		return
	}

	client, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":    microAppName,
		"ModuleName": microAppName,
		"SrvDirName":         toSrvDirName(microAppName),
		"TableName":  m,
		"UpperName":  upper,
	})
	if err != nil {
		fmt.Printf("ERROR executing gRPC client template: %v\n", err)
		return
	}

	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "rpcClient", toCamelFileName(m, "Client.go")), []byte(client), 0644)
}

// genKitexClient 生成 Kitex 客户端
func genKitexClient(projectDir, bffName, m, upper string) {
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "client", "kitex_client.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading Kitex client template: %v\n", err)
		return
	}

	client, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":    microAppName,
		"ModuleName": microAppName,
		"SrvDirName":         toSrvDirName(microAppName),
		"TableName":  m,
		"UpperName":  upper,
	})
	if err != nil {
		fmt.Printf("ERROR executing Kitex client template: %v\n", err)
		return
	}

	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "rpcClient", toCamelFileName(m, "Client.go")), []byte(client), 0644)
}

// genBFFHertz 生成 Hertz BFF 层
func genBFFHertz(projectDir, bffName string, modules []string, protocol string) {
	// main.go
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "main", "hertz_main.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFF Hertz main template: %v\n", err)
		return
	}
	mainGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":      microAppName,
		"BFFName":      bffName,
		"BffDirName":   toBffDirName(bffName),
		"NacosEnabled": microAppConfig == "nacos",
	})
	if err != nil {
		fmt.Printf("ERROR executing BFF Hertz main template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "cmd", "main.go"), []byte(mainGo), 0644)

	// router.go
	type hertzRouteModule struct {
		Name      string
		UpperName string
	}
	var hertzRouteModules []hertzRouteModule
	for _, m := range modules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		hertzRouteModules = append(hertzRouteModules, hertzRouteModule{Name: m, UpperName: upper})
	}
	tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "router", "hertz_router.go.tmpl")
	tmplStr, err = os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFF Hertz router template: %v\n", err)
		return
	}
	routerGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName": microAppName,
		"BFFName": bffName,
		"BffDirName": toBffDirName(bffName),
		"Modules": hertzRouteModules,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFF Hertz router template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "router", "router.go"), []byte(routerGo), 0644)

	// config.yaml
	var bffCfg string
	if microAppRegister == "consul" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: consul\n  address: 127.0.0.1:8500\n"
	} else if microAppRegister == "etcd" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: etcd\n  address: 127.0.0.1:2379\n"
	} else {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n"
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "configs", "config.yaml"), []byte(bffCfg), 0644)

	// middleware
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "middleware", "middleware.go"), []byte(`package middleware

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/cors"
)

func Logger() app.HandlerFunc {
	return func(ctx app.RequestContext) {
		ctx.Next()
	}
}

func CORS() app.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	})
}
`), 0644)

	// generate handlers and rpc clients
	for _, m := range modules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		// rpc client
		if protocol == "kitex" {
			genKitexClient(projectDir, bffName, m, upper)
		} else {
			genGRPCClient(projectDir, bffName, m, upper)
		}

		// handler - Hertz 版本
		tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "handler", "handler_hertz.go.tmpl")
		tmplStr, err := os.ReadFile(tmplPath)
		if err != nil {
			fmt.Printf("ERROR reading BFF Hertz handler template: %v\n", err)
			continue
		}
		handler, err := executeTemplate(string(tmplStr), map[string]interface{}{
			"AppName":     microAppName,
			"BFFName":     bffName,
			"UpperModule": upper,
		})
		if err != nil {
			fmt.Printf("ERROR executing BFF Hertz handler template: %v\n", err)
			continue
		}
		os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "handler", toCamelFileName(m, "Handler.go")), []byte(handler), 0644)
	}
}

func genMicroservice(projectDir, module string, port int, protocol string) {
	upper := strings.ToUpper(module[:1]) + module[1:]

	// main.go - 根据注册中心类型生成不同版本
	var mainGo string
	tmplFile := "main_direct.go.tmpl"
	if microAppRegister == "consul" {
		tmplFile = "main_consul.go.tmpl"
	} else if microAppRegister == "etcd" {
		tmplFile = "main_etcd.go.tmpl"
	}
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "srv", "main", tmplFile)
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading SRV main template: %v\n", err)
		return
	}
	mainGo, err = executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":     microAppName,
		"Module":      module,
		"UpperModule": upper,
		"SrvDirName":  toSrvDirName(module),
		"Otel":        microAppOtel,
	})
	if err != nil {
		fmt.Printf("ERROR executing SRV main template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toSrvDirName(module), "cmd", "main.go"), []byte(mainGo), 0644)

	// config.yaml
	var cfg string
	if microAppRegister != "" {
		cfg = fmt.Sprintf("server:\n  host: 0.0.0.0\n  port: %d\n\ndatabase:\n  host: %s\n  port: %s\n  user: %s\n  password: %s\n  database: %s\n\nregistry:\n  type: %s\n  address: 127.0.0.1:8500\n", port, microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName, microAppRegister)
		if microAppRegister == "etcd" {
			cfg = fmt.Sprintf("server:\n  host: 0.0.0.0\n  port: %d\n\ndatabase:\n  host: %s\n  port: %s\n  user: %s\n  password: %s\n  database: %s\n\nregistry:\n  type: %s\n  address: 127.0.0.1:2379\n", port, microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName, microAppRegister)
		}
	} else {
		cfg = fmt.Sprintf("server:\n  host: 0.0.0.0\n  port: %d\n\ndatabase:\n  host: %s\n  port: %s\n  user: %s\n  password: %s\n  database: %s\n", port, microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName)
	}
	os.WriteFile(filepath.Join(projectDir, toSrvDirName(module), "configs", "config.yaml"), []byte(cfg), 0644)

	// handler - 使用模板渲染
	tmplDir := filepath.Join(getTemplatesDir(), "micro-app", "srv", "handler")
	data := buildTemplateData(module, nil, "", "", 0)
	if err := renderTemplate(
		filepath.Join(tmplDir, "handler.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "handler", toCamelFileName(module, "Handler.go")),
	); err != nil {
		fmt.Printf("ERROR rendering srv handler %s: %v\n", module, err)
	}

	// model - 使用模板渲染
	tmplDir = filepath.Join(getTemplatesDir(), "micro-app", "srv", "model")
	if err := renderTemplate(
		filepath.Join(tmplDir, "model.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "model", module+".go"),
	); err != nil {
		fmt.Printf("ERROR rendering srv model %s: %v\n", module, err)
	}

	// repository - 使用模板渲染
	tmplDir = filepath.Join(getTemplatesDir(), "micro-app", "srv", "repo")
	if err := renderTemplate(
		filepath.Join(tmplDir, "repository.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "repository", toCamelFileName(module, "Repo.go")),
	); err != nil {
		fmt.Printf("ERROR rendering srv repository %s: %v\n", module, err)
	}

	// service
	svc := "package service\n\nimport (\n\t\"context\"\n\n\t\"" + microAppName + "/common/kitexGen/" + module + "\"\n\t\"" + microAppName + "/" + toSrvDirName(module) + "/internal/model\"\n\t\"" + microAppName + "/" + toSrvDirName(module) + "/internal/repository\"\n)\n\ntype " + upper + "Service struct { repo *" + "repository." + upper + "Repo }\n\nfunc New" + upper + "Service(repo *" + "repository." + upper + "Repo) *" + upper + "Service {\n\treturn &" + upper + "Service{repo: repo}\n}\n\nfunc (s *" + upper + "Service) Create(ctx context.Context, req *" + module + ".Create" + upper + "Req) (*" + module + ".Create" + upper + "Resp, error) {\n\tm := &model." + upper + "{Name: req.Name}\n\treturn &" + module + ".Create" + upper + "Resp{Id: int64(m.ID)}, s.repo.Create(ctx, m)\n}\nfunc (s *" + upper + "Service) Get(ctx context.Context, req *" + module + ".Get" + upper + "Req) (*" + module + ".Get" + upper + "Resp, error) {\n\tm, err := s.repo.GetByID(ctx, uint(req.Id))\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn &" + module + ".Get" + upper + "Resp{Id: int64(m.ID), Name: m.Name, Status: int32(m.Status)}, nil\n}\nfunc (s *" + upper + "Service) List(ctx context.Context, req *" + module + ".List" + upper + "Req) (*" + module + ".List" + upper + "Resp, error) {\n\tlist, err := s.repo.List(ctx)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tvar items []*" + module + "." + upper + "Item\n\tfor _, m := range list {\n\t\titems = append(items, &" + module + "." + upper + "Item{Id: int64(m.ID), Name: m.Name, Status: int32(m.Status)})\n\t}\n\treturn &" + module + ".List" + upper + "Resp{Items: items}, nil\n}\nfunc (s *" + upper + "Service) Update(ctx context.Context, req *" + module + ".Update" + upper + "Req) (*" + module + ".Update" + upper + "Resp, error) {\n\tm, err := s.repo.GetByID(ctx, uint(req.Id))\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif req.Name != \"\" {\n\t\tm.Name = req.Name\n\t}\n\treturn &" + module + ".Update" + upper + "Resp{Success: true}, s.repo.Update(ctx, m)\n}\nfunc (s *" + upper + "Service) Delete(ctx context.Context, req *" + module + ".Delete" + upper + "Req) (*" + module + ".Delete" + upper + "Resp, error) {\n\treturn &" + module + ".Delete" + upper + "Resp{Success: true}, s.repo.Delete(ctx, uint(req.Id))\n}\n"
	os.WriteFile(filepath.Join(projectDir, toSrvDirName(module), "internal", "service", toCamelFileName(module, "Service.go")), []byte(svc), 0644)
}

func genScripts(projectDir string, moduleTables ModuleTables) {
	var genProto strings.Builder
	genProto.WriteString("#!/bin/bash\n")
	genProto.WriteString("SCRIPT_DIR=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")\" && pwd)\"\n")
	genProto.WriteString("PROJECT_DIR=\"$(dirname \"$SCRIPT_DIR\")\"\n")
	genProto.WriteString("mkdir -p \"$PROJECT_DIR/common/kitexGen\"\n")
	genProto.WriteString("cd \"$PROJECT_DIR/common/idl\"\n")
	if microAppIDL == "thrift" {
		for _, tables := range moduleTables {
			for _, tbl := range tables {
				genProto.WriteString("kitex -module " + microAppName + " -service " + tbl.TableName + " " + tbl.TableName + ".thrift\n")
			}
		}
	} else {
		for _, tables := range moduleTables {
			for _, tbl := range tables {
				genProto.WriteString("protoc --go_out=../../common/kitexGen --go_opt=module=" + microAppName + "/common/kitexGen --go-grpc_out=../../common/kitexGen --go-grpc_opt=module=" + microAppName + "/common/kitexGen " + tbl.TableName + ".proto\n")
			}
		}
	}
	genProto.WriteString("echo \"Done\"\n")
	os.WriteFile(filepath.Join(projectDir, "scripts", "gen_proto.sh"), []byte(genProto.String()), 0755)

	// build.sh 仍按 module 构建（每个 module 一个 service 目录）
	var build strings.Builder
	build.WriteString("#!/bin/bash\n")
	build.WriteString("SCRIPT_DIR=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")\" && pwd)\"\n")
	for _, tables := range moduleTables {
		// 取第一个表所属的 module（所有表都属于同一 module）
		if len(tables) > 0 {
			// moduleTables 的 key 就是 module 名，直接用
		}
	}
	// build.sh 的 modules 列表通过 microAppModules 传入，这里遍历 moduleTables 的 keys
	for m := range moduleTables {
		build.WriteString("cd \"$SCRIPT_DIR/../" + toSrvDirName(m) + "\" && go build -o ../../bin/" + toSrvDirName(m) + " ./cmd/main.go\n")
	}
	build.WriteString("echo \"Build done\"\n")
	os.WriteFile(filepath.Join(projectDir, "scripts", "build.sh"), []byte(build.String()), 0755)
}

// genMakefile 生成 Makefile（每次生成项目时覆盖）
func genMakefile(projectDir string, modules []string) {
	// 删除旧的 Makefile
	os.Remove(filepath.Join(projectDir, "Makefile"))

	var mf strings.Builder

	// Makefile header
	mf.WriteString("# ========================================\n")
	mf.WriteString("# Auto-generated by gpx - DO NOT EDIT\n")
	mf.WriteString("# ========================================\n\n")

	// 项目变量
	mf.WriteString("APP_NAME := " + microAppName + "\n")
	mf.WriteString("BFF_NAME := " + toBffDirName(microAppBFFName) + "\n")
	mf.WriteString("BIN_DIR := bin\n\n")

	// 服务列表
	mf.WriteString("# Services\n")
	mf.WriteString("SRV_DIRS :=")
	for _, m := range modules {
		mf.WriteString(" " + toSrvDirName(m))
	}
	mf.WriteString("\n")
	mf.WriteString("SRV_BINS :=")
	for _, m := range modules {
		mf.WriteString(" " + toSrvDirName(m))
	}
	mf.WriteString("\n\n")

	// 默认目标
	mf.WriteString(".PHONY: all build run run-bff run-srv stop clean proto test\n\n")
	mf.WriteString("all: build\n\n")

	// build - 编译所有服务
	mf.WriteString("build: proto\n")
	mf.WriteString("\t@mkdir -p $(BIN_DIR)\n")
	mf.WriteString("\t@echo 'Building $(APP_NAME)...'\n")
	mf.WriteString("\t@cd $(BFF_NAME) && go build -o ../$(BIN_DIR)/$(BFF_NAME) ./cmd/main.go\n")
	for _, m := range modules {
		srvDir := toSrvDirName(m)
		mf.WriteString("\t@cd " + srvDir + " && go build -o ../$(BIN_DIR)/" + srvDir + " ./cmd/main.go\n")
	}
	mf.WriteString("\t@echo 'Build done!'\n\n")

	// proto - 编译 proto 文件
	mf.WriteString("proto:\n")
	mf.WriteString("\t@echo 'Generating proto...'\n")
	mf.WriteString("\t@cd scripts && bash gen_proto.sh\n\n")

	// run - 启动所有服务（后台运行）
	mf.WriteString("run: build\n")
	mf.WriteString("\t@echo 'Starting $(APP_NAME)...'\n")
	// 停止已有进程
	mf.WriteString("\t@$(MAKE) stop 2>/dev/null || true\n")
	// 启动 SRV 服务（后台）
	for i, m := range modules {
		srvDir := toSrvDirName(m)
		port := 8001 + i
		mf.WriteString("\t@cd " + srvDir + " && nohup go run ./cmd/main.go > ../log/" + srvDir + ".log 2>&1 &\n")
		_ = port // unused but available for reference
	}
	// 启动 BFF（后台）
	mf.WriteString("\t@sleep 2\n")
	mf.WriteString("\t@cd $(BFF_NAME) && nohup go run ./cmd/main.go > ../log/$(BFF_NAME).log 2>&1 &\n")
	mf.WriteString("\t@echo 'Services started!'\n")
	mf.WriteString("\t@echo 'BFF: http://localhost:8080'\n")
	for i, m := range modules {
		port := 8001 + i
		mf.WriteString("\t@echo '" + strings.ToUpper(m[:1]) + m[1:] + " SRV: localhost:" + fmt.Sprintf("%d", port) + "'\n")
	}
	mf.WriteString("\n")

	// run-bff - 仅启动 BFF
	mf.WriteString("run-bff: build\n")
	mf.WriteString("\t@$(MAKE) stop-bff 2>/dev/null || true\n")
	mf.WriteString("\t@echo 'Starting $(BFF_NAME)...'\n")
	mf.WriteString("\t@cd $(BFF_NAME) && nohup go run ./cmd/main.go > ../log/$(BFF_NAME).log 2>&1 &\n")
	mf.WriteString("\t@echo 'BFF started! http://localhost:8080'\n\n")

	// run-srv - 仅启动所有 SRV
	mf.WriteString("run-srv: build\n")
	mf.WriteString("\t@$(MAKE) stop-srv 2>/dev/null || true\n")
	mf.WriteString("\t@echo 'Starting SRV services...'\n")
	for i, m := range modules {
		srvDir := toSrvDirName(m)
		mf.WriteString("\t@cd " + srvDir + " && nohup go run ./cmd/main.go > ../log/" + srvDir + ".log 2>&1 &\n")
		_ = i
	}
	mf.WriteString("\t@echo 'SRV services started!'\n\n")

	// stop - 停止所有服务
	mf.WriteString("stop:\n")
	mf.WriteString("\t@echo 'Stopping $(APP_NAME)...'\n")
	mf.WriteString("\t@-pkill -f '$(BFF_NAME)/cmd/main.go' 2>/dev/null || true\n")
	for _, m := range modules {
		srvDir := toSrvDirName(m)
		mf.WriteString("\t@-pkill -f '" + srvDir + "/cmd/main.go' 2>/dev/null || true\n")
	}
	mf.WriteString("\t@echo 'All services stopped!'\n\n")

	// stop-bff / stop-srv
	mf.WriteString("stop-bff:\n")
	mf.WriteString("\t@-pkill -f '$(BFF_NAME)/cmd/main.go' 2>/dev/null || true\n")
	mf.WriteString("\t@echo 'BFF stopped!'\n\n")

	mf.WriteString("stop-srv:\n")
	for _, m := range modules {
		srvDir := toSrvDirName(m)
		mf.WriteString("\t@-pkill -f '" + srvDir + "/cmd/main.go' 2>/dev/null || true\n")
	}
	mf.WriteString("\t@echo 'SRV services stopped!'\n\n")

	// clean - 清理编译产物
	mf.WriteString("clean:\n")
	mf.WriteString("\t@rm -rf $(BIN_DIR)\n")
	mf.WriteString("\t@echo 'Clean done!'\n\n")

	// test - 运行测试
	mf.WriteString("test:\n")
	mf.WriteString("\t@echo 'Running tests...'\n")
	mf.WriteString("\t@cd $(BFF_NAME) && go test ./... -v 2>&1 | head -100\n")
	for _, m := range modules {
		srvDir := toSrvDirName(m)
		mf.WriteString("\t@cd " + srvDir + " && go test ./... -v 2>&1 | head -50\n")
	}

	os.WriteFile(filepath.Join(projectDir, "Makefile"), []byte(mf.String()), 0644)
}

func genReadme(projectDir, bffName string, modules []string) {
	var content strings.Builder
	content.WriteString("# " + microAppName + "\n\n")
	content.WriteString("Microservice project.\n\n")
	content.WriteString("## Options\n\n")
	content.WriteString(fmt.Sprintf("- **Protocol**: %s\n", microAppProtocol))
	content.WriteString(fmt.Sprintf("- **HTTP**: %s\n", microAppHTTP))
	content.WriteString(fmt.Sprintf("- **IDL**: %s\n", microAppIDL))
	if microAppRegister != "" {
		content.WriteString(fmt.Sprintf("- **Registry**: %s\n", microAppRegister))
	}
	if microAppMiddleware != "" {
		content.WriteString(fmt.Sprintf("- **Middleware**: %s\n", microAppMiddleware))
	}
	if microAppConfig == "nacos" {
		content.WriteString("- **Config Center**: nacos\n")
	}
	content.WriteString("\n")
	content.WriteString("## Structure\n")
	content.WriteString("```\n")
	content.WriteString(microAppName + "/\n")
	content.WriteString("├── " + toBffDirName(bffName) + "/\n")
	for _, m := range modules {
		content.WriteString("├── " + toSrvDirName(m) + "/\n")
	}
	content.WriteString("├── common/\n")
	content.WriteString("├── pkg/\n")
	if microAppTest {
		content.WriteString("├── " + toBffDirName(bffName) + "/test/     ← BFF 接口测试\n")
		for _, m := range modules {
			content.WriteString("├── " + toSrvDirName(m) + "/test/     ← " + m + " 微服接口测试\n")
		}
	}
	content.WriteString("└── scripts/\n")
	content.WriteString("```\n\n")
	content.WriteString("## Build\n")
	content.WriteString("```bash\n")
	content.WriteString("go mod init github.com/yourorg/" + microAppName + "\n")
	content.WriteString("./scripts/gen_proto.sh\n")
	content.WriteString("./scripts/build.sh\n")
	content.WriteString("```\n")
	if microAppTest {
		content.WriteString("\n## Test\n")
		content.WriteString("```bash\n")
		content.WriteString("# 运行 BFF 层接口测试\n")
		content.WriteString("cd " + toBffDirName(bffName) + " && go test ./test/ -v\n\n")
		for _, m := range modules {
			content.WriteString("# 运行 " + m + " 微服层接口测试\n")
			content.WriteString("cd " + toSrvDirName(m) + " && go test ./test/ -v\n\n")
		}
		content.WriteString("```\n")
	}
	os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(content.String()), 0644)
}

// genGitignore 生成 .gitignore，屏蔽 Windows/macOS 系统隐藏文件及常见临时文件
func genGitignore(projectDir string) {
	content := `# ==========================================
# Auto-generated by gpx - DO NOT EDIT
# ==========================================

# Build output
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Go module cache
vendor/

# IDE / Editor
.idea/
.vscode/
*.swp
*.swo
*~

# macOS
.DS_Store
.AppleDouble
.LSOverride
._*

# Windows
Thumbs.db
Thumbs.db:encryptable
ehthumbs.db
ehthumbs_vista.db
[Dd]esktop.ini
$RECYCLE.BIN/
*.lnk

# Windows Zone.Identifier alternate data streams
*:Zone.Identifier

# Logs
*.log
logs/

# Config overrides (keep template, ignore local)
configs/local*.yaml
`
	// 只有不存在时才生成，避免覆盖用户自定义
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		os.WriteFile(gitignorePath, []byte(content), 0644)
	}
}

// genGoMod 生成 go.mod 文件
func genGoMod(projectDir string) {
	goVersion := getGoVersion()
	content := fmt.Sprintf(`module %s

go %s
`, microAppName, goVersion)

	gomodPath := filepath.Join(projectDir, "go.mod")
	os.WriteFile(gomodPath, []byte(content), 0644)
	fmt.Printf("  Generated: go.mod (go %s)\n", goVersion)
}

func getGoVersion() string {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "1.21"
	}
	re := regexp.MustCompile(`go(\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return "1.21"
}

func runGenProtoScript(projectDir string) {
	scriptPath := filepath.Join(projectDir, "scripts", "gen_proto.sh")
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		fmt.Printf("  Warning: failed to get absolute path: %v\n", err)
		return
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Printf("  Skipped: %s not found\n", absPath)
		return
	}

	cmd := exec.Command("bash", absPath)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  Warning: gen_proto.sh failed: %v\n", err)
		fmt.Printf("  Output: %s\n", string(output))
		return
	}
	fmt.Printf("  Executed: scripts/gen_proto.sh\n")
}

// cleanHiddenFiles 清理项目目录中的 Windows/macOS 系统隐藏文件
// 仅清理已知的系统产生的隐藏文件，不删除 .gitignore、.git 等有用的点文件
func cleanHiddenFiles(projectDir string) {
	// 需要删除的已知系统隐藏文件名（精确匹配）
	knownHiddenFiles := map[string]bool{
		"Thumbs.db":                  true,
		"Thumbs.db:encryptable":      true,
		"ehthumbs.db":                true,
		"ehthumbs_vista.db":          true,
		"desktop.ini":                true,
		"Desktop.ini":                true,
		"[Dd]esktop.ini":             true,
		".DS_Store":                  true,
		".AppleDouble":               true,
		".LSOverride":                true,
	}
	// 需要删除的已知系统隐藏文件名前缀（如 ._filename）
	hiddenPrefixes := []string{"._"}

	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		name := info.Name()

		// 跳过 .git、.gitignore 等有用点文件/目录
		if name == ".git" || name == ".gitignore" || name == ".gitkeep" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 精确匹配已知系统文件
		if !info.IsDir() && knownHiddenFiles[name] {
			os.Remove(path)
			return nil
		}

		// 前缀匹配（如 ._productHandler.go 等 macOS 资源分叉文件）
		for _, prefix := range hiddenPrefixes {
			if strings.HasPrefix(name, prefix) && !info.IsDir() {
				os.Remove(path)
				return nil
			}
		}
		return nil
	})
}

// =============================================================================
// 基于表结构的生成函数
// =============================================================================

// genProtoFilesFromSchema 根据表结构生成 proto 文件
func genProtoFilesFromSchema(projectDir string, moduleTables ModuleTables) {
	// 收集所有表名（用于推导 entity 名）
	var allTableNames []string
	for _, tables := range moduleTables {
		for _, tbl := range tables {
			allTableNames = append(allTableNames, tbl.TableName)
		}
	}

	for moduleName, tables := range moduleTables {
		for _, table := range tables {
			columns := table.Columns
			var proto strings.Builder

			// 为每张表生成独立的 proto 文件，使用表名转换为小驼峰作为文件名
			protoFileName := toCamelCaseFile(table.TableName)
			if protoFileName == "" {
				protoFileName = moduleName
			}

			// 使用 entity 名（从表名推导）作为 service 名称，以避免多个表冲突
			entityName := tableToEntityName(table.TableName, allTableNames)
			entityUpper := strings.ToUpper(entityName[:1]) + entityName[1:]

			// syntax + service 声明
			tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "proto", "syntax.go.tmpl")
			tmplBytes, err := os.ReadFile(tmplPath)
			if err != nil {
				fmt.Printf("Error reading syntax template: %v\n", err)
				continue
			}
			result, err := executeTemplate(string(tmplBytes), map[string]interface{}{
				"Module":    protoFileName,
				"AppName":   microAppName,
				"UpperName": entityUpper, // 使用 entity 名作为 service 名，避免多表冲突
			})
			if err != nil {
				fmt.Printf("Error executing syntax template: %v\n", err)
				continue
			}
			proto.WriteString(result)

			// 添加字段（跳过主键自增字段）
			for _, col := range columns {
				if col.IsPrimary && strings.Contains(strings.ToLower(col.Type), "int") {
					continue // 跳过主键，Create 不需要
				}
				comment := ""
				if col.Comment != "" {
					comment = " // " + col.Comment
				}
				proto.WriteString(fmt.Sprintf("  %s %s = %d;%s\n", col.ProtoType, col.Name, col.ProtoIndex, comment))
			}

			proto.WriteString("}\n\n")

			// Create 响应
			pkField := getPrimaryKeyField(columns)
			proto.WriteString(execTemplateOrFail("proto/create_resp.go.tmpl", map[string]interface{}{
				"UpperName":   entityUpper, // 使用 entity 名
				"PKProtoType": pkField.ProtoType,
				"PKLowerName": getLowerFirst(pkField.Name),
			}))

			// Get 请求 + 响应开头
			proto.WriteString(execTemplateOrFail("proto/get_resp.go.tmpl", map[string]interface{}{
				"UpperName":   entityUpper, // 使用 entity 名
				"PKProtoType": pkField.ProtoType,
				"PKLowerName": getLowerFirst(pkField.Name),
			}))

			// Get 响应 - 包含所有字段
			proto.WriteString(fmt.Sprintf("message Get%sResp {\n", entityUpper))
			for _, col := range columns {
				proto.WriteString(fmt.Sprintf("  %s %s = %d;\n", col.ProtoType, getLowerFirst(col.Name), col.ProtoIndex))
			}
			proto.WriteString("}\n\n")

			// List 请求
			proto.WriteString(execTemplateOrFail("proto/list_req.go.tmpl", map[string]interface{}{
				"UpperName": entityUpper,
			}))

			// List Item 开头
			proto.WriteString(execTemplateOrFail("proto/item.go.tmpl", map[string]interface{}{
				"UpperName": entityUpper,
			}))
			for _, col := range columns {
				proto.WriteString(fmt.Sprintf("  %s %s = %d;\n", col.ProtoType, getLowerFirst(col.Name), col.ProtoIndex))
			}
			proto.WriteString("}\n\n")

			// List 响应
			proto.WriteString(execTemplateOrFail("proto/list_resp.go.tmpl", map[string]interface{}{
				"UpperName": entityUpper,
			}))

			// Update 请求开头
			proto.WriteString(execTemplateOrFail("proto/update_req.go.tmpl", map[string]interface{}{
				"UpperName":   entityUpper,
				"PKProtoType": pkField.ProtoType,
				"PKLowerName": getLowerFirst(pkField.Name),
			}))
			for _, col := range columns {
				if col.IsPrimary {
					continue
				}
				proto.WriteString(fmt.Sprintf("  %s %s = %d;\n", col.ProtoType, col.Name, col.ProtoIndex))
			}
			proto.WriteString("}\n\n")

			// Update 响应
			proto.WriteString(execTemplateOrFail("proto/update_resp.go.tmpl", map[string]interface{}{
				"UpperName": entityUpper,
			}))

			// Delete 请求
			proto.WriteString(execTemplateOrFail("proto/delete_req.go.tmpl", map[string]interface{}{
				"UpperName":   entityUpper,
				"PKProtoType": pkField.ProtoType,
				"PKLowerName": getLowerFirst(pkField.Name),
			}))

			// Delete 响应
			proto.WriteString(execTemplateOrFail("proto/delete_resp.go.tmpl", map[string]interface{}{
				"UpperName": entityUpper,
			}))

			os.WriteFile(filepath.Join(projectDir, "common", "idl", protoFileName+".proto"), []byte(proto.String()), 0644)
			fmt.Printf("Generated proto: %s\n", protoFileName+".proto")
		}
	}
}

// genBFFFromSchema 基于表结构生成 BFF 层 (Gin)
func genBFFFromSchema(projectDir, bffName string, moduleTables ModuleTables, protocol string) {
	// 从 moduleTables 推导 tableNames（供 entityName 计算用）
	var tableNames []string
	for _, tables := range moduleTables {
		for _, tbl := range tables {
			tableNames = append(tableNames, tbl.TableName)
		}
	}

	// main.go
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "main", "gin_main_schema.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFFFromSchema main template: %v\n", err)
		return
	}
	mainGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":      microAppName,
		"BFFName":      bffName,
		"BffDirName":   toBffDirName(bffName),
		"NacosEnabled": microAppConfig == "nacos",
		"Otel":         microAppOtel,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFFFromSchema main template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "cmd", "main.go"), []byte(mainGo), 0644)

	// router.go - 为每张表生成路由，每张表路由到各自的 Handler
	type crudRouteModule struct {
		Name      string
		UpperName string
	}
	var crudRouteModules []crudRouteModule
	for _, tables := range moduleTables {
		for _, table := range tables {
			entityName := tableToEntityName(table.TableName, tableNames)
			upperEntity := strings.ToUpper(entityName[:1]) + entityName[1:]
			crudRouteModules = append(crudRouteModules, crudRouteModule{Name: table.TableName, UpperName: upperEntity})
		}
	}
	tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "router", "gin_router_crud.go.tmpl")
	tmplStr, err = os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFFFromSchema router template: %v\n", err)
		return
	}
	routerGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName": microAppName,
		"BFFName": bffName,
		"BffDirName": toBffDirName(bffName),
		"Modules": crudRouteModules,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFFFromSchema router template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "router", "router.go"), []byte(routerGo), 0644)

	// config.yaml
	var bffCfg string
	if microAppRegister == "consul" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: consul\n  address: 127.0.0.1:8500\n"
	} else if microAppRegister == "etcd" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: etcd\n  address: 127.0.0.1:2379\n"
	} else {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n"
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "configs", "config.yaml"), []byte(bffCfg), 0644)

	// middleware
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "middleware", "middleware.go"), []byte(`package middleware

import "github.com/gin-gonic/gin"

func Logger() gin.HandlerFunc {
	return gin.Logger()
}
`), 0644)

	// 为每个 module 下的每张表生成 rpc_client 和 handler
	srvPort := 50050
	for moduleName, tables := range moduleTables {
		upper := strings.ToUpper(moduleName[:1]) + moduleName[1:]
		for _, table := range tables {
			columns := table.Columns
			pkField := getPrimaryKeyField(columns)

			// 生成 rpc_client - 基于表结构的 CRUD 方法
			entityName := tableToEntityName(table.TableName, tableNames)
			genBFFClientFromSchema(projectDir, bffName, moduleName, entityName, columns, protocol, table.TableName, srvPort)

			// 生成 handler - 入参与 proto 对齐，使用 entityName 命名
			genBFFHandlerFromSchema(projectDir, bffName, moduleName, entityName, upper, columns, pkField, table.TableName, srvPort, tableNames)
		}
		srvPort++
	}
}

// genBFFClientFromSchema 生成 BFF gRPC 客户端（基于表结构）
func genBFFClientFromSchema(projectDir, bffName, m, entityName string, columns []ColumnInfo, protocol string, tableName string, srvPort int) {
	data := buildTemplateData(m, columns, bffName, tableName, srvPort, entityName)
	tmplDir := filepath.Join(getTemplatesDir(), "microservice", "grpc", "schema")

	// 根据注册中心类型选择 RPC client 模板
	var clientTmplFile string
	if microAppRegister == "consul" {
		clientTmplFile = "bff_rpc_client_consul.go.tmpl"
	} else if microAppRegister == "etcd" {
		clientTmplFile = "bff_rpc_client_etcd.go.tmpl"
	} else {
		clientTmplFile = "bff_rpc_client.go.tmpl"
	}

	if err := renderTemplate(
		filepath.Join(tmplDir, clientTmplFile),
		data,
		filepath.Join(projectDir, toBffDirName(bffName), "internal", "rpcClient", toCamelFileName(entityName, "Client.go")),
	); err != nil {
		fmt.Printf("ERROR rendering bff_rpc_client: %v\n", err)
	}
}

func genBFFHandlerFromSchema(projectDir, bffName, m, entityName, upper string, columns []ColumnInfo, pkField ColumnInfo, tableName string, srvPort int, tableNames []string) {
	if entityName == "" {
		entityName = tableToEntityName(tableName, tableNames)
	}
	data := buildTemplateData(entityName, columns, bffName, tableName, srvPort)
	tmplDir := filepath.Join(getTemplatesDir(), "microservice", "grpc", "schema")

	if err := renderTemplate(
		filepath.Join(tmplDir, "bff_handler.go.tmpl"),
		data,
		filepath.Join(projectDir, toBffDirName(bffName), "internal", "handler", toCamelFileName(entityName, "Handler.go")),
	); err != nil {
		fmt.Printf("ERROR rendering bff_handler: %v\n", err)
	}
}

// tableToEntityName 从表名推导 entity 名称（用于文件名和结构体命名）。
// 直接使用表名，转换为小驼峰格式。例如："eb_store_product" → "ebStoreProduct"
// 不再剥离公共前缀，保持表名完整性。
func tableToEntityName(tableName string, allTableNames []string) string {
	// 将表名转为小驼峰：eb_store_product → ebStoreProduct
	return camelCase(tableName)
}

// genSrvMainAndConfig 为一个 module 生成 main.go 和 config.yaml（只调用一次）
func genSrvMainAndConfig(projectDir, module string, port int, tables []TableInfo, tableNames []string) {
	tmplDir := filepath.Join(getTemplatesDir(), "microservice", "grpc", "schema")

	// main.go - 根据注册中心类型选择模板
	var mainTmplFile string
	if microAppRegister == "consul" {
		mainTmplFile = "srv_main_consul.go.tmpl"
	} else if microAppRegister == "etcd" {
		mainTmplFile = "srv_main_etcd.go.tmpl"
	} else {
		mainTmplFile = "srv_main.go.tmpl"
	}

	// 构建 handler 注册代码（支持多表）
	// 注意：proto 使用的是 entity 名（如 EbStoreProductService），不是 module 名（如 ProductService）
	upperModule := strings.ToUpper(module[:1]) + module[1:]
	// 生成多表 proto import 列表（每行一个完整的 import 语句，含唯一别名）
	var protoImports []string
	var handlerRegs []string
	for _, tbl := range tables {
		entityName := tableToEntityName(tbl.TableName, tableNames)
		// 用 camelCase entity 名作 import 别名（如 ebstoreproduct）
		alias := strings.ToLower(entityName[:1]) + entityName[1:]
		protoImports = append(protoImports, fmt.Sprintf("%s \"%s/common/kitexGen/%s\"", alias, microAppName, tbl.TableName))
		upperEntity := strings.ToUpper(entityName[:1]) + entityName[1:]
		handlerRegs = append(handlerRegs, fmt.Sprintf("\t%s.Register%sServiceServer(s, handler.New%sHandler(db))\n", alias, upperEntity, upperEntity))
	}
	// 兼容单表场景
	tableName := ""
	var initRegs string
	if len(tables) > 0 {
		tableName = tables[0].TableName
		alias := strings.ToLower(upperModule[:1]) + upperModule[1:]
		if len(tables) == 1 {
			protoImports = []string{fmt.Sprintf("%s \"%s/common/kitexGen/%s\"", alias, microAppName, tableName)}
			initRegs = fmt.Sprintf("\t%s.Register%sServiceServer(s, handler.New%sHandler(db))\n", alias, upperModule, upperModule)
		} else {
			initRegs = strings.Join(handlerRegs, "")
		}
	}
	if len(protoImports) == 0 {
		protoImports = []string{fmt.Sprintf("pb \"%s/common/kitexGen/%s\"", microAppName, tableName)}
	}
	mainData := TemplateData{
		AppName:           microAppName,
		Module:            module,
		UpperModule:       upperModule,
		LowerModule:       strings.ToLower(module),
		SrvDirName:        toSrvDirName(module),
		SrvPort:          port,
		Register:         microAppRegister,
		UpperEntityName:   upperModule,
		HandlerRegs:      initRegs,
		TableName:        tableName,
		ProtoImports:    protoImports,
	}
	if err := renderTemplate(
		filepath.Join(tmplDir, mainTmplFile),
		mainData,
		filepath.Join(projectDir, toSrvDirName(module), "cmd", "main.go"),
	); err != nil {
		fmt.Printf("ERROR rendering srv_main: %v\n", err)
	}

	// config.yaml
	var cfg string
	if microAppRegister != "" {
		cfg = fmt.Sprintf("server:\n  host: 0.0.0.0\n  port: %d\n\ndatabase:\n  host: %s\n  port: %s\n  user: %s\n  password: %s\n  database: %s\n\nregistry:\n  type: %s\n  address: 127.0.0.1:8500\n", port, microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName, microAppRegister)
		if microAppRegister == "etcd" {
			cfg = fmt.Sprintf("server:\n  host: 0.0.0.0\n  port: %d\n\ndatabase:\n  host: %s\n  port: %s\n  user: %s\n  password: %s\n  database: %s\n\nregistry:\n  type: %s\n  address: 127.0.0.1:2379\n", port, microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName, microAppRegister)
		}
	} else {
		cfg = fmt.Sprintf("server:\n  host: 0.0.0.0\n  port: %d\n\ndatabase:\n  host: %s\n  port: %s\n  user: %s\n  password: %s\n  database: %s\n", port, microAppDBHost, microAppDBPort, microAppDBUser, microAppDBPassword, microAppDBName)
	}
	os.WriteFile(filepath.Join(projectDir, toSrvDirName(module), "configs", "config.yaml"), []byte(cfg), 0644)
}

// genSrvTableFiles 为一张表生成 model/repo/service/handler 文件
func genSrvTableFiles(projectDir, module string, columns []ColumnInfo, tableName string, entityName string, allTableNames []string, srvPort int) {
	// 如果未传入 entityName，通过表名推导（兼容旧调用点）
	if entityName == "" {
		entityName = tableToEntityName(tableName, allTableNames)
	}
	data := buildTemplateData(module, columns, "", tableName, srvPort, entityName)
	tmplDir := filepath.Join(getTemplatesDir(), "microservice", "grpc", "schema")

	// model
	if err := renderTemplate(
		filepath.Join(tmplDir, "srv_model.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "model", entityName+".go"),
	); err != nil {
		fmt.Printf("ERROR rendering srv_model: %v\n", err)
	}

	// repository
	if err := renderTemplate(
		filepath.Join(tmplDir, "srv_repo.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "repository", toCamelFileName(entityName, "Repo.go")),
	); err != nil {
		fmt.Printf("ERROR rendering srv_repo: %v\n", err)
	}

	// service
	if err := renderTemplate(
		filepath.Join(tmplDir, "srv_service.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "service", toCamelFileName(entityName, "Service.go")),
	); err != nil {
		fmt.Printf("ERROR rendering srv_service: %v\n", err)
	}

	// handler
	if err := renderTemplate(
		filepath.Join(tmplDir, "srv_handler.go.tmpl"),
		data,
		filepath.Join(projectDir, toSrvDirName(module), "internal", "handler", toCamelFileName(entityName, "Handler.go")),
	); err != nil {
		fmt.Printf("ERROR rendering srv_handler: %v\n", err)
	}
}

// getTemplatesDir 获取模板目录的绝对路径
func genBFFHertzFromSchema(projectDir, bffName string, moduleTables ModuleTables, protocol string) {
	// main.go
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "main", "hertz_main_schema.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFFHertzFromSchema main template: %v\n", err)
		return
	}
	mainGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName":      microAppName,
		"BFFName":      bffName,
		"BffDirName":   toBffDirName(bffName),
		"NacosEnabled": microAppConfig == "nacos",
		"Otel":        microAppOtel,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFFHertzFromSchema main template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "cmd", "main.go"), []byte(mainGo), 0644)

	// router.go
	type hertzCrudRouteModule struct {
		Name      string
		UpperName string
	}
	var hertzCrudRouteModules []hertzCrudRouteModule
	for m := range moduleTables {
		upper := strings.ToUpper(m[:1]) + m[1:]
		hertzCrudRouteModules = append(hertzCrudRouteModules, hertzCrudRouteModule{Name: m, UpperName: upper})
	}
	tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "router", "hertz_router_crud.go.tmpl")
	tmplStr, err = os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFFHertzFromSchema router template: %v\n", err)
		return
	}
	routerGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName": microAppName,
		"BFFName": bffName,
		"BffDirName": toBffDirName(bffName),
		"Modules": hertzCrudRouteModules,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFFHertzFromSchema router template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "router", "router.go"), []byte(routerGo), 0644)

	// config.yaml
	var bffCfg string
	if microAppRegister == "consul" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: consul\n  address: 127.0.0.1:8500\n"
	} else if microAppRegister == "etcd" {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n\nregistry:\n  type: etcd\n  address: 127.0.0.1:2379\n"
	} else {
		bffCfg = "server:\n  host: 0.0.0.0\n  port: 8080\n"
	}
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "configs", "config.yaml"), []byte(bffCfg), 0644)

	// middleware
	os.WriteFile(filepath.Join(projectDir, toBffDirName(bffName), "internal", "middleware", "middleware.go"), []byte(`package middleware

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/cors"
)

func Logger() app.HandlerFunc {
	return func(ctx app.RequestContext) {
		ctx.Next()
	}
}

func CORS() app.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	})
}
`), 0644)

	// 为每个 module 生成 rpc_client 和 handler
	moduleKeys2 := make([]string, 0, len(moduleTables))
	for m := range moduleTables {
		moduleKeys2 = append(moduleKeys2, m)
	}
	for i, m := range moduleKeys2 {
		columns := moduleTables[m][0].Columns
		upper := strings.ToUpper(m[:1]) + m[1:]
		pkField := getPrimaryKeyField(columns)
		srvPort := 8000 + i + 1

		// 生成 rpc_client
		genBFFClientFromSchema(projectDir, bffName, m, upper, columns, protocol, m, srvPort)

		// 生成 Hertz handler
		genBFFHertzHandlerFromSchema(projectDir, bffName, m, upper, columns, pkField, srvPort)
	}
}

// genBFFHertzHandlerFromSchema 生成 BFF Hertz Handler（入参与 proto 对齐）
func genBFFHertzHandlerFromSchema(projectDir, bffName, m, upper string, columns []ColumnInfo, pkField ColumnInfo, srvPort int) {
	data := buildTemplateData(m, columns, bffName, m, srvPort)
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "handler", "handler_hertz_crud.go.tmpl")
	if err := renderTemplate(tmplPath, data, filepath.Join(projectDir, toBffDirName(bffName), "internal", "handler", toCamelFileName(m, "Handler.go"))); err != nil {
		fmt.Printf("ERROR rendering BFF Hertz CRUD handler %s: %v\n", m, err)
	}
}

// genMicroserviceFromSchema 基于表结构生成微服务 CRUD
func getTemplatesDir() string {
	// 方法1: 通过 runtime.Caller 获取源文件位置
	if _, filename, _, ok := runtime.Caller(0); ok {
		// filename = .../gospacex/internal/cli/microapp_new.go
		// 向上两级到项目根目录
		projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
		tmplDir := filepath.Join(projectRoot, "templates")
		if info, err := os.Stat(tmplDir); err == nil && info.IsDir() {
			return tmplDir
		}
	}

	// 方法2: 从可执行文件路径推导
	exePath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exePath)
		for i := 0; i < 10; i++ {
			tmplDir := filepath.Join(dir, "templates")
			if info, err := os.Stat(tmplDir); err == nil && info.IsDir() {
				return tmplDir
			}
			dir = filepath.Dir(dir)
		}
	}

	// 方法3: GOPATH 回退（处理 src 重复问题）
	gopath := os.Getenv("GOPATH")
	candidates := []string{
		filepath.Join(gopath, "gospacex", "templates"),
		filepath.Join(gopath, "src", "gospacex", "templates"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}

	return filepath.Join(gopath, "gospacex", "templates")
}
// =============================================================================
// 辅助函数
// =============================================================================

// genAuxModel 为附属表生成 model 文件（用于联表场景中副表的 model）
func genAuxModel(projectDir, module, modelName, tableName string, columns []ColumnInfo) {
	if len(columns) == 0 {
		return
	}
	tmplDir := filepath.Join(getTemplatesDir(), "microservice", "grpc", "schema")
	data := buildTemplateData(modelName, columns, "", tableName, 0)
	modelDir := filepath.Join(projectDir, toSrvDirName(module), "internal", "model")
	os.MkdirAll(modelDir, 0755)
	if err := renderTemplate(
		filepath.Join(tmplDir, "srv_model.go.tmpl"),
		data,
		filepath.Join(modelDir, modelName+".go"),
	); err != nil {
		fmt.Printf("ERROR rendering aux model %s: %v\n", modelName, err)
	}
	fmt.Printf("  Generated aux model: %s/internal/model/%s.go\n", toSrvDirName(module), modelName)
}

// getPrimaryKeyField 获取主键字段
func getPrimaryKeyField(columns []ColumnInfo) ColumnInfo {
	for _, col := range columns {
		if col.IsPrimary {
			return col
		}
	}
	// 如果没有主键，返回第一个字段
	if len(columns) > 0 {
		return columns[0]
	}
	return ColumnInfo{Name: "id", ProtoType: "int64"}
}

// getUpperFirst 首字母大写
func getUpperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// toProtoGoFieldName 将 snake_case 字段名转为 proto 生成的 Go 字段名（CamelCase）
// 例如: image_input -> ImageInput, admin_id -> AdminId, id -> Id
func toProtoGoFieldName(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			result.WriteString(part[1:])
		}
	}
	return result.String()
}

// getLowerFirst 首字母小写
func getLowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// protoTypeToGoType Proto 类型转 Go 类型
func protoTypeToGoType(protoType string) string {
	switch protoType {
	case "int64":
		return "int64"
	case "double":
		return "float64"
	case "string":
		return "string"
	case "bytes":
		return "[]byte"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

// isBoolField 判断字段名是否表示 bool 类型（is_ 前缀或 _flag 后缀）
// 前提：调用方已确认列类型是 tinyint（Caller must verify col.Type is "tinyint"）
func isBoolField(colName string) bool {
	lower := strings.ToLower(colName)
	return strings.HasPrefix(lower, "is_") ||
		strings.HasSuffix(lower, "_flag") ||
		lower == "status" || lower == "deleted" || lower == "enabled" ||
		lower == "active" || lower == "locked"
}

// colToModelType 转换列类型为模型类型
func colToModelType(col ColumnInfo) string {
	// 仅当列类型是 tinyint 且字段名为 bool 类命名时，才映射为 bool
	// 注意：不要只看命名，如 is_gift 列是 int 类型，不能转 bool
	if strings.ToLower(col.Type) == "tinyint" && isBoolField(col.Name) {
		return "bool"
	}
	switch strings.ToLower(col.Type) {
	case "int", "smallint", "mediumint", "bigint", "tinyint":
		return "int"
	case "float", "double", "decimal":
		return "float64"
	case "char", "varchar", "text", "longtext", "mediumtext", "tinytext":
		return "string"
	case "blob", "binary", "varbinary":
		return "[]byte"
	case "datetime", "timestamp", "date", "time":
		return "time.Time"
	case "bool", "boolean":
		return "bool"
	default:
		return "string"
	}
}

// pkFieldToModelID 主键字段转模型 ID 类型
func pkFieldToModelID(modelType string) string {
	if modelType == "int64" {
		return "uint64"
	}
	return "uint"
}

// getGormTag 根据列信息生成 GORM tag
func getGormTag(col ColumnInfo) string {
	tag := `gorm:"`
	if col.IsPrimary {
		tag += "primaryKey;"
	}
	tag += "column:" + col.Name
	if col.Comment != "" {
		tag += ";comment:" + col.Comment
	}
	tag += `"`
	return tag
}

// modelToRespField 模型字段转 Proto 响应字段
func modelToRespField(col ColumnInfo) string {
	field := "m." + getUpperFirst(col.Name)
	modelType := colToModelType(col)
	protoType := col.ProtoType

	// 类型转换
	if modelType == "uint" && protoType == "int64" {
		return "int64(" + field + ")"
	}
	if modelType == "int" && protoType == "int64" {
		return "int64(" + field + ")"
	}
	if modelType == "int64" && protoType == "int64" {
		return field
	}
	if modelType == "float64" && protoType == "double" {
		return field
	}
	if modelType == "time.Time" && protoType == "string" {
		return "m." + getUpperFirst(col.Name) + ".Format(\"2006-01-02 15:04:05\")"
	}
	return field
}

// updateReqFieldsWithoutPK 生成不含主键的 Update 请求字段声明
func updateReqFieldsWithoutPK(columns []ColumnInfo) string {
	var buf strings.Builder
	for _, col := range columns {
		if col.IsPrimary {
			continue
		}
		buf.WriteString(fmt.Sprintf("\t\t%s %s `json:\"%s\"`\n", col.Name, protoTypeToGoType(col.ProtoType), col.Name))
	}
	return buf.String()
}

func init() {
	newMicroAppCmd.Flags().StringVar(&microAppName, "name", "", "微应用名称（必填）")
	newMicroAppCmd.Flags().StringVarP(&microAppOutputDir, "output", "o", "", "项目输出目录（必填）")
	newMicroAppCmd.Flags().StringVar(&microAppStyle, "style", "standard", "架构风格: standard (默认)")
	newMicroAppCmd.Flags().StringVar(&microAppIDL, "idl", "proto", "IDL 类型: proto (默认), thrift")
	newMicroAppCmd.Flags().StringVar(&microAppProtocol, "protocol", "grpc", "通信协议: grpc (默认), kitex")
	newMicroAppCmd.Flags().StringVar(&microAppHTTP, "http", "gin", "BFF HTTP 框架: gin (默认), hertz")
	newMicroAppCmd.Flags().StringVar(&microAppBFFName, "bff", "", "BFF 名称（必填）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppModules, "modules", nil, "微服务列表，支持中英文逗号分隔（必填）")
	newMicroAppCmd.Flags().StringVar(&microAppDBHost, "db-host", "127.0.0.1", "数据库主机")
	newMicroAppCmd.Flags().StringVar(&microAppDBPort, "db-port", "3306", "数据库端口")
	newMicroAppCmd.Flags().StringVar(&microAppDBUser, "db-user", "root", "数据库用户")
	newMicroAppCmd.Flags().StringVar(&microAppDBPassword, "db-password", "123456", "数据库密码")
	newMicroAppCmd.Flags().StringVar(&microAppDBName, "db-name", "myshop", "数据库名称")
	newMicroAppCmd.Flags().StringVar(&microAppDBTable, "db-table", "", "数据库表名，用于根据表结构生成 CRUD")
	newMicroAppCmd.Flags().BoolVar(&microAppTest, "test", false, "自动生成 BFF 层和微服层的接口测试代码")
	newMicroAppCmd.Flags().BoolVar(&microAppOtel, "otel", false, "启用 OpenTelemetry 分布式追踪（生成调用链代码）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppSrvs, "srvs", nil, "微服务列表（--modules 的别名），支持中英文逗号分隔")
	newMicroAppCmd.Flags().StringVar(&microAppRegister, "register", "", "服务注册中心: consul|etcd（不指定则 BFF 直连 SRV）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppJoinKey, "db-join-key", nil, "联表条件（兼容旧格式），格式: table1.field1=table2.field2（可多次指定）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppJoinKey, "db-join-condition", nil, "联表条件，格式: table1.field1=table2.field2（可多次指定）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppJoinKey, "djc", nil, "联表条件（简写），格式: table1.field1=table2.field2（可多次指定）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppJoinStyle, "db-join-style", nil, "联表关系（兼容旧格式），格式: table1:table2=1t1|1tn|nt1|ntn（可多次指定）")
	newMicroAppCmd.Flags().StringArrayVar(&microAppJoinStyle, "djs", nil, "联表关系（简写），格式: table1:table2=1t1|1tn|nt1|ntn（可多次指定）")
	newMicroAppCmd.Flags().StringVar(&microAppMiddleware, "middleware", "", "生成中间件（BFF: jwt,ratelimit,blacklist; SRV: ratelimit,retry,timeout,tracing）")
	newMicroAppCmd.Flags().StringVar(&microAppConfig, "config", "", "配置中心: nacos|viper（默认从本地 config.yaml 读取）")
	newMicroAppCmd.Flags().StringVar(&microAppConfigFile, "config-file", "", "从配置文件读取参数，支持 yaml/json/toml 格式")
}

// =============================================================================
// 测试代码生成
// =============================================================================

// genTestDirs 生成 BFF 和微服层的测试目录和接口测试代码
func genTestDirs(projectDir, bffName string, modules []string, hasDBTable bool, tableColumns map[string][]ColumnInfo, moduleTableName map[string]string) {
	fmt.Println("Generating test directories and test code...")

	// ========== BFF 层测试 ==========
	bffTestDir := filepath.Join(projectDir, toBffDirName(bffName), "test")
	os.MkdirAll(bffTestDir, 0755)

	// BFF 接口测试文件
	genBFFTestFile(projectDir, bffName, modules, hasDBTable, tableColumns, moduleTableName, bffTestDir)

	// ========== 微服层测试 ==========
	for i, m := range modules {
		srvTestDir := filepath.Join(projectDir, toSrvDirName(m), "test")
		os.MkdirAll(srvTestDir, 0755)

		upper := strings.ToUpper(m[:1]) + m[1:]
		srvPort := 8001 + i
		if hasDBTable && tableColumns[m] != nil {
			tn := moduleTableName[m]
			if tn == "" {
				tn = m
			}
			genSrvTestFileFromSchema(projectDir, m, upper, tableColumns[m], srvTestDir, tn, srvPort)
		} else {
			genSrvTestFile(projectDir, m, upper, srvTestDir)
		}
	}

	fmt.Println("✓ Test directories and code generated")
}

// genShellScripts 生成测试 shell 脚本（tests/ 目录）
func genShellScripts(projectDir, bffName string, modules []string, bffPort, srvPort int, moduleTableName map[string]string) {
	testsDir := filepath.Join(projectDir, "tests")
	os.MkdirAll(testsDir, 0755)

	type testModule struct {
		Name        string
		UpperName  string
		Package    string
		RouteName  string
		ServiceName string
	}
	var testModules []testModule
	for _, m := range modules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		tableName := m
		if moduleTableName != nil {
			if tn, ok := moduleTableName[m]; ok && tn != "" {
				tableName = tn
			}
		}
		entityName := toProtoGoFieldName(tableName)
		testModules = append(testModules, testModule{
			Name:        m,
			UpperName:   upper,
			Package:     m,
			RouteName:   tableName,
			ServiceName: entityName + "Service",
		})
	}

	// BFF 测试脚本
	bffTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "test", "scripts", "run_bff_tests.sh.tmpl")
	bffTmplBytes, err := os.ReadFile(bffTmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFF test script template: %v\n", err)
	} else {
		bffContent, err := executeTemplate(string(bffTmplBytes), map[string]interface{}{
			"BFFPort": bffPort,
			"Modules": testModules,
		})
		if err != nil {
			fmt.Printf("ERROR executing BFF test script template: %v\n", err)
		} else {
			os.WriteFile(filepath.Join(testsDir, "run_bff_tests.sh"), []byte(bffContent), 0755)
			fmt.Printf("  Generated: tests/run_bff_tests.sh\n")
		}
	}

	// Service 测试脚本
	srvTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "test", "scripts", "run_srv_tests.sh.tmpl")
	srvTmplBytes, err := os.ReadFile(srvTmplPath)
	if err != nil {
		fmt.Printf("ERROR reading Service test script template: %v\n", err)
	} else {
		protoPath := "common/idl/product.proto"
		if len(testModules) > 0 && testModules[0].RouteName != "" {
			protoPath = "common/idl/" + testModules[0].RouteName + ".proto"
		}
		srvContent, err := executeTemplate(string(srvTmplBytes), map[string]interface{}{
			"SRVPort":   srvPort,
			"Modules":   testModules,
			"ProtoPath":  protoPath,
		})
		if err != nil {
			fmt.Printf("ERROR executing Service test script template: %v\n", err)
		} else {
			os.WriteFile(filepath.Join(testsDir, "run_srv_tests.sh"), []byte(srvContent), 0755)
			fmt.Printf("  Generated: tests/run_srv_tests.sh\n")
		}
	}
}

// genBFFTestFile 生成 BFF 层接口测试（基于 httptest）
func genBFFTestFile(projectDir, bffName string, modules []string, hasDBTable bool, tableColumns map[string][]ColumnInfo, moduleTableName map[string]string, testDir string) {
	type testModule struct {
		Name      string
		UpperName string
		RouteName string
	}
	var testModules []testModule
	for _, m := range modules {
		upper := strings.ToUpper(m[:1]) + m[1:]
		tableName := m
		if moduleTableName != nil {
			if tn, ok := moduleTableName[m]; ok && tn != "" {
				tableName = tn
			}
		}
		testModules = append(testModules, testModule{Name: m, UpperName: upper, RouteName: tableName})
	}

	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "test", "handler_test.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading BFF test template: %v\n", err)
		return
	}
	testContent, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"Modules": testModules,
	})
	if err != nil {
		fmt.Printf("ERROR executing BFF test template: %v\n", err)
		return
	}

	// 补充 import
	finalContent := testContent
	if strings.Contains(finalContent, "strings.NewReader") {
		finalContent = strings.Replace(finalContent, `"testing"`, "\"strings\"\n\t\"testing\"", 1)
	}

	os.WriteFile(filepath.Join(testDir, "handler_test.go"), []byte(finalContent), 0644)
	fmt.Printf("  Generated: %s/test/handler_test.go\n", toBffDirName(bffName))
}

// genSrvTestFile 生成微服层接口测试（默认模式，无表结构）
func genSrvTestFile(projectDir, module, upper, testDir string) {
	tmplDir := filepath.Join(getTemplatesDir(), "micro-app", "srv", "test")
	data := buildTemplateData(module, nil, "", "", 0)
	if err := renderTemplate(
		filepath.Join(tmplDir, "handler_test.go.tmpl"),
		data,
		filepath.Join(testDir, "handler_test.go"),
	); err != nil {
		fmt.Printf("ERROR rendering srv test %s: %v\n", module, err)
	} else {
		fmt.Printf("  Generated: %s/test/handler_test.go\n", toSrvDirName(module))
	}
}

// genSrvTestFileFromSchema 生成微服层接口测试（基于表结构，更详细）
func genSrvTestFileFromSchema(projectDir, module, upper string, columns []ColumnInfo, testDir string, tableName string, srvPort int) {
	data := buildTemplateData(module, columns, "", tableName, srvPort, tableName)
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "srv", "test", "handler_test_schema.go.tmpl")
	if err := renderTemplate(tmplPath, data, filepath.Join(testDir, "handler_test.go")); err != nil {
		fmt.Printf("ERROR rendering srv test from schema %s: %v\n", module, err)
	} else {
		fmt.Printf("  Generated: %s/test/handler_test.go (with schema)\n", toSrvDirName(module))
	}
}

// sampleValueForType 根据 Proto 类型和字段名生成示例值
func sampleValueForType(protoType, fieldName string) string {
	nameLower := strings.ToLower(fieldName)
	switch protoType {
	case "int64":
		if strings.Contains(nameLower, "id") {
			return "1"
		}
		if strings.Contains(nameLower, "status") {
			return "1"
		}
		return "0"
	case "double":
		return "0.0"
	case "bool":
		if strings.Contains(nameLower, "is_") || strings.Contains(nameLower, "has_") {
			return "true"
		}
		return "false"
	case "string":
		if strings.Contains(nameLower, "name") {
			return `"test_name"`
		}
		if strings.Contains(nameLower, "title") {
			return `"test_title"`
		}
		if strings.Contains(nameLower, "email") {
			return `"test@example.com"`
		}
		if strings.Contains(nameLower, "phone") {
			return `"13800138000"`
		}
		if strings.Contains(nameLower, "url") || strings.Contains(nameLower, "link") {
			return `"https://example.com"`
		}
		if strings.Contains(nameLower, "image") || strings.Contains(nameLower, "img") || strings.Contains(nameLower, "avatar") {
			return `"https://example.com/img.png"`
		}
		if strings.Contains(nameLower, "desc") || strings.Contains(nameLower, "content") || strings.Contains(nameLower, "description") {
			return `"test description"`
		}
		if strings.Contains(nameLower, "time") || strings.Contains(nameLower, "date") || strings.Contains(nameLower, "at") {
			return `"2026-01-01 00:00:00"`
		}
		return `"test_value"`
	case "bytes":
		return "[]byte(\"test\")"
	default:
		return `"test"`
	}
}

// =============================================================================
// 联表查询代码生成
// =============================================================================

// genJoinCode 生成联表查询代码
func genJoinCode(projectDir string, modules []string, joinCfg *JoinConfig, allTableCols map[string][]ColumnInfo) {
	fmt.Printf("Generating join query code: %s.%s = %s.%s (%s)\n", joinCfg.LeftTable, joinCfg.LeftField, joinCfg.RightTable, joinCfg.RightField, joinCfg.Style)

	// 获取所有实际表名
	var actualTableNames []string
	for tn := range allTableCols {
		actualTableNames = append(actualTableNames, tn)
	}

	// 找到左表和右表对应的 module
	leftModule := ""
	rightModule := ""
	for _, m := range modules {
		if m == joinCfg.LeftTable {
			leftModule = m
		}
		if m == joinCfg.RightTable {
			rightModule = m
		}
	}
	// 如果 module 名和表名不匹配，用第一个 module 作为左表
	if leftModule == "" && len(modules) > 0 {
		leftModule = modules[0]
	}

	// 确定实际的左表和右表名（用于推导 entity 名）
	// 如果 joinCfg 的表名不在 actualTableNames 中，尝试模糊匹配
	leftTableName := joinCfg.LeftTable
	rightTableName := joinCfg.RightTable
	rightIsAux := false

	// 左表：尝试在 actualTableNames 中找到匹配的表
	leftMatched := false
	for _, actualTn := range actualTableNames {
		if actualTn == joinCfg.LeftTable || strings.Contains(actualTn, joinCfg.LeftTable) {
			leftTableName = actualTn
			leftMatched = true
			break
		}
	}
	// 如果左表没有匹配，也尝试第一个 module 对应的表
	if !leftMatched && len(modules) > 0 && len(actualTableNames) > 0 {
		// 使用第一个实际表名
		leftTableName = actualTableNames[0]
	}

	// 右表：尝试在 actualTableNames 中找到匹配的表
	rightMatched := false
	for _, actualTn := range actualTableNames {
		if actualTn == joinCfg.RightTable || strings.Contains(actualTn, joinCfg.RightTable) {
			rightTableName = actualTn
			rightMatched = true
			break
		}
	}
	if !rightMatched {
		// 右表是附属表
		rightIsAux = true
		if len(actualTableNames) > 1 {
			rightTableName = actualTableNames[1]
		}
	}

	// 使用 tableToEntityName 获取正确的 entity 名称
	leftEntityName := tableToEntityName(leftTableName, actualTableNames)
	rightEntityName := tableToEntityName(rightTableName, actualTableNames)
	leftUpper := strings.ToUpper(leftEntityName[:1]) + leftEntityName[1:]
	rightUpper := strings.ToUpper(rightEntityName[:1]) + rightEntityName[1:]

	// 生成联表 Repository
	genJoinRepository(projectDir, leftModule, leftUpper, rightModule, rightUpper, joinCfg, rightIsAux)

	// 生成联表 Service
	genJoinService(projectDir, leftModule, leftUpper, rightModule, rightUpper, joinCfg, rightIsAux)

	// BFF join handler 暂时跳过（原始 myshop 无此功能，且 BFF client 模板未实现 ListByXxx 方法）
	// genJoinHandler(projectDir, leftModule, leftUpper, rightModule, rightUpper, joinCfg)

	fmt.Println("✓ Join query code generated")
}

// tableToModelName 从表名推导 model 名称
// 去掉公共前缀，如 eb_store_product_description → product_description
func tableToModelName(tableName string, allTableCols map[string][]ColumnInfo) string {
	var tableNames []string
	for tn := range allTableCols {
		tableNames = append(tableNames, tn)
	}
	if len(tableNames) < 2 {
		return tableName
	}
	// 找公共前缀
	prefix := tableNames[0]
	for _, tn := range tableNames[1:] {
		minLen := len(prefix)
		if len(tn) < minLen {
			minLen = len(tn)
		}
		for i := 0; i < minLen; i++ {
			if prefix[i] != tn[i] {
				prefix = prefix[:i]
				break
			}
		}
	}
	prefix = strings.TrimRight(prefix, "_")
	if len(prefix) > 0 && strings.HasPrefix(tableName, prefix+"_") {
		return strings.TrimPrefix(tableName, prefix+"_")
	}
	return tableName
}

// genJoinRepository 生成联表查询 Repository 代码
func genJoinRepository(projectDir, leftModule, leftUpper, rightModule, rightUpper string, joinCfg *JoinConfig, rightIsAux bool) {
	var repoContent strings.Builder

	// import 路径：如果右表是附属表，model 在同一个 srv 目录下
	// 注意：rightModule 可能为空（当 module 名和表名不匹配时），此时用 leftModule 的路径
	rightImport := fmt.Sprintf("%s/%s/internal/model", microAppName, toSrvDirName(rightModule))
	if rightIsAux || rightModule == "" {
		rightImport = fmt.Sprintf("%s/%s/internal/model", microAppName, toSrvDirName(leftModule))
	}
	leftImport := fmt.Sprintf("%s/%s/internal/model", microAppName, toSrvDirName(leftModule))

	// 当两个 import 相同时，使用同一 package 模板（避免重复 import 错误）
	useSamePkg := leftImport == rightImport
	if useSamePkg {
		// 同一个 package，合并 import - 使用模板
		headerTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "repository", "header_same_pkg.go.tmpl")
		headerTmplBytes, err := os.ReadFile(headerTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		headerResult, err := executeTemplate(string(headerTmplBytes), map[string]interface{}{
			"LeftImport": leftImport,
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		repoContent.WriteString(headerResult)
	} else {
		// 不同 package - 使用模板
		headerTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "repository", "header_diff_pkg.go.tmpl")
		headerTmplBytes, err := os.ReadFile(headerTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		headerResult, err := executeTemplate(string(headerTmplBytes), map[string]interface{}{
			"LeftImport": leftImport,
			"RightImport": rightImport,
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		repoContent.WriteString(headerResult)
	}

	switch joinCfg.Style {
	case "1t1":
		// 一对一：每条左表记录对应一条右表记录 - 使用模板
		findTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "repository", "find_1t1.go.tmpl")
		findTmplBytes, err := os.ReadFile(findTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		findResult, err := executeTemplate(string(findTmplBytes), map[string]interface{}{
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
			"LeftModule": leftModule,
			"RightModule": rightModule,
			"RightField": joinCfg.RightField,
			"LeftField":  joinCfg.LeftField,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		repoContent.WriteString(findResult)

	case "1tn":
		// 一对多：一条左表记录对应多条右表记录 - 使用模板
		findTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "repository", "find_1tn.go.tmpl")
		findTmplBytes, err := os.ReadFile(findTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		findResult, err := executeTemplate(string(findTmplBytes), map[string]interface{}{
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
			"RightModule": rightModule,
			"RightField": joinCfg.RightField,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		repoContent.WriteString(findResult)

	case "nt1":
		// 多对一：多条左表记录对应一条右表记录 - 使用模板
		findTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "repository", "find_nt1.go.tmpl")
		findTmplBytes, err := os.ReadFile(findTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		findResult, err := executeTemplate(string(findTmplBytes), map[string]interface{}{
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
			"LeftModule": leftModule,
			"LeftField":  joinCfg.LeftField,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		repoContent.WriteString(findResult)
	}

	joinRepoDir := filepath.Join(projectDir, toSrvDirName(leftModule), "internal", "repository")
	os.MkdirAll(joinRepoDir, 0755)
	os.WriteFile(filepath.Join(joinRepoDir, "join"+getUpperFirst(leftModule)+getUpperFirst(rightModule)+"Repo.go"), []byte(repoContent.String()), 0644)
	fmt.Printf("  Generated: %s/internal/repository/join%s%sRepo.go\n", toSrvDirName(leftModule), getUpperFirst(leftModule), getUpperFirst(rightModule))
}

// genJoinService 生成联表查询 Service 代码
func genJoinService(projectDir, leftModule, leftUpper, rightModule, rightUpper string, joinCfg *JoinConfig, rightIsAux bool) {
	var svcContent strings.Builder

	// import 路径：如果右表是附属表，model 在同一个 srv 目录下
	// 注意：rightModule 可能为空（当 module 名和表名不匹配时），此时用 leftModule 的路径
	rightImport := fmt.Sprintf("%s/%s/internal/model", microAppName, toSrvDirName(rightModule))
	rightRepoImport := fmt.Sprintf("%s/%s/internal/repository", microAppName, toSrvDirName(rightModule))
	if rightIsAux || rightModule == "" {
		rightImport = fmt.Sprintf("%s/%s/internal/model", microAppName, toSrvDirName(leftModule))
		rightRepoImport = fmt.Sprintf("%s/%s/internal/repository", microAppName, toSrvDirName(leftModule))
	}
	leftImport := fmt.Sprintf("%s/%s/internal/model", microAppName, toSrvDirName(leftModule))
	leftRepoImport := fmt.Sprintf("%s/%s/internal/repository", microAppName, toSrvDirName(leftModule))

	// 当两个 import 相同时，使用同一 package 模板（避免重复 import 错误）
	useSamePkg := leftImport == rightImport && leftRepoImport == rightRepoImport
	if useSamePkg {
		// 同一个 package - 使用模板
		headerTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "service", "header_same_pkg.go.tmpl")
		headerTmplBytes, err := os.ReadFile(headerTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		headerResult, err := executeTemplate(string(headerTmplBytes), map[string]interface{}{
			"LeftImport":    leftImport,
			"LeftRepoImport": leftRepoImport,
			"LeftUpper":     leftUpper,
			"RightUpper":    rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		svcContent.WriteString(headerResult)
	} else {
		// 不同 package - 使用模板
		headerTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "service", "header_diff_pkg.go.tmpl")
		headerTmplBytes, err := os.ReadFile(headerTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		headerResult, err := executeTemplate(string(headerTmplBytes), map[string]interface{}{
			"LeftImport":     leftImport,
			"RightImport":    rightImport,
			"LeftRepoImport": leftRepoImport,
			"RightRepoImport": rightRepoImport,
			"LeftUpper":     leftUpper,
			"RightUpper":    rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		svcContent.WriteString(headerResult)
	}

	switch joinCfg.Style {
	case "1t1":
		// 一对一 - 使用模板
		getTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "service", "get_1t1.go.tmpl")
		getTmplBytes, err := os.ReadFile(getTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		getResult, err := executeTemplate(string(getTmplBytes), map[string]interface{}{
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		svcContent.WriteString(getResult)

	case "1tn":
		// 一对多 - 使用模板
		getTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "service", "get_1tn.go.tmpl")
		getTmplBytes, err := os.ReadFile(getTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		getResult, err := executeTemplate(string(getTmplBytes), map[string]interface{}{
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		svcContent.WriteString(getResult)

	case "nt1":
		// 多对一 - 使用模板
		getTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "service", "get_nt1.go.tmpl")
		getTmplBytes, err := os.ReadFile(getTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		getResult, err := executeTemplate(string(getTmplBytes), map[string]interface{}{
			"LeftUpper":  leftUpper,
			"RightUpper": rightUpper,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		svcContent.WriteString(getResult)
	}

	joinSvcDir := filepath.Join(projectDir, toSrvDirName(leftModule), "internal", "service")
	os.MkdirAll(joinSvcDir, 0755)
	os.WriteFile(filepath.Join(joinSvcDir, "join"+getUpperFirst(leftModule)+getUpperFirst(rightModule)+"Service.go"), []byte(svcContent.String()), 0644)
	fmt.Printf("  Generated: %s/internal/service/join%s%sService.go\n", toSrvDirName(leftModule), getUpperFirst(leftModule), getUpperFirst(rightModule))
}

// genJoinHandler 生成联表查询 BFF Handler 代码
func genJoinHandler(projectDir, leftModule, leftUpper, rightModule, rightUpper string, joinCfg *JoinConfig) {
	var handlerContent strings.Builder

	// Go 结构体字段名（首字母大写），用于变量命名
	leftVarName := leftUpper  // e.g. Product
	rightVarName := rightUpper // e.g. Description

	// 使用模板生成 Handler 头部
	headerTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "handler", "header.go.tmpl")
	headerTmplBytes, err := os.ReadFile(headerTmplPath)
	if err != nil {
		fmt.Printf("Error reading template: %v\n", err)
		return
	}
	headerResult, err := executeTemplate(string(headerTmplBytes), map[string]interface{}{
		"AppName":      microAppName,
		"BffDirName":   toBffDirName(microAppBFFName),
		"LeftUpper":    leftUpper,
		"RightUpper":   rightUpper,
		"LeftVarName":  leftVarName,
		"RightVarName": rightVarName,
		"LeftModule":   leftModule,
		"RightModule":  rightModule,
	})
	if err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		return
	}
	handlerContent.WriteString(headerResult)

	switch joinCfg.Style {
	case "1t1":
		// 一对一 - 使用模板
		getTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "handler", "get_1t1.go.tmpl")
		getTmplBytes, err := os.ReadFile(getTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		getResult, err := executeTemplate(string(getTmplBytes), map[string]interface{}{
			"LeftUpper":    leftUpper,
			"RightUpper":   rightUpper,
			"LeftVarName":  leftVarName,
			"RightVarName": rightVarName,
			"LeftModule":   leftModule,
			"RightModule":  rightModule,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		handlerContent.WriteString(getResult)

	case "1tn":
		// 一对多 - 使用模板
		getTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "handler", "get_1tn.go.tmpl")
		getTmplBytes, err := os.ReadFile(getTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		getResult, err := executeTemplate(string(getTmplBytes), map[string]interface{}{
			"LeftUpper":    leftUpper,
			"RightUpper":   rightUpper,
			"LeftVarName":  leftVarName,
			"RightVarName": rightVarName,
			"LeftModule":   leftModule,
			"RightModule":  rightModule,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		handlerContent.WriteString(getResult)

	case "nt1":
		// 多对一 - 使用模板
		getTmplPath := filepath.Join(getTemplatesDir(), "micro-app", "join", "handler", "get_nt1.go.tmpl")
		getTmplBytes, err := os.ReadFile(getTmplPath)
		if err != nil {
			fmt.Printf("Error reading template: %v\n", err)
			return
		}
		getResult, err := executeTemplate(string(getTmplBytes), map[string]interface{}{
			"LeftUpper":    leftUpper,
			"RightUpper":   rightUpper,
			"LeftVarName":  leftVarName,
			"RightVarName": rightVarName,
			"LeftModule":   leftModule,
			"RightModule":  rightModule,
		})
		if err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			return
		}
		handlerContent.WriteString(getResult)
	}

	joinHandlerDir := filepath.Join(projectDir, toBffDirName(microAppBFFName), "internal", "handler")
	os.MkdirAll(joinHandlerDir, 0755)
	os.WriteFile(filepath.Join(joinHandlerDir, "join"+getUpperFirst(leftModule)+getUpperFirst(rightModule)+"Handler.go"), []byte(handlerContent.String()), 0644)
	fmt.Printf("  Generated: %s/internal/handler/join%s%sHandler.go\n", toBffDirName(microAppBFFName), getUpperFirst(leftModule), getUpperFirst(rightModule))
}

func GetMicroAppCmd() *cobra.Command {
	return newMicroAppCmd
}

func GetMicroAppNewCmd() *cobra.Command {
	return newMicroAppCmd
}

// execTemplateOrFail reads a template file relative to the micro-app templates dir and executes it.
// If there is an error, it panics (should only be used in code generation where errors are fatal).
func execTemplateOrFail(relPath string, data interface{}) string {
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", relPath)
	tmplBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("Error reading template %s: %v\n", relPath, err)
		return ""
	}
	result, err := executeTemplate(string(tmplBytes), data)
	if err != nil {
		fmt.Printf("Error executing template %s: %v\n", relPath, err)
		return ""
	}
	return result
}

// executeTemplate executes a Go text template with the given data
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

// toSrvDirName 生成 srv 目录名（小驼峰）：srvProduct, srvOrder
func toSrvDirName(module string) string {
	if module == "" || module == "srv" {
		return "srv"
	}
	// 仅当模块名长度 > 1 且不是 srv 时添加前缀
	return "srv" + strings.ToUpper(module[:1]) + module[1:]
}

// toBffDirName 生成 BFF 目录名（小驼峰）：bffApi, bffH5, bff
func toBffDirName(bffName string) string {
	if bffName == "" || bffName == "bff" {
		return "bff"
	}
	// 仅当 bff 名称长度 > 1 且不是 bff 时添加前缀
	return "bff" + strings.ToUpper(bffName[:1]) + bffName[1:]
}

// toCamelFileName 生成小驼峰文件名：productHandler.go, productRepo.go
func toCamelFileName(module, suffix string) string {
	if suffix == "" {
		return module
	}
	return module + strings.ToUpper(suffix[:1]) + suffix[1:]
}

// =============================================================================
// R6: --middleware 中间件/拦截器生成
// =============================================================================

// genMiddleware 生成 BFF 中间件和 SRV 拦截器
// --middleware jwt,ratelimit,blacklist
func genMiddleware(projectDir, bffName string, modules []string) {
	fmt.Println("Generating middleware/interceptor code...")

	middlewareList := strings.Split(microAppMiddleware, ",")
	for i := range middlewareList {
		middlewareList[i] = strings.TrimSpace(middlewareList[i])
	}

	// === BFF 中间件 ===
	bffMiddlewareDir := filepath.Join(projectDir, toBffDirName(bffName), "internal", "middleware")
	os.MkdirAll(bffMiddlewareDir, 0755)

	// 生成 middleware.go (Builder 入口)
	middlewareBuilderPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "middleware.go.tmpl")
	if _, err := os.Stat(middlewareBuilderPath); err == nil {
		middlewareBuilderStr, err := os.ReadFile(middlewareBuilderPath)
		if err != nil {
			fmt.Printf("ERROR reading middleware builder template: %v\n", err)
		} else {
			middlewareGo, err := executeTemplate(string(middlewareBuilderStr), map[string]interface{}{
				"AppName": microAppName,
				"BFFName": bffName,
			})
			if err != nil {
				fmt.Printf("ERROR executing middleware builder template: %v\n", err)
			} else {
				os.WriteFile(filepath.Join(bffMiddlewareDir, "middleware.go"), []byte(middlewareGo), 0644)
				fmt.Printf("  Generated BFF middleware builder: %s/internal/middleware/middleware.go\n", toBffDirName(bffName))
			}
		}
	}

	// 根据 --middleware 生成对应的适配器
	for _, m := range middlewareList {
		var tmplPath string
		var outputFile string

		switch m {
		case "jwt":
			if microAppHTTP == "hertz" {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "hertz_jwt.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "hertz_jwt.go")
			} else {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "gin_jwt.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "gin_jwt.go")
			}
		case "ratelimit":
			if microAppHTTP == "hertz" {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "hertz_ratelimit.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "hertz_ratelimit.go")
			} else {
				tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "gin_ratelimit.go.tmpl")
				outputFile = filepath.Join(bffMiddlewareDir, "gin_ratelimit.go")
			}
		case "blacklist":
			if microAppHTTP == "hertz" {
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
			"AppName": microAppName,
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
				"AppName": microAppName,
			})
			if err == nil {
				configDir := filepath.Join(projectDir, toBffDirName(bffName), "configs")
				os.MkdirAll(configDir, 0755)
				os.WriteFile(filepath.Join(configDir, "middleware.yaml"), []byte(configContent), 0644)
				fmt.Printf("  Generated BFF middleware config: %s/configs/middleware.yaml\n", toBffDirName(bffName))
			}
		}
	}

	// === SRV 拦截器 ===
	for _, module := range modules {
		srvInterceptorDir := filepath.Join(projectDir, toSrvDirName(module), "internal", "interceptor")
		os.MkdirAll(srvInterceptorDir, 0755)

		// 生成 interceptor.go (Builder 入口)
		interceptorBuilderPath := filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "interceptor.go.tmpl")
		if _, err := os.Stat(interceptorBuilderPath); err == nil {
			interceptorBuilderStr, err := os.ReadFile(interceptorBuilderPath)
			if err == nil {
				interceptorGo, err := executeTemplate(string(interceptorBuilderStr), map[string]interface{}{
					"AppName": microAppName,
					"Module":  module,
				})
				if err == nil {
					os.WriteFile(filepath.Join(srvInterceptorDir, "interceptor.go"), []byte(interceptorGo), 0644)
					fmt.Printf("  Generated SRV interceptor builder: %s/internal/interceptor/interceptor.go\n", toSrvDirName(module))
				}
			}
		}

		// 根据 --middleware 生成对应的拦截器
		for _, m := range middlewareList {
			var tmplPath string
			var outputFile string

			switch m {
			case "ratelimit":
				if microAppProtocol == "kitex" {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "kitex_ratelimit.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "kitex_ratelimit.go")
				} else {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "grpc_ratelimit.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "grpc_ratelimit.go")
				}
			case "retry":
				if microAppProtocol == "kitex" {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "kitex_retry.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "kitex_retry.go")
				} else {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "grpc_retry.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "grpc_retry.go")
				}
			case "timeout":
				if microAppProtocol == "kitex" {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "kitex_timeout.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "kitex_timeout.go")
				} else {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "grpc_timeout.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "grpc_timeout.go")
				}
			case "tracing":
				if microAppProtocol == "kitex" {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "kitex_tracing.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "kitex_tracing.go")
				} else {
					tmplPath = filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "grpc_tracing.go.tmpl")
					outputFile = filepath.Join(srvInterceptorDir, "grpc_tracing.go")
				}
			default:
				continue
			}

			tmplStr, err := os.ReadFile(tmplPath)
			if err != nil {
				fmt.Printf("ERROR reading SRV interceptor template %s: %v\n", tmplPath, err)
				continue
			}

			interceptorGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
				"AppName": microAppName,
				"Module":  module,
			})
			if err != nil {
				fmt.Printf("ERROR executing SRV interceptor template: %v\n", err)
				continue
			}
			os.WriteFile(outputFile, []byte(interceptorGo), 0644)
			fmt.Printf("  Generated SRV %s interceptor: %s\n", m, outputFile)
		}
	}
}

// =============================================================================
// R7: --config nacos 配置中心生成
// =============================================================================

// genNacosConfig 生成 Nacos 配置中心支持
func genNacosConfig(projectDir, bffName string, modules []string) {
	fmt.Println("Generating Nacos config center support...")

	// 生成 nacos_config.go 到 pkg/config/
	tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "config", "nacos_config.go.tmpl")
	tmplStr, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Printf("ERROR reading nacos config template: %v\n", err)
		return
	}

	nacosGo, err := executeTemplate(string(tmplStr), map[string]interface{}{
		"AppName": microAppName,
	})
	if err != nil {
		fmt.Printf("ERROR executing nacos config template: %v\n", err)
		return
	}
	os.WriteFile(filepath.Join(projectDir, "pkg", "config", "nacos_config.go"), []byte(nacosGo), 0644)
	fmt.Println("  Generated: pkg/config/nacos_config.go")

	// 更新 config.yaml 添加 nacos 配置段
	nacosYaml := "\n# Nacos 配置中心\nnacos:\n  server_addr: 127.0.0.1\n  port: 8848\n  namespace: \"\"\n  group: DEFAULT_GROUP\n  data_id: " + microAppName + ".yaml\n"

	// BFF config.yaml
	bffCfgPath := filepath.Join(projectDir, toBffDirName(bffName), "configs", "config.yaml")
	if existing, err := os.ReadFile(bffCfgPath); err == nil {
		os.WriteFile(bffCfgPath, append(existing, []byte(nacosYaml)...), 0644)
		fmt.Printf("  Updated BFF config: %s/configs/config.yaml (added nacos section)\n", toBffDirName(bffName))
	}

	// SRV config.yaml
	for _, m := range modules {
		srvCfgPath := filepath.Join(projectDir, toSrvDirName(m), "configs", "config.yaml")
		if existing, err := os.ReadFile(srvCfgPath); err == nil {
			os.WriteFile(srvCfgPath, append(existing, []byte(nacosYaml)...), 0644)
			fmt.Printf("  Updated SRV config: %s/configs/config.yaml (added nacos section)\n", toSrvDirName(m))
		}
	}

	// 更新 pkg/config/config.go 添加 NacosConfig 字段
	configGoPath := filepath.Join(projectDir, "pkg", "config", "config.go")
	if content, err := os.ReadFile(configGoPath); err == nil {
		cfgStr := string(content)
		// 在 Config struct 中添加 Nacos 字段
		cfgStr = strings.Replace(cfgStr,
			"Registry RegistryConfig `yaml:\"registry\"`",
			"Registry RegistryConfig `yaml:\"registry\"`\n\tNacos   NacosConfig   `yaml:\"nacos\"`",
			1,
		)
		// 在 RegistryConfig 之后添加 NacosConfig struct
		cfgStr = strings.Replace(cfgStr,
			"func Load(path string)",
			"// NacosConfig Nacos 配置中心\ntype NacosConfig struct {\n\tServerAddr string `yaml:\"server_addr\"`\n\tPort       uint64 `yaml:\"port\"`\n\tNamespace  string `yaml:\"namespace\"`\n\tGroup      string `yaml:\"group\"`\n\tDataID     string `yaml:\"data_id\"`\n}\n\nfunc Load(path string)",
			1,
		)
		os.WriteFile(configGoPath, []byte(cfgStr), 0644)
		fmt.Println("  Updated: pkg/config/config.go (added NacosConfig)")
	}
}
