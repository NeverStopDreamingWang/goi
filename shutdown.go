package goi

// 关闭服务回调
type ShutdownCallback interface {
	// Name 获取回调名称
	Name() string
	// OnShutdown 关闭服务处理程序
	OnShutdown() error
}

var shutdownCallbacks []ShutdownCallback // 用户定义的关闭服务回调处理程序

// RegisterOnShutdown 注册关闭服务回调处理程序
//
// 参数:
//   - shutdownCallback ShutdownCallback: 回调函数
//
// 注意:
//   - 回调函数会逆序执行（先注册的后执行）
func RegisterOnShutdown(shutdownCallback ShutdownCallback) {
	shutdownCallbacks = append(shutdownCallbacks, shutdownCallback)
}
