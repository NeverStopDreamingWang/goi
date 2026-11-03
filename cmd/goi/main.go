package main

import (
	"fmt"
	"os"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/spf13/cobra"
)

// InitFile 定义一个生成文件，内容和路径均由本次创建参数决定。
type InitFile struct {
	Name    string
	Content func(createConfig) string
	Path    func(createConfig) string
}

var goiHelp = `goi version（版本）: %s
使用"goi help <command>"获取命令的更多信息。

Usage（用法）:
	goi <command> [arguments]
The commands are（命令如下）:
	create-project  [path]<project-name>   创建项目配置包，同时在当前目录生成 go.mod 和 server/main.go
	create-app      [path]<app-name>       创建 app，当前目录必须存在 go.mod

语法说明：
	[path] 为可选的相对父路径前缀，包含目录分隔符，按当前系统的路径格式填写。
	<project-name>、<app-name> 为必填名称；方括号和尖括号仅用于说明，不需要输入。

示例:
	mkdir project
	cd project
	goi create-project path/myproject
	goi create-app path/myapp

详细使用说明：https://github.com/NeverStopDreamingWang/goi/blob/main/cmd/goi/README.md

`

// 根命令
var GoiCmd = &cobra.Command{
	Use:   "goi",
	Short: `goi 一款 web 框架`,
	RunE: func(cmd *cobra.Command, args []string) error {
		help_txt := fmt.Sprintf(goiHelp, goi.Version())
		cmd.Print(help_txt)
		return nil
	},
}

// help
var HelpCmd = &cobra.Command{
	Use:   "help [command]",
	Short: "help 帮助",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			target, _, err := GoiCmd.Find(args)
			if err != nil {
				return err
			}
			return target.Help()
		}
		help_txt := fmt.Sprintf(goiHelp, goi.Version())
		cmd.Print(help_txt)
		return nil
	},
}

func init() {
	defaultHelp := GoiCmd.HelpFunc()
	GoiCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == GoiCmd {
			cmd.Print(fmt.Sprintf(goiHelp, goi.Version()))
			return
		}
		defaultHelp(cmd, args)
	})
	GoiCmd.AddCommand(HelpCmd)
}

func main() {
	if err := GoiCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
