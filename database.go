package goi

import (
	"database/sql"
	"reflect"
	"sync"
)

type DBConnectFunc func() *sql.DB

// DatabaseConfig 将数据库专属配置写入指定目标，目标类型由实现校验。
type DatabaseConfig interface {
	GetConfig(target any) error
}

// Database 数据库连接管理器
//
// 字段:
//   - Engine: string 数据库引擎名称
//   - Config: DatabaseConfig 数据库专属配置，默认 nil
//   - Connect: func(string) *sql.DB 连接池创建函数，首次调用 DB 时执行
//
// 说明:
//   - Config 不参与连接池管理，应在初始化 ORM 前设置，使用期间不要修改共享配置
//   - Close 后再次调用 DB 会重新创建连接池
type Database struct {
	Engine  string
	Config  DatabaseConfig
	Connect func(Engine string) *sql.DB // 使用时自动连接
	db      *sql.DB
	mu      sync.RWMutex
}

// GetConfig 委托配置对象写入原始目标，错误原样返回。
// Config 为 nil 或 nil 指针时保留目标默认值；不复制字段或深拷贝共享策略。
func (database *Database) GetConfig(target any) error {
	if database.Config == nil {
		return nil
	}
	value := reflect.ValueOf(database.Config)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil
	}
	return database.Config.GetConfig(target)
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
