package crypto

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// 常量定义
const (
	// CHARSET 定义了用于生成随机字符串的字符集
	// 包含小写字母、数字和特殊字符
	CHARSET = "abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*(-_=+)"

	// SECRET_KEY_INSECURE_PREFIX 定义密钥的前缀标识
	SECRET_KEY_INSECURE_PREFIX = "goi-insecure-"

	// SECRET_KEY_MIN_LENGTH 定义了密钥的最小长度要求
	SECRET_KEY_MIN_LENGTH = 50
)

// GenerateRandomString 生成指定长度的随机字符串
//
// 参数:
//   - length int: 要生成的随机字符串长度
//   - allowedChars string: 允许使用的字符集
//
// 返回:
//   - string: 生成的随机字符串
//   - error: 生成过程中的错误
//
// 生成过程:
//  1. 预分配内存以提高性能
//  2. 使用密码学安全的随机数生成器
//  3. 从允许的字符集中随机选择字符
//  4. 拼接生成最终的随机字符串
func GenerateRandomString(length int, allowedChars string) (string, error) {
	var builder strings.Builder
	builder.Grow(length) // 预分配内存，提高性能

	maxIndex := big.NewInt(int64(len(allowedChars)))

	// 根据长度生成随机字符串
	for i := 0; i < length; i++ {
		// 生成一个 [0, len(allowedChars)-1] 范围内的随机数
		index, err := rand.Int(rand.Reader, maxIndex)
		if err != nil {
			return "", err
		}

		// 将随机字符添加到 builder 中
		builder.WriteByte(allowedChars[index.Int64()])
	}

	return builder.String(), nil
}

// GenerateRandomSecretKey 生成随机密钥字符串
//
// 返回:
//   - string: 生成的密钥字符串，包含不安全前缀
//   - error: 生成过程中的错误
//
// 生成过程:
//  1. 使用 CHARSET 字符集生成随机字符串
//  2. 添加不安全前缀标识
//  3. 确保密钥长度符合最小长度要求
func GenerateRandomSecretKey() (string, error) {
	randomStr, err := GenerateRandomString(SECRET_KEY_MIN_LENGTH, CHARSET)
	if err != nil {
		return "", err
	}
	return SECRET_KEY_INSECURE_PREFIX + randomStr, nil
}

// GenerateRead 生成随机字节序列
//
// 参数:
//   - Bytes []byte: 用于存储随机字节的切片，长度决定生成的字节数
//
// 返回:
//   - error: 生成过程中的错误
//
// 说明:
//   - 使用密码学安全的随机数生成器
//   - 生成的随机字节直接写入传入的切片
func GenerateRead(Bytes []byte) error {
	// 生成随机字节
	_, err := rand.Read(Bytes)
	if err != nil {
		return err
	}
	return nil
}
