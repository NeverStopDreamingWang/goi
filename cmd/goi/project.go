package main

import (
	"fmt"
	"go/version"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/crypto"
	"github.com/NeverStopDreamingWang/goi/v3/crypto/rsa"
	"github.com/spf13/cobra"
)

var CreateProject = &cobra.Command{
	Use:   "create-project [path]<project-name>",
	Short: "创建项目",
	Long: "先创建并进入项目根目录，再运行创建命令；当前目录即 Go 模块根目录。\n" +
		"go.mod 写入当前目录，服务入口为 server/main.go，配置包写入指定相对路径。\n" +
		"module 默认使用当前项目根目录名，可通过 --module 指定。\n" +
		"[path] 为可选的相对父路径前缀，包含当前系统的目录分隔符；<project-name> 为必填配置包名。\n" +
		"方括号和尖括号仅用于说明，不需要输入。\n" +
		"例如 myproject 或 path/myproject；创建后无需切换目录。",
	Example: "  mkdir project\n" +
		"  cd project\n" +
		"  goi create-project path/myproject\n" +
		"  goi create-app path/myapp\n" +
		"  go run ./server",
	Args: cobra.ExactArgs(1),
	RunE: GoiCreateProject,
}

func init() {
	CreateProject.Flags().String("module", "", "Go 模块路径，默认使用当前项目根目录名")
	GoiCmd.AddCommand(CreateProject) // 创建项目
}

var MainAppFileList = []InitFile{
	mainGo,
	goMod,
	settings,
	projectURLs,
	converter,
	validator,
	logger,
	ssl,
}

// 创建项目
func GoiCreateProject(cmd *cobra.Command, args []string) error {
	rootDir, err := os.Getwd()
	if err != nil {
		return err
	}
	projectPath, projectName, err := parseCreationPath(args[0])
	if err != nil {
		return err
	}
	if strings.EqualFold(projectPath, "server") {
		return fmt.Errorf("配置包路径不能为 server，该目录用于服务入口")
	}
	modulePath, err := cmd.Flags().GetString("module")
	if err != nil {
		return err
	}
	if modulePath == "" {
		modulePath = filepath.Base(rootDir)
	}
	if err := validateModulePath(modulePath); err != nil {
		return err
	}
	for _, target := range []string{
		filepath.Join(rootDir, "go.mod"),
		filepath.Join(rootDir, filepath.FromSlash(projectPath)),
		filepath.Join(rootDir, "server", "main.go"),
	} {
		if err := ensureNewDirectory(target); err != nil {
			return err
		}
	}

	// 生成密钥
	secretKey, err := crypto.GenerateRandomSecretKey()
	if err != nil {
		return err
	}

	// 生成 RSA 密钥
	privateKeyBytes, publicKeyBytes, err := rsa.GenerateKey(2048)
	if err != nil {
		return err
	}
	config := createConfig{
		RootDir:        rootDir,
		ProjectName:    projectName,
		ProjectPath:    projectPath,
		ModulePath:     modulePath,
		SettingsImport: modulePath + "/" + projectPath,
		SecretKey:      secretKey,
		PrivateKey:     string(privateKeyBytes),
		PublicKey:      string(publicKeyBytes),
	}
	return generateFiles(config, MainAppFileList)
}

// main.go 文件
var mainGo = InitFile{
	Name: "main.go",
	Content: func(config createConfig) string {
		content := `package main

import (
	"%s"

	// 创建应用后，取消注释并导入对应应用包。
	// "context"
	// "%s/path/myapp"
)

func main() {
	// 需要建表时，在启动服务前显式执行迁移。
	// myapp.Migrate(context.Background())

	// 启动服务
	%s.Server.RunServer()
}
`
		return fmt.Sprintf(content, config.SettingsImport, config.ModulePath, config.ProjectName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, "server")
	},
}

