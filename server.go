package goi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

// Http 服务
var version = "1.3.1"
var engine *Engine
var Settings *metaSettings
var Cache *metaCache
var Log *metaLog

var exitChan = make(chan os.Signal, 1)

// 版本
func Version() string {
	return version
}

// HandlerFunc 定义 goi 使用的请求处理程序
type HandlerFunc func(*Request) interface{}

// type HandlerFunc func(*http.Request)

// Engine 实现 ServeHTTP 接口
type Engine struct {
	startTime   *time.Time
	server      http.Server
	Router      *metaRouter
	MiddleWares *metaMiddleWares
	Settings    *metaSettings
	Cache       *metaCache
	Log         *metaLog
}

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	Settings = newSettings()
	Cache = newCache()
	Log = newLog()
	engine = &Engine{
		startTime:   nil,
		server:      http.Server{},
		Router:      newRouter(),
		MiddleWares: newMiddleWares(),
		Settings:    Settings,
		Cache:       Cache,
		Log:         Log,
	}
	return engine
}

// 启动 http 服务
func (engine *Engine) RunServer() {
	var err error
	
	// 初始化时区
	location, err := time.LoadLocation(engine.Settings.TIME_ZONE)
	if err != nil {
		panic(fmt.Sprintf("初始化时区错误: %v\n", err))
	}
	engine.Settings.LOCATION = location
	
	// 初始化日志
	engine.Log.InitLogger()
	
	engine.Log.Log(meta, "---start---")
	engine.Log.Log(meta, fmt.Sprintf("启动时间: %s", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05")))
	engine.Log.Log(meta, fmt.Sprintf("goi 版本: %v", version))
	
	engine.Log.Log(meta, fmt.Sprintf("DEBUG: %v", engine.Log.DEBUG))
	for _, logger := range engine.Log.LOGGERS {
		log := fmt.Sprintf("- [%v]", logger.Name)
		if logger.SPLIT_SIZE != 0 {
			log += fmt.Sprintf(" 切割大小: %v", formatBytes(logger.SPLIT_SIZE))
		}
		if logger.SPLIT_TIME != "" {
			log += fmt.Sprintf(" 切割日期: %v", logger.SPLIT_TIME)
		}
		engine.Log.Log(meta, log)
	}
	
	engine.Log.Log(meta, fmt.Sprintf("当前时区: %v", engine.Settings.TIME_ZONE))
	
	// 初始化缓存
	engine.Cache.initCache()
	
	startTime := time.Now().In(Settings.LOCATION)
	engine.startTime = &startTime
	
	// 注册关闭信号
	signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	
	go func() {
		// 等待关闭信号
		sig, _ := <-exitChan
		switch sig {
		case os.Kill, os.Interrupt:
			err = engine.StopServer()
			engine.Log.Error(err)
		default:
			engine.Log.Log(meta, "无效操作！")
		}
	}()
	
	engine.server = http.Server{
		Addr:    fmt.Sprintf("%v:%v", engine.Settings.BIND_ADDRESS, engine.Settings.PORT),
		Handler: engine,
	}
	ln, err := net.Listen(engine.Settings.NET_WORK, engine.server.Addr)
	if err != nil {
		engine.Log.Error(err)
	}
	if engine.Settings.SSL.STATUS == true {
		certificate, err := os.ReadFile(engine.Settings.SSL.CERT_PATH)
		if err != nil {
			engine.Log.Error(err)
		}
		key, err := os.ReadFile(engine.Settings.SSL.KEY_PATH)
		if err != nil {
			engine.Log.Error(err)
		}
		cert, err := tls.X509KeyPair(certificate, key)
		if err != nil {
			engine.Log.Error(err)
		}
		engine.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		
		engine.Log.Log(meta, fmt.Sprintf("监听地址: https://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK))
		if engine.Settings.Domain != "" {
			engine.Log.Log(meta, fmt.Sprintf("监听地址: https://%v:%v [%v]", engine.Settings.Domain, engine.Settings.PORT, engine.Settings.NET_WORK))
		}
		err = engine.server.ServeTLS(ln, engine.Settings.SSL.CERT_PATH, engine.Settings.SSL.KEY_PATH)
	} else {
		engine.Log.Log(meta, fmt.Sprintf("监听地址: http://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK))
		if engine.Settings.Domain != "" {
			engine.Log.Log(meta, fmt.Sprintf("监听地址: http://%v:%v [%v]", engine.Settings.Domain, engine.Settings.PORT, engine.Settings.NET_WORK))
		}
		err = engine.server.Serve(ln)
	}
	if err != nil && err != http.ErrServerClosed {
		engine.Log.Error(err)
	}
}

// 停止 http 服务
func (engine *Engine) StopServer() error {
	engine.Log.Log(meta, "服务已停止！")
	engine.Log.Log(meta, fmt.Sprintf("停止时间: %s", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05")))
	engine.Log.Log(meta, fmt.Sprintf("共运行: %v", engine.RunTimeStr()))
	engine.Log.Log(meta, "---end---")
	// 关闭服务器
	err := engine.server.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// 获取当前运行时间 返回时间间隔
func (engine *Engine) RunTime() time.Duration {
	// 获取当前时间并设置时区
	currentTime := time.Now().In(Settings.LOCATION)
	
	// 计算时间间隔
	return currentTime.Sub(*engine.startTime)
}

// 获取当前运行时间 返回字符串
func (engine *Engine) RunTimeStr() string {
	// 获取时间间隔
	elapsed := engine.RunTime()
	seconds := int(elapsed.Seconds()) % 60
	minutes := int(elapsed.Minutes()) % 60
	hours := int(elapsed.Hours()) % 24
	days := int(elapsed.Hours() / 24)
	
	elapsedStr := fmt.Sprintf("%02d秒", seconds)
	if minutes > 0 {
		elapsedStr = fmt.Sprintf("%02d分", minutes) + elapsedStr
	}
	if hours > 0 {
		elapsedStr = fmt.Sprintf("%02d时", hours) + elapsedStr
	}
	if days > 0 {
		elapsedStr = fmt.Sprintf("%d天", days) + elapsedStr
	}
	return elapsedStr
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestID := time.Now().In(engine.Settings.LOCATION)
	ctx := context.WithValue(request.Context(), "requestID", requestID)
	request = request.WithContext(ctx)
	
	var log string
	var err error
	// 初始化请求
	requestContext := &Request{
		Object:      request,
		Context:     request.Context(),
		PathParams:  make(metaValues),
		QueryParams: make(metaValues),
		BodyParams:  make(metaValues),
	}
	defer metaRecovery(requestContext, response) // 异常处理
	
	requestContext.parseRequestParams()
	
	// 处理 HTTP 请求
	StatusCode := http.StatusOK
	var ResponseData []byte
	var write interface{}
	
	responseData := engine.HandlerHTTP(requestContext, response)
	
	// 文件处理
	fileObject, isFile := responseData.(*os.File)
	if isFile {
		// 返回文件内容
		write, ResponseData = metaResponseStatic(fileObject, requestContext, response)
		log = fmt.Sprintf("- %v - %v %v %v byte status %v %v",
			request.Host,
			request.Method,
			request.URL.Path,
			write,
			request.Proto,
			StatusCode,
		)
		engine.Log.Info(log)
		return
	}
	
	// 判断 responseData 是否为指针类型，如果是，则解引用获取指向的值
	responseDataValue := reflect.ValueOf(responseData)
	if responseDataValue.Kind() == reflect.Ptr {
		responseData = responseDataValue.Elem().Interface()
	}
	
	// 响应处理
	responseObject, isResponse := responseData.(Response)
	if isResponse {
		StatusCode = responseObject.Status
		responseData = responseObject.Data
	}
	
	switch value := responseData.(type) {
	case nil:
		ResponseData, err = json.Marshal(value)
	case bool:
		ResponseData, err = json.Marshal(value)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		ResponseData, err = json.Marshal(value)
	case float32, float64:
		ResponseData, err = json.Marshal(value)
	case string:
		ResponseData = []byte(value)
	case []byte:
		ResponseData = responseData.([]byte)
	default:
		ResponseData, err = json.Marshal(value)
	}
	// 返回响应
	if err != nil {
		panic(fmt.Sprintf("响应 json 数据错误: %v\n", err))
	}
	// 返回响应
	response.WriteHeader(StatusCode)
	write, err = response.Write(ResponseData)
	if err != nil {
		panic(fmt.Sprintf("响应错误: %v\n", err))
	}
	log = fmt.Sprintf("- %v - %v %v %v byte status %v %v",
		request.Host,
		request.Method,
		request.URL.Path,
		write,
		request.Proto,
		StatusCode,
	)
	engine.Log.Info(log)
}

// 处理 HTTP 请求
func (engine *Engine) HandlerHTTP(request *Request, response http.ResponseWriter) (result interface{}) {
	// 处理请求前的中间件
	result = engine.MiddleWares.processRequest(request)
	if result != nil {
		return result
	}
	
	// 路由处理
	handlerFunc, err := engine.Router.routerHandlers(request)
	if err != "" {
		return Response{Status: http.StatusNotFound, Data: err}
	}
	
	// 视图前的中间件
	result = engine.MiddleWares.processView(request, handlerFunc)
	if result != nil {
		return result
	}
	// 视图处理
	viewResponse := handlerFunc(request)
	
	// 返回响应前的中间件
	result = engine.MiddleWares.processResponse(request, response)
	if result != nil {
		return result
	}
	return viewResponse
}
