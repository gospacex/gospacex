package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.0.9"
)

var rootCmd = &cobra.Command{
	Use:   "gpx",
	Short: "gpx - Go 项目脚手架生成器",
	Long: `gpx 是一个基于 Cobra CLI 的通用脚手架工具，能够生成四类 Go 项目：
- 微服务项目（支持标准/DDD 架构，istio,Protobuf/Thrift IDL）
- 单体项目（传统 MVC 架构）
- 脚本中心（基于 gocron 的定时任务框架）
- Agent 项目（基于 CloudWeGo Eino）`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing help: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() error {
	// 注册子命令
	rootCmd.AddCommand(
		GetMonolithCmd(),
		GetMicroAppCmd(),
		GetMicroBffCmd(),
		GetCRUDCmd(),
		GetGenProtoCmd(),
		GetGenGRPCCmd(),
		GetScriptCenterCmd(),
		GetPkgCmd(),
	)

	return rootCmd.Execute()
}

func GetGenProtoCmd() *cobra.Command {
	return genProtoCmd
}

func GetScriptCenterCmd() *cobra.Command {
	return scriptCenterCmd
}
