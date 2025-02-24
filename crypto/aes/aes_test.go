package aes_test

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/crypto/aes"
)

func ExampleEncrypt() {
	plainText := "Hello, World!"
	key := "my-secret-key-123"

	encrypted, err := aes.Encrypt(plainText, []byte(key))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	decrypted, err := aes.Decrypt(encrypted, []byte(key))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("原文:", plainText)
	// 由于加密结果每次都不同（因为IV是随机的），所以这里不输出加密结果
	fmt.Println("解密:", decrypted)

	// Output:
	//
	// 原文: Hello, World!
	// 解密: Hello, World!
}

func ExampleDecrypt() {
	plainText := "Hello, World!"
	key := "my-secret-key-123"

	encrypted, err := aes.Encrypt(plainText, []byte(key))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	decrypted, err := aes.Decrypt(encrypted, []byte(key))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("原文:", plainText)
	// 由于加密结果每次都不同（因为IV是随机的），所以这里不输出加密结果
	fmt.Println("解密:", decrypted)

	// Output:
	//
	// 原文: Hello, World!
	// 解密: Hello, World!
}
