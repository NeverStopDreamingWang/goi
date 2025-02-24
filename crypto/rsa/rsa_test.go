package rsa_test

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/crypto/rsa"
)

func ExampleGenerateKey() {
	// 生成2048位的RSA密钥对
	_, _, err := rsa.GenerateKey(2048)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("私钥类型: %s\n", "RSA PRIVATE KEY")
	fmt.Printf("公钥类型: %s\n", "RSA PUBLIC KEY")

	// Output:
	// 私钥类型: RSA PRIVATE KEY
	// 公钥类型: RSA PUBLIC KEY
}

func ExampleEncrypt() {
	// 生成密钥对
	privateKey, publicKey, err := rsa.GenerateKey(2048)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 原文
	plaintext := []byte("Hello, RSA!")

	// 加密
	encrypted, err := rsa.Encrypt(publicKey, plaintext)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 解密
	decrypted, err := rsa.Decrypt(privateKey, encrypted)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("原文:", string(plaintext))
	// 由于加密结果每次都不同，所以这里不输出加密结果
	fmt.Println("解密:", string(decrypted))

	// Output:
	// 原文: Hello, RSA!
	// 解密: Hello, RSA!
}

func ExampleDecrypt() {
	// 生成密钥对
	privateKey, publicKey, err := rsa.GenerateKey(2048)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 原文
	plaintext := []byte("Hello, RSA!")

	// 加密
	encrypted, err := rsa.Encrypt(publicKey, plaintext)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 解密
	decrypted, err := rsa.Decrypt(privateKey, encrypted)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("原文:", string(plaintext))
	// 由于加密结果每次都不同，所以这里不输出加密结果
	fmt.Println("解密:", string(decrypted))

	// Output:
	// 原文: Hello, RSA!
	// 解密: Hello, RSA!
}
