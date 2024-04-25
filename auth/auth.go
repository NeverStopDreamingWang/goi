package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// MakePassword 生成密码的哈希值
func MakePassword(password string, cost int) (string, error) {
	// 生成随机盐值并加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword 验证密码
func CheckPassword(password, hashedPassword string) bool {
	// 验证密码是否匹配
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
