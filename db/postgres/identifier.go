package postgres

import "github.com/NeverStopDreamingWang/goi/v2/db"

// identifierQuoter 保存后续新建 PostgreSQL Engine 使用的标识符引用器。
var identifierQuoter db.IdentifierQuoter = db.NewIdentifierQuoter("\"", "\"")

// SetIdentifierQuoter 设置后续新建实例使用的标识符引用器
//
// 参数:
//   - quoter: db.IdentifierQuoter 默认或自定义引用器；nil 时忽略，保留原配置
//
// 说明:
//   - 在连接前设置，勿与 Connect、ConnectDatabase 并发调用；已有实例不变
//   - 注册后不要修改引用器；自定义实现需保证内部并发安全
func SetIdentifierQuoter(quoter db.IdentifierQuoter) {
	if quoter == nil {
		return
	}
	identifierQuoter = quoter
}

// GetIdentifierQuoter 获取当前包级标识符引用器
//
// 返回:
//   - db.IdentifierQuoter: 当前引用器，默认使用双引号
//
// 说明:
//   - 新建实例保存当前引用器，后续包级替换不影响已有实例
func GetIdentifierQuoter() db.IdentifierQuoter {
	return identifierQuoter
}
