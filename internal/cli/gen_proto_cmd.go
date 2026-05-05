package cli

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/gospacex/gpx/internal/generator"
	_ "github.com/go-sql-driver/mysql"
)

var (
	genProtoTableName  string
	genProtoOutputPath string
	genProtoDBHost     string
	genProtoDBPort     string
	genProtoDBUser     string
	genProtoDBPassword string
	genProtoDBName     string
	genProtoDryRun     bool
)

var genProtoCmd = &cobra.Command{
	Use:   "gen-proto",
	Short: "从数据库表生成 Proto 文件",
	Long: `从数据库表结构自动生成 Proto IDL 文件

示例:
  gospacex gen-proto -t user -o ./idl/user.proto
  gospacex gen-proto -t article --dry-run
  gospacex gen-proto -t "t_user" -H localhost -P 3306 -u root -p password -d mydb`,
	RunE: runGenProto,
}

func init() {
	genProtoCmd.Flags().StringVarP(&genProtoTableName, "table", "t", "", "数据库表名 (必填)")
	genProtoCmd.Flags().StringVarP(&genProtoOutputPath, "output", "o", "", "输出文件路径")
	genProtoCmd.Flags().StringVar(&genProtoDBHost, "host", "127.0.0.1", "数据库主机")
	genProtoCmd.Flags().StringVar(&genProtoDBPort, "port", "3306", "数据库端口")
	genProtoCmd.Flags().StringVar(&genProtoDBUser, "user", "root", "数据库用户名")
	genProtoCmd.Flags().StringVar(&genProtoDBPassword, "password", "", "数据库密码")
	genProtoCmd.Flags().StringVar(&genProtoDBName, "database", "", "数据库名")
	genProtoCmd.Flags().BoolVar(&genProtoDryRun, "dry-run", false, "仅预览, 不生成文件")

	genProtoCmd.MarkFlagRequired("table")
}

func runGenProto(cmd *cobra.Command, args []string) error {
	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		genProtoDBUser, genProtoDBPassword, genProtoDBHost, genProtoDBPort, genProtoDBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 创建生成器
	gen := generator.NewProtoGenerator(db, "", "github.com/example")

	// 生成 proto
	info, err := gen.GenerateFromTable(genProtoTableName)
	if err != nil {
		return fmt.Errorf("生成 proto 信息失败: %w", err)
	}

	// 预览模式
	if genProtoDryRun {
		fmt.Println("=== 生成的 Proto 内容预览 ===")
		fmt.Printf("表名: %s\n", info.TableName)
		fmt.Printf("服务名: %sService\n", info.ServiceName)
		fmt.Printf("模块名: %s\n\n", info.ModuleName)
		fmt.Println("字段列表:")
		for _, f := range info.Fields {
			pk := ""
			if f.IsPrimary {
				pk = " [PK]"
			}
			fmt.Printf("  - %s (%s)%s - %s\n", f.Name, f.ProtoType, pk, f.Comment)
		}
		return nil
	}

	// 生成文件
	if genProtoOutputPath == "" {
		genProtoOutputPath = fmt.Sprintf("./%s.proto", info.ModuleName)
	}

	if err := gen.GenerateProtoFile(info, genProtoOutputPath); err != nil {
		return fmt.Errorf("生成 proto 文件失败: %w", err)
	}

	fmt.Printf("Proto 文件已生成: %s\n", genProtoOutputPath)
	return nil
}
