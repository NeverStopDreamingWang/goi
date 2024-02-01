package goi

import (
	"fmt"
	defaultLog "log"
	"os"
	"path"
	"path/filepath"
	"time"
)

type logLevel uint8

// 日志等级
const (
	DEBUG   logLevel = iota // 打印所有日志
	INFO                    // 打印所有日志
	WARNING                 // 打印 Warning、Error 日志
	ERROR                   // 打印 Error 日志
	MeteLog                 // 日志
)

type MetaLogger struct {
	DEBUG            bool               // 默认为 true
	INFO_OUT_PATH    string             // 所有日志输出路径
	ACCESS_OUT_PATH  string             // 访问日志输出路径
	ERROR_OUT_PATH   string             // 错误日志输出路径
	OUT_DATABASE     string             // 所有日志输出到数据库
	SplitSize        int64              // 日志切割大小,默认为 1G 1024 * 1024 * 1024
	SplitTime        string             // 日志切割大小,默认按天切割
	loggerConsole    *defaultLog.Logger // 控制台日志对象
	loggerInfoFile   *defaultLog.Logger // 所有日志对象
	loggerAccessFile *defaultLog.Logger // 访问日志对象
	loggerErrorFile  *defaultLog.Logger // 错误日志对象
	createInfoTime   time.Time          // 所有日志创建时间
	createAccessTime time.Time          // 访问日志创建时间
	createErrorTime  time.Time          // 错误日志创建时间
	outInfoFile      *os.File           // 所有日志文件对象
	outAccessFile    *os.File           // 访问日志文件对象
	outErrorFile     *os.File           // 错误日志文件对象
}

// NewLogger 创建一个logger
func NewLogger() *MetaLogger {
	return &MetaLogger{
		DEBUG:            true,
		INFO_OUT_PATH:    "",
		ACCESS_OUT_PATH:  "",
		ERROR_OUT_PATH:   "",
		OUT_DATABASE:     "",
		SplitSize:        1024 * 1024 * 1024,
		SplitTime:        "2006-01-02",
		loggerConsole:    nil,
		loggerInfoFile:   nil,
		loggerAccessFile: nil,
		loggerErrorFile:  nil,
		createInfoTime:   time.Time{},
		createAccessTime: time.Time{},
		createErrorTime:  time.Time{},
		outInfoFile:      nil,
		outAccessFile:    nil,
		outErrorFile:     nil,
	}
}

// 日志初始化
func (logger *MetaLogger) InitLogger() {
	// DEBUG
	if logger.DEBUG == true {
		// 初始化控制台日志
		logger.loggerConsole = defaultLog.New(os.Stdout, "", 0)
		logger.loggerConsole.SetFlags(0)
	}

	// INFO
	if logger.INFO_OUT_PATH != "" {
		logger.loggerInfoFileInit()  // 初始化
		logger.loggerInfoFileSplit() // 日志切割
	}
	// ACCESS
	if logger.ACCESS_OUT_PATH != "" {
		logger.loggerAccessFileInit()
		logger.loggerAccessFileSplit()
	}
	// ERROR
	if logger.ERROR_OUT_PATH != "" {
		logger.loggerErrorFileInit()
		logger.loggerErrorFileSplit()
	}
}

// INFO 日志初始化
func (logger *MetaLogger) loggerInfoFileInit() {
	OutPath := logger.INFO_OUT_PATH
	if string(logger.INFO_OUT_PATH[0]) != "/" {
		OutPath = path.Join(Settings.BASE_DIR, logger.INFO_OUT_PATH)
	}
	var err error

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	logger.outInfoFile, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[INFO]日志错误: ", err))
	}
	logger.loggerInfoFile = defaultLog.New(logger.outInfoFile, "", 0)
	logger.createInfoTime = time.Now().In(Settings.LOCATION)
}

