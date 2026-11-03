package sqlite3

import (
	"errors"
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// Config 定义连接级策略，可赋给 goi.Database.Config。
// 非 nil 字段优先于数据库包级配置，不修改全局默认值。
type Config struct {
	EngineOptions db.EngineOptions
}

// GetConfig 将配置浅复制到非 nil 的 *Config，策略接口保留原对象引用。
// 目标为 nil 或类型不匹配时返回本地化错误。
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
