package goi

import (
	"context"
	"sync"
)

type startupManager struct {
	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup *sync.WaitGroup
	startups  []Startup
}

var startup = newStartupManager()

func newStartupManager() *startupManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &startupManager{
		ctx:       ctx,
		cancel:    cancel,
		waitGroup: &sync.WaitGroup{},
		startups:  make([]Startup, 0),
	}
}

// Startup 表示一个在服务启动时运行的后台任务
type Startup interface {
	// Name 返回任务名称，用于日志或调试
	Name() string
	// OnStartup 启动任务（通常内部是一个循环，监听 ctx.Done）
	OnStartup(ctx context.Context, wg *sync.WaitGroup)
}

// RegisterOnStartup 注册一个在服务启动时运行的后台任务
//
// 参数:
//   - task Startup: 要注册的任务
//
// 注意:
//   - 请在 RunServer 执行之前注册，之后注册的任务不会执行
func RegisterOnStartup(task Startup) {
	startup.startups = append(startup.startups, task)
}
