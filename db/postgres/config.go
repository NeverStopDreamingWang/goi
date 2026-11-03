package postgres

import (
	"errors"
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// Config 定义 PostgreSQL ORM 的连接级策略，可赋给 goi.Database.Config。
// 非 nil 策略覆盖包级配置，仅影响本次创建的实例。
type Config struct {
	EngineOptions db.EngineOptions
}

// GetConfig 将配置写入非 nil 的 *Config 目标。
// 按值复制配置，策略接口仍引用原对象，不做深拷贝。
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
