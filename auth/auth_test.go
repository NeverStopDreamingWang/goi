package auth

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

// 调试
func TestAuth(t *testing.T) {
	// 原始密码
	password := "goi123456"

	// 生成密码的哈希值
	hashedPassword, err := MakePassword(password, bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("密码加密失败:", err)
		return
	}

	// 输出加密后的密码
	fmt.Println("加密后的密码:", hashedPassword)

	// 验证密码
	isValid := CheckPassword(password, hashedPassword)
	if isValid {
		fmt.Println("密码验证成功")
	} else {
		fmt.Println("密码验证失败")
	}
}
