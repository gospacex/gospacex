package cli

import (
	"fmt"
	"os"

	"github.com/gospacex/gpx/internal/config"
	pkggen "github.com/gospacex/gpx/internal/generator/pkg"
	"github.com/spf13/cobra"
)

// pkg 命令参数
var (
	pkgOutputDir  string
	pkgSnowflake  bool
)

var pkgCmd = &cobra.Command{
	Use:   "pkg [flags]",
	Short: "在当前目录（或指定目录）的 pkg 下添加通用组件",
	Long: `在项目的 pkg 目录下生成通用组件代码。

支持的组件:
  --snowflake   Snowflake 分布式唯一 ID 生成器

组件说明:
  snowflake     基于 bwmarrin/snowflake，提供 int64/string/base64 三种格式，
                支持自定义 Epoch、动态更新 NodeID、默认全局单例，内含完整测试。

示例:

  # 在当前目录的 pkg/ 下生成 snowflake 组件
  gpx pkg --snowflake

  # 指定输出目录（会在该目录下的 pkg/snowflake/ 中写入文件）
  gpx pkg --output ./myapp --snowflake`,
	RunE: runPkg,
}

func runPkg(cmd *cobra.Command, args []string) error {
	// 确定输出目录，默认当前目录
	outputDir := pkgOutputDir
	if outputDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("获取当前目录失败: %w", err)
		}
		outputDir = cwd
	}

	// 至少需要选择一个组件
	if !pkgSnowflake {
		return fmt.Errorf("请至少指定一个组件，例如 --snowflake\n运行 'gpx pkg --help' 查看所有可用组件")
	}

	cfg := &config.ProjectConfig{
		ProjectType:  "pkg",
		OutputDir:    outputDir,
		PkgSnowflake: pkgSnowflake,
	}

	gen := pkggen.NewGenerator(cfg)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("生成组件失败: %w", err)
	}

	// 打印结果
	fmt.Printf("✓ pkg 组件生成成功（输出目录: %s）\n", outputDir)
	if pkgSnowflake {
		fmt.Printf("  pkg/snowflake/snowflake.go\n")
		fmt.Printf("  pkg/snowflake/snowflake_test.go\n")
	}
	fmt.Println()
	fmt.Println("使用提示:")
	if pkgSnowflake {
		fmt.Println("  go get github.com/bwmarrin/snowflake")
		fmt.Println("  go test ./pkg/snowflake/...")
	}

	return nil
}

func init() {
	pkgCmd.Flags().StringVarP(&pkgOutputDir, "output", "o", "", "目标项目根目录（默认: 当前目录）")
	pkgCmd.Flags().BoolVar(&pkgSnowflake, "snowflake", false, "生成 Snowflake 分布式唯一 ID 组件")
}

// GetPkgCmd 返回 pkg 子命令
func GetPkgCmd() *cobra.Command {
	return pkgCmd
}
