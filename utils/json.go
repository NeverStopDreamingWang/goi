package utils

import (
	"encoding/json"
	"os"
)

// LoadJSON 通用的读取 JSON 文件到指定的结构体
//
// 参数:
//   - filePath string: JSON 文件路径
//   - value any: 目标结构体指针
//
// 返回:
//   - error: 错误信息
func LoadJSON(filePath string, value any) error {
	// 打开 JSON 文件
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建 JSON 解码器
	decoder := json.NewDecoder(file)
	// 解码到目标结构
	err = decoder.Decode(value)
	if err != nil {
		return err
	}
	return nil
}

// SaveJSON 通用的将指定的结构体保存到 JSON 文件
//
// 参数:
//   - filePath string: JSON 文件路径
//   - value any: 目标结构体
//
// 返回:
//   - error: 错误信息
func SaveJSON(filePath string, value any) error {
	// 打开 JSON 文件

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用 json 编码器将数据写入文件
	encoder := json.NewEncoder(file)
	err = encoder.Encode(value)
	if err != nil {
		return err
	}
	return nil
}
