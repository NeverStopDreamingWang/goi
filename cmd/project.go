package main

import (
	"encoding/base64"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/crypto"
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
	Args:  cobra.ExactArgs(1),
}

func init() {
	GoiCmd.AddCommand(CreateProject) // 创建项目
}

// AES 密钥
var secretKey string

// RSA 私钥
var privateKey string

// RSA 公钥
var publicKey string

var MainAppFileList = []InitFile{
	mainGo,
	goMod,
	settings,
	converter,
	logs,
	ssl,
}

// 创建项目
func GoiCreateProject(cmd *cobra.Command, args []string) error {
	projectName = args[0]

	// 生成 AES 密钥
	secretKeyBytes := make([]byte, 32)
	err := crypto.GenerateAES(secretKeyBytes)
	if err != nil {
		return err
	}
	// 将随机字节转换为Base64编码的字符串
	secretKey = base64.StdEncoding.EncodeToString(secretKeyBytes)

	// 生成 RSA 密钥
	var privateKeyBytes, publicKeyBytes []byte
	err = crypto.GenerateRSA(2048, &privateKeyBytes, &publicKeyBytes)
	if err != nil {
		return err
	}
	privateKey = string(privateKeyBytes)
	publicKey = string(publicKeyBytes)

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
	"%s/%s"

	// 注册app
	// _ "xxx"
)

func main() {

	// 启动服务
	%s.Server.RunServer()
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
	github.com/mattn/go-sqlite3 v1.14.17 // indirectgo.sum
)

require (
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
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
	
	// 设置网络协议
	Server.Settings.NET_WORK = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 运行地址
	Server.Settings.BIND_ADDRESS = "0.0.0.0"
	// 端口
	Server.Settings.PORT = 8080

	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()

	// 项目 AES 密钥
	Server.Settings.SECRET_KEY = "%s"

	// 项目 RSA 私钥
	Server.Settings.PRIVATE_KEY = %s

	// 项目 RSA 公钥
	Server.Settings.PUBLIC_KEY = %s

	// 设置 SSL
	Server.Settings.SSL = goi.MetaSSL{
		STATUS:    false, // SSL 开关
		CERT_PATH: path.Join(Server.Settings.BASE_DIR, "ssl/%s.crt"),
		KEY_PATH:  path.Join(Server.Settings.BASE_DIR, "ssl/%s.key"),
		CSR_PATH:  path.Join(Server.Settings.BASE_DIR, "ssl/%s.csr"), // 可选
	}
	
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
	Server.Settings.TIME_ZONE = "Asia/Shanghai" // 默认 Asia/Shanghai，America/New_York
	
	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用
	
	// 日志设置
	Server.Log.DEBUG = true
	// 日志列表
	defaultLog := newDefaultLog()
	accessLog := newAccessLog()
	errorLog := newErrorLog()
	Server.Log.LOGGERS = []*goi.MetaLogger{
		defaultLog, // 默认日志
		accessLog, // 访问日志
		errorLog, // 错误日志
	}
	
	// 日志打印
	// Server.Log.Log() = goi.Log.Log()
	// Server.Log.Info() = goi.Log.Info()
	// Server.Log.Warning() = goi.Log.Warning()
	// Server.Log.Error() = goi.Log.Error()

	
	// 设置自定义配置
	// redis配置
	Server.Settings.Set("REDIS_HOST", "127.0.0.1")
	Server.Settings.Set("REDIS_PORT", 6379)
	Server.Settings.Set("REDIS_PASSWORD", "123")
	Server.Settings.Set("REDIS_DB", 0)
}
`
		return fmt.Sprintf(content, projectName, secretKey, "`"+privateKey+"`", "`"+publicKey+"`", projectName, projectName, projectName)
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
	goi.RegisterConverter("phone", %s)
	
}
`
		return fmt.Sprintf(content, projectName, "`(1[3456789]\\d{9})`")
	},
	Path: func() string {
		return path.Join(baseDir, projectName, projectName)
	},
}

// 日志文件
var logs = InitFile{
	Name: "logs.go",
	Content: func() string {
		content := `package %s

import (
	"os"
	"fmt"
	"log"
	"path"
	"time"
	"github.com/NeverStopDreamingWang/goi"
)

// 日志输出
func LogPrintln(logger *goi.MetaLogger, log ...any) {
	timeStr := fmt.Sprintf("[%v]", time.Now().In(Server.Settings.LOCATION).Format("2006-01-02 15:04:05"))
	log = append([]any{timeStr}, log...)
	logger.Logger.Println(log...)
}

// 默认日志
func newDefaultLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/server.log")
	defaultLog := &goi.MetaLogger{
		Name: "默认日志",
		Path: OutPath,
		Level: []goi.Level{ // 所有等级的日志
			goi.INFO,
			goi.WARNING,
			goi.ERROR,
		},
		Logger:        nil,
		File:          nil,
		CreateTime:    time.Now().In(Server.Settings.LOCATION),
		SPLIT_SIZE:    1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:    "2006-01-02",    // 切割日期，每天
		NewLoggerFunc: newDefaultLog,
		LoggerPrint:   LogPrintln,
	}

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[%v]日志错误: ", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 访问日志
func newAccessLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/access.log")
	defaultLog := &goi.MetaLogger{
		Name: "访问日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入正确的日志
			goi.INFO,
		},
		Logger:        nil,
		File:          nil,
		CreateTime:    time.Now().In(Server.Settings.LOCATION),
		SPLIT_SIZE:    1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:    "2006-01-02",    // 切割日期，每天
		NewLoggerFunc: newAccessLog,
		LoggerPrint:   LogPrintln,
	}

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[%v]日志错误: ", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 错误日志
func newErrorLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/error.log")
	defaultLog := &goi.MetaLogger{
		Name: "错误日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入错误的日志
			goi.ERROR,
		},
		Logger:        nil,
		File:          nil,
		CreateTime:    time.Now().In(Server.Settings.LOCATION),
		SPLIT_SIZE:    1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:    "2006-01-02",    // 切割日期，每天
		NewLoggerFunc: newErrorLog,
		LoggerPrint:   LogPrintln,
	}

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[%v]日志错误: ", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}
`
		return fmt.Sprintf(content, projectName)
	},
	Path: func() string {
		return path.Join(baseDir, projectName, projectName)
	},
}

// SSL 文件
var ssl = InitFile{
	Name: "ssl.go",
	Content: func() string {
		content := `package %s

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"math/big"
	"os"
	"path"
	"time"
)

func init() {
	SSLPath := path.Join(Server.Settings.BASE_DIR, "ssl")
	if Server.Settings.SSL.STATUS == false {
		return
	}

	_, certErr := os.Stat(Server.Settings.SSL.CERT_PATH)
	_, keyErr := os.Stat(Server.Settings.SSL.KEY_PATH)
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
			panic(fmt.Sprintf("创建 SSL 目录错误: ", err))
		}
	}

	// 生成 128 位随机序列号
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
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
		NotBefore:             time.Now().In(Server.Settings.LOCATION),
		NotAfter:              time.Now().In(Server.Settings.LOCATION).Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	err = goi.GenerateRSACertificate(2048, certificateTemplate, SSLPath) // RSA 算法
	// err = GenerateECCCertificate(certificateTemplate, SSLPath) // ECC 算法
	if err != nil {
		panic(fmt.Sprintf("生成SSL证书和私钥时发生错误: ", err))
	}
}
`
		return fmt.Sprintf(content, projectName, projectName)
	},
	Path: func() string {
		return path.Join(baseDir, projectName, projectName)
	},
}
