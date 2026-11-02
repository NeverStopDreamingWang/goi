package oracle

import "github.com/NeverStopDreamingWang/goi/v2/db"

// identifierQuoter 保存后续新建 Oracle Engine 使用的标识符引用器。
var identifierQuoter db.IdentifierQuoter = db.NewIdentifierQuoter("\"", "\"")

// SetIdentifierQuoter 设置全局标识符引用器
//
// 参数:
//   - quoter: db.IdentifierQuoter 默认或自定义引用器；nil 时忽略，保留原配置
//
// 说明:
//   - 仅影响新建 Engine，已有 Engine 继续使用原对象
//   - 连接前完成配置，不可与连接操作并发修改
//   - 注册后不应修改引用器；自定义实现负责内部并发安全
func SetIdentifierQuoter(quoter db.IdentifierQuoter) {
	if quoter == nil {
		return
	}
	identifierQuoter = quoter
}

// GetIdentifierQuoter 获取当前全局标识符引用器
//
// 返回:
//   - db.IdentifierQuoter: 当前引用器
//
// 说明:
//   - 默认使用 Oracle 双引号
func GetIdentifierQuoter() db.IdentifierQuoter {
	return identifierQuoter
}
