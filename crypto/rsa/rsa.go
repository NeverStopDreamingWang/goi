package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// GenerateKey 生成 RSA 密钥对
//
// 参数:
//   - bits int: 密钥长度（建议2048或4096位）
//
// 返回:
//   - []byte: PEM格式的私钥
//   - []byte: PEM格式的公钥
//   - error: 生成过程中的错误
//
// 生成过程:
//  1. 生成RSA密钥对
//  2. 将私钥转换为PKCS1格式
//  3. 将公钥转换为PKIX格式
//  4. 分别编码为PEM格式
func GenerateKey(bits int) (privateKey []byte, publicKey []byte, err error) {
	// 生成 RSA 密钥对
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}

	// 将私钥编码为 PEM 格式
	privateKey = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaPrivateKey),
	})

	// 将公钥编码为 PEM 格式
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&rsaPrivateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	publicKey = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKey, publicKey, nil
}

// Encrypt 使用RSA公钥加密数据
//
// 参数:
//   - publicKeyBytes []byte: PEM格式的公钥数据
//   - plaintext []byte: 需要加密的原文数据
//
// 返回:
//   - []byte: 加密后的密文数据
//   - error: 加密过程中的错误
//
// 加密过程:
//  1. 解码PEM格式的公钥
//  2. 解析PKIX格式的公钥
//  3. 使用OAEP填充方案加密(SHA-256哈希)
func Encrypt(publicKeyBytes []byte, plaintext []byte) ([]byte, error) {
	// 解析PEM格式的公钥
	block, _ := pem.Decode(publicKeyBytes)
	if block == nil {
		errMsg := i18n.T("crypto.rsa.public_key_decode_error")
		return nil, errors.New(errMsg)
	}

	// 解析公钥
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		errMsg := i18n.T("crypto.rsa.public_key_parse_error", map[string]any{
			"err": err,
		})
		return nil, errors.New(errMsg)
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		errMsg := i18n.T("crypto.rsa.public_key_type_error")
		return nil, errors.New(errMsg)
	}

	// 使用OAEP加密(SHA-256哈希)
	hash := sha256.New()
	return rsa.EncryptOAEP(hash, rand.Reader, publicKey, plaintext, nil)
}

// Decrypt 使用RSA私钥解密数据
//
// 参数:
//   - privateKeyBytes []byte: PEM格式的私钥数据
//   - ciphertext []byte: 需要解密的密文数据
//
// 返回:
//   - []byte: 解密后的原文数据
//   - error: 解密过程中的错误
//
// 解密过程:
//  1. 解码PEM格式的私钥
//  2. 解析PKCS1格式的私钥
//  3. 使用OAEP填充方案解密(SHA-256哈希)
func Decrypt(privateKeyBytes []byte, ciphertext []byte) ([]byte, error) {
	// 解析PEM格式的私钥
	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		errMsg := i18n.T("crypto.rsa.private_key_decode_error")
		return nil, errors.New(errMsg)
	}

	// 解析私钥
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		errMsg := i18n.T("crypto.rsa.private_key_parse_error", map[string]any{
			"err": err,
		})
		return nil, errors.New(errMsg)
	}

	// 使用OAEP解密(SHA-256哈希)
	hash := sha256.New()
	return rsa.DecryptOAEP(hash, rand.Reader, privateKey, ciphertext, nil)
}
