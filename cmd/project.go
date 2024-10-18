package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/crypto"
	"github.com/spf13/cobra"
)

var CreateProject = &cobra.Command{
	Use:   "create-project",
	Short: "创建项目",
	Args:  cobra.ExactArgs(1),
	RunE:  GoiCreateProject,
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
	validator,
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
		file, err := os.Create(filePath)

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

require github.com/NeverStopDreamingWang/goi %s

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
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
	"database/sql"
	"os"
	"path"
	"time"

	"github.com/NeverStopDreamingWang/goi"
)

// Http 服务
var Server *goi.Engine

func init() {
	var err error
	
	// 创建 http 服务
	Server = goi.NewHttpServer()
	
	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()
	// 网络协议
	Server.Settings.NET_WORK = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 监听地址
	Server.Settings.BIND_ADDRESS = "0.0.0.0" // 默认 127.0.0.1
	// 端口
	Server.Settings.PORT = 8080
	// 域名
	Server.Settings.BIND_DOMAIN = ""

	// AES 密钥
	Server.Settings.SECRET_KEY = "%s"

	// RSA 私钥
	Server.Settings.PRIVATE_KEY = ` + "`%s`" + `

	// RSA 公钥
	Server.Settings.PUBLIC_KEY = ` + "`%s`" + `

	// 设置 SSL
	Server.Settings.SSL = goi.MetaSSL{
		STATUS:    false, // SSL 开关
		TYPE:      "自签证书", // 证书类型
		CERT_PATH: path.Join(Server.Settings.BASE_DIR, "ssl/%s.crt"),
		KEY_PATH:  path.Join(Server.Settings.BASE_DIR, "ssl/%s.key"),
	}
	
	// 数据库配置
	Server.Settings.DATABASES["default"] = &goi.DataBase{
		ENGINE:         "mysql",
		DataSourceName: "root:123@tcp(127.0.0.1:3306)/test_goi",
		Connect: func(ENGINE string, DataSourceName string) (*sql.DB, error) {
			mysqlDB, err := sql.Open(ENGINE, DataSourceName)
			if err != nil {
				panic(err)
			}
			// 设置连接池参数
			mysqlDB.SetMaxOpenConns(10)           // 设置最大打开连接数
			mysqlDB.SetMaxIdleConns(5)            // 设置最大空闲连接数
			mysqlDB.SetConnMaxLifetime(time.Hour) // 设置连接的最大存活时间
			return mysqlDB, nil
		},
	}
	Server.Settings.DATABASES["sqlite"] = &goi.DataBase{
		ENGINE:         "sqlite3",
		DataSourceName: path.Join(Server.Settings.BASE_DIR, "data/test_goi.db"),
		Connect: func(ENGINE string, DataSourceName string) (*sql.DB, error) {
			sqliteDB, err := sql.Open(ENGINE, DataSourceName)
			if err != nil {
				panic(err)
			}
			return sqliteDB, nil
		},
	}
	
	// 设置时区
	err = Server.Settings.SetTimeZone("Asia/Shanghai") // 默认为空字符串 ''，本地时间
	if err != nil {
		panic(err)
	}
	//  Server.Settings.GetLocation() 获取时区 Location

	// 设置框架语言
	Server.Settings.SetLanguage(goi.ZH_CN) // 默认 ZH_CN
	
	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用
	
	// 日志 DEBUG 设置
	Server.Log.DEBUG = true
	// 注册日志
	defaultLog := newDefaultLog() // 默认日志
	err = Server.Log.RegisterLogger(defaultLog)
	if err != nil {
		panic(err)
	}
	accessLog := newAccessLog() // 访问日志
	err = Server.Log.RegisterLogger(accessLog)
	if err != nil {
		panic(err)
	}
	errorLog := newErrorLog() // 错误日志
	err = Server.Log.RegisterLogger(errorLog)
	if err != nil {
		panic(err)
	}
	
	// 日志打印
	// Server.Log.Log() = goi.Log.Log()
	// Server.Log.Info() = goi.Log.Info()
	// Server.Log.Warning() = goi.Log.Warning()
	// Server.Log.Error() = goi.Log.Error()
	
	// 设置验证器错误，不指定则使用默认
	Server.Validator.SetValidationError(&validationError{})
	
	// 设置自定义配置
	// Server.Settings.Set(key string, value interface{})
	// Server.Settings.Get(key string, dest interface{})
}
`
		return fmt.Sprintf(content, projectName, secretKey, privateKey, publicKey, projectName, projectName)
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

func init() {
	// 注册路由转换器
	
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

// 验证器文件
var validator = InitFile{
	Name: "validator.go",
	Content: func() string {
		content := `package %s

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 注册验证器
	// 手机号
	goi.RegisterValidate("phone", phoneValidate)
}

// phone 类型
func phoneValidate(value string) goi.ValidationError {
	var IntRe = %s
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%%v", value))
	}
	return nil
}

// 自定义参数验证错误
type validationError struct {
	Status  int
	Message string
}

// 创建参数验证错误方法
func (validationErr *validationError) NewValidationError(status int, message string, args ...interface{}) goi.ValidationError {
	return &validationError{
		Status:  status,
		Message: message,
	}
}

// 参数验证错误响应格式
func (validationErr *validationError) Response() goi.Response {
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status:  validationErr.Status,
			Message: validationErr.Message,
			Results:    nil,
		},
	}
}

