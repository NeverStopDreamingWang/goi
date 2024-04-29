package example

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/NeverStopDreamingWang/goi"
)

// 日志输出
func LogPrintln(logger *goi.MetaLogger, log ...interface{}) {
	timeStr := fmt.Sprintf("[%v]", time.Now().In(Server.Settings.LOCATION).Format("2006-01-02 15:04:05"))
	log = append([]interface{}{timeStr}, log...)
	logger.Logger.Println(log...)
}

// 默认日志
func newDefaultLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/server.log")
	defaultLog := &goi.MetaLogger{
		Name: "默认日志",
		Path: OutPath,
		Level: []goi.Level{ // 所有等级的日志
			goi.INFO,
			goi.WARNING,
			goi.ERROR,
		},
		Logger:        nil,
		File:          nil,
		CreateTime:    time.Now().In(Server.Settings.LOCATION),
		SPLIT_SIZE:    1024 * 5,     // 切割大小
		SPLIT_TIME:    "2006-01-02", // 切割日期，每天
		NewLoggerFunc: newDefaultLog,
		LoggerPrint:   LogPrintln,
	}

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[%v]日志错误: ", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 访问日志
func newAccessLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/access.log")
	defaultLog := &goi.MetaLogger{
		Name: "访问日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入正确的日志
			goi.INFO,
		},
		Logger:        nil,
		File:          nil,
		CreateTime:    time.Now().In(Server.Settings.LOCATION),
		SPLIT_SIZE:    1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:    "2006-01-02",    // 切割日期，每天
		NewLoggerFunc: newAccessLog,
		LoggerPrint:   LogPrintln,
	}

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[%v]日志错误: ", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}

// 错误日志
func newErrorLog() *goi.MetaLogger {
	var err error

	OutPath := path.Join(Server.Settings.BASE_DIR, "logs/error.log")
	defaultLog := &goi.MetaLogger{
		Name: "错误日志",
		Path: OutPath,
		Level: []goi.Level{ // 仅输入错误的日志
			goi.ERROR,
		},
		Logger:        nil,
		File:          nil,
		CreateTime:    time.Now().In(Server.Settings.LOCATION),
		SPLIT_SIZE:    1024 * 1024 * 5, // 切割大小
		SPLIT_TIME:    "2006-01-02",    // 切割日期，每天
		NewLoggerFunc: newErrorLog,
		LoggerPrint:   LogPrintln,
	}

	OutDir := path.Dir(OutPath) // 检查目录
	_, err = os.Stat(OutDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(OutDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建日志目录错误: ", err))
		}
	}
	defaultLog.File, err = os.OpenFile(OutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintln("初始化[%v]日志错误: ", defaultLog.Name, err))
	}
	defaultLog.Logger = log.New(defaultLog.File, "", 0)

	return defaultLog
}