// go.mod 文件
var goMod = InitFile{
	Name: "go.mod",
	Content: func(config createConfig) string {
		content := `module %s

go %s

require (
	github.com/NeverStopDreamingWang/goi/v3 %s
	github.com/mattn/go-sqlite3 v1.14.32
)
`
		goVersion := strings.TrimPrefix(version.Lang(runtime.Version()), "go")
		if goVersion == "" {
			goVersion = "1.24.0"
		}
		return fmt.Sprintf(content, config.ModulePath, goVersion, goi.Version())
	},
	Path: func(config createConfig) string {
		return config.RootDir
	},
}

// 配置文件
var settings = InitFile{
	Name: "settings.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/clickjacking"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/common"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/security"
	_ "github.com/mattn/go-sqlite3"
)

// Http 服务
var Server *goi.Engine

func init() {
	var err error

	// 创建 http 服务
	Server = goi.NewHTTPServer()

	// version := goi.Version() // 获取版本信息
	// fmt.Println("goi 版本", version)

	// 项目路径
	Server.Settings.BaseDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	// Debug
	Server.Settings.Debug = true
	// 网络协议
	Server.Settings.Network = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 监听地址
	Server.Settings.BindAddress = "0.0.0.0" // 默认 0.0.0.0
	// 端口
	Server.Settings.Port = 8080
	// 域名
	Server.Settings.BindDomain = ""

	// 密钥
	Server.Settings.SecretKey = "%s"

	// RSA 私钥
	Server.Settings.PrivateKey = ` + "`%s`" + `

	// RSA 公钥
	Server.Settings.PublicKey = ` + "`%s`" + `

	// 设置 SSL
	Server.Settings.SSL = goi.SSL{
		Enabled:    false,  // SSL 开关
		Type:      "自签证书", // 证书类型
		CertPath: filepath.Join(Server.Settings.BaseDir, "ssl", "%s.crt"),
		KeyPath:  filepath.Join(Server.Settings.BaseDir, "ssl", "%s.key"),
	}

	// 注册中间件（根路由）
	Server.Router.Use(
		security.Default(), // 安全中间件
		common.Default(),   // 通用中间件
		// corsheaders.Default(), // CORS 跨域中间件
		clickjacking.Default(), // 点击劫持中间件
	)

	// 数据库配置
	Server.Settings.Databases["default"] = &goi.Database{
		Engine: "sqlite3",
		Connect: func(Engine string) *sql.DB {
			dataDir := filepath.Join(Server.Settings.BaseDir, "data")
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			DataSourceName := filepath.Join(dataDir, "%s.db")
			sqliteDB, err := sql.Open(Engine, DataSourceName)
			if err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			return sqliteDB
		},
	}

	Server.Settings.UseTZ = true
	// 设置时区
	err = Server.Settings.SetTimeZone("Asia/Shanghai") // 默认为空字符串 ''，本地时间
	if err != nil {
		panic(err)
	}
	//  goi.GetLocation() 获取时区 Location
	//  goi.GetTime() 获取当前时区的时间

	// 设置框架语言
	Server.Settings.SetLanguage(goi.ZhCN) // 默认 ZhCN

	// 设置最大缓存大小
	Server.Cache.EvictPolicy = goi.AllKeysLRU   // 缓存淘汰策略
	Server.Cache.ExpirationPolicy = goi.Periodic // 过期策略
	Server.Cache.MaxSize = 0                     // 单位为字节，0 为不限制使用

	// 创建日志，或者修改全局日志配置
	Server.Log = newDefaultLog() // 默认日志
	// 设置全局日志
	goi.Log = Server.Log
	// 注册日志切割任务
	goi.RegisterOnStartup(goi.Log)

	// 设置验证器错误，不指定则使用默认
	Server.Validation.SetValidationError(validationError{})

	// 设置自定义配置
	// Server.Settings.Set(key string, value any)
	// Server.Settings.Get(key string, dest any)

	// 注册后台任务，在 RunServer 之前注册，之后注册的任务不会执行
	// RunServer 内部会启动注册 goroutine 执行任务
	// goi.RegisterOnStartup(&Task{})

	// 注册关闭回调处理程序，可以在服务运行期间注册，停机开始后注册的回调不会执行
	goi.RegisterOnShutdown(&Shutdown{})
}

