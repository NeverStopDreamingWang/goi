package crypto_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3/crypto"
)

func ExampleGenerateRandomString() {
	randomStr, err := crypto.GenerateRandomString(16, crypto.Charset)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("length:", len(randomStr))

	// Output:
	// length: 16
}

func ExampleGenerateRandomSecretKey() {
	secretKey, err := crypto.GenerateRandomSecretKey()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("prefix:", strings.HasPrefix(secretKey, crypto.SecretKeyInsecurePrefix))
	fmt.Println("random length:", len(strings.TrimPrefix(secretKey, crypto.SecretKeyInsecurePrefix)))

	// Output:
	// prefix: true
	// random length: 50
}

func TestGenerateRandomString(t *testing.T) {
	for _, test := range []struct {
		name    string
		length  int
		charset string
	}{
		{"zero_length", 0, crypto.Charset},
		{"one_character", 1, crypto.Charset},
		{"default_charset", 64, crypto.Charset},
		{"custom_charset", 32, "ab09"},
		{"single_character_charset", 16, "x"},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, err := crypto.GenerateRandomString(test.length, test.charset)
			if err != nil {
				t.Fatal(err)
			}
			if len(value) != test.length {
				t.Fatalf("length = %d, want %d", len(value), test.length)
			}
			assertAllowedCharacters(t, value, test.charset)
		})
	}
}

func TestGenerateRandomSecretKey(t *testing.T) {
	key, err := crypto.GenerateRandomSecretKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, crypto.SecretKeyInsecurePrefix) {
		t.Fatal("secret key is missing its prefix")
	}
	randomPart := strings.TrimPrefix(key, crypto.SecretKeyInsecurePrefix)
	if len(randomPart) != crypto.SecretKeyMinLength {
		t.Fatalf("random length = %d, want %d", len(randomPart), crypto.SecretKeyMinLength)
	}
	assertAllowedCharacters(t, randomPart, crypto.Charset)
}

func assertAllowedCharacters(t *testing.T, value string, charset string) {
	t.Helper()
	for index := range value {
		if !strings.ContainsRune(charset, rune(value[index])) {
			t.Fatalf("character at index %d is not in the allowed charset", index)
		}
	}
}
