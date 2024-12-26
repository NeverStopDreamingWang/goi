package goi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type Level string

// 日志等级
const (
	meta    Level = "" // 框架日志
	DEBUG   Level = "DEBUG"
	INFO    Level = "INFO"
	WARNING Level = "WARNING"
	ERROR   Level = "ERROR"
)

// 日志
type MetaLogger struct {
	Name            string                                                    // 日志名称
	Path            string                                                    // 日志输出路径
	Level           []Level                                                   // 日志等级
	Logger          *log.Logger                                               // 日志对象
	File            *os.File                                                  // 日志文件对象
	LoggerPrint     func(logger *MetaLogger, level Level, log ...interface{}) // 自定义日志输出
	CreateTime      time.Time                                                 // 日志创建时间
	SPLIT_SIZE      int64                                                     // 日志切割大小，默认为 1G 1024 * 1024 * 1024
	SPLIT_TIME      string                                                    // 日志切割大小，默认按天切割
	GetFileFunc     func(filePath string) (*os.File, error)                   // 创建文件对象方法
	SplitLoggerFunc func(OldLogger *MetaLogger) error                         // 自定义日志切割：符合切割条件时，传入日志对象，关闭旧文件对象，Logger.SetOutput 设置新的输出对象
	lock            sync.Mutex
}

type metaLog struct {
	DEBUG          bool          // 默认为 true
	console_logger *MetaLogger   // DEBUG 日志
	loggers        []*MetaLogger // 日志列表
}

func newLog() *metaLog {
	return &metaLog{
		DEBUG:   true,            // 开发模式，开启后自动输出到控制台
		loggers: []*MetaLogger{}, // 日志
	}
}

// 注册日志
func (metaLog *metaLog) RegisterLogger(logger *MetaLogger) error {
	if logger.Path == "" {
		invalidPathMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_path",
		})
		return errors.New(invalidPathMsg)
	}
	if len(logger.Level) == 0 {
		invalidLevelMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_level",
		})
		return errors.New(invalidLevelMsg)
	}
	if logger.File == nil {
		invalidFileMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_file",
		})
		return errors.New(invalidFileMsg)
	}
	if logger.Logger == nil {
		invalidObjectMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_object",
		})
		return errors.New(invalidObjectMsg)
	}
	if logger.GetFileFunc == nil {
		invalidNilMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.invalid_nil",
			TemplateData: map[string]interface{}{
				"name": "GetFileFunc",
			},
		})
		return errors.New(invalidNilMsg)
	}
	err := checkSplitLoggerFunc(logger)
	if err != nil {
		return err
	}

	metaLog.loggers = append(metaLog.loggers, logger)
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
func (metaLog *metaLog) Log(level Level, logs ...interface{}) {
	// DEBUG
	if metaLog.DEBUG == true {
		if metaLog.console_logger == nil {
			// 初始化控制台日志
			metaLog.console_logger = &MetaLogger{
				Name:            "Debug-console",
				Path:            "",
				Level:           []Level{DEBUG, INFO, WARNING, ERROR},
				Logger:          log.New(os.Stdout, "", 0),
				File:            nil,
				CreateTime:      time.Now().In(Settings.GetLocation()),
				SPLIT_SIZE:      0,
				SPLIT_TIME:      "",
				GetFileFunc:     nil,
				SplitLoggerFunc: nil,
				LoggerPrint:     defaultLoggerPrint,
			}
			metaLog.console_logger.Logger.SetFlags(0)
		}
		metaLog.console_logger.LoggerPrint(metaLog.console_logger, level, logs...)
	} else if metaLog.console_logger != nil {
		metaLog.console_logger = nil
	}

	// 输出到所有符合的日志中
	for _, logger := range metaLog.loggers {
		logger.lock.Lock()
		for _, loggerLever := range logger.Level {
			if level == loggerLever || level == meta {
				if logger.LoggerPrint != nil {
					logger.LoggerPrint(logger, level, logs...) // 调用自定义输出
				} else {
					defaultLoggerPrint(logger, level, logs...) // 调用默认输出
				}
				break
			}
		}
		logger.lock.Unlock()
	}
}

// 日志初始化
func (metaLog *metaLog) splitLogger(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done() // 确保 goroutine 完成时减少 waitGroup 计数
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var err error
			for _, logger := range metaLog.loggers {
				err = checkSplitLoggerFunc(logger)
				if err != nil {
					panic(err)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// 默认日志输出格式
func defaultLoggerPrint(logger *MetaLogger, level Level, logs ...interface{}) {
	timeStr := fmt.Sprintf("[%v]", time.Now().In(Settings.GetLocation()).Format("2006-01-02 15:04:05"))
	if level != "" {
		timeStr += fmt.Sprintf(" %v", level)
	}
	logs = append([]interface{}{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// 检查日志切割
func checkSplitLoggerFunc(metaLog *MetaLogger) error {
	var err error
	if metaLog.Path == "" {
		return nil
	}
	fileInfo, err := os.Stat(metaLog.Path)
	if err != nil {
		splitLogStatErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.split_log_stat_error",
			TemplateData: map[string]interface{}{
				"name": metaLog.Name,
				"err":  err,
			},
		})
		panic(splitLogStatErrorMsg)
		return nil
	}
	fileSize := fileInfo.Size()
	nowTime := time.Now().In(Settings.GetLocation())
	isSplit := false
	if metaLog.SPLIT_TIME != "" {
		if metaLog.CreateTime.Format(metaLog.SPLIT_TIME) != nowTime.Format(metaLog.SPLIT_TIME) {
			isSplit = true
		}
	}
	if metaLog.SPLIT_SIZE != 0 {
		if metaLog.SPLIT_SIZE <= fileSize {
			isSplit = true
		}
	}
	if isSplit == false {
		return nil
	}

	metaLog.lock.Lock()
	defer metaLog.lock.Unlock()

	if metaLog.SplitLoggerFunc != nil {
		return metaLog.SplitLoggerFunc(metaLog)
	}

	var (
		fileName string
		baseName string
		fileExt  string
		fileDir  string
	)

	fileName = filepath.Base(metaLog.Path)
	fileDir = filepath.Dir(metaLog.Path)
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		if fileName[i] == '.' {
			baseName = fileName[:i]
			fileExt = fileName[i:]
			break
		}
	}
	// 自动加 _n
	oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", baseName, metaLog.CreateTime.Format(metaLog.SPLIT_TIME), fileExt))
	_, err = os.Stat(oldInfoFile)
	for idx := 1; err == nil || os.IsNotExist(err) == false; idx++ {
		oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", baseName, metaLog.CreateTime.Format(metaLog.SPLIT_TIME), idx, fileExt))
		_, err = os.Stat(oldInfoFile)
	}

	_ = metaLog.File.Close()
	err = os.Rename(metaLog.Path, oldInfoFile)
	if err != nil {
		splitLogReNameErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "log.split_log_rename_error",
			TemplateData: map[string]interface{}{
				"name": metaLog.Name,
				"err":  err,
			},
		})
		return errors.New(splitLogReNameErrorMsg)
	}

	// 初始化新的文件对象
	metaLog.File, err = metaLog.GetFileFunc(metaLog.Path)
	if err != nil {
		return err
	}
	metaLog.Logger.SetOutput(metaLog.File)
	return nil
}