type Shutdown struct{}

// ShutdownName 关闭回调名称
func (self *Shutdown) ShutdownName() string {
	return "自定义关闭操作"
}

// OnShutdown 关闭回调处理程序
func (self *Shutdown) OnShutdown() error {
	return nil
}

type Task struct{}

func (self *Task) StartupName() string {
	return "任务名称"
}

// OnStartup 启动任务
func (self *Task) OnStartup(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			goi.Log.Info("任务结束")
			return
		case <-ticker.C:
			goi.Log.Info("执行任务")
		}
	}
}

`
		return fmt.Sprintf(content, config.ProjectName, config.SecretKey, config.PrivateKey, config.PublicKey,
			config.ProjectName, config.ProjectName, config.ProjectName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.ProjectPath))
	},
}

// 项目路由文件，预留给用户集中配置路由。
var projectURLs = InitFile{
	Name: "urls.go",
	Content: func(config createConfig) string {
		return fmt.Sprintf("package %s\n", config.ProjectName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.ProjectPath))
	},
}

// 路由转换器文件
var converter = InitFile{
	Name: "converter.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"github.com/NeverStopDreamingWang/goi/v3"
)

func init() {
	// 注册路由转换器
	// 手机号
	goi.RegisterConverter("phone", phoneConverter)
}

// 手机号
var phoneConverter = goi.Converter{
	Regex: %s,
	ToGo:  func(value string) (any, error) { return value, nil },
}

`
		return fmt.Sprintf(content, config.ProjectName, "`(1[3456789]\\d{9})`")
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.ProjectPath))
	},
}

// 验证器文件
var validator = InitFile{
	Name: "validator.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/response"
)

func init() {
	// 注册验证器
	// 手机号
	goi.RegisterValidator("phone", phoneValidator{})
}

// 自定义参数验证错误
type validationError struct {
	Status  int
	Message string
}

// 创建参数验证错误方法
func (validationErr validationError) NewValidationError(status int, message string, args ...any) goi.ValidationError {
	return &validationError{
		Status:  status,
		Message: message,
	}
}

// 实现 error 接口，返回错误消息
func (validationErr validationError) Error() string {
	return validationErr.Message
}

// 参数验证错误响应格式
func (validationErr validationError) Response() goi.Response {
	return goi.Response{
		Status: http.StatusOK,
		Data: response.Data{
			Code:    validationErr.Status,
			Message: validationErr.Message,
			Data: nil,
		},
	}
}

type phoneValidator struct{}

// Parse 验证手机号，并将有效值转换为整数。
func (validator phoneValidator) Parse(value any) (any, goi.ValidationError) {
	var valueStr string
	switch typeValue := value.(type) {
	case int:
		valueStr = strconv.Itoa(typeValue)
	case string:
		valueStr = typeValue
	default:
		return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%%v", value))
	}
	re := regexp.MustCompile(` + "`^(1[3456789]\\d{9})$`" + `)
	if !re.MatchString(valueStr) {
		return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%%v", value))
	}
	intValue, err := strconv.Atoi(valueStr)
	if err != nil {
		return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%%v", value))
	}
	return intValue, nil
}

