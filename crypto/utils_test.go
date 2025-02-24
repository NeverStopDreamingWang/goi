package crypto_test

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/crypto"
)

func ExampleGenerateRandomString() {
	randomStr, err := crypto.GenerateRandomString(16, crypto.CHARSET)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(randomStr)

	// Output:
	//
	// cl%4_08!l%&hpup9
}

func ExampleGenerateRandomSecretKey() {
	SecretKey, err := crypto.GenerateRandomSecretKey()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(SecretKey)

	// Output:
	//
	// goi-insecure-6!p6pwq&x!+x-7m9ta=vc9u4gh$#-49+5b=i2t6qb=j%f^)34t
}
