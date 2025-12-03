package goi

// shutdownManager 关闭服务管理器
type shutdownManager struct {
	callbacks []ShutdownCallback
}

var shutdown = newShutdownManager()

func newShutdownManager() *shutdownManager {
	return &shutdownManager{
		callbacks: make([]ShutdownCallback, 0),
	}
}

// 关闭服务回调
type ShutdownCallback interface {
	// ShutdownName 关闭服务名称
	ShutdownName() string
	// OnShutdown 关闭服务处理程序
	OnShutdown() error
}

// RegisterOnShutdown 注册关闭服务回调处理程序
//
// 参数:
//   - shutdownCallback ShutdownCallback: 回调函数
//
// 注意:
//   - 回调函数会逆序执行（先注册的后执行）
func RegisterOnShutdown(shutdownCallback ShutdownCallback) {
	shutdown.callbacks = append(shutdown.callbacks, shutdownCallback)
}
