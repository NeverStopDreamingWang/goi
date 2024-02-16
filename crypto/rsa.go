package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func GenerateRSA(bits int, privateKeyBytes *[]byte, publicKeyBytes *[]byte) error {
	// 生成 RSA 密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, bits) // bits 2048
	if err != nil {
		return err
	}

	// 将私钥编码为PEM格式
	privateKeyPEMBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyPEMBytes,
	}
	*privateKeyBytes = pem.EncodeToMemory(privateKeyPEM)

	// 将公钥编码为PEM格式
	publicKey := &privateKey.PublicKey
	publicKeyPEMBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPEMBytes,
	}
	*publicKeyBytes = pem.EncodeToMemory(publicKeyPEM)
	return nil
}
