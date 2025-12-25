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

// NewLogger 创建新的日志器
//
// 参数:
//   - path string: 日志输出路径，例如 "logs/server.log"
//
// 返回:
//   - *Logger: 日志器实例
func NewLogger(path string) *Logger {
	if path == "" {
		return nil
	}
	pathDir := filepath.Dir(path) // 检查目录
	_, err := os.Stat(pathDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(pathDir, 0644)
		if err != nil {
			panic(err)
		}
	}
	logger := &Logger{
		Name:            "默认日志",
		Path:            path,
		File:            nil,
		Logger:          nil,
		SplitSize:       1024 * 1024 * 10, // 切割大小
		SplitTime:       "2006-01-02",     // 切割日期，每天
		CreateTime:      GetTime(),
		PrintFunc:       nil, // 自定义日志输出格式
		GetFileFunc:     nil, // 创建文件对象方法
		SplitLoggerFunc: nil, // 自定义日志切割：符合切割条件时，传入日志对象，返回新的文件对象
	}
	logger.File, err = logger.GetFile()
	if err != nil {
		panic(err)
	}
	logger.Logger = log.New(logger.File, "", 0)
	return logger
}

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

// Logger 日志管理器
type Logger struct {
	Name          string        // 日志名称
	Path          string        // 日志输出路径
	File          *os.File      // 日志文件对象
	Logger        *log.Logger   // 日志对象
	SplitSize     int64         // 日志切割大小，默认为 1G 1024 * 1024 * 1024
	SplitTime     string        // 日志切割大小，默认按天切割
	SplitInterval time.Duration // 日志切割检查间隔
	CreateTime    time.Time     // 日志文件创建时间，每次日志切割时需要重置
	lock          sync.Mutex    // 互斥锁
	// 自定义方法
	PrintFunc       func(logger *Logger, level Level, logs ...any) // 自定义日志输出格式
	GetFileFunc     func(filePath string) (*os.File, error)        // 创建文件对象方法
	SplitLoggerFunc func(metaLogger *Logger) (*os.File, error)     // 自定义日志切割：符合切割条件时，传入日志对象，返回新的文件对象

}

// DebugF 记录调试级别日志
//
// 参数:
//   - format string: 日志格式
//   - logs ...any: 日志内容
func (self *Logger) DebugF(format string, logs ...any) {
	self.Debug(fmt.Sprintf(format, logs...))
}

// Debug 记录调试级别日志
//
// 参数:
//   - logs ...any: 日志内容
func (self *Logger) Debug(logs ...any) {
	self.Log(DEBUG, logs...)
}

// InfoF 记录信息级别日志
//
// 参数:
//   - format string: 日志格式
//   - logs ...any: 日志内容
func (self *Logger) InfoF(format string, logs ...any) {
	self.Info(fmt.Sprintf(format, logs...))
}

// Info 记录信息级别日志
//
// 参数:
//   - logs ...any: 日志内容
func (self *Logger) Info(logs ...any) {
	self.Log(INFO, logs...)
}

// WarningF 记录警告级别日志
//
// 参数:
//   - format string: 日志格式
//   - logs ...any: 日志内容
func (self *Logger) WarningF(format string, logs ...any) {
	self.Warning(fmt.Sprintf(format, logs...))
}

// Warning 记录警告级别日志
//
// 参数:
//   - logs ...any: 日志内容
func (self *Logger) Warning(logs ...any) {
	self.Log(WARNING, logs...)
}

// ErrorF 记录错误级别日志
//
// 参数:
//   - format string: 日志格式
//   - logs ...any: 日志内容
func (self *Logger) ErrorF(format string, logs ...any) {
	self.Error(fmt.Sprintf(format, logs...))
}

// Error 记录错误级别日志
//
// 参数:
//   - logs ...any: 日志内容
func (self *Logger) Error(logs ...any) {
	self.Log(ERROR, logs...)
}

// LogF 记录指定级别的日志
//
// 参数:
//   - level Level: 日志级别
//   - format string: 日志格式
//   - logs ...any: 日志内容
func (self *Logger) LogF(level Level, format string, logs ...any) {
	self.Log(level, fmt.Sprintf(format, logs...))
}

// Log 记录指定级别的日志
//
// 参数:
//   - level Level: 日志级别
//   - logs ...any: 日志内容
func (self *Logger) Log(level Level, logs ...any) {
	// DEBUG 模式下所有日志都输出到控制台
	if Settings.DEBUG == true {
		// 初始化控制台日志
		consoleLogger = getConsoleLogger()
		consoleLogger.lock.Lock()
		consoleLogger.Print(consoleLogger, level, logs...)
		consoleLogger.lock.Unlock()
	} else {
		consoleLogger = nil
		// DEBUG 模式关闭时，跳过 DEBUG 级别日志
		if level == DEBUG {
			return
		}
	}

	self.lock.Lock()
	self.Print(self, level, logs...)
	self.lock.Unlock()
}

