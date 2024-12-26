package goi

import (
	"encoding/json"
	"os"
)

// 通用的 JSON 文件加载函数
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
