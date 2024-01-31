package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var projectName string
var appName string

type InitFile struct {
	Name    string
	Content string
	Args    []any
}

var goiHelp = `goi version（版本）：%s
使用"goi help <command>"获取命令的更多信息。

Usage（用法）:
	goi <command> [arguments]
The commands are（命令如下）:
	create-project 	myproject	创建项目
	create-app 		myapp		创建app

`

// 根命令
var GoiCmd = &cobra.Command{
	Use:   "goi",
	Short: `goi 一款 web 框架`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// help_txt := fmt.Sprintf(goiHelp, goi.Version())
		// fmt.Print(help_txt)
		fmt.Print("root", args)
		return nil
	},
}

func main() {
	if err := GoiCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
