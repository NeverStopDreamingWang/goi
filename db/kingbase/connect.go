package kingbase

import (
	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/mysql"
	"github.com/NeverStopDreamingWang/goi/v3/db/oracle"
	"github.com/NeverStopDreamingWang/goi/v3/db/postgres"
	"github.com/NeverStopDreamingWang/goi/v3/db/sqlserver"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

const driverName = "kingbase"

func init() {
	db.Register(driverName, factory)
}

// factory 按兼容模式创建对应方言的操作实例。
func factory(name string, database *goi.Database) db.Engine {
	return ConnectDatabase(name, database)
}

// ConnectDatabase 按 Kingbase 配置创建对应数据库方言的操作实例
//
// 参数:
//   - name: string 连接配置名称
//   - database: *goi.Database 连接管理器，通过 GetConfig 读取配置
//
// 返回:
//   - db.Engine: Mode 对应的 postgres、mysql、oracle 或 sqlserver 引擎
//
// 说明:
//   - 不校验 Engine 名称；底层连接仍由 database.DB() 提供，不切换驱动
//   - 未指定模式时使用 PostgreSQL；配置读取失败或模式未知时 panic
//   - Config 为 nil 或 nil 指针时使用默认配置，策略对象不做深拷贝
//   - 默认将 ORM 占位符 ? 转为 Gokb 的 $1、$2 等，可通过 EngineOptions 覆盖
//   - 模型与校验应使用返回引擎所属数据库包的接口和方法
func ConnectDatabase(name string, database *goi.Database) db.Engine {
	var config Config
	if err := database.GetConfig(&config); err != nil {
		panic(err)
	}
	options := (db.EngineOptions{PlaceholderParser: gokbPlaceholderParser{}}).Merge(config.EngineOptions)
	switch config.Mode {
	case "", ModePostgreSQL:
		return postgres.ConnectDatabase(name, database, options)
	case ModeMySQL:
		return mysql.ConnectDatabase(name, database, options)
	case ModeOracle:
		return oracle.ConnectDatabase(name, database, options)
	case ModeSQLServer:
		return sqlserver.ConnectDatabase(name, database, options)
	default:
		panic(i18n.T("db.compatibility_mode_is_not_supported", map[string]any{
			"engine": driverName,
			"mode":   config.Mode,
		}))
	}
}

// Connect 按全局配置创建 Kingbase 操作实例
//
// 参数:
//   - name: string 连接配置名称
//
// 返回:
//   - db.Engine: 兼容模式对应的具体引擎，完整 ORM 方法需通过对应类型调用
//
// 说明:
//   - 配置不存在或 Engine 不是 kingbase 时 panic
//   - 可用 db.Connect[*mysql.Engine] 等泛型入口直接取得指定类型；类型须与 Mode 一致
func Connect(name string) db.Engine {
	database, ok := goi.Settings.Databases[name]
	if !ok {
		panic(i18n.T("db.databases_not_error", map[string]any{"name": name}))
	}
	if database.Engine != driverName {
		panic(i18n.T("db.engine_type_is_not_match", map[string]any{
			"want_type": driverName,
			"got_type":  database.Engine,
		}))
	}
	return ConnectDatabase(name, database)
}
