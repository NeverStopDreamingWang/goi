package hgee

import (
	"fmt"
	defaultLog "log"
	"os"
	"path"
	"path/filepath"
	"time"
)

// 公共方法
func Debug(log ...any) {
	engine.Log.Debug(log...)
}
func Info(log ...any) {
	engine.Log.Info(log...)
}
func Warning(log ...any) {
	engine.Log.Warning(log...)
}
func Error(log ...any) {
	engine.Log.Error(log...)
}

type metaLevel uint8

// 日志等级
const (
	DEBUG   metaLevel = iota // 打印所有日志
	INFO    metaLevel = iota // 打印所有日志
	WARNING                  // 打印 Warning、Error 日志
	ERROR                    // 打印 Error 日志
)

type metaLogger struct {
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
func newLogger() *metaLogger {
	return &metaLogger{
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
func (logger *metaLogger) initLogger() {
	serverInfo := fmt.Sprintf("hgee version: %v\nserver run: %v:%v", engine.Settings.version, engine.Settings.SERVER_ADDRESS, engine.Settings.SERVER_PORT)
	// DEBUG
	if logger.DEBUG == true {
		// 初始化控制台日志
		logger.loggerConsole = defaultLog.New(os.Stdout, "", 0)
		logger.loggerConsole.SetFlags(0)
		logger.loggerConsole.Println(serverInfo)
	}

	// INFO
	if logger.INFO_OUT_PATH != "" {
		logger.loggerInfoFileInit()  // 初始化
		logger.loggerInfoFileSplit() // 日志切割
		logger.loggerInfoFile.Println(serverInfo)
	}
	// ACCESS
	if logger.ACCESS_OUT_PATH != "" {
		logger.loggerAccessFileInit()
		logger.loggerAccessFileSplit()
		logger.loggerAccessFile.Println(serverInfo)
	}
	// ERROR
	if logger.ERROR_OUT_PATH != "" {
		logger.loggerErrorFileInit()
		logger.loggerErrorFileSplit()
		logger.loggerErrorFile.Println(serverInfo)
	}
}

// 检查目录
func (logger *metaLogger) checkDir(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0644)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误：", err))
		}
	}
}

// INFO 日志初始化
func (logger *metaLogger) loggerInfoFileInit() {
	OutPath := logger.INFO_OUT_PATH
	if string(logger.INFO_OUT_PATH[0]) != "/" {
		OutPath = path.Join(engine.Settings.BASE_DIR, logger.INFO_OUT_PATH)
	}
	logger.checkDir(path.Dir(OutPath)) // 检查目录
	var err error
	logger.outInfoFile, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[INFO]日志错误：", err))
	}
	logger.loggerInfoFile = defaultLog.New(logger.outInfoFile, "", 0)
	logger.loggerInfoFile.SetFlags(0)
	logger.createInfoTime = time.Now()
}

// INFO 日志切割
func (logger *metaLogger) loggerInfoFileSplit() {
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
		panic(fmt.Sprintf("日志切割-获取[INFO]日志文件信息错误：%v", err))
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
	nowTime := time.Now()
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v_1%v", name, logger.createInfoTime.Format(logger.SplitTime), ext))
	_, err = os.Stat(oldInfoFile)
	for idx := 2; err == nil; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v_%v%v", name, logger.createInfoTime.Format(logger.SplitTime), idx, ext))
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
			panic(fmt.Sprintln("关闭[INFO]日志错误：", err))
		}
		err = os.Rename(logger.INFO_OUT_PATH, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintln("切割[INFO]日志错误：", err))
		}
		logger.loggerInfoFileInit()
	}
}

// ACCESS 日志初始化
func (logger *metaLogger) loggerAccessFileInit() {
	OutPath := logger.ACCESS_OUT_PATH
	if string(logger.ACCESS_OUT_PATH[0]) != "/" {
		OutPath = path.Join(engine.Settings.BASE_DIR, logger.ACCESS_OUT_PATH)
	}
	logger.checkDir(path.Dir(OutPath)) // 检查目录
	var err error
	logger.outAccessFile, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[ACCESS]日志错误：", err))
	}
	logger.loggerAccessFile = defaultLog.New(logger.outAccessFile, "", 0)
	logger.loggerAccessFile.SetFlags(0)
	logger.createAccessTime = time.Now()
}

