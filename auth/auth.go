package auth

import (
	"crypto/rand"

	"github.com/NeverStopDreamingWang/goi/crypto/pbkdf2_sha256"
)

// MakePassword 生成密码的哈希值
func MakePassword(password string) (string, error) {
	salt := make([]byte, 16) // 生成 16 字节的盐值
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	iterations := 870000
	return pbkdf2_sha256.Encode(password, salt, iterations), nil
}

// CheckPassword 验证密码
func CheckPassword(password, encoded string) bool {
	var err error
	var Info pbkdf2_sha256.Crypto

	err = pbkdf2_sha256.Decode(encoded, &Info)
	if err != nil {
		return false
	}
	encoded_2 := pbkdf2_sha256.Encode(password, Info.Salt, Info.Iterations)
	return encoded == encoded_2
}
