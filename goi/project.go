package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/spf13/cobra"
	"os"
	"path"
	"regexp"
	"runtime"
)

var CreateProject = &cobra.Command{
	Use:   "create-project",
	Short: "创建项目",
	RunE:  GoiCreateProject,
}

func init() {
	GoiCmd.AddCommand(CreateProject) // 创建项目
}

// 创建-项目
var secretKey string

var MainAppFileList = []InitFile{
	settings,
	converter,
	mainGo,
	goMod,
}

// 创建项目
func GoiCreateProject(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("请输入项目名称")
	}
	projectName = args[0]

	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return err
	}
	// 将随机字节转换为Base64编码的字符串
	secretKey = base64.StdEncoding.EncodeToString(randomBytes)

	// 将随机字节串转换为十六进制字符串
	// secretKey = hex.EncodeToString(randomBytes)

	for _, itemFile := range MainAppFileList {
		itemPath := itemFile.Path()
		_, err = os.Stat(itemPath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(itemPath, 0755)
			if err != nil {
				return err
			}
		}

		// 创建文件
		filePath := path.Join(itemPath, itemFile.Name)
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0755)

		Content := itemFile.Content()
		_, err = file.WriteString(Content)
		if err != nil {
			return err
		}
	}

	return nil
}

// main.go 文件
var mainGo = InitFile{
	Name: "main.go",
	Content: func() string {
		content := `package main

import (
	"fmt"
	"%s/%s"

	// 注册app
	// _ "xxx"
)

func main() {

	// 启动服务
	err := %s.Server.RunServer()
	if err != nil {
		fmt.Printf("服务已停止！", err)
		// panic(fmt.Sprintf("停止: %v\n", err))

	}
}
`
		return fmt.Sprintf(content, projectName, projectName, projectName)
	},
	Path: func() string {
		return path.Join(baseDir, projectName)
	},
}

// go.mod 文件
var goMod = InitFile{
	Name: "go.mod",
	Content: func() string {
		content := `module %s

go %s

require (
	github.com/NeverStopDreamingWang/goi v%s
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirectgo.sum
)

`
		var version string
		versionStr := runtime.Version()

		regexpStr := `(\d+.\d+)`
		re := regexp.MustCompile(regexpStr)
		result := re.FindAllStringSubmatch(versionStr, -1)
		if len(result) == 1 {
			version = result[0][0]
		}
		return fmt.Sprintf(content, projectName, version, goi.Version())
	},
	Path: func() string {
		return path.Join(baseDir, projectName)
	},
}

// 配置文件
var settings = InitFile{
	Name: "settings.go",
	Content: func() string {
		content := `package %s

import (
	"os"
	"path"
	"github.com/NeverStopDreamingWang/goi"
)

// Http 服务
var Server *goi.Engine

func init() {
	// 创建 http 服务
	Server = goi.NewHttpServer()
	
	// version := goi.Version() // 获取版本信息
	// fmt.Println("goi 版本", version)
	
	// 运行地址
	Server.Settings.SERVER_ADDRESS = "0.0.0.0"
	// 端口
	Server.Settings.SERVER_PORT = 8080
	
	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()
	
	// 项目
	Server.Settings.SECRET_KEY = "%s"
	
	// 数据库配置
	Server.Settings.DATABASES["default"] = goi.MetaDataBase{
		ENGINE:   "mysql",
		NAME:     "test_goi",
		USER:     "root",
		PASSWORD: "123",
		HOST:     "127.0.0.1",
		PORT:     3306,
	}
	Server.Settings.DATABASES["sqlite_1"] = goi.MetaDataBase{
		ENGINE:   "sqlite3",
		NAME:     path.Join(Server.Settings.BASE_DIR, "data/test_goi.db"),
		USER:     "",
		PASSWORD: "",
		HOST:     "",
		PORT:     0,
	}
	
	// 设置时区
	Server.Settings.TIME_ZONE = "Asia/Shanghai" // 默认 Asia/Shanghai
	// Server.Settings.TIME_ZONE = "America/New_York"
	
	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用
	
	// 日志设置
	Server.Log.DEBUG = true
	Server.Log.INFO_OUT_PATH = "logs/info.log"      // 输出所有日志到文件
	Server.Log.ACCESS_OUT_PATH = "logs/asccess.log" // 输出访问日志文件
	Server.Log.ERROR_OUT_PATH = "logs/error.log"    // 输出错误日志文件
	// Server.Log.OUT_DATABASE = ""  // 输出到的数据库 例: default
	Server.Log.SplitSize = 1024 * 1024
	Server.Log.SplitTime = "2006-01-02"
	// 日志打印
	// Server.Log.Debug() = goi.Log.Debug() = goi.Log.Log(goi.DEBUG, "")
	// Server.Log.Info() = goi.Log.Info()
	// Server.Log.Warning() = goi.Log.Warning()
	// Server.Log.Error() = goi.Log.Error()
	
	// Server.Settings.LOG_OUTPATH = "/log/server.log"
	// log_path := path.Join(BASE_DIR, LOG_OUTPATH)
	// log_file, err := os.OpenFile(log_path, os.O_CREATE|os.O_APPEND, 0777)
	// if err != nil {
	// 	panic(fmt.Sprintf("异常【日志创建】: %v\n", err))
	// 	return
	// }
	// log.SetOutput(log_file)
	// // 设置日志前缀为空
	// log.SetFlags(0)
	// // defer log_file.Close()
	
	// 设置自定义配置
	// redis配置
	Server.Settings.Set("REDIS_HOST", "127.0.0.1")
	Server.Settings.Set("REDIS_PORT", 6379)
	Server.Settings.Set("REDIS_PASSWORD", "123")
	Server.Settings.Set("REDIS_DB", 0)
	
	Server.Init()
}
`
		return fmt.Sprintf(content, projectName, secretKey)
	},
	Path: func() string {
		return path.Join(baseDir, projectName, projectName)
	},
}

// 路由转换器文件
var converter = InitFile{
	Name: "converter.go",
	Content: func() string {
		content := `package %s

import "github.com/NeverStopDreamingWang/goi"

func RegisterMyConverter() {
	// 注册一个路由转换器
	
	// 手机号
	goi.RegisterConverter("phone", ` + "`(1[3456789]\\d{9})`)" + `
	
}
`
		return fmt.Sprintf(content, projectName)
	},
	Path: func() string {
		return path.Join(baseDir, projectName, projectName)
	},
}
