package commands

import (
	"fmt"
	"os"

	"github.com/gospacex/gpx/internal/cli"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gpx",
	Short: "gpx - Go 项目脚手架生成器",
	Long: `gpx 是一个基于 Cobra CLI 的通用脚手架工具，能够生成四类 Go 项目：
- 微服务项目（standard/DDD/Istio，Protobuf/Thrift IDL）
- 单体项目（传统 MVC 架构）
- 脚本中心（基于 gocron 的定时任务框架）
- Agent 项目（基于 CloudWeGo Eino）

Version: 0.1.0`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing help: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute executes the root command
func Execute() error {
	// Add subcommands
	rootCmd.AddCommand(

		cli.GetCRUDCmd(),

	)

	return rootCmd.Execute()
}
