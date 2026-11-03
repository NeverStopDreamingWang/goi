# 密码哈希与密码学工具

**简体中文** | [English](security.en.md) · [文档目录](README.md)

本页说明当前 `auth`、`crypto` 子包的实际能力和格式，不代表完整的登录系统或安全协议。框架不自动提供 JWT、session、权限检查、登录限流或密钥管理；这些由应用和[中间件](middleware.md)实现。TLS 证书见[配置与 HTTPS](configuration.md)。

## 密码哈希

密码存储使用 `auth.MakePassword` 和 `auth.CheckPassword`，不要用可逆的 AES/RSA 加密替代这里的密码哈希流程。

```go
package main

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3/auth"
)

func main() {
	encoded, err := auth.MakePassword("example-password")
	if err != nil {
		panic(err)
	}
	// 将 encoded 保存到服务端密码字段，不保存明文。
	fmt.Println(auth.CheckPassword("example-password", encoded))
	fmt.Println(auth.CheckPassword("wrong-password", encoded))
	// true
	// false
}
```

当前实现使用 PBKDF2-SHA256、16 字节随机盐、870000 次迭代和 32 字节派生结果，存储格式为：

```text
pbkdf2_sha256$iterations$base64(salt)$base64(hash)
```

| 方法 | 输入与返回 |
| --- | --- |
| `auth.MakePassword(password)` | 空密码返回 error；成功返回包含算法、迭代数、盐和哈希的字符串。 |
| `auth.CheckPassword(password, encoded)` | 解析存储值、按其中的盐和迭代数重新计算，再比较结果；解析失败或不匹配返回 false。 |

验证工作量来自 `encoded` 中的迭代数，因此它应来自受控的服务端存储，不要直接接收客户端提供的完整哈希格式作为验证参数。当前没有自动重哈希、迭代升级或密码强度策略，也没有常量时间字符串比较的实现保证。

底层编码和解析位于 [crypto/pbkdf2_sha256](../crypto/pbkdf2_sha256/hashers.go)；直接使用底层方法时，调用方还需负责参数约束和格式管理。

## 随机数据与加解密示例

下面示例只在内存中生成数据，不打印密钥或明文密码：

```go
package main

import (
	"fmt"

	goicrypto "github.com/NeverStopDreamingWang/goi/v3/crypto"
	goiaes "github.com/NeverStopDreamingWang/goi/v3/crypto/aes"
	goirsa "github.com/NeverStopDreamingWang/goi/v3/crypto/rsa"
)

func main() {
	key := make([]byte, 32)
	if err := goicrypto.GenerateRead(key); err != nil {
		panic(err)
	}
	ciphertext, err := goiaes.Encrypt("example", key)
	if err != nil {
		panic(err)
	}
	plaintext, err := goiaes.Decrypt(ciphertext, key)
	if err != nil {
		panic(err)
	}

	privateKey, publicKey, err := goirsa.GenerateKey(2048)
	if err != nil {
		panic(err)
	}
	rsaCiphertext, err := goirsa.Encrypt(publicKey, []byte("example"))
	if err != nil {
		panic(err)
	}
	rsaPlaintext, err := goirsa.Decrypt(privateKey, rsaCiphertext)
	if err != nil {
		panic(err)
	}

	token, err := goicrypto.GenerateRandomString(32, "abcdefghijklmnopqrstuvwxyz0123456789")
	if err != nil {
		panic(err)
	}
	fmt.Println(plaintext == "example", string(rsaPlaintext) == "example", len(token))
	// true true 32
}
```

## AES 格式与边界

`crypto/aes.Encrypt(plainText string, keyBytes []byte)` 使用 AES-CBC 和 PKCS7 填充，生成随机 IV，将 `IV || ciphertext` 一起作标准 Base64 编码。`Decrypt` 接收该 Base64 字符串和相同密钥，返回原字符串或 error。

密钥处理是本包的自定义行为：

- 16、24、32 字节密钥原样使用。
- 较短或中间长度的密钥补齐到下一个有效长度，补的是字符 `'0'`，不是密码派生函数。
- 超过 32 字节会截断到 32 字节。

示例主动生成 32 字节随机密钥，避免触发隐式补齐或截断。`Settings.SecretKey` 不会自动注入这些方法，调用方需要显式传入密钥并自行保管。

当前 CBC helper 没有 MAC 或认证标签，不能检测所有密文篡改，也不能当作签名或带认证的加密协议。`Decrypt` 并未完整检查密文块长度，错误构造的密文可能触发 panic，而不只是返回 error；不要把“往返示例成功”当作可直接处理任意不可信密文的保证。

## RSA 格式与边界

| 方法 | 格式 |
| --- | --- |
| `rsa.GenerateKey(bits)` | 返回 PEM 私钥、公钥和 error；私钥 DER 为 PKCS1，公钥 DER 为 PKIX。 |
| `rsa.Encrypt(publicKey, plaintext)` | 解析 PKIX 公钥，使用 RSA-OAEP/SHA-256，返回二进制密文。 |
| `rsa.Decrypt(privateKey, ciphertext)` | 解析 PKCS1 私钥，按同一 OAEP 参数解密，返回二进制明文。 |

本包公钥 PEM 标签虽然写作 `RSA PUBLIC KEY`，其正文仍是 PKIX，而不是 PKCS1 公钥；与外部工具互通时应按实际 DER 格式处理。本包不自动解析任意 PKCS8 私钥或证书 PEM，也不提供这里未列出的签名/验签接口。

RSA 方法不做长消息分段，超出 OAEP 可用长度时返回错误。密文是 `[]byte`，用于 JSON 或文本传输时由应用显式选择 Base64 等编码；不要把任意二进制直接当成可显示字符串。生成的私钥不带口令加密，持久化和访问控制由调用方负责。

## 随机工具

| 方法 | 行为 |
| --- | --- |
| `GenerateRead(buffer)` | 使用 `crypto/rand.Read` 填满已有 byte 切片；切片长度决定输出字节数，返回 error。 |
| `GenerateRandomString(length, allowedChars)` | 使用密码学随机索引逐字节选择字符；调用方应保证 length 非负，字符集非空且适合按字节使用。 |
| `GenerateRandomSecretKey()` | 使用 `Charset` 生成 `SecretKeyMinLength` 个字符，再加上 `SecretKeyInsecurePrefix`；这是实际返回格式，不代表框架会自动校验或轮换该设置。 |

`allowedChars` 按 byte 索引，多字节字符集合可能产生无效 UTF-8；示例使用 ASCII 字符集。不符合长度或字符集前提的输入未统一转换为 error，可能 panic。所有示例数据和模板中的固定密钥都不应直接作为部署凭据。

## 源码与相关指南

[auth/auth.go](../auth/auth.go) · [crypto/aes/aes.go](../crypto/aes/aes.go) · [crypto/rsa/rsa.go](../crypto/rsa/rsa.go) · [crypto/utils.go](../crypto/utils.go)

[服务配置与 HTTPS](configuration.md) · [中间件](middleware.md) · [请求与响应](http.md)