// INFO 日志切割
func (logger *MetaLogger) loggerInfoFileSplit() {
	var err error
	if logger.INFO_OUT_PATH == "" {
		return
	}
	fileName := filepath.Base(logger.INFO_OUT_PATH)
	fileDir := filepath.Dir(logger.INFO_OUT_PATH)
	fileInfo, err := os.Stat(logger.INFO_OUT_PATH)
	if os.IsNotExist(err) {
		logger.loggerInfoFileInit()
		return
	} else if err != nil {
		panic(fmt.Sprintf("日志切割-获取[INFO]日志文件信息错误: %v", err))
		return
	}
	fileSize := fileInfo.Size()
	var name string
	var ext string
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		if fileName[i] == '.' {
			name = fileName[:i]
			ext = fileName[i:]
		}
	}
	nowTime := time.Now().In(Settings.LOCATION)
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", name, logger.createInfoTime.Format(logger.SplitTime), ext))
	_, err = os.Stat(oldInfoFile)
	for idx := 1; err == nil; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", name, logger.createInfoTime.Format(logger.SplitTime), idx, ext))
		_, err = os.Stat(oldInfoFile)
	}
	isSplit := false
	if logger.SplitTime != "" {
		if logger.createInfoTime.Format(logger.SplitTime) != nowTime.Format(logger.SplitTime) {
			isSplit = true
		}
	}
	if logger.SplitSize != 0 {
		if logger.SplitSize <= fileSize {
			isSplit = true
		}
	}
	if isSplit == true {
		err = logger.outInfoFile.Close()
		if err != nil {
			panic(fmt.Sprintln("关闭[INFO]日志错误: ", err))
		}
		err = os.Rename(logger.INFO_OUT_PATH, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintln("切割[INFO]日志错误: ", err))
		}
		logger.loggerInfoFileInit()
	}
}

// ACCESS 日志初始化
func (logger *MetaLogger) loggerAccessFileInit() {
	OutPath := logger.ACCESS_OUT_PATH
	if string(logger.ACCESS_OUT_PATH[0]) != "/" {
		OutPath = path.Join(Settings.BASE_DIR, logger.ACCESS_OUT_PATH)
	}
	var err error

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	logger.outAccessFile, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[ACCESS]日志错误: ", err))
	}
	logger.loggerAccessFile = defaultLog.New(logger.outAccessFile, "", 0)
	logger.createAccessTime = time.Now().In(Settings.LOCATION)
}

// ACCESS 日志切割
func (logger *MetaLogger) loggerAccessFileSplit() {
	var err error
	if logger.ACCESS_OUT_PATH == "" {
		return
	}
	fileName := filepath.Base(logger.ACCESS_OUT_PATH)
	fileDir := filepath.Dir(logger.ACCESS_OUT_PATH)
	fileInfo, err := os.Stat(logger.ACCESS_OUT_PATH)
	if os.IsNotExist(err) {
		logger.loggerAccessFileInit()
		return
	} else if err != nil {
		panic(fmt.Sprintf("日志切割-获取[ACCESS]日志文件信息错误: %v", err))
		return
	}
	fileSize := fileInfo.Size()
	var name string
	var ext string
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		if fileName[i] == '.' {
			name = fileName[:i]
			ext = fileName[i:]
		}
	}
	nowTime := time.Now().In(Settings.LOCATION)
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", name, logger.createInfoTime.Format(logger.SplitTime), ext))
	_, err = os.Stat(oldInfoFile)
	for idx := 1; err == nil; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", name, logger.createInfoTime.Format(logger.SplitTime), idx, ext))
		_, err = os.Stat(oldInfoFile)
	}
	isSplit := false
	if logger.SplitTime != "" {
		if logger.createAccessTime.Format(logger.SplitTime) != nowTime.Format(logger.SplitTime) {
			isSplit = true
		}
	}
	if logger.SplitSize != 0 {
		if logger.SplitSize <= fileSize {
			isSplit = true
		}
	}
	if isSplit == true {
		err = logger.outAccessFile.Close()
		if err != nil {
			panic(fmt.Sprintln("关闭[ACCESS]日志错误: ", err))
		}
		err = os.Rename(logger.ACCESS_OUT_PATH, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintln("切割[ACCESS]日志错误: ", err))
		}
		logger.loggerAccessFileInit()
	}
}

// ERROR 错误日志
func (logger *MetaLogger) loggerErrorFileInit() {
	OutPath := logger.ERROR_OUT_PATH
	if string(logger.ERROR_OUT_PATH[0]) != "/" {
		OutPath = path.Join(Settings.BASE_DIR, logger.ERROR_OUT_PATH)
	}
	var err error

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	logger.outErrorFile, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[ERROR]日志错误: ", err))
	}
	logger.loggerErrorFile = defaultLog.New(logger.outErrorFile, "", 0)
	logger.createErrorTime = time.Now().In(Settings.LOCATION)
}

