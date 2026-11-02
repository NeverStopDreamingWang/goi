package mysql

import (
	"errors"
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v2/db"
	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// Config 定义 MySQL ORM 的连接级配置，可赋给 goi.Database.Config。
// 非 nil 策略覆盖包级配置，不修改其它连接或全局默认值。
type Config struct {
	EngineOptions db.EngineOptions // 当前连接的标识符、占位符和列解析策略
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
