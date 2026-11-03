package db

import (
	"database/sql"
	"errors"
	"reflect"
	"sync"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// DBProvider 提供共享连接池，ORM 不负责关闭该连接池。
type DBProvider interface {
	DB() *sql.DB
}

// ConnectFunc 根据连接配置名称和连接管理器创建 ORM 引擎，须返回有效的非 nil 实例。
type ConnectFunc func(UseDatabases string, database *goi.Database) Engine

// engineMu 保护 engines
var engineMu sync.RWMutex

// engines 存储所有可用的数据库引擎
// key: 数据库引擎名称
// value: 数据库引擎连接函数
var engines = map[string]ConnectFunc{}

// Register 注册 ORM 引擎工厂
//
// 参数:
//   - name: string 引擎名称，对应 goi.Database.Engine
//   - connect: ConnectFunc 引擎工厂，不能为 nil
//
// 说明:
//   - 同名引擎重复注册或工厂为 nil 时 panic
func Register(name string, connect ConnectFunc) {
	engineMu.Lock()
	defer engineMu.Unlock()
	if connect == nil {
		registerEngineConnectIsNotNilMsg := i18n.T("db.register_engine_is_not_nil")
		panic(errors.New(registerEngineConnectIsNotNilMsg))
	}
	if _, dup := engines[name]; dup {
		registerEngineConnectTwiceMsg := i18n.T("db.register_engine_twice", map[string]any{
			"name": name,
		})
		panic(errors.New(registerEngineConnectTwiceMsg))
	}
	engines[name] = connect
}

// GetConnectFunc 获取已注册的 ORM 引擎工厂
//
// 参数:
//   - name: string 引擎名称
//
// 返回:
//   - ConnectFunc: 引擎工厂，未注册时为 nil
//   - bool: 是否已注册
func GetConnectFunc(name string) (ConnectFunc, bool) {
	engineMu.RLock()
	defer engineMu.RUnlock()
	connect, ok := engines[name]
	return connect, ok
}

// Connect 按全局连接配置创建指定类型的 ORM 引擎
//
// 参数:
//   - UseDatabases: string 数据库配置名称
//
// 返回:
//   - T: 指定类型的引擎，例如 *mysql.Engine
//
// 说明:
//   - 从 goi.Settings.Databases 查找配置，再调用 ConnectDatabase
//   - 配置不存在、引擎未注册或返回类型与 T 不匹配时 panic
func Connect[T Engine](UseDatabases string) T {
	database, ok := goi.Settings.Databases[UseDatabases]
	if !ok {
		databasesNotErrorMsg := i18n.T("db.databases_not_error", map[string]any{
			"name": UseDatabases,
		})
		panic(errors.New(databasesNotErrorMsg))
	}
	return ConnectDatabase[T](UseDatabases, database)
}

// ConnectDatabase 使用连接管理器创建指定类型的 ORM 引擎
//
// 参数:
//   - UseDatabases: string 数据库配置名称
//   - database: *goi.Database 数据库连接管理器
//
// 返回:
//   - T: 指定类型的引擎，例如 *mysql.Engine
//
// 说明:
//   - database 不能为 nil；按 database.Engine 查找工厂并传入连接管理器
//   - 引擎未注册或工厂返回类型与 T 不匹配时 panic
func ConnectDatabase[T Engine](UseDatabases string, database *goi.Database) T {
	connect, ok := GetConnectFunc(database.Engine)
	if !ok {
		engineNotRegisteredMsg := i18n.T("db.engine_not_registered", map[string]any{
			"engine": database.Engine,
		})
		panic(errors.New(engineNotRegisteredMsg))
	}

	// 构造具体引擎实例
	engine := connect(UseDatabases, database)

	// 泛型断言为调用方期望的类型 T
	v, ok := any(engine).(T)
	if !ok {
		engineTypeIsNotMatchMsg := i18n.T("db.engine_type_is_not_match", map[string]any{
			"want_type": reflect.TypeFor[T]().String(),
			"got_type":  reflect.TypeOf(engine).String(),
		})
		panic(errors.New(engineTypeIsNotMatchMsg))
	}
	return v
}
