package goi

import "sync"

// shutdownManager 关闭服务管理器
type shutdownManager struct {
	mutex     sync.Mutex         // 保护 claimed 和 callbacks
	claimed   bool               // 是否已经关闭停机回调注册入口
	callbacks []ShutdownCallback // 按注册顺序保存本轮停机回调
}

// shutdown 保存通过 RegisterOnShutdown 注册的包级停机回调。
var shutdown = newShutdownManager()

// newShutdownManager 创建停机回调管理器。
func newShutdownManager() *shutdownManager {
	return &shutdownManager{
		callbacks: make([]ShutdownCallback, 0),
	}
}

// claim 关闭本轮生命周期的停机回调注册入口。
//
// 返回后 callbacks 不再发生变化，可以在不持有 mutex 的情况下安全读取。
func (manager *shutdownManager) claim() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.claimed = true
}

// register 在注册入口开放时追加停机回调；入口锁定后返回 false。
func (manager *shutdownManager) register(callback ShutdownCallback) bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if manager.claimed {
		return false
	}
	manager.callbacks = append(manager.callbacks, callback)
	return true
}

// ShutdownCallback 表示服务停止阶段执行的回调。
type ShutdownCallback interface {
	// ShutdownName 关闭服务名称
	ShutdownName() string
	// OnShutdown 执行停机处理；调用时后台任务已退出、数据库仍可使用。
	// 返回的错误会被记录，但不会阻止后续回调继续执行
	OnShutdown() error
}

// RegisterOnShutdown 注册关闭服务回调处理程序
//
// 参数:
//   - shutdownCallback ShutdownCallback: 回调函数
//
// 注意:
//   - 回调函数会逆序执行（先注册的后执行）
//   - 可以在服务运行期间注册，停机开始后注册的回调会被忽略
func RegisterOnShutdown(shutdownCallback ShutdownCallback) {
	shutdown.register(shutdownCallback)
}
