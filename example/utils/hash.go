package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func CalculateSHA256(file io.Reader) (string, error) {
	// 创建一个 SHA256 hash 对象
	hash := sha256.New()

	// 将文件内容复制到 hash 对象
	_, err := io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	// 获取文件的 SHA256 哈希值并转换为字符串
	hashInBytes := hash.Sum(nil)
	sha256String := hex.EncodeToString(hashInBytes)

	return sha256String, nil
}
