package goi

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"path"
)

func GenerateRSACertificate(certificateTemplate x509.Certificate, outPath string) error {
	// 生成 RSA 密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	privateKeyFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".key"))
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()

	// 将 RSA 私钥编码为 PEM 格式并写入文件
	err = pem.Encode(privateKeyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})
	if err != nil {
		return err
	}

	// 使用证书请求模板和密钥生成证书
	certificateDER, err := x509.CreateCertificate(rand.Reader, &certificateTemplate, &certificateTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certificateFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".crt"))
	if err != nil {
		return err
	}
	defer certificateFile.Close()

	err = pem.Encode(certificateFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	if err != nil {
		return err
	}

	certificatePemFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".pem"))
	if err != nil {
		return err
	}
	defer certificatePemFile.Close()

	err = pem.Encode(certificatePemFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	if err != nil {
		return err
	}

	// 生成证书签名请求 (CSR)
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{CommonName: certificateTemplate.Subject.CommonName},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return err
	}

	csrFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".csr"))
	if err != nil {
		return err
	}
	defer csrFile.Close()

	err = pem.Encode(csrFile, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	if err != nil {
		return err
	}

	return nil
}

func GenerateECCCertificate(certificateTemplate x509.Certificate, outPath string) error {
	// 生成 ECC 密钥对
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return err
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}

	privateKeyFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".key"))
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()

	// 将 ECC 私钥编码为 PEM 格式并写入文件
	err = pem.Encode(privateKeyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes})
	if err != nil {
		return err
	}

	// 使用证书请求模板和密钥生成证书
	certificateDER, err := x509.CreateCertificate(rand.Reader, &certificateTemplate, &certificateTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certificateFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".crt"))
	if err != nil {
		return err
	}
	defer certificateFile.Close()

	// 将证书编码为 PEM 格式并写入文件
	err = pem.Encode(certificateFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	if err != nil {
		return err
	}

	certificatePemFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".pem"))
	if err != nil {
		return err
	}
	defer certificatePemFile.Close()

	err = pem.Encode(certificatePemFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	if err != nil {
		return err
	}

	// 生成证书签名请求 (CSR)
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{CommonName: certificateTemplate.Subject.CommonName},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return err
	}

	csrFile, err := os.Create(path.Join(outPath, certificateTemplate.Subject.CommonName+".csr"))
	if err != nil {
		return err
	}
	defer csrFile.Close()

	err = pem.Encode(csrFile, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	if err != nil {
		return err
	}

	return nil
}
