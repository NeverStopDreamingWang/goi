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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// Http 服务
var Settings = newSettings()
var Cache = newCache()
var Log *Logger
var Validation = newValidation()

// localizedServerError 保存国际化文案键，并在读取错误信息时按当前语言翻译。
type localizedServerError string

// Error 返回当前语言对应的服务错误信息。
func (err localizedServerError) Error() string {
	return i18n.T(string(err))
}

// ErrServerNotRunning 表示服务尚未进入运行生命周期，无法执行停机。
var ErrServerNotRunning error = localizedServerError("server.not_running")

// errServerAlreadyStarted 表示同一个 Engine 已经进入过启动生命周期。
var errServerAlreadyStarted error = localizedServerError("server.already_started")

// engineState 表示 Engine 从创建、启动到停机结束的单向生命周期状态。
type engineState uint8

const (
	engineCreated  engineState = iota // 已创建，尚未启动
	engineStarting                    // 正在初始化监听器和后台任务
	engineRunning                     // 启动完成，可以接收停机请求
	engineStopping                    // 已有调用者取得停机执行权
	engineStopped                     // 停机流程已完成
	engineFailed                      // 启动失败且清理工作已完成
)

// lifecycleState 通过互斥锁串行化生命周期状态的读取和条件切换。
type lifecycleState struct {
	mutex sync.Mutex
	value engineState
}

// get 并发安全地读取当前生命周期状态。
func (state *lifecycleState) get() engineState {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return state.value
}

// set 仅在当前状态与 expected 一致时原子地切换到 next，并返回是否成功。
func (state *lifecycleState) set(expected, next engineState) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	if state.value != expected {
		return false
	}
	state.value = next
	return true
}

// Engine 实现 ServeHTTP 接口
type Engine struct {
	startTime  *time.Time     // 启动时间
	server     http.Server    // net/http 服务
	Router     *Router        // 路由
	Settings   *settings      // 设置
	Cache      *cache         // 缓存
	Log        *Logger        // 日志
	Validation *validation    // 验证管理器
	state      lifecycleState // 受锁保护的生命周期状态
	startDone  chan struct{}  // 启动成功或失败时关闭，唤醒等待启动结果的调用者
	stopDone   chan struct{}  // 停机完成或启动失败时关闭，唤醒所有停机等待者
	stopErrors []error        // 汇总启动失败和停机各阶段错误，供所有 StopServer 调用返回
}

// NewHTTPServer 创建一个 HTTP 服务，并初始化完整的生命周期状态。
func NewHTTPServer() *Engine {
	return &Engine{
		startTime:  nil,
		server:     http.Server{},
		Router:     newRouter(),
		Settings:   Settings,
		Cache:      Cache,
		Log:        Log,
		Validation: Validation,
		startDone:  make(chan struct{}),
		stopDone:   make(chan struct{}),
	}
}

