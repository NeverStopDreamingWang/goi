# 服务配置、HTTPS、国际化与时间

**简体中文** | [English](configuration.en.md) · [文档目录](README.md)

## 初始化与共享配置

`goi.NewHTTPServer()` 创建服务生命周期对象，并复制创建时的包级 `Settings`、`Cache`、`Validation`、`Log` 指针。修改所指对象的字段仍会共享；重新赋包级变量不会自动更新已有 Engine。它不是完全隔离的多租户容器，也不会自动加载配置文件、创建默认日志或推断项目目录。

先完成日志和设置初始化，再注册应用和启动服务。设置使用普通字段和 map，运行中并发修改不受统一保护。

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
)

func main() {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	goi.Log = goi.NewLogger(filepath.Join(baseDir, "server.log"))
	defer func() {
		if err := goi.Log.Close(); err != nil {
			log.Print(err)
		}
	}()
	server := goi.NewHTTPServer()
	server.Settings.BaseDir = baseDir
	server.Settings.Debug = false
	server.Settings.Network = "tcp"
	server.Settings.BindAddress = "127.0.0.1"
	server.Settings.Port = 8080
	server.Settings.Params.Set("service_name", "demo")
	if err := server.Settings.SetTimeZone("Asia/Shanghai"); err != nil {
		panic(err)
	}
	server.Settings.SetLanguage(goi.EnUS)
	server.Router.Path("ping", "health", goi.ViewSet{GET: func(*goi.Request) any { return "pong" }})
	fmt.Println(goi.GetTime().Format(time.RFC3339))
	// 完成配置后，取消下一行注释以启动服务。
	// server.RunServer()
}
```

此例只完成配置；`RunServer` 会阻塞。日志应在整轮停机结束后由调用方关闭，详见[日志](logging.md)和[生命周期](lifecycle.md)。

## 主要设置

`settings` 类型不直接对外构造，通过 `server.Settings` 或 `goi.Settings` 访问。

| 字段 | 默认值与用途 |
| --- | --- |
| `Debug` | `true`；影响框架调试输出，不代表启用或关闭所有安全策略。 |
| `Network` | `"tcp"`；HTTP 监听使用 `net.Listen`，常用 `tcp`、`tcp4`、`tcp6`，不是 UDP/HTTP3 服务。 |
| `BindAddress` / `Port` | `127.0.0.1` / `8080`；直接组合为监听地址。IPv6 字面地址需带方括号，例如 `[::1]`。 |
| `BindDomain` | 空；用于启动时显示地址，不配置 DNS，也不限制 HTTP Host。 |
| `BaseDir` | 空；应用自行设置根目录。证书路径不会自动拼接它，建议用 `filepath.Join` 明确路径。 |
| `SSL` | 默认不启用 HTTPS；见下一节。 |
| `Databases` | 已初始化的连接配置 map；通过[数据库指南](../db/README.md)注册连接并交给框架管理。 |
| `SecretKey` / `PrivateKey` / `PublicKey` | 默认空字符串，供应用保存密钥设置；不会自动加密请求、启用登录鉴权或配置 TLS 证书。 |
| `Params` | 已初始化的自定义 `Params` map；在配置阶段写入，应用按键读取。 |
| `UseTZ` | `true`；控制 `GetTime` 的时间表示方式。 |

框架核心不规定 YAML/JSON 配置文件格式；CLI 生成项目和示例项目的配置加载属于应用层实现。

## HTTPS

在前述服务配置中设置：

```go
server.Settings.SSL = goi.SSL{
	Enabled:  true,
	CertPath: filepath.Join(baseDir, "certs", "localhost.crt"),
	KeyPath:  filepath.Join(baseDir, "certs", "localhost.key"),
}
```

`RunServer` 在创建监听器前调用 `tls.LoadX509KeyPair`，把证书加载到 `TLSConfig`，随后通过 `ServeTLS` 提供 HTTPS。文件缺失、格式错误或证书与私钥不匹配会导致启动 panic；框架不自动续期、重新加载证书或获取 CA 证书。

`SSL.Type` 不参与核心 `RunServer` 的证书算法选择，实际使用所加载的证书。CLI 生成的应用可以用此字段选择自己的证书生成逻辑。

### 生成本地自签名证书

下面示例创建 `certs/localhost.key`、`.crt`、`.pem` 和 `.csr`。自签名证书不等于浏览器已信任的证书；应自行配置本地信任，或为部署准备适用证书。

```go
package main

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
)

func main() {
	if err := os.MkdirAll("certs", 0700); err != nil {
		panic(err)
	}
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if err := goi.GenerateECCCertificate(template, "certs"); err != nil {
		panic(err)
	}
	fmt.Println("local certificate generated")
}
```

RSA 对应 `GenerateRSACertificate(bits, template, outPath)`，ECC 使用固定的 P-384 曲线。两个函数都使用传入模板自身签名，并非向 CA 提交签发。

输出目录必须存在，文件名来自 `Subject.CommonName`；该值应由本地配置提供，不能直接使用不受信任的请求输入。生成会覆盖同名文件，失败时可能留下部分文件；私钥未加口令保护，调用方负责目录、文件权限和密钥保管。CSR 当前只复制 CommonName，不复制模板中的全部 SAN 等字段。

## 国际化

`server.Settings.SetLanguage(goi.ZhCN)` 和 `SetLanguage(goi.EnUS)` 切换框架内置文案，分别对应 `zh-CN`、`en-US`。它修改的是包级翻译器，不是每个请求独立的语言协商；不会根据 `Accept-Language` 自动切换，也不负责应用业务文案。

请在处理请求前配置，避免运行中并发替换全局翻译器。当前 `SetLanguage` 是值接收者：翻译器会切换，但 `GetLanguage()` 可能仍返回原设置值，不应将其作为翻译器当前状态的可靠查询。

## 时间与时区

| API | 行为 |
| --- | --- |
| `SetTimeZone(name)` | 使用 `time.LoadLocation` 加载时区，失败返回 error；保存配置并设置进程环境变量 `TZ`，不直接替换 `time.Local`。 |
| `GetLocation()` | 返回配置时区；全局 Settings 为 nil 时退回 `time.Local`。 |
| `GetTime()`，`UseTZ=true` | 返回配置时区的当前时间，保留时区信息和实际时间瞬间。 |
| `GetTime()`，`UseTZ=false` | 保留配置时区的墙钟年月日时分秒，但重新标记为 UTC；这可能改变所表示的时间瞬间，不是普通时区转换。 |
| `Localize(t)` | 调用 `t.In(GetLocation())`，只改变时区表示，不改变时间瞬间，也不受 `UseTZ` 的重新标记规则影响。 |

默认时区位置是 `time.Local`。`SetTimeZone` 会在调用当时根据 `UseTZ` 设置 `TZ`；之后单独更改 `UseTZ` 不会再次设置环境变量。跨服务传输、数据库存储和比较时间时，应明确选择“时间瞬间”还是“本地墙钟”，不要混用这两种语义。

## 源码入口

[settings.go](../settings.go) · [time.go](../time.go) · [certificate.go](../certificate.go) · [server.go](../server.go)
