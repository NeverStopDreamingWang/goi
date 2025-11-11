package goi

// Data 自定义标准的API响应格式
//
// 字段:
//   - Code int: 响应状态码
//   - Message string: 响应消息
//   - Results interface{}: 响应数据
type Data struct {
	Code    int         `json:"code"`    // 响应状态码
	Message string      `json:"message"` // 响应消息
	Results interface{} `json:"results"` // 响应数据
}
