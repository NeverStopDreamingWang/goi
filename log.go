package goi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// Level 日志等级类型
type Level string

// 日志等级
const (
	meta    Level = "" // 框架日志
	DEBUG   Level = "DEBUG"
	INFO    Level = "INFO"
	WARNING Level = "WARNING"
	ERROR   Level = "ERROR"
)

// MetaLogger 日志管理器
type MetaLogger struct {
	Name            string                                                    // 日志名称
	Path            string                                                    // 日志输出路径
	Level           []Level                                                   // 日志等级
	File            *os.File                                                  // 日志文件对象
	CreateTime      time.Time                                                 // 日志文件创建时间，每次日志切割时需要重置
	Logger          *log.Logger                                               // 日志对象
	LoggerPrint     func(logger *MetaLogger, level Level, log ...interface{}) // 自定义日志输出
	SPLIT_SIZE      int64                                                     // 日志切割大小，默认为 1G 1024 * 1024 * 1024
	SPLIT_TIME      string                                                    // 日志切割大小，默认按天切割
	GetFileFunc     func(filePath string) (*os.File, error)                   // 创建文件对象方法
	SplitLoggerFunc func(metaLogger *MetaLogger) (*os.File, error)            // 自定义日志切割：符合切割条件时，传入日志对象，返回新的文件对象
	lock            sync.Mutex                                                // 互斥锁
}

// metaLog 日志系统管理器
//
// 字段:
//   - DEBUG bool: 是否开启调试模式
//   - console_logger *MetaLogger: 控制台日志器
//   - loggers []*MetaLogger: 日志器列表
type metaLog struct {
	DEBUG          bool          // 开发模式，开启后自动输出到控制台
	console_logger *MetaLogger   // DEBUG 日志
	loggers        []*MetaLogger // 日志列表
}

// newLog 创建新的日志系统管理器
//
// 返回:
//   - *metaLog: 日志系统管理器实例
func newLog() *metaLog {
	return &metaLog{
		DEBUG:   true,            // 开发模式，开启后自动输出到控制台
		loggers: []*MetaLogger{}, // 日志
	}
}

// RegisterLogger 注册日志器
//
// 参数:
//   - logger *MetaLogger: 要注册的日志器
//
// 返回:
//   - error: 注册过程中的错误信息
func (metaLog *metaLog) RegisterLogger(logger *MetaLogger) error {
	if logger.Path == "" {
		invalidPathMsg := i18n.T("log.invalid_path")
		return errors.New(invalidPathMsg)
	}
	if len(logger.Level) == 0 {
		invalidLevelMsg := i18n.T("log.invalid_level")
		return errors.New(invalidLevelMsg)
	}
	if logger.File == nil {
		invalidFileMsg := i18n.T("log.invalid_file")
		return errors.New(invalidFileMsg)
	}
	if logger.Logger == nil {
		invalidObjectMsg := i18n.T("log.invalid_object")
		return errors.New(invalidObjectMsg)
	}
	// 检查是否满足切割条件
	err := checkSplitLoggerFunc(logger)
	if err != nil {
		return err
	}

	metaLog.loggers = append(metaLog.loggers, logger)
	return nil
}

// DebugF 记录调试级别日志
//
// 参数:
//   - format string: 日志格式
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) DebugF(format string, log ...interface{}) {
	metaLog.Debug(fmt.Sprintf(format, log...))
}

// Debug 记录调试级别日志
//
// 参数:
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) Debug(log ...interface{}) {
	metaLog.Log(DEBUG, log...)
}

// InfoF 记录信息级别日志
//
// 参数:
//   - format string: 日志格式
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) InfoF(format string, log ...interface{}) {
	metaLog.Info(fmt.Sprintf(format, log...))
}

// Info 记录信息级别日志
//
// 参数:
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) Info(log ...interface{}) {
	metaLog.Log(INFO, log...)
}

// WarningF 记录警告级别日志
//
// 参数:
//   - format string: 日志格式
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) WarningF(format string, log ...interface{}) {
	metaLog.Warning(fmt.Sprintf(format, log...))
}

// Warning 记录警告级别日志
//
// 参数:
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) Warning(log ...interface{}) {
	metaLog.Log(WARNING, log...)
}

// ErrorF 记录错误级别日志
//
// 参数:
//   - format string: 日志格式
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) ErrorF(format string, log ...interface{}) {
	metaLog.Error(fmt.Sprintf(format, log...))
}

// Error 记录错误级别日志
//
// 参数:
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) Error(log ...interface{}) {
	metaLog.Log(ERROR, log...)
}

// LogF 记录指定级别的日志
//
// 参数:
//   - level Level: 日志级别
//   - format string: 日志格式
//   - log ...interface{}: 日志内容
func (metaLog *metaLog) LogF(level Level, format string, log ...interface{}) {
	metaLog.Log(level, fmt.Sprintf(format, log...))
}

// Log 记录指定级别的日志
//
// 参数:
//   - level Level: 日志级别
//   - logs ...interface{}: 日志内容
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
				CreateTime:      GetTime(),
				SPLIT_SIZE:      0,
				SPLIT_TIME:      "",
				GetFileFunc:     nil,
				SplitLoggerFunc: nil,
				LoggerPrint:     defaultLoggerPrint,
			}
			metaLog.console_logger.Logger.SetFlags(0)
		}
		metaLog.console_logger.lock.Lock()
		metaLog.console_logger.LoggerPrint(metaLog.console_logger, level, logs...)
		metaLog.console_logger.lock.Unlock()
	} else if metaLog.console_logger != nil {
		metaLog.console_logger = nil
	}

	// 输出到所有符合的日志中
	for _, logger := range metaLog.loggers {
		if !isLevel(logger, level) {
			continue
		}
		logger.lock.Lock()
		if logger.LoggerPrint != nil {
			logger.LoggerPrint(logger, level, logs...) // 调用自定义输出
		} else {
			defaultLoggerPrint(logger, level, logs...) // 调用默认输出
		}
		logger.lock.Unlock()
	}
}

