package goi

import (
	"errors"
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
	meta Level = iota // 框架日志
	DEBUG
	INFO
	WARNING
	ERROR
)

// 日志
type MetaLogger struct {
	Name            string                                       // 日志名称
	Path            string                                       // 日志输出路径
	Level           []Level                                      // 日志等级
	Logger          *log.Logger                                  // 日志对象
	File            *os.File                                     // 日志文件对象
	LoggerPrint     func(logger *MetaLogger, log ...interface{}) // 自定义日志输出
	CreateTime      time.Time                                    // 日志创建时间
	SPLIT_SIZE      int64                                        // 日志切割大小，默认为 1G 1024 * 1024 * 1024
	SPLIT_TIME      string                                       // 日志切割大小，默认按天切割
	NewLoggerFunc   func() *MetaLogger                           // 初始化日志，同时用于自动切割日志后初始化新日志
	SplitLoggerFunc func(OldLogger *MetaLogger) *MetaLogger      // 自定义日志切割：传入旧的日志对象，返回新日志对象
}

type metaLog struct {
	DEBUG   bool          // 默认为 true
	loggers []*MetaLogger // 日志列表
}

func newLog() *metaLog {
	return &metaLog{
		DEBUG:   true,            // 开发模式，开启后自动输出到控制台
		loggers: []*MetaLogger{}, // 日志
	}
}

// 注册日志
func (metaLog *metaLog) RegisterLogger(loggers *MetaLogger) error {
	if loggers.Path == "" {
		invalidPathMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_path",
		})
		return errors.New(invalidPathMsg)
	}
	if len(loggers.Level) == 0 {
		invalidLevelMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_level",
		})
		return errors.New(invalidLevelMsg)
	}
	if loggers.File == nil {
		invalidFileMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_file",
		})
		return errors.New(invalidFileMsg)
	}
	if loggers.Logger == nil {
		invalidObjectMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_object",
		})
		return errors.New(invalidObjectMsg)
	}
	if loggers.NewLoggerFunc == nil {
		invalidNewLoggerFuncMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_NewLoggerFunc",
		})
		return errors.New(invalidNewLoggerFuncMsg)
	}

	metaLog.loggers = append(metaLog.loggers, loggers)
	return nil
}

// 日志初始化
func (metaLog *metaLog) InitLogger() {
	// DEBUG
	if metaLog.DEBUG == true {
		// 初始化控制台日志
		consoleLogger := &MetaLogger{
			Name:            "Debug-console",
			Path:            "",
			Level:           []Level{DEBUG, INFO, WARNING, ERROR},
			Logger:          log.New(os.Stdout, "", 0),
			File:            nil,
			CreateTime:      time.Now().In(Settings.GetLocation()),
			SPLIT_SIZE:      0,
			SPLIT_TIME:      "",
			SplitLoggerFunc: nil,
			NewLoggerFunc:   nil,
			LoggerPrint: func(logger *MetaLogger, log ...interface{}) {
				timeStr := fmt.Sprintf("[%v]", time.Now().In(Settings.GetLocation()).Format("2006-01-02 15:04:05"))
				log = append([]interface{}{timeStr}, log...)
				logger.Logger.Println(log...)
			},
		}
		metaLog.loggers = append(metaLog.loggers, consoleLogger)
		consoleLogger.Logger.SetFlags(0)
	}
	for i, logger := range metaLog.loggers {
		var newLogger *MetaLogger
		if logger.SplitLoggerFunc != nil {
			newLogger = logger.SplitLoggerFunc(logger)
		} else {
			newLogger = defaultSplitLoggerFunc(logger)
		}
		if newLogger != nil {
			metaLog.loggers[i] = newLogger
		}
	}
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
	for i, logger := range metaLog.loggers {
		for _, loggerLever := range logger.Level {
			if level == loggerLever || level == meta {
				newLogger := defaultSplitLoggerFunc(logger)
				if newLogger != nil {
					metaLog.loggers[i] = newLogger
					logger = newLogger
				}
				logger.LoggerPrint(logger, log...) // 调用自定义输出函数
				break
			}
		}
	}
}

// 日志切割
func defaultSplitLoggerFunc(OldLogger *MetaLogger) *MetaLogger {
	var err error
	if OldLogger.Path == "" {
		return nil
	}
	fileInfo, err := os.Stat(OldLogger.Path)
	if err != nil {
		splitLogStatErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.split_log_stat_error",
			TemplateData: map[string]interface{}{
				"name": OldLogger.Name,
				"err":  err,
			},
		})
		panic(splitLogStatErrorMsg)
		return nil
	}
	fileSize := fileInfo.Size()
	nowTime := time.Now().In(Settings.GetLocation())
	isSplit := false
	if OldLogger.SPLIT_TIME != "" {
		if OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME) != nowTime.Format(OldLogger.SPLIT_TIME) {
			isSplit = true
		}
	}
	if OldLogger.SPLIT_SIZE != 0 {
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
		oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", fileName, OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME), fileExt))
		_, err = os.Stat(oldInfoFile)
		for idx := 1; err == nil; idx++ {
			oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", fileName, OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME), idx, fileExt))
			_, err = os.Stat(oldInfoFile)
		}

		OldLogger.File.Close()
		err = os.Rename(OldLogger.Path, oldInfoFile)
		if err != nil {
			splitLogReNameErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "log.split_log_rename_error",
				TemplateData: map[string]interface{}{
					"name": OldLogger.Name,
					"err":  err,
				},
			})
			panic(splitLogReNameErrorMsg)
		}
		// 重新初始化
		return OldLogger.NewLoggerFunc()
	}
	return nil
}
