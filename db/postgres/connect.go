package postgres

import (
	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// driverName 与 goi.Settings.Databases[*].Engine 保持一致
const driverName = "postgres"

func init() {
	db.Register(driverName, factory)
}

// factory 读取连接配置并创建引擎，配置错误在创建连接池前原样 panic。
func factory(UseDatabases string, database *goi.Database) db.Engine {
	var config Config
	if err := database.GetConfig(&config); err != nil {
		panic(err)
	}
	return ConnectDatabase(UseDatabases, database, config.EngineOptions)
}

// ConnectDatabase 创建 PostgreSQL 操作实例
//
// 参数:
//   - UseDatabases: string 连接配置名称
//   - database: db.DBProvider 提供连接池的对象
//   - options: ...db.EngineOptions 实例策略，按顺序覆盖非 nil 项
//
// 返回:
//   - *Engine: 新的操作实例
//
// 说明:
//   - 本入口不读取 Database.Config；连接级配置请使用 Connect 或 db.Connect/db.ConnectDatabase 泛型入口
//   - 连接和驱动由 database 提供，调用方负责保证 SQL 方言兼容
//   - 未指定的策略使用当前包级配置，不校验连接的 Engine 名称
//   - 列解析器优先使用连接选项，其次是 PostgreSQL 包级配置，最后是 db 包级规则
//   - 实例保存创建时的引用器、占位符和列解析器，不受后续包级配置替换影响
func ConnectDatabase(UseDatabases string, database db.DBProvider, options ...db.EngineOptions) *Engine {
	strategies := (db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}).Merge(options...)
	return &Engine{
		name:              UseDatabases,
		DB:                database.DB(),
		placeholderParser: strategies.PlaceholderParser,
		identifierQuoter:  strategies.IdentifierQuoter,
		columnParser:      strategies.ColumnParser,
	}
}

// Connect 按全局配置创建 PostgreSQL 操作实例
//
// 参数:
//   - UseDatabases: string 连接配置名称
//
// 返回:
//   - *Engine: 新的操作实例
//
// 说明:
//   - 从 goi.Settings.Databases 读取连接管理器及本包 Config，连接级非 nil 策略优先
//   - 配置读取失败时原样 panic，且不会创建连接池
//   - 配置不存在或 Engine 名称不匹配时 panic；应用需自行导入数据库驱动
func Connect(UseDatabases string) *Engine {
	database, ok := goi.Settings.Databases[UseDatabases]
	if !ok {
		databasesNotErrorMsg := i18n.T("db.databases_not_error", map[string]any{
			"name": UseDatabases,
		})
		panic(databasesNotErrorMsg)
	}
	if database.Engine != driverName {
		engineNotMatchErrorMsg := i18n.T("db.engine_type_is_not_match", map[string]any{
			"want_type": driverName,
			"got_type":  database.Engine,
		})
		panic(engineNotMatchErrorMsg)
	}
	return factory(UseDatabases, database).(*Engine)
}
