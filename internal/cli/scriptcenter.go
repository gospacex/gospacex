package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/generator"
	"github.com/gospacex/gpx/internal/variables"
	"github.com/spf13/cobra"
)

var (
	scOutputDir   string
	scProjectName string
	scDbTypes     []string
	scDbHost      string
	scDbPort      string
	scDbUser      string
	scDbPassword  string
	scDbName      string
	scDbTable     string
	scMQTypes     string
	scDTMEnabled  bool
	scDTMServer   string
	scDTMMode     string
	scOutputFlag  string
)

var scriptCenterCmd = &cobra.Command{
	Use:     "script [output-dir]",
	Short:   "创建脚本中心项目（基于 Cobra CLI 的任务调度）",
	Long:    "创建脚本中心项目 - 基于 Cobra CLI 的定时任务调度平台，支持多种数据库和消息队列集成.\n\n脚本中心适用于:\n  - 数据同步任务\n  - 消息队列消费者/生产者\n  - 定时批处理作业\n  - 工具类 CLI 应用\n\n支持数据库: mysql, postgresql, redis, mongodb, elasticsearch\n支持消息队列: kafka, rabbitmq, rocketmq\n支持分布式事务: DTM",
	Args:    cobra.RangeArgs(0, 1),
	RunE:    runScriptCenter,
	Example: "  # 创建基础脚本中心（仅 MySQL）\n  gpx script data-sync --db mysql\n\n  # 创建带 Kafka 的消费者脚本中心\n  gpx script kafka-consumer --db mysql --mq kafka\n\n  # 完整配置指定数据库连接\n  gpx script my-sync --db mysql --db-host 192.168.1.100 --db-user root --db-password secret --db-name syncdb\n\n  # 使用 --output 标志指定输出目录\n  gpx script --output my-scripts --db mysql --db-host 127.0.0.1 --db-user root --db-password secret --db-name syncdb",
}

func runScriptCenter(cmd *cobra.Command, args []string) error {
	// 确定输出目录：优先使用 --output 标志，其次使用位置参数
	if scOutputFlag != "" {
		scOutputDir = scOutputFlag
	} else if len(args) > 0 {
		scOutputDir = args[0]
	} else {
		return fmt.Errorf("必须指定输出目录：使用 --output 标志或位置参数")
	}

	baseName := filepath.Base(scOutputDir)
	if scProjectName == "" {
		scProjectName = "github.com/gospacex/" + baseName
	}

	validator := variables.NewValidator()
	if err := validator.Validate(scProjectName, scOutputDir, scDbTypes, scMQTypes, "", "", ""); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if _, err := os.Stat(scOutputDir); err == nil {
		return fmt.Errorf("output directory already exists: %s", scOutputDir)
	}

	cfg := &config.ProjectConfig{
		ProjectType:   "script",
		ProjectName:   scProjectName,
		OutputDir:     scOutputDir,
		DB:            scDbTypes,
		MQ:            scMQTypes,
		MySQLHost:     scDbHost,
		MySQLPort:     scDbPort,
		MySQLUser:     scDbUser,
		MySQLPassword: scDbPassword,
		MySQLDatabase: scDbName,
		MySQLTable:    scDbTable,
		DTMEnabled:    scDTMEnabled,
		DTMServer:     scDTMServer,
		DTMMode:       scDTMMode,
	}

	fmt.Printf("Creating script project in %s...\n", scOutputDir)

	gen := generator.NewScriptCenterGenerator(cfg)
	if err := gen.Generate(scOutputDir); err != nil {
		return fmt.Errorf("generate project: %w", err)
	}

	fmt.Printf("✓ script project created successfully in %s\n", scOutputDir)
	fmt.Printf("  Project: %s\n", scProjectName)
	fmt.Printf("  Databases: %s\n", strings.Join(scDbTypes, ", "))
	if scMQTypes != "" {
		fmt.Printf("  Message Queue: %s\n", scMQTypes)
	}

	return nil
}

func init() {
	scriptCenterCmd.Flags().StringVar(&scProjectName, "name", "", "Go 模块名称 (默认: github.com/gospacex/<output-dir>)")
	scriptCenterCmd.Flags().StringVar(&scOutputFlag, "output", "", "脚本中心项目输出目录（可选，默认为位置参数）")
	scriptCenterCmd.Flags().StringSliceVar(&scDbTypes, "db", []string{"mysql"}, "启用的数据库类型（可多次指定）")
	scriptCenterCmd.Flags().StringVar(&scMQTypes, "mq", "", "启用的消息队列类型 (kafka,rabbitmq,rocketmq，逗号分隔)")
	scriptCenterCmd.Flags().StringVar(&scDbHost, "db-host", "", "MySQL 主机地址")
	scriptCenterCmd.Flags().StringVar(&scDbPort, "db-port", "3306", "MySQL 端口")
	scriptCenterCmd.Flags().StringVar(&scDbUser, "db-user", "", "MySQL 用户名")
	scriptCenterCmd.Flags().StringVar(&scDbPassword, "db-password", "", "MySQL 密码")
	scriptCenterCmd.Flags().StringVar(&scDbName, "db-name", "", "MySQL 数据库名称")
	scriptCenterCmd.Flags().StringVar(&scDbTable, "db-table", "", "MySQL 表名（自动生成 CRUD）")
	scriptCenterCmd.Flags().BoolVar(&scDTMEnabled, "dtm", false, "启用 DTM 分布式事务")
	scriptCenterCmd.Flags().StringVar(&scDTMServer, "dtm-server", "http://localhost:36789", "DTM 服务器地址")
	scriptCenterCmd.Flags().StringVar(&scDTMMode, "dtm-mode", "saga", "事务模式：saga, tcc, msg, workflow")
}
