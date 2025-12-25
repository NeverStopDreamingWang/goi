package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

type TransactionFunc func(engine Engine, args ...any) error

// Engine 结构体用于管理MySQL数据库连接和操作
type Engine interface {
	Name() string
	Execute(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

type ConnectFunc func(UseDataBases string, database goi.DataBase) Engine

// engineMu 保护 engines
var engineMu sync.RWMutex

// engines 存储所有可用的数据库引擎
// key: 数据库引擎名称
// value: 数据库引擎连接函数
var engines = map[string]ConnectFunc{}

// Register 注册 ORM 数据库引擎
//
// 参数:
//   - name string: 数据库引擎名称
//   - connect ConnectFunc: 数据库引擎连接函数
func Register(name string, connect ConnectFunc) {
	engineMu.Lock()
	defer engineMu.Unlock()
	if connect == nil {
		registerEngineConnectIsNotNilMsg := i18n.T("db.register_engine_is_not_nil")
		panic(fmt.Errorf(registerEngineConnectIsNotNilMsg))
	}
	if _, dup := engines[name]; dup {
		registerEngineConnectTwiceMsg := i18n.T("db.register_engine_twice", map[string]any{
			"name": name,
		})
		panic(fmt.Errorf(registerEngineConnectTwiceMsg))
	}
	engines[name] = connect
}

// GetConnectFunc 获取 ORM 数据库引擎连接函数
//
// 参数:
//   - name string: 数据库引擎名称
//
// 返回:
//   - ConnectFunc: 数据库引擎连接函数
//   - bool: 是否存在
func GetConnectFunc(name string) (ConnectFunc, bool) {
	engineMu.RLock()
	defer engineMu.RUnlock()
	connect, ok := engines[name]
	return connect, ok
}

// Connect 连接 ORM 数据库引擎（泛型）
//
// 参数:
//   - UseDataBases: string 数据库配置名称
//
// 返回:
//   - T: 调用方期望的具体引擎类型（例如 *mysql.Engine、*sqlite3.Engine）
//
// 说明:
//   - 从全局配置中获取数据库连接信息
//   - 根据 DataBase.Engine（应为驱动名字符串）查找注册的引擎工厂
//   - 使用工厂基于已建立的 *sql.DB 构造具体引擎实例
//   - 将具体实例断言为 T；若类型不匹配则 panic
func Connect[T Engine](UseDataBases string) T {
	database, ok := goi.Settings.DATABASES[UseDataBases]
	if !ok {
		databasesNotErrorMsg := i18n.T("db.databases_not_error", map[string]any{
			"name": UseDataBases,
		})
		panic(fmt.Errorf(databasesNotErrorMsg))
	}
	return ConnectDatabase[T](UseDataBases, *database)
}

// ConnectDatabase 连接 ORM 数据库引擎（泛型）
//
// 参数:
//   - database: *goi.DataBase 数据库连接管理器
//
// 返回:
//   - T: 调用方期望的具体引擎类型（例如 *mysql.Engine、*sqlite3.Engine）
//
// 说明:
//   - 根据 DataBase.Engine（应为驱动名字符串）查找注册的引擎工厂
//   - 使用工厂基于已建立的 *sql.DB 构造具体引擎实例
//   - 将具体实例断言为 T；若类型不匹配则 panic
func ConnectDatabase[T Engine](UseDataBases string, database goi.DataBase) T {
	connect, ok := GetConnectFunc(database.ENGINE)
	if !ok {
		engineNotRegisteredMsg := i18n.T("db.engine_not_registered", map[string]any{
			"engine": database.ENGINE,
		})
		panic(fmt.Errorf(engineNotRegisteredMsg))
	}

	// 构造具体引擎实例
	engine := connect(UseDataBases, database)

	// 泛型断言为调用方期望的类型 T
	v, ok := any(engine).(T)
	if !ok {
		var zero T
		engineTypeIsNotMatchMsg := i18n.T("db.engine_type_is_not_match", map[string]any{
			"want_type": reflect.TypeOf(zero).String(),
			"got_type":  reflect.TypeOf(engine).String(),
		})
		panic(fmt.Errorf(engineTypeIsNotMatchMsg))
	}
	return v
}
