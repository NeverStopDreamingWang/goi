package goi

import (
	"database/sql"
	"sync"
)

type DBConnectFunc func() *sql.DB

// Database 数据库连接管理器
//
// 字段:
//   - Engine string: 数据库引擎名称
//   - Connect func: 数据库连接函数，使用时才会连接，且仅会调用一次
//   - db *sql.DB: 数据库连接实例
type Database struct {
	Engine  string
	Connect func(Engine string) *sql.DB // 使用时自动连接
	db      *sql.DB
	mu      sync.RWMutex
}

// DB 获取数据库连接实例
//
// 返回:
//   - *sql.DB: 数据库连接对象，如果连接不存在则自动创建
func (database *Database) DB() *sql.DB {
	// 先用读锁检查（高性能路径）
	database.mu.RLock()
	if database.db != nil {
		defer database.mu.RUnlock()
		return database.db
	}
	database.mu.RUnlock()

	// 需要初始化，获取写锁
	database.mu.Lock()
	defer database.mu.Unlock()

	// 双重检查：防止等待写锁期间其他 goroutine 已完成初始化
	if database.db == nil {
		database.db = database.Connect(database.Engine)
	}
	return database.db
}

// Close 关闭数据库连接
//
// 返回:
//   - error: 关闭过程中的错误信息，如果连接不存在则返回nil
func (database *Database) Close() error {
	database.mu.Lock()
	defer database.mu.Unlock()
	if database.db == nil {
		return nil
	}
	err := database.db.Close()
	database.db = nil // 重置为 nil，允许下次重新连接
	return err
}
