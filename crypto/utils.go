package crypto

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	SECRET_KEY_INSECURE_PREFIX = "goi-insecure-"
	SECRET_KEY_MIN_LENGTH      = 50
)

// GenerateRandomString generates a securely generated random string.
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

func GenerateRandomSecretKey() (string, error) {
	// 字符集：大写字母、小写字母、数字和特殊字符
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*(-_=+)"
	return GenerateRandomString(SECRET_KEY_MIN_LENGTH, charset)
}

func GenerateRead(Bytes []byte) error {
	// 生成所随机字节
	_, err := rand.Read(Bytes)
	if err != nil {
		return err
	}
	return nil
}
