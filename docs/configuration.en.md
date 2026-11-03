# Server Configuration, HTTPS, Language, and Time

[简体中文](configuration.md) | **English** · [Documentation](README.en.md)

## Initialization and Shared Settings

`goi.NewHTTPServer()` creates lifecycle state and copies the package-level `Settings`, `Cache`, `Validation`, and `Log` pointers at construction time. Mutations of the referenced objects remain shared; reassigning a package variable does not automatically update an existing Engine. An Engine is not a fully isolated multi-tenant container and does not load a configuration file, create a default logger, or infer the project directory.

Initialize logging and settings before registering applications and serving requests. Ordinary fields and maps have no common protection against concurrent configuration changes.

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
	// Uncomment after configuration to start serving.
	// server.RunServer()
}
```

This example only configures the server; `RunServer` blocks. The caller closes logging after shutdown completes. See [Logging](logging.en.md) and [Lifecycle](lifecycle.en.md).

## Main Settings

The `settings` type is not constructed directly by applications. Access it through `server.Settings` or `goi.Settings`.

| Field | Default and purpose |
| --- | --- |
| `Debug` | `true`; affects framework debug output, not every security policy. |
| `Network` | `"tcp"`; HTTP uses `net.Listen`, normally `tcp`, `tcp4`, or `tcp6`. It is not a UDP/HTTP3 server. |
| `BindAddress` / `Port` | `127.0.0.1` / `8080`; combined directly into the listen address. IPv6 literals need brackets, for example `[::1]`. |
| `BindDomain` | Empty; used in startup address output. It neither configures DNS nor restricts HTTP Host. |
| `BaseDir` | Empty; set the application root explicitly. Certificate paths are not automatically joined to it; use `filepath.Join` for explicit paths. |
| `SSL` | HTTPS is disabled by default; see below. |
| `Databases` | An initialized connection configuration map. Register connections through the [Database Guide (CN)](../db/README.md) and use the framework lifecycle. |
| `SecretKey` / `PrivateKey` / `PublicKey` | Empty by default. Application key settings do not automatically encrypt requests, enable authentication, or configure TLS certificates. |
| `Params` | An initialized custom `Params` map. Populate it during configuration and read values in application code. |
| `UseTZ` | `true`; controls the representation returned by `GetTime`. |

The core does not prescribe a YAML/JSON configuration file. Configuration loading in generated and example projects belongs to application code.

## HTTPS

Add this to the server configuration above:

```go
server.Settings.SSL = goi.SSL{
	Enabled:  true,
	CertPath: filepath.Join(baseDir, "certs", "localhost.crt"),
	KeyPath:  filepath.Join(baseDir, "certs", "localhost.key"),
}
```

Before opening the listener, `RunServer` calls `tls.LoadX509KeyPair`, stores the certificate in `TLSConfig`, and then serves HTTPS with `ServeTLS`. Missing files, invalid formats, or mismatched keys cause a startup panic. The framework does not renew or reload certificates or obtain CA-issued certificates automatically.

Core `RunServer` does not use `SSL.Type` to choose a certificate algorithm; it uses the loaded certificate. Generated application code may use this field for its own certificate-generation choice.

### Generate a Local Self-signed Certificate

The following example creates `certs/localhost.key`, `.crt`, `.pem`, and `.csr`. A self-signed certificate is not automatically trusted by browsers. Configure local trust yourself or provide an appropriate deployment certificate.

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

The RSA counterpart is `GenerateRSACertificate(bits, template, outPath)`. ECC uses the fixed P-384 curve. Both functions self-sign with the supplied template; they do not request issuance from a CA.

The output directory must exist. File names come from `Subject.CommonName`, which should be trusted local configuration rather than untrusted request input. Generation overwrites matching files and may leave partial output on failure. Private keys are not passphrase-protected; callers manage file permissions and key storage. The CSR currently copies only CommonName, not all SAN and other template fields.

## Language

`server.Settings.SetLanguage(goi.ZhCN)` and `SetLanguage(goi.EnUS)` select built-in framework messages for `zh-CN` and `en-US`. This replaces a package-level translator, not a per-request locale. It does not automatically negotiate `Accept-Language` or translate application messages.

Configure it before serving requests rather than replacing the global translator concurrently. The current `SetLanguage` has a value receiver: translation changes, but `GetLanguage()` may still return the previous setting. Do not use that getter as a reliable query of the translator's current state.

## Time and Time Zones

| API | Behavior |
| --- | --- |
| `SetTimeZone(name)` | Loads a location through `time.LoadLocation`, returns loading errors, stores the configuration, and sets process environment variable `TZ`. It does not directly replace `time.Local`. |
| `GetLocation()` | Returns the configured location, falling back to `time.Local` when global Settings is nil. |
| `GetTime()`, `UseTZ=true` | Returns the current instant in the configured time zone, retaining its location information. |
| `GetTime()`, `UseTZ=false` | Keeps the configured location's wall-clock components but relabels them as UTC. This may change the represented instant; it is not an ordinary time-zone conversion. |
| `Localize(t)` | Calls `t.In(GetLocation())`, preserving the instant. It does not apply the `UseTZ=false` relabeling rule. |

The initial location is `time.Local`. `SetTimeZone` sets `TZ` according to `UseTZ` at the time of the call; changing `UseTZ` alone later does not update that environment variable. Decide whether an integration stores instants or local wall-clock values and do not mix those meanings.

## Source

[settings.go](../settings.go) · [time.go](../time.go) · [certificate.go](../certificate.go) · [server.go](../server.go)