// Print 默认日志打印格式化函数
//
// 参数:
//   - logger *Logger: 日志器
//   - level Level: 日志级别
//   - logs ...any: 日志内容
func (self *Logger) Print(logger *Logger, level Level, logs ...any) {
	if self.Logger == nil {
		invalidObjectMsg := i18n.T("log.invalid_object")
		panic(invalidObjectMsg)
	}

	// 自定义日志输出
	if self.PrintFunc != nil {
		self.PrintFunc(self, level, logs...)
		return
	}

	timeStr := fmt.Sprintf("[%v]", GetTime().Format(time.DateTime))
	if level != "" {
		timeStr += fmt.Sprintf(" %v", level)
	}
	logs = append([]any{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// CheckSplit 检查是否需要进行日志切割
//
// 返回:
//   - error: 检查过程中的错误信息
func (self *Logger) CheckSplit() error {
	if self.File == nil {
		invalidFileMsg := i18n.T("log.invalid_file")
		return errors.New(invalidFileMsg)
	}

	var err error
	nowTime := GetTime()
	isSplit := false
	if self.SplitTime != "" {
		if self.CreateTime.Format(self.SplitTime) != nowTime.Format(self.SplitTime) {
			isSplit = true
		}
	}
	if isSplit == false && self.SplitSize != 0 {
		fileInfo, err := self.File.Stat()
		if err != nil {
			splitLogStatErrorMsg := i18n.T("log.split_log_stat_error", map[string]any{
				"name": self.Name,
				"err":  err,
			})
			panic(splitLogStatErrorMsg)
			return nil
		}
		fileSize := fileInfo.Size()
		if self.SplitSize <= fileSize {
			isSplit = true
		}
	}
	if isSplit == false {
		return nil
	}

	self.lock.Lock()
	defer self.lock.Unlock()
	file, err := self.SplitLogger()
	if err != nil {
		return err
	}
	if file != nil {
		self.File = file                 // 更新日志文件对象
		self.Logger.SetOutput(self.File) // 设置新的输出文件对象
		self.CreateTime = nowTime        // 更新日志文件时间
	}
	return nil
}

// SplitLogger 默认日志切割实现
//
// 返回:
//   - *os.File: 新的日志文件对象
//   - error: 切割过程中的错误信息
func (self *Logger) SplitLogger() (*os.File, error) {
	if self.File == nil {
		invalidFileMsg := i18n.T("log.invalid_file")
		return nil, errors.New(invalidFileMsg)
	}

	// 关闭旧的文件对象
	_ = self.File.Close()

	// 自定义日志切割
	if self.SplitLoggerFunc != nil {
		return self.SplitLoggerFunc(self)
	}

	var (
		fileName string
		baseName string
		fileExt  string
		fileDir  string
		err      error
	)

	fileName = filepath.Base(self.Path)
	fileDir = filepath.Dir(self.Path)
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		if fileName[i] == '.' {
			baseName = fileName[:i]
			fileExt = fileName[i:]
			break
		}
	}
	// 自动加 _n
	oldFilePath := filepath.Join(fileDir, fmt.Sprintf("%v_%v%v", baseName, self.CreateTime.Format(self.SplitTime), fileExt))
	_, err = os.Stat(oldFilePath)
	for idx := 1; err == nil || os.IsNotExist(err) == false; idx++ {
		oldFilePath = filepath.Join(fileDir, fmt.Sprintf("%v_%v_%v%v", baseName, self.CreateTime.Format(self.SplitTime), idx, fileExt))
		_, err = os.Stat(oldFilePath)
	}
	err = os.Rename(self.Path, oldFilePath)
	if err != nil {
		splitLogReNameErrorMsg := i18n.T("log.split_log_rename_error", map[string]any{
			"name": self.Name,
			"err":  err,
		})
		return nil, errors.New(splitLogReNameErrorMsg)
	}
	// 初始化新的文件对象
	return self.GetFile()
}

// GetFile 默认文件创建函数
//
// 返回:
//   - *os.File: 文件对象
//   - error: 创建过程中的错误信息
func (self *Logger) GetFile() (*os.File, error) {
	if self.Path == "" {
		invalidPathMsg := i18n.T("log.invalid_path")
		return nil, errors.New(invalidPathMsg)
	}

	// 自定义文件创建
	if self.GetFileFunc != nil {
		return self.GetFileFunc(self.Path)
	}

	var file *os.File
	var err error

	_, err = os.Stat(self.Path)
	if os.IsNotExist(err) { // 不存在则创建
		file, err = os.OpenFile(self.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// 文件存在，打开文件
		file, err = os.OpenFile(self.Path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

// StartupName 启动任务名称
//
// 返回:
//   - string: 启动任务名称
func (self *Logger) StartupName() string {
	return fmt.Sprintf("日志切割-%v", self.Name)
}

// OnStartup 日志切割管理协程
//
// 参数:
//   - ctx context.Context: 上下文对象，用于控制协程退出
//   - wg *sync.WaitGroup: 等待组，用于同步协程
func (self *Logger) OnStartup(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done() // 确保 goroutine 完成时减少 waitGroup 计数

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := self.CheckSplit()
			if err != nil {
				consoleLogger.Error(err)
				panic(err)
			}
		}
	}
}

var consoleLogger *Logger

// 默认日志
func getConsoleLogger() *Logger {
	if consoleLogger != nil {
		return consoleLogger
	}
	// 初始化控制台日志
	consoleLogger = &Logger{
		Name:            "Debug-console",
		Path:            "",
		Logger:          log.New(os.Stdout, "", 0),
		File:            nil,
		SplitSize:       0,
		SplitTime:       "",
		CreateTime:      GetTime(),
		PrintFunc:       nil,
		GetFileFunc:     nil,
		SplitLoggerFunc: nil,
	}
	consoleLogger.Logger.SetFlags(0)
	return consoleLogger
}
