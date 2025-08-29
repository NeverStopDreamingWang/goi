package example

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/NeverStopDreamingWang/goi"
)

// 日志输出
func LogPrintln(logger *goi.MetaLogger, level goi.Level, logs ...interface{}) {
	timeStr := fmt.Sprintf("[%v]", goi.GetTime().Format(time.DateTime))
	if level != "" {
		timeStr += fmt.Sprintf(" %v", level)
	}
	logs = append([]interface{}{timeStr}, logs...)
	logger.Logger.Println(logs...)
}

// 获取 *os.File 对象
func getFileFunc(filePath string) (*os.File, error) {
	var file *os.File
	var err error

	_, err = os.Stat(filePath)
	if os.IsNotExist(err) { // 不存在则创建
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
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
		CreateTime:      goi.GetTime(),
		SPLIT_SIZE:      1024 * 1024 * 10, // 切割大小
		SPLIT_TIME:      "2006-01-02",     // 切割日期，每天
		GetFileFunc:     getFileFunc,      // 创建文件对象方法
		SplitLoggerFunc: nil,              // 自定义日志切割：符合切割条件时，传入日志对象，返回新的文件对象
	}
	defaultLog.File, err = getFileFunc(defaultLog.Path)
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

	accessLog := &goi.MetaLogger{
		Name: "访问日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入正确的日志
			goi.INFO,
		},
		File:            nil,
		Logger:          nil,
		LoggerPrint:     LogPrintln,
		CreateTime:      goi.GetTime(),
		SPLIT_SIZE:      1024 * 1024 * 10,
		SPLIT_TIME:      "2006-01-02",
		GetFileFunc:     getFileFunc,
		SplitLoggerFunc: nil,
	}
	accessLog.File, err = getFileFunc(accessLog.Path)
	if err != nil {
		panic(fmt.Sprintf("初始化[%v]日志错误: %v", accessLog.Name, err))
	}
	accessLog.Logger = log.New(accessLog.File, "", 0)
	return accessLog
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
	errorLog := &goi.MetaLogger{
		Name: "错误日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入错误的日志
			goi.ERROR,
		},
		File:            nil,
		Logger:          nil,
		LoggerPrint:     LogPrintln,
		CreateTime:      goi.GetTime(),
		SPLIT_SIZE:      1024 * 1024 * 10,
		SPLIT_TIME:      "2006-01-02",
		GetFileFunc:     getFileFunc,
		SplitLoggerFunc: mySplitLoggerFunc,
	}
	errorLog.File, err = getFileFunc(errorLog.Path)
	if err != nil {
		panic(fmt.Sprintf("初始化[%v]日志错误: %v", errorLog.Name, err))
	}
	errorLog.Logger = log.New(errorLog.File, "", 0)
	return errorLog
}

// 自定义日志切割
func mySplitLoggerFunc(metaLogger *goi.MetaLogger) (*os.File, error) {
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
		return nil, errors.New(fmt.Sprintf("日志切割-[%v]日志重命名错误: %v", metaLogger.Name, err))
	}
	// 初始化新的文件对象
	return getFileFunc(metaLogger.Path)
}