// RunServer 初始化监听器和后台任务并阻塞提供 HTTP 服务，直到整轮停机完成。
//
// Engine 必须由 NewHTTPServer 创建。
// 同一个 Engine 实例停止后不能重新启动。
// 启动或 Serve 失败时会在完成对应清理后继续抛出原始 panic 或错误。
func (engine *Engine) RunServer() {
	// 条件切换保证只有第一个调用者能够执行启动流程。
	if !engine.state.set(engineCreated, engineStarting) {
		panic(errServerAlreadyStarted)
	}

	// 启动完成前发生异常时，停止已启动的后台任务并发布失败终态，
	// 避免并发 StopServer 永久等待 startDone 或 stopDone。
	defer func() {
		if recovered := recover(); recovered != nil {
			startErr, ok := recovered.(error)
			if !ok {
				startErr = fmt.Errorf("%v", recovered)
			}
			if engine.state.get() == engineStarting {
				engine.stopErrors = append(engine.stopErrors, startErr)
				engine.Log.Log(ERROR, startErr)
				startup.claim()
				startup.cancel()
				startup.waitGroup.Wait()
				// 先收集错误再切换终态，使观察到 engineFailed 的调用可以读取结果。
				if engine.state.set(engineStarting, engineFailed) {
					close(engine.startDone)
					close(engine.stopDone)
				}
			}
			panic(recovered)
		}
	}()

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

	// Debug
	engine.Log.Log(meta, fmt.Sprintf("Debug: %v", engine.Settings.Debug))

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

	// 在启动后台任务前初始化服务器，避免任务已启动但监听器初始化失败
	engine.server = http.Server{
		Addr:    fmt.Sprintf("%v:%v", engine.Settings.BindAddress, engine.Settings.Port),
		Handler: engine,
	}

	// 统一选择 HTTP 或 HTTPS 的监听函数，后续启动流程无需重复分支。
	scheme := "http"
	serve := engine.server.Serve
	if engine.Settings.SSL.Enabled {
		cert, err := tls.LoadX509KeyPair(
			engine.Settings.SSL.CertPath,
			engine.Settings.SSL.KeyPath,
		)
		if err != nil {
			panic(err)
		}
		engine.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		scheme = "https"
		serve = func(listener net.Listener) error {
			// 证书已经在启动阶段加载到 TLSConfig，传入空路径避免再次读取文件。
			return engine.server.ServeTLS(listener, "", "")
		}
	}

	// 先成功创建监听器，再启动后台任务，避免监听失败后遗留运行中的任务。
	ln, err := net.Listen(engine.Settings.Network, engine.server.Addr)
	if err != nil {
		panic(err)
	}

	// 初始化缓存并完成框架内部任务注册后，关闭本轮启动任务注册入口。
	engine.Cache.initCache()
	startup.claim()
	// 注册入口锁定后，startups 切片不会再变化，可以稳定地启动全部任务。
	for _, task := range startup.startups {
		taskNameMsg := i18n.T("server.startup_task", map[string]any{
			"name": task.StartupName(),
		})
		engine.Log.Log(meta, taskNameMsg)
		startup.waitGroup.Add(1)
		go task.OnStartup(startup.ctx, startup.waitGroup)
	}

	// 后台任务已经启动，输出监听地址后立即进入服务监听阶段。
	listenAddressMsg := i18n.T("server.listen_address", map[string]any{
		"bind_address": fmt.Sprintf("%v://%v:%v [%v]", scheme, engine.Settings.BindAddress, engine.Settings.Port, engine.Settings.Network),
	})
	engine.Log.Log(meta, listenAddressMsg)
	if engine.Settings.BindDomain != "" {
		listenDomainMsg := i18n.T("server.listen_address", map[string]any{
			"bind_address": fmt.Sprintf("%v://%v:%v [%v]", scheme, engine.Settings.BindDomain, engine.Settings.Port, engine.Settings.Network),
		})
		engine.Log.Log(meta, listenDomainMsg)
	}

	// 发布运行状态后关闭 startDone，唤醒启动期间调用 StopServer 的调用者。
	if !engine.state.set(engineStarting, engineRunning) {
		panic(errServerAlreadyStarted)
	}
	close(engine.startDone)

	// 后台任务启动后再接管退出信号，收到信号时统一进入 StopServer。
	serverChan := make(chan os.Signal, 1)
	signal.Notify(serverChan, os.Interrupt, syscall.SIGTERM)
	signalExit := make(chan struct{})
	// 信号协程只负责将系统信号转换为统一的 StopServer 调用；手动停机时，
	// signalExit 会在 RunServer 返回前通知该协程退出。
	go func() {
		select {
		case _, ok := <-serverChan:
			if ok {
				_ = engine.StopServer()
			}
		case <-signalExit:
		}
	}()
	// RunServer 离开时停止向本地通道投递信号，并终止仍在等待信号的协程。
	defer func() {
		signal.Stop(serverChan)
		close(signalExit)
	}()

	// Serve 正常停机时返回 http.ErrServerClosed；其他错误也进入同一停机流程，
	// 等待资源清理完成后再把原始监听错误以 panic 形式交给调用方。
	err = serve(ln)
	serveFailed := err != nil && !errors.Is(err, http.ErrServerClosed)
	if serveFailed {
		func() {
			defer func() {
				_ = recover()
			}()
			engine.Log.Log(ERROR, err)
		}()
		_ = engine.StopServer()
	}

	// 信号、业务代码或 Serve 异常都会进入 StopServer；统一等待完整停机。
	<-engine.stopDone
	if serveFailed {
		panic(err)
	}
}

