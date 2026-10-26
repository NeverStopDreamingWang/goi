package goi

import (
	"context"
	"sync"
)

// startupManager 启动服务管理器
type startupManager struct {
	mutex     sync.Mutex         // 保护 claimed 和 startups
	claimed   bool               // 是否已经关闭启动任务注册入口
	ctx       context.Context    // 传递给所有启动任务的生命周期上下文
	cancel    context.CancelFunc // 通知所有启动任务结束
	waitGroup *sync.WaitGroup    // 等待所有启动任务完成收尾
	startups  []Startup          // 按注册顺序保存本轮启动任务
}

// claim 关闭本轮生命周期的启动任务注册入口。
//
// 返回后 startups 不再发生变化，可以在不持有 mutex 的情况下安全读取。
func (manager *startupManager) claim() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.claimed = true
}

// register 在注册入口开放时追加启动任务；入口锁定后返回 false。
func (manager *startupManager) register(task Startup) bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if manager.claimed {
		return false
	}
	manager.startups = append(manager.startups, task)
	return true
}

// startup 保存通过 RegisterOnStartup 注册的包级启动任务及其运行上下文。
var startup = newStartupManager()

// newStartupManager 创建可统一取消并等待全部任务退出的启动管理器。
func newStartupManager() *startupManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &startupManager{
		ctx:       ctx,
		cancel:    cancel,
		waitGroup: &sync.WaitGroup{},
		startups:  make([]Startup, 0),
	}
}

// Startup 表示由服务生命周期管理的后台任务。
type Startup interface {
	// StartupName 启动任务名称
	StartupName() string
	// OnStartup 启动任务；实现必须在 ctx.Done 后自行完成收尾并返回，
	// 退出时调用一次 wg.Done，不能依赖随后才会执行的 OnShutdown
	OnStartup(ctx context.Context, wg *sync.WaitGroup)
}

// RegisterOnStartup 注册一个在服务启动时运行的后台任务
//
// 参数:
//   - task Startup: 要注册的任务
//
// 注意:
//   - 应在调用 RunServer 前完成注册；注册入口关闭后的任务会被忽略
func RegisterOnStartup(task Startup) {
	startup.register(task)
}
