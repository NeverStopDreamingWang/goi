package auth_test

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/auth"
)

func ExampleMakePassword() {
	// 原始密码
	password := "goi123456"

	// 生成密码的哈希值
	hashedPassword, err := auth.MakePassword(password)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("原始密码: %s\n", password)

	// 验证密码
	isValid := auth.CheckPassword(password, hashedPassword)
	fmt.Printf("密码验证: %v\n", isValid)

	// Output:
	//
	// 原始密码: goi123456
	// 密码验证: true
}

func ExampleCheckPassword() {
	// 原始密码
	password := "goi123456"
	// 错误的密码
	wrongPassword := "wrong123456"

	// 生成密码的哈希值
	hashedPassword, err := auth.MakePassword(password)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 验证正确密码
	isValid := auth.CheckPassword(password, hashedPassword)
	fmt.Printf("正确密码验证: %v\n", isValid)

	// 验证错误密码
	isValid = auth.CheckPassword(wrongPassword, hashedPassword)
	fmt.Printf("错误密码验证: %v\n", isValid)

	// Output:
	//
	// 正确密码验证: true
	// 错误密码验证: false
}
