package hgee

import (
	"fmt"
	defaultLog "log"
	"os"
	"path"
	"time"
)

type metaLevel uint8

// 日志等级
const (
	DEBUG   metaLevel = iota // 打印所有日志
	INFO    metaLevel = iota // 打印所有日志
	WARNING                  // 打印 Warning、Error 日志
	ERROR                    // 打印 Error 日志
)

type metaLogger struct {
	DEBUG            bool
	INFO_OUT_PATH    string // 所有日志输出路径
	ACCESS_OUT_PATH  string // 访问日志输出路径
	ERROR_OUT_PATH   string // 错误日志输出路径
	OUT_DATABASE     string // 所有日志输出到数据库
	loggerInfoFile   *defaultLog.Logger
	loggerAccessFile *defaultLog.Logger
	loggerErrorFile  *defaultLog.Logger
	loggerConsole    *defaultLog.Logger
}

// 对外方法
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

// NewLogger 创建一个logger
func newLogger() *metaLogger {
	return &metaLogger{
		DEBUG:            true,
		INFO_OUT_PATH:    "",
		ACCESS_OUT_PATH:  "",
		ERROR_OUT_PATH:   "",
		OUT_DATABASE:     "",
		loggerInfoFile:   nil,
		loggerAccessFile: nil,
		loggerErrorFile:  nil,
		loggerConsole:    nil,
	}
}

func checkDir(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0644)
		if err != nil {
			panic(fmt.Sprintf("创建目录错误", err))
		}
	}
}

func (engine *Engine) initLogger() {
	// DEBUG
	if engine.Log.DEBUG {
		// 初始化控制台日志
		engine.Log.loggerConsole = defaultLog.New(os.Stdout, "", 0)
		engine.Log.loggerConsole.SetFlags(0)
	}

	// INFO
	if engine.Log.INFO_OUT_PATH != "" {
		infoOutPath := engine.Log.INFO_OUT_PATH
		if string(engine.Log.INFO_OUT_PATH[0]) != "/" {
			infoOutPath = path.Join(engine.Settings.BASE_DIR, engine.Log.INFO_OUT_PATH)
		}
		checkDir(path.Dir(infoOutPath)) // 检查目录
		infoOutFile, err := os.OpenFile(infoOutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			panic(fmt.Sprintln("初始化[INFO]日志错误:", err))
		}
		engine.Log.loggerInfoFile = defaultLog.New(infoOutFile, "", 0)
		engine.Log.loggerInfoFile.SetFlags(0)
	}
	// ACCESS
	if engine.Log.ACCESS_OUT_PATH != "" {
		accessOutPath := engine.Log.ACCESS_OUT_PATH
		if string(engine.Log.ACCESS_OUT_PATH[0]) != "/" {
			accessOutPath = path.Join(engine.Settings.BASE_DIR, engine.Log.ACCESS_OUT_PATH)
		}
		checkDir(path.Dir(accessOutPath)) // 检查目录
		accessOutFile, err := os.OpenFile(accessOutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			panic(fmt.Sprintln("初始化[ACCESS]日志错误:", err))
		}
		engine.Log.loggerAccessFile = defaultLog.New(accessOutFile, "", 0)
		engine.Log.loggerAccessFile.SetFlags(0)
	}
	// ERROR
	if engine.Log.ERROR_OUT_PATH != "" {
		errorOutPath := engine.Log.ERROR_OUT_PATH
		if string(engine.Log.ERROR_OUT_PATH[0]) != "/" {
			errorOutPath = path.Join(engine.Settings.BASE_DIR, engine.Log.ERROR_OUT_PATH)
		}
		checkDir(path.Dir(errorOutPath)) // 检查目录
		errorOutFile, err := os.OpenFile(errorOutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			panic(fmt.Sprintln("初始化[ERROR]日志错误:", err))
		}
		engine.Log.loggerErrorFile = defaultLog.New(errorOutFile, "", 0)
		engine.Log.loggerErrorFile.SetFlags(0)
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

// Debug 通用debug日志(按天切分)
func (logger *metaLogger) Log(level metaLevel, log ...any) {
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
