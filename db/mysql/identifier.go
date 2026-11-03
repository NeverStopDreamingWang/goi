package mysql

import "github.com/NeverStopDreamingWang/goi/v3/db"

// identifierQuoter 保存后续新建 MySQL Engine 使用的标识符引用器。
var identifierQuoter db.IdentifierQuoter = db.NewIdentifierQuoter("`", "`")

// SetIdentifierQuoter 设置后续 MySQL Engine 使用的标识符引用器。
//
// 参数:
//   - quoter: db.IdentifierQuoter 默认或自定义的标识符引用器；nil 时保留原配置
//
// 说明:
//   - 仅影响新建 Engine，已有实例不变
//   - 须在连接前配置，不可与 Connect 或 ConnectDatabase 并发调用
//   - 注册后的实现不应再修改，共享状态的并发安全由实现者保证
func SetIdentifierQuoter(quoter db.IdentifierQuoter) {
	if quoter == nil {
		return
	}
	identifierQuoter = quoter
}

// GetIdentifierQuoter 获取当前 MySQL 包级标识符引用器。
//
// 返回:
//   - db.IdentifierQuoter: 当前 MySQL 包级标识符引用器
//
// 说明:
//   - 默认使用 MySQL 反引号；Engine 在创建时保存引用器
func GetIdentifierQuoter() db.IdentifierQuoter {
	return identifierQuoter
}
