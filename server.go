package goi

import (
	"context"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Http 服务
var version = "v1.4.3"
var Settings *metaSettings
var Cache *metaCache
var Log *metaLog
var Validator *metaValidator

var serverChan = make(chan os.Signal, 1)

// 关闭服务处理程序
type ShutdownHandler func(engine *Engine) error

// 关闭服务回调
type ShutdownCallback struct {
	name    string
	handler ShutdownHandler
}

// 版本
func Version() string {
	return version
}

// Engine 实现 ServeHTTP 接口
type Engine struct {
	startTime               *time.Time         // 启动时间
	server                  http.Server        // net/http 服务
	Router                  *MetaRouter        // 路由
	MiddleWares             *metaMiddleWares   // 中间件
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
	Settings = newSettings()
	Cache = newCache()
	Log = newLog()
	Validator = newValidator()
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		startTime:   nil,
		server:      http.Server{},
		Router:      newRouter(),
		MiddleWares: newMiddleWares(),
		Settings:    Settings,
		Cache:       Cache,
		Log:         Log,
		Validator:   Validator,
		ctx:         ctx,
		cancel:      cancel,
		waitGroup:   &sync.WaitGroup{},
	}
}

// 启动 http 服务
func (engine *Engine) RunServer() {
	var err error
	startTime := GetTime()
	engine.startTime = &startTime

	// 日志切割
	go engine.Log.splitLogger(engine.ctx, engine.waitGroup)

	startMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.start",
	})
	engine.Log.Log(meta, startMsg)
	startTimeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.start_time",
		TemplateData: map[string]interface{}{
			"start_time": engine.startTime.Format("2006-01-02 15:04:05"),
		},
	})
	engine.Log.Log(meta, startTimeMsg)

	goiVersionMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.goi_version",
		TemplateData: map[string]interface{}{
			"version": version,
		},
	})
	engine.Log.Log(meta, goiVersionMsg)

	engine.Log.Log(meta, fmt.Sprintf("DEBUG: %v", engine.Log.DEBUG))

	if engine.Settings.GetTimeZone() != "" {
		currentTimeZoneMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.current_time_zone",
			TemplateData: map[string]interface{}{
				"time_zone": engine.Settings.GetTimeZone(),
			},
		})
		engine.Log.Log(meta, currentTimeZoneMsg)
	}

	for _, logger := range engine.Log.loggers {
		logInfoMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.log_info",
			TemplateData: map[string]interface{}{
				"name":       logger.Name,
				"split_size": FormatBytes(logger.SPLIT_SIZE),
				"split_time": logger.SPLIT_TIME,
			},
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
			panic(err)
			return
		default:
			invalidOperationMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.invalid_operation",
				TemplateData: map[string]interface{}{
					"name": sig,
				},
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

		listenAddressMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.listen_address",
			TemplateData: map[string]interface{}{
				"bind_address": fmt.Sprintf("https://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK),
			},
		})
		engine.Log.Log(meta, listenAddressMsg)
		if engine.Settings.BIND_DOMAIN != "" {
			listenDomainMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.listen_address",
				TemplateData: map[string]interface{}{
					"bind_address": fmt.Sprintf("https://%v:%v [%v]", engine.Settings.BIND_DOMAIN, engine.Settings.PORT, engine.Settings.NET_WORK),
				},
			})
			engine.Log.Log(meta, listenDomainMsg)
		}
		err = engine.server.ServeTLS(ln, engine.Settings.SSL.CERT_PATH, engine.Settings.SSL.KEY_PATH)
	} else {
		listenAddressMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.listen_address",
			TemplateData: map[string]interface{}{
				"bind_address": fmt.Sprintf("http://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK),
			},
		})
		engine.Log.Log(meta, listenAddressMsg)
		if engine.Settings.BIND_DOMAIN != "" {
			listenDomainMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.listen_address",
				TemplateData: map[string]interface{}{
					"bind_address": fmt.Sprintf("http://%v:%v [%v]", engine.Settings.BIND_DOMAIN, engine.Settings.PORT, engine.Settings.NET_WORK),
				},
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
		shutdownHandlerMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.shutdown_handler",
			TemplateData: map[string]interface{}{
				"name": shutdownCallback.name,
			},
		})
		engine.Log.Log(meta, shutdownHandlerMsg)

		err = shutdownCallback.handler(engine)
		if err != nil {
			shutdownHandlerErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.shutdown_handler_error",
				TemplateData: map[string]interface{}{
					"err": err,
				},
			})
			engine.Log.Log(meta, shutdownHandlerErrorMsg)
		}
	}

	if len(engine.Settings.DATABASES) != 0 {
		closeDatabaseMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.close_database",
		})
		engine.Log.Log(meta, closeDatabaseMsg)
	}
	for name, database := range engine.Settings.DATABASES {
		err = database.Close()
		if err != nil {
			closeDatabaseErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.close_database_error",
				TemplateData: map[string]interface{}{
					"engine": database.ENGINE,
					"name":   name,
					"err":    err,
				},
			})
			engine.Log.Log(meta, closeDatabaseErrorMsg)
		}
	}

	// 取消所有 goroutine 任务
	engine.cancel()
	// 等待所有 goroutine 任务结束
	engine.waitGroup.Wait()

	stopTimeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.stop_time",
		TemplateData: map[string]interface{}{
			"stop_time": GetTime().Format("2006-01-02 15:04:05"),
		},
	})
	engine.Log.Log(meta, stopTimeMsg)
	runTimeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.run_time",
		TemplateData: map[string]interface{}{
			"run_time": engine.runTimeStr(),
		},
	})
	engine.Log.Log(meta, runTimeMsg)
	stopMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.stop",
	})
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
		dayMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "day",
		})
		elapsedStr += fmt.Sprintf("%d %v ", days, dayMsg)
	}
	if hours > 0 {
		hourMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "hour",
		})
		elapsedStr += fmt.Sprintf("%02d %v ", hours, hourMsg)
	}
	if minutes > 0 {
		minuteMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "minute",
		})
		elapsedStr += fmt.Sprintf("%02d %v ", minutes, minuteMsg)
	}
	if seconds > 0 {
		secondMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "second",
		})
		elapsedStr += fmt.Sprintf("%02d %v ", seconds, secondMsg)
	}
	return elapsedStr
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := GetTime()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	r = r.WithContext(ctx)

	var log string
	var err error
	// 初始化请求
	request := &Request{
		Object:     r,
		PathParams: make(Params),
	}
	// 创建自定义响应写入器
	response := &customResponseWriter{
		ResponseWriter: w,
	}

	defer metaRecovery(request, response) // 异常处理

	// 处理 HTTP 请求
	StatusCode := http.StatusOK
	var ResponseData []byte

	responseData := engine.HandlerHTTP(request, response)

	defer func() {
		log = fmt.Sprintf("- %v - %v %v => generated %v byte status (%v %v)",
			request.Object.Host,
			request.Object.Method,
			request.Object.URL.Path,
			response.Bytes,
			request.Object.Proto,
			response.StatusCode,
		)
		engine.Log.Info(log)
	}()

	// 判断 responseData 是否为指针类型，如果是，则解引用获取指向的值
	responseDataValue := reflect.ValueOf(responseData)
	if responseDataValue.Kind() == reflect.Ptr {
		responseData = responseDataValue.Elem().Interface()
	}

	// 文件处理
	fileObject, isFile := responseData.(os.File)
	if isFile {
		// 返回文件内容
		metaResponseStatic(fileObject, request, response)
		return
	}

	// 文件系统处理
	fs, isFileFS := responseData.(embed.FS)
	if isFileFS {
		staticServer := http.FileServer(http.FS(fs))
		staticServer.ServeHTTP(response, request.Object)
		return
	}

	// 响应处理
	responseObject, isResponse := responseData.(Response)
	if isResponse {
		StatusCode = responseObject.Status
		responseData = responseObject.Data
	}

	switch value := responseData.(type) {
	case string:
		ResponseData = []byte(value)
	case []byte:
		ResponseData = value
	default:
		ResponseData, err = json.Marshal(value)
	}
	// 返回响应
	if err != nil {
		responseJsonErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.response_json_error",
			TemplateData: map[string]interface{}{
				"err": err,
			},
		})
		panic(responseJsonErrorMsg)
	}
	// 返回响应
	response.WriteHeader(StatusCode)
	_, err = response.Write(ResponseData)
	if err != nil {
		responseErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.response_error",
			TemplateData: map[string]interface{}{
				"err": err,
			},
		})
		panic(responseErrorMsg)
	}
}

