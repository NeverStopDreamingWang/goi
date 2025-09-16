package goi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// Http 服务
var Settings = newSettings()
var Cache = newCache()
var Log = newLog()
var Validator = newValidator()

var serverChan = make(chan os.Signal, 1)

// 关闭服务处理程序
type ShutdownHandler func(engine *Engine) error

// 关闭服务回调
type ShutdownCallback struct {
	name    string
	handler ShutdownHandler
}

// Engine 实现 ServeHTTP 接口
type Engine struct {
	startTime               *time.Time         // 启动时间
	server                  http.Server        // net/http 服务
	Router                  *MetaRouter        // 路由
	MiddleWare              []MiddleWare       // 中间件
	Settings                *metaSettings      // 设置
	Cache                   *metaCache         // 缓存
	Log                     *metaLog           // 日志
	Validator               *metaValidator     // 验证器
	ShutdownCallbackHandler []ShutdownCallback // 用户定义的关闭服务回调处理程序
	ctx                     context.Context
	cancel                  context.CancelFunc
	waitGroup               *sync.WaitGroup
}

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		startTime:  nil,
		server:     http.Server{},
		Router:     newRouter(),
		MiddleWare: make([]MiddleWare, 0),
		Settings:   Settings,
		Cache:      Cache,
		Log:        Log,
		Validator:  Validator,
		ctx:        ctx,
		cancel:     cancel,
		waitGroup:  &sync.WaitGroup{},
	}
}

