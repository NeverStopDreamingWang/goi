package sqlserver

import (
	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// driverName 与 goi.Settings.Databases[*].Engine 保持一致
const driverName = "sqlserver"

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

// ConnectDatabase 创建 SQL Server 操作实例
//
// 参数:
//   - UseDatabases: string 数据库配置名称
//   - database: db.DBProvider 连接池提供者
//   - options: ...db.EngineOptions 实例策略，按顺序覆盖非 nil 选项
//
// 返回:
//   - *Engine: 新的操作实例，复用提供者的连接池
//
// 说明:
//   - 默认使用当前包级策略；创建后不受全局替换影响
//   - 列解析器优先使用连接选项，其次是 SQL Server 包级配置，最后是 db 包级规则
//   - 此底层入口只使用显式 options，不读取 goi.Database.Config；Connect 和注册工厂会读取连接配置
//   - 驱动由连接池提供者决定，不检查其 Engine 名称
func ConnectDatabase(UseDatabases string, database db.DBProvider, options ...db.EngineOptions) *Engine {
	engineOptions := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}.Merge(options...)
	return &Engine{
		name:              UseDatabases,
		DB:                database.DB(),
		placeholderParser: engineOptions.PlaceholderParser,
		identifierQuoter:  engineOptions.IdentifierQuoter,
		columnParser:      engineOptions.ColumnParser,
	}
}

// Connect 从全局配置创建 SQL Server 操作实例
//
// 参数:
//   - UseDatabases: string goi.Settings.Databases 中的配置名称
//
// 返回:
//   - *Engine: 新的操作实例
//
// 说明:
//   - 配置不存在、Engine 不匹配或配置读取失败时 panic
//   - 通过 GetConfig 读取连接级策略，驱动由连接管理器决定
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
