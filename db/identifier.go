package db

import "strings"

// IdentifierQuoter 为 SQL 表名、列名等添加定界符，可自行实现以替换默认规则。
type IdentifierQuoter interface {
	// QuoteIdentifier 引用单个 SQL 标识符。
	QuoteIdentifier(identifier string) string
	// QuoteIdentifiers 引用并使用逗号连接一组 SQL 标识符。
	QuoteIdentifiers(identifiers []string) string
}

// DefaultIdentifierQuoter 使用创建时指定的定界符引用标识符。
//
// 说明:
//   - 零值仅去除首尾空白，不添加定界符
//   - 不同数据库应分别保存引用器，避免方言互相覆盖
type DefaultIdentifierQuoter struct {
	leftDelimiter  string
	rightDelimiter string
}

// NewIdentifierQuoter 创建使用指定左右定界符的引用器
//
// 参数:
//   - leftDelimiter: string 标识符左定界符
//   - rightDelimiter: string 标识符右定界符
//
// 返回:
//   - DefaultIdentifierQuoter: 定界符固定的引用器
func NewIdentifierQuoter(leftDelimiter string, rightDelimiter string) DefaultIdentifierQuoter {
	return DefaultIdentifierQuoter{
		leftDelimiter:  leftDelimiter,
		rightDelimiter: rightDelimiter,
	}
}

// QuoteIdentifier 为单个 SQL 标识符添加定界符
//
// 参数:
//   - identifier: string 表名或列名等单个标识符
//
// 返回:
//   - string: 添加左右定界符后的标识符
//
// 说明:
//   - 先去除首尾空白，再按定界符的字符集合清理两端，最后添加一层定界符
//   - 名称本身位于首尾的定界符字符也会被删除，不做转义或 SQL 校验
//   - 不拆分 schema.table 或 SQL 表达式；特殊规则需自行实现 IdentifierQuoter
func (quoter DefaultIdentifierQuoter) QuoteIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	identifier = strings.TrimLeft(identifier, quoter.leftDelimiter)
	identifier = strings.TrimRight(identifier, quoter.rightDelimiter)
	return quoter.leftDelimiter + identifier + quoter.rightDelimiter
}

// QuoteIdentifiers 为一组标识符添加定界符并用逗号连接
//
// 参数:
//   - identifiers: []string 标识符列表
//
// 返回:
//   - string: 按原顺序引用并用逗号连接的结果
//
// 说明:
//   - 逐个调用 QuoteIdentifier，跳过空白项；空列表返回空字符串
func (quoter DefaultIdentifierQuoter) QuoteIdentifiers(identifiers []string) string {
	quoted := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		identifier = strings.TrimSpace(identifier)
		if identifier == "" {
			continue
		}
		quoted = append(quoted, quoter.QuoteIdentifier(identifier))
	}
	return strings.Join(quoted, ",")
}
