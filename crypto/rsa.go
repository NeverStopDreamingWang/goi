package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

// GenerateRSA 生成 RSA 密钥对并返回 PEM 格式的密钥字节
// bits: 密钥长度(建议 2048 或 4096)
// 返回值: 私钥字节, 公钥字节, 错误信息
func GenerateRSA(bits int) (privateKey []byte, publicKey []byte, err error) {
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
