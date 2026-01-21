package response

// Data 自定义标准的API响应格式
//
// 字段:
//   - Code int: 响应状态码
//   - Message string: 响应消息
//   - Data any: 响应数据
type Data struct {
	Code    int    `json:"code"`    // 响应状态码
	Message string `json:"message"` // 响应消息
	Data    any    `json:"data"`    // 响应数据
}

// Result 自定义标准的API响应格式
//
// 字段:
//   - Code int: 响应状态码
//   - Message string: 响应消息
//   - Result any: 响应数据
type Result struct {
	Code    int    `json:"code"`    // 响应状态码
	Message string `json:"message"` // 响应消息
	Result  any    `json:"result"`  // 响应数据
}
