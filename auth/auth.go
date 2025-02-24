package auth

import (
	"crypto/rand"
	"errors"

	"github.com/NeverStopDreamingWang/goi/crypto/pbkdf2_sha256"
)

// MakePassword 使用 PBKDF2-SHA256 算法对密码进行加密
//
// 参数:
//   - password string: 原始密码字符串
//
// 返回:
//   - string: 加密后的密码字符串，格式为: <algorithm>$<iterations>$<salt>$<hash>
//   - error: 错误信息，如果密码为空或生成盐值失败时返回错误
func MakePassword(password string) (string, error) {
	// 验证密码不为空
	if password == "" {
		return "", errors.New("密码不能为空")
	}

	// 生成 16 字节的随机盐值
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	// 设置迭代次数为 870000
	iterations := 870000

	// 使用 PBKDF2-SHA256 算法加密密码
	return pbkdf2_sha256.Encode(password, salt, iterations), nil
}

// CheckPassword 验证密码是否匹配
//
// 参数:
//   - password string: 待验证的原始密码
//   - encoded string: 已加密的密码字符串
//
// 返回:
//   - bool: 如果密码匹配返回 true，否则返回 false
func CheckPassword(password, encoded string) bool {
	var err error
	var Info pbkdf2_sha256.Crypto

	// 解析加密的密码字符串
	err = pbkdf2_sha256.Decode(encoded, &Info)
	if err != nil {
		return false
	}

	// 使用相同的盐值和迭代次数加密待验证的密码
	encoded_2 := pbkdf2_sha256.Encode(password, Info.Salt, Info.Iterations)

	// 比较两个加密结果是否相同
	return encoded == encoded_2
}
