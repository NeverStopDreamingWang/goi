package example

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"math/big"
	"os"
	"path"
	"time"
)

func init() {
	SSLPath := path.Join(Server.Settings.BASE_DIR, "ssl")
	if Server.Settings.SSL.STATUS == false {
		return
	}

	_, certErr := os.Stat(Server.Settings.SSL.CERT_PATH)
	_, keyErr := os.Stat(Server.Settings.SSL.KEY_PATH)
	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		generateCertificate(SSLPath)
	}
}

// 生成自签证书
func generateCertificate(SSLPath string) {
	var err error
	_, err = os.Stat(SSLPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(SSLPath, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建 SSL 目录错误: ", err))
		}
	}

	// 生成 128 位随机序列号
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}

	// 创建证书请求模板
	certificateTemplate := x509.Certificate{
		SerialNumber: serialNumber, // 证书本身的序列号
		Subject: pkix.Name{
			Country:            []string{"CN"},  // 国家
			Organization:       []string{"goi"}, // 证书持有者的组织名称
			OrganizationalUnit: nil,             // 证书持有者的组织单元名称
			Locality:           nil,             // 证书持有者所在地区（城市）的名称
			Province:           nil,             // 证书持有者所在省/州的名称
			StreetAddress:      nil,             // 证书持有者的街道地址
			PostalCode:         nil,             // 证书持有者的邮政编码
			SerialNumber:       "",              // 证书持有者的序列号
			CommonName:         "example",       //  证书的通用名称
		},
		NotBefore:             time.Now().In(Server.Settings.LOCATION),
		NotAfter:              time.Now().In(Server.Settings.LOCATION).Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	err = goi.GenerateRSACertificate(2048, certificateTemplate, SSLPath) // RSA 算法
	// err = GenerateECCCertificate(certificateTemplate, SSLPath) // ECC 算法
	if err != nil {
		panic(fmt.Sprintf("生成SSL证书和私钥时发生错误: ", err))
	}
}
