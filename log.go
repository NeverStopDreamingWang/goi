package goi

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
)

type Level uint8

// 日志等级
const (
	DEBUG   Level = iota // 打印所有日志
	INFO                 // 打印所有日志
	WARNING              // 打印 Warning、Error 日志
	ERROR                // 打印 Error 日志
	meta
)

// 日志
type MetaLogger struct {
	Name          string                                       // 日志名称
	Path          string                                       // 日志输出路径
	Level         []Level                                      // 日志等级
	Logger        *log.Logger                                  // 日志对象
	File          *os.File                                     // 日志文件对象
	CreateTime    time.Time                                    // 日志创建时间
	SPLIT_SIZE    int64                                        // 日志切割大小，默认为 1G 1024 * 1024 * 1024
	SPLIT_TIME    string                                       // 日志切割大小，默认按天切割
	NewLoggerFunc func() *MetaLogger                           // 自定义初始化日志
	LoggerPrint   func(logger *MetaLogger, log ...interface{}) // 自定义日志输出
}

type metaLog struct {
	DEBUG   bool          // 默认为 true
	LOGGERS []*MetaLogger // 日志列表
}

func newLog() *metaLog {
	return &metaLog{
		DEBUG:   true,            // 开发模式，开启后自动输出到控制台
		LOGGERS: []*MetaLogger{}, // 日志
	}
}

// 日志初始化
func (metaLog *metaLog) InitLogger() {
	// DEBUG
	if metaLog.DEBUG == true {
		// 初始化控制台日志
		consoleLogger := &MetaLogger{
			Name:          "Debug-console",
			Path:          "",
			Level:         []Level{DEBUG, INFO, WARNING, ERROR},
			Logger:        log.New(os.Stdout, "", 0),
			File:          nil,
			CreateTime:    time.Now().In(Settings.LOCATION),
			SPLIT_SIZE:    0,
			SPLIT_TIME:    "",
			NewLoggerFunc: nil,
			LoggerPrint: func(logger *MetaLogger, log ...interface{}) {
				timeStr := fmt.Sprintf("[%v]", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05"))
				log = append([]interface{}{timeStr}, log...)
				logger.Logger.Println(log...)
			},
		}
		metaLog.LOGGERS = append(metaLog.LOGGERS, consoleLogger)
		consoleLogger.Logger.SetFlags(0)
	}
	for i, logger := range metaLog.LOGGERS {
		newLogger := logger.splitLogger()
		if newLogger != nil {
			metaLog.LOGGERS[i] = newLogger
		}
	}
}

// 日志切割
func (logger *MetaLogger) splitLogger() *MetaLogger {
	var err error
	if logger.Path == "" {
		return nil
	}
	fileInfo, err := os.Stat(logger.Path)
	if err != nil {
		panic(fmt.Sprintf("日志切割-[%v]获取日志文件信息错误: %v", logger.Name, err))
		return nil
	}
	fileSize := fileInfo.Size()
	nowTime := time.Now().In(Settings.LOCATION)
	isSplit := false
	if logger.SPLIT_TIME != "" {
		if logger.CreateTime.Format(logger.SPLIT_TIME) != nowTime.Format(logger.SPLIT_TIME) {
			isSplit = true
		}
	}
	if logger.SPLIT_SIZE != 0 {
		if logger.SPLIT_SIZE <= fileSize {
			isSplit = true
		}
	}
	if isSplit == true && logger.Path != "" && logger.File != nil {
		var (
			fileName string
			fileExt  string
			fileDir  string
		)

		fileName = filepath.Base(logger.Path)
		fileDir = filepath.Dir(logger.Path)
		for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
			if fileName[i] == '.' {
				fileExt = fileName[i:]
				fileName = fileName[:i]
				break
			}
		}
		// 自动加 _n
		oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", fileName, logger.CreateTime.Format(logger.SPLIT_TIME), fileExt))
		_, err = os.Stat(oldInfoFile)
		for idx := 1; err == nil; idx++ {
			oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", fileName, logger.CreateTime.Format(logger.SPLIT_TIME), idx, fileExt))
			_, err = os.Stat(oldInfoFile)
		}

		logger.File.Close()
		err = os.Rename(logger.Path, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintf("日志切割-[%v]日志重命名错误: %v", logger.Name, err))
		}
		// 重新初始化
		return logger.NewLoggerFunc()
	}
	return nil
}

// Info 日志
func (metaLog *metaLog) Info(log ...interface{}) {
	metaLog.Log(INFO, log...)
}

// WARNING 日志
func (metaLog *metaLog) Warning(log ...interface{}) {
	metaLog.Log(WARNING, log...)
}

// ERROR 日志
func (metaLog *metaLog) Error(log ...interface{}) {
	metaLog.Log(ERROR, log...)
}

// log 打印日志
func (metaLog *metaLog) Log(level Level, log ...interface{}) {
	// 输出到所有符合的日志中
	for i, logger := range metaLog.LOGGERS {
		for _, loggerLever := range logger.Level {
			if level == loggerLever || level == meta {
				newLogger := logger.splitLogger()
				if newLogger != nil {
					metaLog.LOGGERS[i] = newLogger
					logger = newLogger
				}
				logger.LoggerPrint(logger, log...) // 调用自定义输出函数
				break
			}
		}
	}
}
