package crypto

import (
	"crypto/rand"
)

func GenerateAES(secretKeyBytes []byte) error {
	// 生成 AES 密钥
	_, err := rand.Read(secretKeyBytes)
	if err != nil {
		return err
	}
	// 将随机字节转换为Base64编码的字符串
	// secretKey = base64.StdEncoding.EncodeToString(randomBytes)
	return nil
}
