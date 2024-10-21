package example

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/NeverStopDreamingWang/goi"
)

// 日志输出
func LogPrintln(logger *goi.MetaLogger, level goi.Level, logs ...interface{}) {
	timeStr := fmt.Sprintf("[%v]", time.Now().In(Server.Settings.GetLocation()).Format("2006-01-02 15:04:05"))
	if level != "" {
		timeStr += fmt.Sprintf(" %v", level)
	}
	logs = append([]interface{}{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// 默认日志
func newDefaultLog() *goi.MetaLogger {
	var err error

	OutPath := filepath.Join(Server.Settings.BASE_DIR, "logs", "server.log")
	OutDir := filepath.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: %v", err))
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
		File:            nil,
		Logger:          nil,
		LoggerPrint:     LogPrintln, // 日志输出格式
		CreateTime:      time.Now().In(Server.Settings.GetLocation()),
		SPLIT_SIZE:      1024 * 5,      // 切割大小
		SPLIT_TIME:      "2006-01-02",  // 切割日期，每天
		NewLoggerFunc:   newDefaultLog, // 初始化日志，同时用于自动切割日志后初始化新日志
		SplitLoggerFunc: nil,           // 自定义日志切割：传入旧的日志对象，返回新日志对象
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("初始化[%v]日志错误: %v", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 访问日志
func newAccessLog() *goi.MetaLogger {
	var err error

	OutPath := filepath.Join(Server.Settings.BASE_DIR, "logs", "access.log")
	OutDir := filepath.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: %v", err))
		}
	}

	defaultLog := &goi.MetaLogger{
		Name: "访问日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入正确的日志
			goi.INFO,
		},
		File:            nil,
		Logger:          nil,
		LoggerPrint:     LogPrintln, // 日志输出格式
		CreateTime:      time.Now().In(Server.Settings.GetLocation()),
		SPLIT_SIZE:      1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:      "2006-01-02",    // 切割日期，每天
		NewLoggerFunc:   newAccessLog,    // 初始化日志，同时用于自动切割日志后初始化新日志
		SplitLoggerFunc: nil,             // 自定义日志切割：传入旧的日志对象，返回新日志对象
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("初始化[%v]日志错误: %v", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 错误日志
func newErrorLog() *goi.MetaLogger {
	var err error

	OutPath := filepath.Join(Server.Settings.BASE_DIR, "logs", "error.log")
	OutDir := filepath.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: %v", err))
		}
	}

	defaultLog := &goi.MetaLogger{
		Name: "错误日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入错误的日志
			goi.ERROR,
		},
		File:            nil,
		Logger:          nil,
		LoggerPrint:     LogPrintln, // 日志输出格式
		CreateTime:      time.Now().In(Server.Settings.GetLocation()),
		SPLIT_SIZE:      1024 * 1024 * 5,   // 切割大小
		SPLIT_TIME:      "2006-01-02",      // 切割日期，每天
		NewLoggerFunc:   newErrorLog,       // 初始化日志，同时用于自动切割日志后初始化新日志
		SplitLoggerFunc: mySplitLoggerFunc, // 自定义日志切割：传入旧的日志对象，返回新日志对象，nil 根据 SPLIT_SIZE,SPLIT_TIME 自动切割
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("初始化[%v]日志错误: %v", defaultLog.Name, err))
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
		panic(fmt.Sprintf("日志切割-[%v]获取日志文件信息错误: %v", OldLogger.Name, err))
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
		oldInfoFile := path.Join(fileDir, fmt.Sprintf("%v_%v%v", fileName, OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME), fileExt))
		_, err = os.Stat(oldInfoFile)
		for idx := 1; err == nil; idx++ {
			oldInfoFile = path.Join(fileDir, fmt.Sprintf("%v_%v(%v)%v", fileName, OldLogger.CreateTime.Format(OldLogger.SPLIT_TIME), idx, fileExt))
			_, err = os.Stat(oldInfoFile)
		}

		_ = OldLogger.File.Close()
		err = os.Rename(OldLogger.Path, oldInfoFile)
		if err != nil {
			panic(fmt.Sprintf("日志切割-[%v]日志重命名错误: %v", OldLogger.Name, err))
		}
		// 重新初始化
		return OldLogger.NewLoggerFunc()
	}
	return nil
}