// 启动 http 服务
func (engine *Engine) RunServer() {
	var err error
	startTime := GetTime()
	engine.startTime = &startTime

	// 日志切割
	go engine.Log.splitLogger(engine.ctx, engine.waitGroup)

	startMsg := i18n.T("server.start")
	engine.Log.Log(meta, startMsg, fmt.Sprintf("DEBUG: %v", engine.Log.DEBUG))

	startTimeMsg := i18n.T("server.start_time", map[string]interface{}{
		"start_time": engine.startTime.Format(time.DateTime),
	})
	engine.Log.Log(meta, startTimeMsg)

	if engine.Log.DEBUG == true {
		goiVersionMsg := i18n.T("server.goi_version", map[string]interface{}{
			"version": version,
		})
		engine.Log.Log(meta, goiVersionMsg)
	}

	if engine.Settings.GetTimeZone() != "" {
		currentTimeZoneMsg := i18n.T("server.current_time_zone", map[string]interface{}{
			"time_zone": engine.Settings.GetTimeZone(),
		})
		engine.Log.Log(meta, currentTimeZoneMsg)
	}

	for _, logger := range engine.Log.loggers {
		logInfoMsg := i18n.T("server.log_info", map[string]interface{}{
			"name":       logger.Name,
			"split_size": FormatBytes(logger.SPLIT_SIZE),
			"split_time": logger.SPLIT_TIME,
		})
		engine.Log.Log(meta, logInfoMsg)
	}

	// 初始化缓存
	engine.Cache.initCache()
	if engine.Cache.EXPIRATION_POLICY == PERIODIC {
		go engine.Cache.cachePeriodicDeleteExpires(engine.ctx, engine.waitGroup)
	}

	// 注册关闭信号
	signal.Notify(serverChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		// 等待关闭信号
		sig, _ := <-serverChan
		switch sig {
		case os.Kill, os.Interrupt, syscall.SIGTERM:
			err = engine.StopServer()
			if err != nil {
				panic(err)
			}
			return
		default:
			invalidOperationMsg := i18n.T("server.invalid_operation", map[string]interface{}{
				"name": sig,
			})
			engine.Log.Log(meta, invalidOperationMsg)
			return
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

		listenAddressMsg := i18n.T("server.listen_address", map[string]interface{}{
			"bind_address": fmt.Sprintf("https://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK),
		})
		engine.Log.Log(meta, listenAddressMsg)
		if engine.Settings.BIND_DOMAIN != "" {
			listenDomainMsg := i18n.T("server.listen_address", map[string]interface{}{
				"bind_address": fmt.Sprintf("https://%v:%v [%v]", engine.Settings.BIND_DOMAIN, engine.Settings.PORT, engine.Settings.NET_WORK),
			})
			engine.Log.Log(meta, listenDomainMsg)
		}
		err = engine.server.ServeTLS(ln, engine.Settings.SSL.CERT_PATH, engine.Settings.SSL.KEY_PATH)
	} else {
		listenAddressMsg := i18n.T("server.listen_address", map[string]interface{}{
			"bind_address": fmt.Sprintf("http://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK),
		})
		engine.Log.Log(meta, listenAddressMsg)
		if engine.Settings.BIND_DOMAIN != "" {
			listenDomainMsg := i18n.T("server.listen_address", map[string]interface{}{
				"bind_address": fmt.Sprintf("http://%v:%v [%v]", engine.Settings.BIND_DOMAIN, engine.Settings.PORT, engine.Settings.NET_WORK),
			})
			engine.Log.Log(meta, listenDomainMsg)
		}
		err = engine.server.Serve(ln)
	}
	if err != nil && err != http.ErrServerClosed {
		engine.Log.Error(err)
	}
}

// 注册停止回调函数
func (engine *Engine) RegisterShutdownHandler(name string, shutdownHandler ShutdownHandler) {
	engine.ShutdownCallbackHandler = append(engine.ShutdownCallbackHandler, ShutdownCallback{
		name:    name,
		handler: shutdownHandler,
	})
}

// 停止 http 服务
func (engine *Engine) StopServer() error {
	var err error

	// 执行用户定义的回调函数
	for _, shutdownCallback := range engine.ShutdownCallbackHandler {
		shutdownHandlerMsg := i18n.T("server.shutdown_handler", map[string]interface{}{
			"name": shutdownCallback.name,
		})
		engine.Log.Log(meta, shutdownHandlerMsg)

		err = shutdownCallback.handler(engine)
		if err != nil {
			shutdownHandlerErrorMsg := i18n.T("server.shutdown_handler_error", map[string]interface{}{
				"err": err,
			})
			engine.Log.Log(meta, shutdownHandlerErrorMsg)
		}
	}

	if len(engine.Settings.DATABASES) != 0 {
		closeDatabaseMsg := i18n.T("server.close_database")
		engine.Log.Log(meta, closeDatabaseMsg)
	}
	for name, database := range engine.Settings.DATABASES {
		err = database.Close()
		if err != nil {
			closeDatabaseErrorMsg := i18n.T("server.close_database_error", map[string]interface{}{
				"engine": database.ENGINE,
				"name":   name,
				"err":    err,
			})
			engine.Log.Log(meta, closeDatabaseErrorMsg)
		}
	}

	// 取消所有 goroutine 任务
	engine.cancel()
	// 等待所有 goroutine 任务结束
	engine.waitGroup.Wait()

	stopTimeMsg := i18n.T("server.stop_time", map[string]interface{}{
		"stop_time": GetTime().Format(time.DateTime),
	})
	engine.Log.Log(meta, stopTimeMsg)
	runTimeMsg := i18n.T("server.run_time", map[string]interface{}{
		"run_time": engine.runTimeStr(),
	})
	engine.Log.Log(meta, runTimeMsg)
	stopMsg := i18n.T("server.stop")
	engine.Log.Log(meta, stopMsg)
	// 关闭服务器
	err = engine.server.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// 获取当前运行时间 返回时间间隔
func (engine *Engine) RunTime() time.Duration {
	// 获取当前时间并设置时区
	currentTime := GetTime()

	// 计算时间间隔
	return currentTime.Sub(*engine.startTime)
}

// 获取当前运行时间 返回字符串
func (engine *Engine) runTimeStr() string {
	// 获取时间间隔
	elapsed := engine.RunTime()
	seconds := int(elapsed.Seconds()) % 60
	minutes := int(elapsed.Minutes()) % 60
	hours := int(elapsed.Hours()) % 24
	days := int(elapsed.Hours() / 24)

	var elapsedStr string
	if days > 0 {
		dayMsg := i18n.T("day")
		elapsedStr += fmt.Sprintf("%d %v ", days, dayMsg)
	}
	if hours > 0 {
		hourMsg := i18n.T("hour")
		elapsedStr += fmt.Sprintf("%02d %v ", hours, hourMsg)
	}
	if minutes > 0 {
		minuteMsg := i18n.T("minute")
		elapsedStr += fmt.Sprintf("%02d %v ", minutes, minuteMsg)
	}
	if seconds > 0 {
		secondMsg := i18n.T("second")
		elapsedStr += fmt.Sprintf("%02d %v ", seconds, secondMsg)
	}
	return elapsedStr
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := GetTime()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	r = r.WithContext(ctx)

	// 初始化请求
	request := &Request{
		Object:     r,
		PathParams: make(Params),
		Params:     make(Params),
	}
	responseWriter := &ResponseWriter{ResponseWriter: w}

	defer engine.recovery(request, responseWriter, GetTime().Sub(requestID))

	// 处理请求：构建洋葱链后获取响应
	middlewareChain := engine.loadMiddleware(engine.getResponse)
	response := middlewareChain(request)
	_, err := response.write(request, responseWriter)
	if err != nil {
		panic(err)
	}
}

// getResponseFunc 定义“获取响应”的函数类型。
// 我们用该类型来承载“被中间件层层包装后的最终处理器”。
type getResponseFunc func(*Request) Response

// getResponse 为最内层处理器：只负责路由解析、获取视图、执行视图并封装为 Response。
func (engine *Engine) getResponse(request *Request) (response Response) {
	defer func() {
		// 异常捕获
		err := recover()
		if err != nil {
			result := engine.processExceptionByMiddleware(request, err)
			if result == nil {
				panic(err)
			}
			response = toResponse(result)
			return
		}
	}()

	// 路由解析
	viewSet, err := engine.Router.resolveRequest(request)
	if err != nil {
		return Response{Status: http.StatusNotFound, Data: err.Error()}
	}

	// 获取视图处理函数
	handlerFunc, err := viewSet.GetHandlerFunc(request.Object.Method)
	if handlerFunc == nil || err != nil {
		return Response{Status: http.StatusMethodNotAllowed, Data: err.Error()}
	}

	// 视图处理
	content := handlerFunc(request)
	return toResponse(content)
}

// 统一把结果转换为 Response
func toResponse(content interface{}) Response {
	switch value := content.(type) {
	case Response:
		return value
	case *Response:
		if value == nil {
			return Response{Status: http.StatusOK, Data: nil}
		}
		return *value
	default:
		return Response{Status: http.StatusOK, Data: value}
	}
}