`
		return fmt.Sprintf(content, projectName, "`^(1[3456789]\\d{9})$`")
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
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/NeverStopDreamingWang/goi"
)

// 日志输出
func LogPrintln(logger *goi.MetaLogger, level goi.Level, logs ...interface{}) {
	timeStr := fmt.Sprintf("[%%v]", time.Now().In(Server.Settings.GetLocation()).Format("2006-01-02 15:04:05"))
	if level != "" {
		timeStr += fmt.Sprintf(" %%v", level)
	}
	logs = append([]interface{}{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// 默认日志
func newDefaultLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/server.log")
	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: %%v", err))
		}
	}

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
		LoggerPrint:     LogPrintln, // 日志输出格式
		CreateTime:    time.Now().In(Server.Settings.GetLocation()),
		SPLIT_SIZE:      1024 * 5,      // 切割大小
		SPLIT_TIME:      "2006-01-02",  // 切割日期，每天
		NewLoggerFunc:   newDefaultLog, // 初始化日志，同时用于自动切割日志后初始化新日志
		SplitLoggerFunc: nil,           // 自定义日志切割：传入旧的日志对象，返回新日志对象
	}

	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("初始化[%%v]日志错误: %%v", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 访问日志
func newAccessLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/access.log")
	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: %%v", err))
		}
	}

	defaultLog := &goi.MetaLogger{
		Name: "访问日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入正确的日志
			goi.INFO,
		},
		Logger:        nil,
		File:          nil,
		LoggerPrint:     LogPrintln, // 日志输出格式
		CreateTime:    time.Now().In(Server.Settings.GetLocation()),
		SPLIT_SIZE:      1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:      "2006-01-02",    // 切割日期，每天
		NewLoggerFunc:   newAccessLog,    // 初始化日志，同时用于自动切割日志后初始化新日志
		SplitLoggerFunc: nil,             // 自定义日志切割：传入旧的日志对象，返回新日志对象
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("初始化[%%v]日志错误: %%v", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 错误日志
func newErrorLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/error.log")
	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: %%v", err))
		}
	}

	defaultLog := &goi.MetaLogger{
		Name: "错误日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入错误的日志
			goi.ERROR,
		},
		Logger:        nil,
		File:          nil,
		LoggerPrint:     LogPrintln, // 日志输出格式
		CreateTime:    time.Now().In(Server.Settings.GetLocation()),
		SPLIT_SIZE:      1024 * 1024 * 5,   // 切割大小
		SPLIT_TIME:      "2006-01-02",      // 切割日期，每天
		NewLoggerFunc:   newErrorLog,       // 初始化日志，同时用于自动切割日志后初始化新日志
		SplitLoggerFunc: mySplitLoggerFunc, // 自定义日志切割：传入旧的日志对象，返回新日志对象，nil 根据 SPLIT_SIZE,SPLIT_TIME 自动切割
	}

	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("初始化[%%v]日志错误: %%v", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 自定义日志切割
func mySplitLoggerFunc(OldLogger *goi.MetaLogger) *goi.MetaLogger {
	var err error
	if OldLogger.Path == "" {
		return nil
	}
	fileInfo, err := os.Stat(OldLogger.Path)
	if err != nil {
		panic(fmt.Sprintf("日志切割-[%%v]获取日志文件信息错误: %%v", OldLogger.Name, err))
		return nil
	}
	fileSize := fileInfo.Size()
	nowTime := time.Now().In(Server.Settings.GetLocation())
	isSplit := false
	if OldLogger.SPLIT_TIME != "" { // 按照日志大小
		if OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME) != nowTime.Format(OldLogger.SPLIT_TIME) {
			isSplit = true
		}
	}
	if OldLogger.SPLIT_SIZE != 0 { // 按照日期
		if OldLogger.SPLIT_SIZE <= fileSize {
			isSplit = true
		}
	}
	if isSplit == true && OldLogger.Path != "" && OldLogger.File != nil {
		var (
			fileName string
			fileExt  string
			fileDir  string
		)

		fileName = filepath.Base(OldLogger.Path)
		fileDir = filepath.Dir(OldLogger.Path)
		for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
			if fileName[i] == '.' {
				fileExt = fileName[i:]
				fileName = fileName[:i]
				break
			}
		}
		// 自动加 _n
		oldInfoFile := path.Join(fileDir, fmt.Sprintf("%%v_%%v%%v", fileName, OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME), fileExt))
		_, err = os.Stat(oldInfoFile)
		for idx := 1; err == nil; idx++ {
			oldInfoFile = path.Join(fileDir, fmt.Sprintf("%%v_%%v(%%v)%%v", fileName, OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME), idx, fileExt))
			_, err = os.Stat(oldInfoFile)
		}

		OldLogger.File.Close()
		err = os.Rename(OldLogger.Path, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintf("日志切割-[%%v]日志重命名错误: %%v", OldLogger.Name, err))
		}
		// 重新初始化
		return OldLogger.NewLoggerFunc()
	}
	return nil
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
			panic(fmt.Sprintf("创建 SSL 目录错误: %%v", err))
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
		NotBefore:             time.Now().In(Server.Settings.GetLocation()),
		NotAfter:              time.Now().In(Server.Settings.GetLocation()).Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	err = goi.GenerateRSACertificate(2048, certificateTemplate, SSLPath) // RSA 算法
	// err = GenerateECCCertificate(certificateTemplate, SSLPath) // ECC 算法
	if err != nil {
		panic(fmt.Sprintf("生成SSL证书和私钥时发生错误: %%v", err))
	}
}
`
		return fmt.Sprintf(content, projectName, projectName)
	},
	Path: func() string {
		return path.Join(baseDir, projectName, projectName)
	},
}
