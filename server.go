package goi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// Http 服务
var Settings = newSettings()
var Cache = newCache()
var Log = NewLogger(filepath.Join(Settings.BASE_DIR, "logs", "server.log"))
var Validator = newValidator()

var serverChan = make(chan os.Signal, 1)

// Engine 实现 ServeHTTP 接口
type Engine struct {
	startTime *time.Time  // 启动时间
	server    http.Server // net/http 服务
	Router    *Router     // 路由
	Settings  *settings   // 设置
	Cache     *cache      // 缓存
	Log       *Logger     // 日志
	Validator *validator  // 验证器
}

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	return &Engine{
		startTime: nil,
		server:    http.Server{},
		Router:    newRouter(),
		Settings:  Settings,
		Cache:     Cache,
		Log:       Log,
		Validator: Validator,
	}
}

// 启动 http 服务
func (engine *Engine) RunServer() {
	startedMsg := i18n.T("server.started")
	engine.Log.Log(meta, startedMsg)

	// goi 版本
	goiVersionMsg := i18n.T("server.goi_version", map[string]any{
		"version": version,
	})
	engine.Log.Log(meta, goiVersionMsg)

	// 启动时间
	startTime := GetTime()
	engine.startTime = &startTime
	startTimeMsg := i18n.T("server.start_time", map[string]any{
		"start_time": engine.startTime.Format(time.DateTime),
	})
	engine.Log.Log(meta, startTimeMsg)

	// DEBUG
	engine.Log.Log(meta, fmt.Sprintf("DEBUG: %v", engine.Settings.DEBUG))

	// 时区
	if engine.Settings.GetTimeZone() != "" {
		currentTimeZoneMsg := i18n.T("server.current_time_zone", map[string]any{
			"time_zone": engine.Settings.GetTimeZone(),
		})
		engine.Log.Log(meta, currentTimeZoneMsg)
	}

	// 全局默认日志
	logInfoMsg := i18n.T("server.log_info", map[string]any{
		"name":       engine.Log.Name,
		"split_size": FormatBytes(engine.Log.SplitSize),
		"split_time": engine.Log.SplitTime,
	})
	engine.Log.Log(meta, logInfoMsg)

	// 初始化缓存
	engine.Cache.initCache()

	// 启动后台任务
	for _, task := range startup.startups {
		taskNameMsg := i18n.T("server.startup_task", map[string]any{
			"name": task.StartupName(),
		})
		engine.Log.Log(meta, taskNameMsg)
		go task.OnStartup(startup.ctx, startup.waitGroup)
	}

	// 注册关闭信号
	signal.Notify(serverChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		// 等待关闭信号
		sig, _ := <-serverChan
		switch sig {
		case os.Kill, os.Interrupt, syscall.SIGTERM:
			err := engine.StopServer()
			if err != nil {
				engine.Log.Error(err)
				panic(err)
			}
		default:
			invalidOperationMsg := i18n.T("server.invalid_operation", map[string]any{
				"name": sig,
			})
			panic(invalidOperationMsg)
		}
	}()

	engine.server = http.Server{
		Addr:    fmt.Sprintf("%v:%v", engine.Settings.BIND_ADDRESS, engine.Settings.PORT),
		Handler: engine,
	}
	ln, err := net.Listen(engine.Settings.NET_WORK, engine.server.Addr)
	if err != nil {
		engine.Log.Error(err)
		panic(err)
	}
	if engine.Settings.SSL.STATUS == true {
		certificate, err := os.ReadFile(engine.Settings.SSL.CERT_PATH)
		if err != nil {
			engine.Log.Error(err)
			panic(err)
		}
		key, err := os.ReadFile(engine.Settings.SSL.KEY_PATH)
		if err != nil {
			engine.Log.Error(err)
			panic(err)
		}
		cert, err := tls.X509KeyPair(certificate, key)
		if err != nil {
			engine.Log.Error(err)
			panic(err)
		}
		engine.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		listenAddressMsg := i18n.T("server.listen_address", map[string]any{
			"bind_address": fmt.Sprintf("https://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK),
		})
		engine.Log.Log(meta, listenAddressMsg)
		if engine.Settings.BIND_DOMAIN != "" {
			listenDomainMsg := i18n.T("server.listen_address", map[string]any{
				"bind_address": fmt.Sprintf("https://%v:%v [%v]", engine.Settings.BIND_DOMAIN, engine.Settings.PORT, engine.Settings.NET_WORK),
			})
			engine.Log.Log(meta, listenDomainMsg)
		}
		err = engine.server.ServeTLS(ln, engine.Settings.SSL.CERT_PATH, engine.Settings.SSL.KEY_PATH)
	} else {
		listenAddressMsg := i18n.T("server.listen_address", map[string]any{
			"bind_address": fmt.Sprintf("http://%v:%v [%v]", engine.Settings.BIND_ADDRESS, engine.Settings.PORT, engine.Settings.NET_WORK),
		})
		engine.Log.Log(meta, listenAddressMsg)
		if engine.Settings.BIND_DOMAIN != "" {
			listenDomainMsg := i18n.T("server.listen_address", map[string]any{
				"bind_address": fmt.Sprintf("http://%v:%v [%v]", engine.Settings.BIND_DOMAIN, engine.Settings.PORT, engine.Settings.NET_WORK),
			})
			engine.Log.Log(meta, listenDomainMsg)
		}
		err = engine.server.Serve(ln)
	}
	if err != nil && err != http.ErrServerClosed {
		engine.Log.Error(err)
		panic(err)
	}
}

// 停止 http 服务
func (engine *Engine) StopServer() (err error) {
	// 执行用户定义的回调函数（逆序执行：先进后出）
	for i := len(shutdown.callbacks) - 1; i >= 0; i-- {
		shutdownCallback := shutdown.callbacks[i]
		shutdownHandlerMsg := i18n.T("server.shutdown_handler", map[string]any{
			"name": shutdownCallback.ShutdownName(),
		})
		engine.Log.Log(meta, shutdownHandlerMsg)

		err = shutdownCallback.OnShutdown()
		if err != nil {
			shutdownHandlerErrorMsg := i18n.T("server.shutdown_handler_error", map[string]any{
				"err": err,
			})
			return errors.New(shutdownHandlerErrorMsg)
		}
	}

	if len(engine.Settings.DATABASES) != 0 {
		closeDatabaseMsg := i18n.T("server.close_database")
		engine.Log.Log(meta, closeDatabaseMsg)
	}
	for name, database := range engine.Settings.DATABASES {
		err = database.Close()
		if err != nil {
			closeDatabaseErrorMsg := i18n.T("server.close_database_error", map[string]any{
				"engine": database.ENGINE,
				"name":   name,
				"err":    err,
			})
			return errors.New(closeDatabaseErrorMsg)
		}
	}

	// 取消所有 goroutine 任务
	startup.cancel()
	// 等待所有 goroutine 任务结束
	startup.waitGroup.Wait()

	stopTimeMsg := i18n.T("server.stop_time", map[string]any{
		"stop_time": GetTime().Format(time.DateTime),
	})
	engine.Log.Log(meta, stopTimeMsg)
	runTimeMsg := i18n.T("server.run_time", map[string]any{
		"run_time": engine.runTimeStr(),
	})
	engine.Log.Log(meta, runTimeMsg)
	stopMsg := i18n.T("server.stopped")
	engine.Log.Log(meta, stopMsg)
	// 关闭服务器
	err = engine.server.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// 获取当前运行时间 返回时间间隔
//
// 返回:
//   - time.Duration: 运行时间间隔
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
//
// 参数:
//   - w http.ResponseWriter: 响应写入器
//   - r *http.Request: HTTP请求对象
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

	response := engine.handler(request)
	_, err := response.write(request, responseWriter)
	if err != nil {
		panic(err)
	}
}

// handler 为 Engine 的核心请求处理函数
//
// 参数:
//   - request *Request: HTTP请求对象
//
// 返回:
//   - *Response: HTTP响应对象指针
func (engine *Engine) handler(request *Request) (response *Response) {
	// 路由解析
	viewSet, middlewares, isPattern := engine.Router.resolve(request.Object.URL.Path, request.PathParams)
	if isPattern == false || viewSet == nil {
		urlNotAllowedMsg := i18n.T("server.url_not_allowed", map[string]any{
			"path": request.Object.URL.Path,
		})
		return &Response{Status: http.StatusNotFound, Data: urlNotAllowedMsg}
	}

	// 设置 Allow 响应头
	defer func() {
		allowHeader := strings.Join(viewSet.GetMethods(), ", ")
		response.Header().Set("Allow", allowHeader)
	}()

	// 获取视图处理函数
	handlerFunc, err := viewSet.GetHandlerFunc(request.Object.Method)
	if handlerFunc == nil || err != nil {
		return &Response{Status: http.StatusMethodNotAllowed, Data: err.Error()}
	}

	// 中间件，处理请求之前：构建洋葱链后获取响应
	middlewareChain := middlewares.loadMiddleware(handlerFunc)
	return middlewareChain(request)
}

// 统一把结果转换为 Response
//
// 参数:
//   - content any: 视图处理结果
//
// 返回:
//   - *Response: HTTP响应对象指针
func toResponse(content any) *Response {
	switch value := content.(type) {
	case Response:
		return &value
	case *Response:
		if value == nil {
			return &Response{Status: http.StatusOK, Data: nil}
		}
		return value
	default:
		return &Response{Status: http.StatusOK, Data: value}
	}
}