// 处理 HTTP 请求
func (engine *Engine) HandlerHTTP(request *Request, response http.ResponseWriter) (result interface{}) {
	viewSet, isPattern := engine.Router.routeResolution(request.Object.URL.Path, request.PathParams)
	if isPattern == false {
		urlNotAllowedMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.url_not_allowed",
			TemplateData: map[string]interface{}{
				"path": request.Object.URL.Path,
			},
		})
		return Response{
			Status: http.StatusNotFound,
			Data:   urlNotAllowedMsg,
		}
	}
	var handlerFunc HandlerFunc
	switch request.Object.Method {
	case "GET":
		handlerFunc = viewSet.GET
	case "HEAD":
		handlerFunc = viewSet.HEAD
	case "POST":
		handlerFunc = viewSet.POST
	case "PUT":
		handlerFunc = viewSet.PUT
	case "PATCH":
		handlerFunc = viewSet.PATCH
	case "DELETE":
		handlerFunc = viewSet.DELETE
	case "CONNECT":
		handlerFunc = viewSet.CONNECT
	case "OPTIONS":
		handlerFunc = viewSet.OPTIONS
	case "TRACE":
		handlerFunc = viewSet.TRACE
	default:
		methodNotAllowedMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.method_not_allowed",
			TemplateData: map[string]interface{}{
				"method": request.Object.Method,
			},
		})
		return Response{
			Status: http.StatusNotFound,
			Data:   methodNotAllowedMsg,
		}
	}

	if viewSet.file != "" {
		return metaStaticFileHandler(viewSet.file, request)
	} else if viewSet.dir != "" {
		return metaStaticDirHandler(viewSet.dir, request)
	} else if viewSet.fileFS != nil {
		return *viewSet.fileFS
	} else if viewSet.dirFS != nil {
		return *viewSet.dirFS
	}

	if handlerFunc == nil {
		methodNotAllowedMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.method_not_allowed",
			TemplateData: map[string]interface{}{
				"method": request.Object.Method,
			},
		})
		return Response{
			Status: http.StatusNotFound,
			Data:   methodNotAllowedMsg,
		}
	}

	// 处理请求前的中间件
	result = engine.MiddleWares.processRequest(request)
	if result != nil {
		return result
	}

	// 视图前的中间件
	result = engine.MiddleWares.processView(request)
	if result != nil {
		return result
	}

	// 视图处理
	viewResponse := handlerFunc(request)

	// 返回响应前的中间件
	result = engine.MiddleWares.processResponse(request, viewResponse)
	if result != nil {
		return result
	}
	return viewResponse
}
