package db

// EngineOptions 设置单个 Engine 的策略，nil 字段保留原配置。
// 连接级选项覆盖数据库包级配置；列解析器未单独配置时使用 db 包级规则。
// 策略对象只保存引用，自定义实现的共享状态由调用方管理。
type EngineOptions struct {
	PlaceholderParser PlaceholderParser // 占位符解析器
	IdentifierQuoter  IdentifierQuoter  // 标识符引用器
	ColumnParser      ColumnParser      // 列定义解析器
}

// Merge 返回合并后的副本，按传入顺序覆盖非 nil 字段，不修改原配置。
func (options EngineOptions) Merge(overrides ...EngineOptions) EngineOptions {
	for _, override := range overrides {
		if override.PlaceholderParser != nil {
			options.PlaceholderParser = override.PlaceholderParser
		}
		if override.IdentifierQuoter != nil {
			options.IdentifierQuoter = override.IdentifierQuoter
		}
		if override.ColumnParser != nil {
			options.ColumnParser = override.ColumnParser
		}
	}
	return options
}
