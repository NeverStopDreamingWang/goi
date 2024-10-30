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
	"syscall"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Http 服务
var version = "v1.3.3"
var Settings *metaSettings
var Cache *metaCache
var Log *metaLog
var Validator *metaValidator

var exitChan = make(chan os.Signal, 1)

// 版本
func Version() string {
	return version
}

// Engine 实现 ServeHTTP 接口
type Engine struct {
	startTime   *time.Time
	server      http.Server
	Router      *metaRouter
	MiddleWares *metaMiddleWares
	Settings    *metaSettings
	Cache       *metaCache
	Log         *metaLog
	Validator   *metaValidator
}

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	Settings = newSettings()
	Cache = newCache()
	Log = newLog()
	Validator = newValidator()
	return &Engine{
		startTime:   nil,
		server:      http.Server{},
		Router:      newRouter(),
		MiddleWares: newMiddleWares(),
		Settings:    Settings,
		Cache:       Cache,
		Log:         Log,
		Validator:   Validator,
	}
}

// 启动 http 服务
func (engine *Engine) RunServer() {
	var err error
	startTime := time.Now().In(engine.Settings.GetLocation())
	engine.startTime = &startTime

	// 初始化日志
	engine.Log.InitLogger()

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
	for _, logger := range engine.Log.loggers {
		log := fmt.Sprintf("- [%v]", logger.Name)
		if logger.SPLIT_SIZE != 0 {
			logSplitSizeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.log_split_size",
				TemplateData: map[string]interface{}{
					"split_size": formatBytes(logger.SPLIT_SIZE),
				},
			})
			log += " " + logSplitSizeMsg
		}
		if logger.SPLIT_TIME != "" {
			logSplitSizeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.log_split_size",
				TemplateData: map[string]interface{}{
					"split_size": logger.SPLIT_TIME,
				},
			})
			log += " " + logSplitSizeMsg
		}
		engine.Log.Log(meta, log)
	}

	if engine.Settings.GetTimeZone() != "" {
		currentTimeZoneMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.log_split_size",
			TemplateData: map[string]interface{}{
				"time_zone": engine.Settings.GetTimeZone(),
			},
		})
		engine.Log.Log(meta, currentTimeZoneMsg)
	}

	// 初始化缓存
	engine.Cache.initCache()

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
			invalidOperationMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "server.invalid_operation",
			})
			engine.Log.Log(meta, invalidOperationMsg)
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

// 停止 http 服务
func (engine *Engine) StopServer() error {
	var err error

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

	stopTimeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "server.stop_time",
		TemplateData: map[string]interface{}{
			"stop_time": time.Now().In(engine.Settings.GetLocation()).Format("2006-01-02 15:04:05"),
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
	currentTime := time.Now().In(engine.Settings.GetLocation())

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
	requestID := time.Now().In(engine.Settings.GetLocation())
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	r = r.WithContext(ctx)

	var log string
	var err error
	// 初始化请求
	request := &Request{
		Object:     r,
		Context:    ctx,
		PathParams: make(ParamsValues),
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
