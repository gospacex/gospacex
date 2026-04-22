package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/generator/monolith"
	"github.com/spf13/cobra"
)

// monolith 命令参数
var (
	monolithModule     string
	monolithOutputDir  string
	monolithDB         string
	monolithDBHost     string
	monolithDBPort     string
	monolithDBUser     string
	monolithDBPassword string
	monolithDBName     string
	monolithDBTable    string
	monolithRedis      bool
	monolithRedisAddr  string
	monolithES         bool
	monolithESAddrs    string
	monolithESIndex    string
	monolithMQ         string
	monolithMQBrokers  string
	monolithMQTopic    string
	monolithObjectDB   string
	monolithOSSEndpoint string
	monolithOSSBucket   string
	monolithMinioEndpoint string
	monolithMinioBucket   string
	monolithVector     string
	monolithZVecHost   string
	monolithZVecPort   string
)

var monolithCmd = &cobra.Command{
	Use:   "monolith [flags]",
	Short: "创建单体应用项目 (MVC架构)",
	Long: `创建单体应用项目 - 基于传统 MVC 架构，适合中小型项目快速启动。

支持的数据库: mysql, postgresql, sqlite
支持的缓存: redis
支持的搜索: elasticsearch
支持的MQ: kafka, rabbitmq, rocketmq
支持的对象存储: oss, minio, rustfs
支持的向量数据库: zvec

示例:

  # 创建基础单体项目
  gpx monolith --module shop --db mysql --db-name myshop

  # 创建带 Redis 的单体项目
  gpx monolith --module shop --db mysql --redis

  # 创建带完整技术栈的单体项目
  gpx monolith --module shop \
    --db mysql \
    --db-name myshop \
    --redis \
    --elasticsearch \
    --mq kafka \
    --objectdb minio`,
	RunE: runMonolith,
}

func runMonolith(cmd *cobra.Command, args []string) error {
	if monolithModule == "" {
		return fmt.Errorf("--module 参数必填")
	}

	// 确定输出目录
	outputDir := monolithOutputDir
	if outputDir == "" {
		outputDir = "./" + monolithModule
	}

	// 检查目录是否存在
	if _, err := os.Stat(outputDir); err == nil {
		return fmt.Errorf("输出目录已存在: %s", outputDir)
	}

	// 构建模块名
	moduleName := monolithModule
	if !strings.Contains(moduleName, "/") {
		moduleName = "github.com/gospacex/" + moduleName
	}

	// 构建数据库类型列表
	dbTypes := []string{monolithDB}
	if monolithDB == "" {
		dbTypes = []string{"mysql"} // 默认 MySQL
	}

	// 构建配置
	cfg := &config.ProjectConfig{
		ProjectType:  "monolith",
		ModuleName:   monolithModule,
		ProjectName:  moduleName,
		OutputDir:    outputDir,
		GoModuleName: moduleName,
		DB:           dbTypes,
		MySQLHost:    monolithDBHost,
		MySQLPort:    monolithDBPort,
		MySQLUser:    monolithDBUser,
		MySQLPassword: monolithDBPassword,
		MySQLDatabase: monolithDBName,
		MySQLTable:   monolithDBTable,
	}

	// 添加 Redis
	if monolithRedis {
		cfg.DB = append(cfg.DB, "redis")
		cfg.Cache = "redis"
		cfg.RedisAddr = monolithRedisAddr
		cfg.RedisPrefix = monolithModule + ":"
	}

	// 添加 ES
	if monolithES {
		cfg.DB = append(cfg.DB, "elasticsearch")
		cfg.ESHost = monolithESAddrs
		cfg.ESIndex = monolithESIndex
	}

	// 添加 MQ
	if monolithMQ != "" {
		cfg.DB = append(cfg.DB, monolithMQ)
		cfg.MQ = monolithMQ
		cfg.MQType = "basic"
		cfg.MQBrokers = monolithMQBrokers
	}

	// 添加对象存储
	if monolithObjectDB != "" {
		cfg.DB = append(cfg.DB, monolithObjectDB)
		cfg.ObjectDB = monolithObjectDB
		switch monolithObjectDB {
		case "oss":
			cfg.OSSEndpoint = monolithOSSEndpoint
			cfg.OSSBucket = monolithOSSBucket
		case "minio":
			cfg.MinioEndpoint = monolithMinioEndpoint
			cfg.MinioBucket = monolithMinioBucket
		case "rustfs":
			cfg.RustfsPath = "/data/rustfs"
		}
	}

	// 添加向量数据库
	if monolithVector != "" {
		cfg.DB = append(cfg.DB, monolithVector)
		cfg.ZVecHost = monolithZVecHost
		cfg.ZVecPort = monolithZVecPort
		cfg.ZVecCollection = monolithVector
	}

	fmt.Printf("创建单体项目: %s\n", monolithModule)
	fmt.Printf("  输出目录: %s\n", outputDir)
	fmt.Printf("  数据库: %s\n", strings.Join(cfg.DB, ", "))
	if monolithRedis {
		fmt.Printf("  Redis: %s\n", monolithRedisAddr)
	}
	if monolithES {
		fmt.Printf("  Elasticsearch: %s\n", monolithESAddrs)
	}
	if monolithMQ != "" {
		fmt.Printf("  MQ: %s\n", monolithMQ)
	}
	if monolithObjectDB != "" {
		fmt.Printf("  对象存储: %s\n", monolithObjectDB)
	}
	if monolithVector != "" {
		fmt.Printf("  向量数据库: %s\n", monolithVector)
	}
	fmt.Println()

	// 生成项目
	gen := monolith.NewGenerator(cfg)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("生成项目失败: %w", err)
	}

	// 生成 Docker 文件
	if err := gen.GenerateDocker(outputDir); err != nil {
		fmt.Printf("Warning: 生成 Docker 文件失败: %v\n", err)
	}

	fmt.Printf("\n✓ 单体项目创建成功!\n")
	fmt.Printf("  cd %s && go mod tidy\n", filepath.Base(outputDir))
	fmt.Printf("  cd %s && go run cmd/main.go\n", filepath.Base(outputDir))

	return nil
}

