package kingbase

import (
	"errors"
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// CompatibilityMode 标识 Kingbase 的数据库兼容模式，不用于设置服务端参数。
type CompatibilityMode string

const (
	ModePostgreSQL CompatibilityMode = "postgres"  // PostgreSQL 兼容模式
	ModeOracle     CompatibilityMode = "oracle"    // Oracle 兼容模式
	ModeMySQL      CompatibilityMode = "mysql"     // MySQL 兼容模式
	ModeSQLServer  CompatibilityMode = "sqlserver" // SQL Server 兼容模式
)

// Config 定义 Kingbase ORM 的连接级配置，可赋给 goi.Database.Config。
// 策略仅影响本次创建的实例，不修改其它数据库包的默认配置。
type Config struct {
	Mode          CompatibilityMode // 服务端已启用的兼容模式，空值按 PostgreSQL 模式处理
	EngineOptions db.EngineOptions  // 当前连接的策略，非 nil 字段覆盖数据库包级配置
}

// GetConfig 将配置写入目标
//
// 参数:
//   - target: any 非 nil 的 *Config
//
// 返回:
//   - error: 目标为 nil 或类型不匹配时的错误
//
// 说明:
//   - 按值复制配置，策略接口仍引用原对象，不做深拷贝
func (config Config) GetConfig(target any) error {
	result, ok := target.(*Config)
	if target == nil || ok && result == nil {
		return errors.New(i18n.T("params.params_is_not_null", map[string]any{"name": "target"}))
	}
	if !ok {
		return errors.New(i18n.T("db.config_type_is_not_match", map[string]any{
			"engine": driverName,
			"type":   fmt.Sprintf("%T", target),
		}))
	}
	*result = config
	return nil
}
