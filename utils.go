package goi

import (
	"encoding/json"
	"os"
)

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

// 通用的读取 JSON 文件到指定的结构体
//
// 参数:
//   - filePath string: JSON 文件路径
//   - value interface{}: 目标结构体指针
func LoadJSON(filePath string, value interface{}) error {
	// 打开 JSON 文件
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	// 创建 JSON 解码器
	decoder := json.NewDecoder(file)
	// 解码到目标结构
	err = decoder.Decode(value)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}