// ERROR 日志切割
func (logger *MetaLogger) loggerErrorFileSplit() {
	var err error
	if logger.ERROR_OUT_PATH == "" {
		return
	}
	fileName := filepath.Base(logger.ERROR_OUT_PATH)
	fileDir := filepath.Dir(logger.ERROR_OUT_PATH)
	fileInfo, err := os.Stat(logger.ERROR_OUT_PATH)
	if os.IsNotExist(err) {
		logger.loggerErrorFileInit()
		return
	} else if err != nil {
		panic(fmt.Sprintf("日志切割-获取[ERROR]日志文件信息错误: %v", err))
		return
	}
	fileSize := fileInfo.Size()
	var name string
	var ext string
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		if fileName[i] == '.' {
			name = fileName[:i]
			ext = fileName[i:]
		}
	}
	nowTime := time.Now().In(Settings.LOCATION)
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", name, logger.createInfoTime.Format(logger.SplitTime), ext))
	_, err = os.Stat(oldInfoFile)
	for idx := 1; err == nil; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", name, logger.createInfoTime.Format(logger.SplitTime), idx, ext))
		_, err = os.Stat(oldInfoFile)
	}
	isSplit := false
	if logger.SplitTime != "" {
		if logger.createErrorTime.Format(logger.SplitTime) != nowTime.Format(logger.SplitTime) {
			isSplit = true
		}
	}
	if logger.SplitSize != 0 {
		if logger.SplitSize <= fileSize {
			isSplit = true
		}
	}
	if isSplit == true {
		err = logger.outErrorFile.Close()
		if err != nil {
			panic(fmt.Sprintln("关闭[ERROR]日志错误: ", err))
		}
		err = os.Rename(logger.ERROR_OUT_PATH, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintln("切割[ERROR]日志错误: ", err))
		}
		logger.loggerErrorFileInit()
	}
}

// 日志方法
// Debug 通用 Debug 日志
func (logger *MetaLogger) Debug(log ...any) {
	logger.log(DEBUG, log...)
}

// Info 通用 Info 日志
func (logger *MetaLogger) Info(log ...any) {
	logger.log(INFO, log...)
}

// WARNING 通用 WARNING 日志
func (logger *MetaLogger) Warning(log ...any) {
	logger.log(WARNING, log...)
}

// ERROR 通用 ERROR 日志
func (logger *MetaLogger) Error(log ...any) {
	logger.log(ERROR, log...)
}

// Log 日志
func (logger *MetaLogger) MetaLog(log ...any) {
	logger.log(MeteLog, log...)
}

// log 打印日志
func (logger *MetaLogger) log(level logLevel, log ...any) {
	// 日志切割
	logger.loggerInfoFileSplit()
	logger.loggerAccessFileSplit()
	logger.loggerErrorFileSplit()

	switch level {
	case INFO:
		prefix := fmt.Sprintf("[%s] - [INFO]", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05"))
		log = append([]any{prefix}, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(log...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(log...)
		}
		// ACCESS 日志
		if logger.loggerAccessFile != nil {
			logger.loggerAccessFile.Println(log...)
		}
	case WARNING:
		prefix := fmt.Sprintf("[%s] - [WARNING]", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05"))
		log = append([]any{prefix}, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(log...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(log...)
		}
	case ERROR:
		prefix := fmt.Sprintf("[%s] - [ERROR]", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05"))
		log = append([]any{prefix}, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(log...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(log...)
		}
		// ERROR 日志
		if logger.loggerErrorFile != nil {
			logger.loggerErrorFile.Println(log...)
		}
	case DEBUG:
		prefix := fmt.Sprintf("[%s] - [DEBUG]", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05"))
		log = append([]any{prefix}, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(log...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(log...)
		}
		// ACCESS 日志
		if logger.loggerAccessFile != nil {
			logger.loggerAccessFile.Println(log...)
		}
		// ERROR 日志
		if logger.loggerErrorFile != nil {
			logger.loggerErrorFile.Println(log...)
		}
	default: // log
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(log...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(log...)
		}
		// ACCESS 日志
		if logger.loggerAccessFile != nil {
			logger.loggerAccessFile.Println(log...)
		}
		// ERROR 日志
		if logger.loggerErrorFile != nil {
			logger.loggerErrorFile.Println(log...)
		}
	}
}