func init() {
	// 基础参数
	monolithCmd.Flags().StringVar(&monolithModule, "module", "", "单体模块名 (必填)")
	monolithCmd.Flags().StringVarP(&monolithOutputDir, "output", "o", "", "输出目录 (默认: ./<module>)")

	// 数据库参数
	monolithCmd.Flags().StringVar(&monolithDB, "db", "mysql", "数据库类型: mysql, pg, sqlite")
	monolithCmd.Flags().StringVar(&monolithDBHost, "db-host", "localhost", "数据库主机")
	monolithCmd.Flags().StringVar(&monolithDBPort, "db-port", "3306", "数据库端口")
	monolithCmd.Flags().StringVar(&monolithDBUser, "db-user", "root", "数据库用户")
	monolithCmd.Flags().StringVar(&monolithDBPassword, "db-password", "", "数据库密码")
	monolithCmd.Flags().StringVar(&monolithDBName, "db-name", "", "数据库名称")
	monolithCmd.Flags().StringVar(&monolithDBTable, "db-table", "", "数据表名")

	// Redis 参数
	monolithCmd.Flags().BoolVar(&monolithRedis, "redis", false, "启用 Redis 缓存")
	monolithCmd.Flags().StringVar(&monolithRedisAddr, "redis-addr", "localhost:6379", "Redis 地址")

	// Elasticsearch 参数
	monolithCmd.Flags().BoolVar(&monolithES, "elasticsearch", false, "启用 Elasticsearch")
	monolithCmd.Flags().StringVar(&monolithESAddrs, "es-addrs", "localhost:9200", "ES 地址")
	monolithCmd.Flags().StringVar(&monolithESIndex, "es-index", "", "ES 索引名")

	// MQ 参数
	monolithCmd.Flags().StringVar(&monolithMQ, "mq", "", "消息队列: kafka, rabbitmq, rocketmq")
	monolithCmd.Flags().StringVar(&monolithMQBrokers, "mq-brokers", "localhost:9092", "MQ brokers")
	monolithCmd.Flags().StringVar(&monolithMQTopic, "mq-topic", "", "MQ topic")

	// 对象存储参数
	monolithCmd.Flags().StringVar(&monolithObjectDB, "objectdb", "", "对象存储: oss, minio, rustfs")
	monolithCmd.Flags().StringVar(&monolithOSSEndpoint, "oss-endpoint", "", "OSS endpoint")
	monolithCmd.Flags().StringVar(&monolithOSSBucket, "oss-bucket", "", "OSS bucket")
	monolithCmd.Flags().StringVar(&monolithMinioEndpoint, "minio-endpoint", "", "MinIO endpoint")
	monolithCmd.Flags().StringVar(&monolithMinioBucket, "minio-bucket", "", "MinIO bucket")

	// 向量数据库参数
	monolithCmd.Flags().StringVar(&monolithVector, "vector", "", "向量数据库: zvec")
	monolithCmd.Flags().StringVar(&monolithZVecHost, "zvec-host", "localhost", "ZVec 主机")
	monolithCmd.Flags().StringVar(&monolithZVecPort, "zvec-port", "8080", "ZVec 端口")
}

// GetMonolithCmd returns the monolith command
func GetMonolithCmd() *cobra.Command {
	return monolithCmd
}