// splitLogger 日志切割管理协程
//
// 参数:
//   - ctx context.Context: 上下文对象，用于控制协程退出
//   - wg *sync.WaitGroup: 等待组，用于同步协程
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

// isLevel 是否为当前日志器的日志等级
func isLevel(logger *MetaLogger, level Level) bool {
	for _, loggerLever := range logger.Level {
		if level == loggerLever || level == meta {
			return true
		}
	}
	return false
}

// defaultLoggerPrint 默认日志打印格式化函数
//
// 参数:
//   - logger *MetaLogger: 日志器
//   - level Level: 日志级别
//   - logs ...interface{}: 日志内容
func defaultLoggerPrint(logger *MetaLogger, level Level, logs ...interface{}) {
	timeStr := fmt.Sprintf("[%v]", GetTime().Format(time.DateTime))
	if level != "" {
		timeStr += fmt.Sprintf(" %v", level)
	}
	logs = append([]interface{}{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// checkSplitLoggerFunc 检查是否需要进行日志切割
//
// 参数:
//   - metaLogger *MetaLogger: 日志器
//
// 返回:
//   - error: 检查过程中的错误信息
func checkSplitLoggerFunc(metaLogger *MetaLogger) error {
	var err error
	nowTime := GetTime()
	isSplit := false
	if metaLogger.SPLIT_TIME != "" {
		if metaLogger.CreateTime.Format(metaLogger.SPLIT_TIME) != nowTime.Format(metaLogger.SPLIT_TIME) {
			isSplit = true
		}
	}
	if isSplit == false && metaLogger.SPLIT_SIZE != 0 {
		if metaLogger.File == nil {
			return nil
		}
		fileInfo, err := metaLogger.File.Stat()
		if err != nil {
			splitLogStatErrorMsg := i18n.T("log.split_log_stat_error", map[string]interface{}{
				"name": metaLogger.Name,
				"err":  err,
			})
			panic(splitLogStatErrorMsg)
			return nil
		}
		fileSize := fileInfo.Size()
		if metaLogger.SPLIT_SIZE <= fileSize {
			isSplit = true
		}
	}
	if isSplit == false {
		return nil
	}

	metaLogger.lock.Lock()
	defer metaLogger.lock.Unlock()

	_ = metaLogger.File.Close() // 关闭旧的文件对象
	var file *os.File
	if metaLogger.SplitLoggerFunc != nil {
		file, err = metaLogger.SplitLoggerFunc(metaLogger)
		if err != nil {
			return err
		}
	} else {
		file, err = defaultSplitLoggerFunc(metaLogger)
		if err != nil {
			return err
		}
	}
	metaLogger.File = file                       // 更新日志文件对象
	metaLogger.Logger.SetOutput(metaLogger.File) // 设置新的输出文件对象
	metaLogger.CreateTime = nowTime              // 更新日志文件时间
	return nil
}

// defaultSplitLoggerFunc 默认日志切割实现
//
// 参数:
//   - metaLogger *MetaLogger: 日志器
//
// 返回:
//   - *os.File: 新的日志文件对象
//   - error: 切割过程中的错误信息
func defaultSplitLoggerFunc(metaLogger *MetaLogger) (*os.File, error) {
	var (
		fileName string
		baseName string
		fileExt  string
		fileDir  string
		err      error
	)

	fileName = filepath.Base(metaLogger.Path)
	fileDir = filepath.Dir(metaLogger.Path)
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		if fileName[i] == '.' {
			baseName = fileName[:i]
			fileExt = fileName[i:]
			break
		}
	}
	// 自动加 _n
	oldFilePath := filepath.Join(fileDir, fmt.Sprintf("%v_%v%v", baseName, metaLogger.CreateTime.Format(metaLogger.SPLIT_TIME), fileExt))
	_, err = os.Stat(oldFilePath)
	for idx := 1; err == nil || os.IsNotExist(err) == false; idx++ {
		oldFilePath = filepath.Join(fileDir, fmt.Sprintf("%v_%v_%v%v", baseName, metaLogger.CreateTime.Format(metaLogger.SPLIT_TIME), idx, fileExt))
		_, err = os.Stat(oldFilePath)
	}
	err = os.Rename(metaLogger.Path, oldFilePath)
	if err != nil {
		splitLogReNameErrorMsg := i18n.T("log.split_log_rename_error", map[string]interface{}{
			"name": metaLogger.Name,
			"err":  err,
		})
		return nil, errors.New(splitLogReNameErrorMsg)
	}

	if metaLogger.GetFileFunc != nil {
		return metaLogger.GetFileFunc(metaLogger.Path)
	}
	// 初始化新的文件对象
	return defaultGetFileFunc(metaLogger.Path)
}

// defaultGetFileFunc 默认文件创建函数
//
// 参数:
//   - filePath string: 文件路径
//
// 返回:
//   - *os.File: 文件对象
//   - error: 创建过程中的错误信息
func defaultGetFileFunc(filePath string) (*os.File, error) {
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
		file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}