// StopServer 停止 HTTP 服务，并阻塞到整轮停机流程完成。
//
// 停机顺序为：停止 HTTP 监听并等待在途请求、停止并等待后台协程、
// 逆序执行停机回调，最后关闭数据库。HTTP Shutdown、OnShutdown 和
// Database.Close 返回的错误会立即写入日志并继续后续清理，停机完成后合并返回；
// panic 保持 Go 的默认语义，不会转换为 error。并发调用时只有一个调用者执行
// 清理，其余调用者等待同一轮停机完成。因为本方法会等待在途请求且不设置超时，
// 不要在 HTTP 请求处理程序、受管理的后台任务或停机回调中同步调用，
// 应从其他协程发起停机。
func (engine *Engine) StopServer() error {
	for {
		switch engine.state.get() {
		case engineCreated:
			return ErrServerNotRunning
		case engineStarting:
			// 等待启动成功或失败的结果，再按新的状态决定是否执行停机。
			<-engine.startDone
		case engineRunning:
			// 停机真正开始前一直允许注册回调；在状态切换前关闭注册入口，
			// 保证后续遍历 callbacks 时切片保持稳定。
			shutdown.claim()
			if !engine.state.set(engineRunning, engineStopping) {
				continue
			}

			// 停机执行者退出时发布终态并唤醒等待者；panic 仍按 Go 默认语义向外传播。
			defer func() {
				engine.state.set(engineStopping, engineStopped)
				close(engine.stopDone)
			}()

			// 先关闭服务器，停止接收新请求并等待在途请求处理完成。
			if err := engine.server.Shutdown(context.Background()); err != nil {
				engine.stopErrors = append(engine.stopErrors, err)
				engine.Log.Log(ERROR, err)
			}

			// 再取消并等待所有后台任务退出。
			startup.cancel()
			startup.waitGroup.Wait()

			// 逆序执行用户注册的停机回调。
			for i := len(shutdown.callbacks) - 1; i >= 0; i-- {
				shutdownCallback := shutdown.callbacks[i]
				shutdownHandlerMsg := i18n.T("server.shutdown_callback", map[string]any{
					"name": shutdownCallback.ShutdownName(),
				})
				engine.Log.Log(meta, shutdownHandlerMsg)
				// OnShutdown 的 error 属于可预期失败：记录后继续清理其他资源。
				if err := shutdownCallback.OnShutdown(); err != nil {
					engine.stopErrors = append(engine.stopErrors, err)
					shutdownHandlerErrorMsg := i18n.T("server.shutdown_callback_error", map[string]any{
						"err": err,
					})
					shutdownHandlerErr := errors.New(shutdownHandlerErrorMsg)
					engine.Log.Log(ERROR, shutdownHandlerErr)
				}
			}

			// 所有请求、后台任务和停机回调结束后，最后关闭数据库。
			if len(engine.Settings.Databases) != 0 {
				closeDatabaseMsg := i18n.T("server.close_database")
				engine.Log.Log(meta, closeDatabaseMsg)
			}
			for name, database := range engine.Settings.Databases {
				if err := database.Close(); err != nil {
					closeDatabaseErrorMsg := i18n.T("server.close_database_error", map[string]any{
						"engine": database.Engine,
						"name":   name,
						"err":    err,
					})
					closeDatabaseErr := errors.New(closeDatabaseErrorMsg)
					engine.stopErrors = append(engine.stopErrors, closeDatabaseErr)
					engine.Log.Log(ERROR, closeDatabaseErr)
				}
			}

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

			// 将本轮停机收集的全部错误作为同一结果返回。
			return errors.Join(engine.stopErrors...)
		case engineStopping:
			// 其他调用者等待当前停机执行者完成，避免重复关闭相同资源。
			<-engine.stopDone
		case engineStopped, engineFailed:
			// 生命周期结束后重复调用时，返回与本轮生命周期一致的结果。
			return errors.Join(engine.stopErrors...)
		default:
			return ErrServerNotRunning
		}
	}
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
