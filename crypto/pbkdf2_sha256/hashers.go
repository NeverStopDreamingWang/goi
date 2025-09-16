package pbkdf2_sha256

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
	"golang.org/x/crypto/pbkdf2"
)

const Algorithm = "pbkdf2_sha256"

type Crypto struct {
	Algorithm  string
	Hash       string
	Iterations int
	Salt       []byte
}

func Encode(password string, salt []byte, iterations int) string {
	// 使用 PBKDF2 生成密码哈希
	hash := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// 将盐值和哈希一起编码为一个字符串，方便存储
	return fmt.Sprintf("%s$%d$%s$%s", Algorithm, iterations, base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(hash))
}

func Decode(encoded string, Info *Crypto) error {
	var err error
	parts := strings.SplitN(encoded, "$", 4)
	if len(parts) != 4 {
		errMsg := i18n.T("crypto.pbkdf2.invalid_format")
		return errors.New(errMsg)
	}

	Info.Algorithm = parts[0]
	iterations64, err := strconv.ParseInt(parts[1], 10, 64) // 10 表示十进制，64 表示结果为 int64
	if err != nil {
		errMsg := i18n.T("crypto.pbkdf2.invalid_iterations", map[string]interface{}{
			"err": err,
		})
		return errors.New(errMsg)
	}

	Info.Iterations = int(iterations64)
	Info.Hash = parts[3]

	Info.Salt, err = base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		errMsg := i18n.T("crypto.pbkdf2.invalid_salt", map[string]interface{}{
			"err": err,
		})
		return errors.New(errMsg)
	}
	return nil
}