// ACCESS 日志切割
func (logger *metaLogger) loggerAccessFileSplit() {
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
		panic(fmt.Sprintf("日志切割-获取[ACCESS]日志文件信息错误：%v", err))
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
	nowTime := time.Now()
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v_1%v", name, logger.createInfoTime.Format(logger.SplitTime), ext))
	_, err = os.Stat(oldInfoFile)
	for idx := 2; err == nil; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v_%v%v", name, logger.createInfoTime.Format(logger.SplitTime), idx, ext))
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
			panic(fmt.Sprintln("关闭[ACCESS]日志错误：", err))
		}
		err = os.Rename(logger.ACCESS_OUT_PATH, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintln("切割[ACCESS]日志错误：", err))
		}
		logger.loggerAccessFileInit()
	}
}

// ERROR 错误日志
func (logger *metaLogger) loggerErrorFileInit() {
	OutPath := logger.ERROR_OUT_PATH
	if string(logger.ERROR_OUT_PATH[0]) != "/" {
		OutPath = path.Join(engine.Settings.BASE_DIR, logger.ERROR_OUT_PATH)
	}
	logger.checkDir(path.Dir(OutPath)) // 检查目录
	var err error
	logger.outErrorFile, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[ERROR]日志错误：", err))
	}
	logger.loggerErrorFile = defaultLog.New(logger.outErrorFile, "", 0)
	logger.loggerErrorFile.SetFlags(0)
	logger.createErrorTime = time.Now()
}

// ERROR 日志切割
func (logger *metaLogger) loggerErrorFileSplit() {
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
		panic(fmt.Sprintf("日志切割-获取[ERROR]日志文件信息错误：%v", err))
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
	nowTime := time.Now()
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v_1%v", name, logger.createInfoTime.Format(logger.SplitTime), ext))
	_, err = os.Stat(oldInfoFile)
	for idx := 2; err == nil; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v_%v%v", name, logger.createInfoTime.Format(logger.SplitTime), idx, ext))
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
			panic(fmt.Sprintln("关闭[ERROR]日志错误：", err))
		}
		err = os.Rename(logger.ERROR_OUT_PATH, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintln("切割[ERROR]日志错误：", err))
		}
		logger.loggerErrorFileInit()
	}
}

// 日志方法
// Debug 通用 Debug 日志
func (logger *metaLogger) Debug(log ...any) {
	logger.Log(DEBUG, log...)
}

// Info 通用 Info 日志
func (logger *metaLogger) Info(log ...any) {
	logger.Log(INFO, log...)
}

// WARNING 通用 WARNING 日志
func (logger *metaLogger) Warning(log ...any) {
	logger.Log(WARNING, log...)
}

// ERROR 通用 ERROR 日志
func (logger *metaLogger) Error(log ...any) {
	logger.Log(ERROR, log...)
}

// Log 打印日志
func (logger *metaLogger) Log(level metaLevel, log ...any) {
	// 日志切割
	logger.loggerInfoFileSplit()
	logger.loggerAccessFileSplit()
	logger.loggerErrorFileSplit()

	logData := make([]any, 1, len(log)+1)
	switch level {
	case INFO:
		logData[0] = fmt.Sprintf("[%s] - [INFO]", time.Now().Format("2006-01-02 15:04:05"))
		logData = append(logData, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(logData...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(logData...)
		}
		// ACCESS 日志
		if logger.loggerAccessFile != nil {
			logger.loggerAccessFile.Println(logData...)
		}
	case WARNING:
		logData[0] = fmt.Sprintf("[%s] - [WARNING]", time.Now().Format("2006-01-02 15:04:05"))
		logData = append(logData, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(logData...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(logData...)
		}
		// ACCESS 日志
		if logger.loggerAccessFile != nil {
			logger.loggerAccessFile.Println(logData...)
		}
	case ERROR:
		logData[0] = fmt.Sprintf("[%s] - [ERROR]", time.Now().Format("2006-01-02 15:04:05"))
		logData = append(logData, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(logData...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(logData...)
		}
		// ERROR 日志
		if logger.loggerErrorFile != nil {
			logger.loggerErrorFile.Println(logData...)
		}
	default:
		logData[0] = fmt.Sprintf("[%s] - [DEBUG]", time.Now().Format("2006-01-02 15:04:05"))
		logData = append(logData, log...)
		// 控制台 日志
		if logger.loggerConsole != nil {
			logger.loggerConsole.Println(logData...)
		}
		// INFO 日志
		if logger.loggerInfoFile != nil {
			logger.loggerInfoFile.Println(logData...)
		}
		// ACCESS 日志
		if logger.loggerAccessFile != nil {
			logger.loggerAccessFile.Println(logData...)
		}
		// ERROR 日志
		if logger.loggerErrorFile != nil {
			logger.loggerErrorFile.Println(logData...)
		}
	}
}
