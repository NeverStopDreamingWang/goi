package main

import (
	"fmt"
	"os"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/spf13/cobra"
)

var baseDir, _ = os.Getwd()
var projectName string
var appName string

type InitFile struct {
	Name    string
	Content func() string
	Path    func() string
}

var goiHelp = `goi version（版本）: %s
使用"goi help <command>"获取命令的更多信息。

Usage（用法）:
	goi <command> [arguments]
The commands are（命令如下）:
	create-project  myproject   创建项目
	create-app      myapp       创建app

`

// 根命令
var GoiCmd = &cobra.Command{
	Use:   "goi",
	Short: `goi 一款 web 框架`,
	RunE: func(cmd *cobra.Command, args []string) error {
		help_txt := fmt.Sprintf(goiHelp, goi.Version())
		fmt.Print(help_txt)
		return nil
	},
}

// help
var HelpCmd = &cobra.Command{
	Use:   "help",
	Short: "help 帮助",
	RunE: func(cmd *cobra.Command, args []string) error {
		help_txt := fmt.Sprintf(goiHelp, goi.Version())
		fmt.Print(help_txt)
		return nil
	},
}

func main() {
	GoiCmd.AddCommand(HelpCmd) // 帮助信息
	if err := GoiCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