`
		return fmt.Sprintf(content, config.ProjectName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.ProjectPath))
	},
}

// 日志文件
var logger = InitFile{
	Name: "logger.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
)

// 默认日志
func newDefaultLog() *goi.Logger {
	// 使用默认
	// return goi.NewLogger(filepath.Join(Server.Settings.BaseDir, "logs", "server.log"))

	var err error

	LogDir := filepath.Join(Server.Settings.BaseDir, "logs") // 检查目录
	err = os.MkdirAll(LogDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("创建日志目录错误: %%v", err))
	}
	path := filepath.Join(LogDir, "server.log")

	logger := &goi.Logger{
		Name:       "默认日志",
		Path:       path,
		File:       nil,
		Logger:     nil,
		SplitSize:  1024 * 1024 * 10, // 切割大小
		SplitTime:  "2006-01-02",     // 切割日期，每天
		CreateTime: goi.GetTime(),
		PrintFunc:  nil, // 自定义日志输出格式，为 nil 则使用默认方法
		// PrintFunc:       printlnFunc,
		GetFileFunc: nil, // 创建文件对象方法 为 nil 则使用默认方法
		// GetFileFunc:     getFileFunc,
		SplitLoggerFunc: nil, // 自定义日志切割：符合切割条件时，传入日志对象，返回新的文件对象
	}

	logger.File, err = logger.GetFile()
	if err != nil {
		panic(fmt.Sprintf("初始化[%%v]日志错误: %%v", logger.Name, err))
	}
	logger.Logger = log.New(logger.File, "", 0)
	return logger
}

// 日志输出格式
func printlnFunc(logger *goi.Logger, level goi.Level, logs ...any) {
	timeStr := fmt.Sprintf("[%%v]", goi.GetTime().Format(time.DateTime))
	if level != "" {
		timeStr += fmt.Sprintf(" %%v", level)
	}
	logs = append([]any{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// 获取 *os.File 对象
func getFileFunc(filePath string) (*os.File, error) {
	var file *os.File
	var err error

	_, err = os.Stat(filePath)
	if os.IsNotExist(err) { // 不存在则创建
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// 文件存在，打开文件
		file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

`
		return fmt.Sprintf(content, config.ProjectName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.ProjectPath))
	},
}

// SSL 文件
var ssl = InitFile{
	Name: "ssl.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
)

// 检查生成自签证书
func init() {
	SSLPath := filepath.Join(Server.Settings.BaseDir, "ssl")
	if Server.Settings.SSL.Enabled == false {
		return
	}

	_, certErr := os.Stat(Server.Settings.SSL.CertPath)
	_, keyErr := os.Stat(Server.Settings.SSL.KeyPath)
	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		generateCertificate(SSLPath)
	}
}

// 生成自签证书
func generateCertificate(SSLPath string) {
	var err error
	_, err = os.Stat(SSLPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(SSLPath, 0755)
		if err != nil {
			msg := fmt.Sprintf("创建 SSL 目录错误: %%v", err)
			goi.Log.Error(msg)
			panic(msg)
		}
	}

	// 生成 128 位随机序列号
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		goi.Log.Error(err)
		panic(err)
	}

	// 创建证书请求模板
	certificateTemplate := x509.Certificate{
		SerialNumber: serialNumber, // 证书本身的序列号
		Subject: pkix.Name{
			Country:            []string{"CN"},  // 国家
			Organization:       []string{"goi"}, // 证书持有者的组织名称
			OrganizationalUnit: nil,             // 证书持有者的组织单元名称
			Locality:           nil,             // 证书持有者所在地区（城市）的名称
			Province:           nil,             // 证书持有者所在省/州的名称
			StreetAddress:      nil,             // 证书持有者的街道地址
			PostalCode:         nil,             // 证书持有者的邮政编码
			SerialNumber:       "",              // 证书持有者的序列号
			CommonName:         "%s",       //  证书的通用名称
		},
		NotBefore:             goi.GetTime(),
		NotAfter:              goi.GetTime().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	err = goi.GenerateRSACertificate(2048, certificateTemplate, SSLPath) // RSA 算法
	// err = GenerateECCCertificate(certificateTemplate, SSLPath) // ECC 算法
	if err != nil {
		msg := fmt.Sprintf("生成SSL证书和私钥时发生错误: %%v", err) 
		goi.Log.Error(msg)
		panic(msg)
	}
}
`
		return fmt.Sprintf(content, config.ProjectName, config.ProjectName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.ProjectPath))
	},
}
