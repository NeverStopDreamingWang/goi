package db

import "database/sql"

// TransactionFunc 定义使用通用引擎的事务回调，本身不负责开启或提交事务。
type TransactionFunc func(engine Engine, args ...any) error

// Engine 定义 ORM 引擎的基础操作，Name 返回连接配置名称。
type Engine interface {
	// Name 返回连接配置名称。
	Name() string
	// Execute 执行 SQL，返回执行结果。
	Execute(query string, args ...any) (sql.Result, error)
	// QueryRow 查询一行，错误由返回值的 Scan 方法报告。
	QueryRow(query string, args ...any) *sql.Row
	// Query 查询多行，调用方负责关闭返回的 Rows。
	Query(query string, args ...any) (*sql.Rows, error)
}
