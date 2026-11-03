# Password Hashing and Cryptographic Utilities

[简体中文](security.md) | **English** · [Documentation](README.en.md)

This page describes the current `auth` and `crypto` APIs and formats, not a complete authentication system or security protocol. The framework does not automatically provide JWT, sessions, permission checks, login rate limits, or key management. Applications and [middleware](middleware.en.md) provide those features. For TLS certificates, see [Configuration and HTTPS](configuration.en.md).

## Password Hashing

Store passwords with `auth.MakePassword` and verify them with `auth.CheckPassword`. Reversible AES/RSA encryption is not a substitute for this password-hashing flow.

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
	// Store encoded in the server-side password field, not the plaintext.
	fmt.Println(auth.CheckPassword("example-password", encoded))
	fmt.Println(auth.CheckPassword("wrong-password", encoded))
	// true
	// false
}
```

The current implementation uses PBKDF2-SHA256, a random 16-byte salt, 870000 iterations, and a 32-byte derived result. Storage format:

```text
pbkdf2_sha256$iterations$base64(salt)$base64(hash)
```

| Method | Input and result |
| --- | --- |
| `auth.MakePassword(password)` | Returns an error for an empty password; otherwise returns a string containing the algorithm, iterations, salt, and hash. |
| `auth.CheckPassword(password, encoded)` | Parses the stored value, recomputes with its salt and iteration count, and compares the result. Parsing failures and mismatches return false. |

Verification cost comes from the iteration count in `encoded`. Use controlled server-side stored values rather than accepting a complete encoded hash from the client as the verification parameter. The implementation does not automatically rehash, upgrade iteration counts, or enforce password strength, and does not promise a constant-time string comparison.

The lower-level encoding and decoding APIs are in [crypto/pbkdf2_sha256](../crypto/pbkdf2_sha256/hashers.go). Direct callers manage their own argument constraints and formats.

## Random Data and Encryption Example

This example generates data in memory and does not print keys or plaintext passwords:

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

## AES Format and Limits

`crypto/aes.Encrypt(plainText string, keyBytes []byte)` uses AES-CBC with PKCS7 padding, generates a random IV, and encodes `IV || ciphertext` with standard Base64. `Decrypt` accepts that Base64 string and the same key, returning plaintext or an error.

Key handling is specific to this package:

- Keys of 16, 24, or 32 bytes are used unchanged.
- Shorter or intermediate-length keys are padded to the next supported length with the character `'0'`. This is not password-based key derivation.
- Keys longer than 32 bytes are truncated to 32 bytes.

The example explicitly generates a random 32-byte key, avoiding implicit padding or truncation. `Settings.SecretKey` is not automatically injected into these functions; callers pass and manage their keys explicitly.

The CBC helper has no MAC or authentication tag. It cannot detect every ciphertext modification and is not a signature or authenticated encryption protocol. `Decrypt` does not fully validate ciphertext block lengths; malformed ciphertext can panic instead of returning an error. A successful round trip is not a guarantee that arbitrary untrusted ciphertext can be passed directly to this API.

## RSA Format and Limits

| Method | Format |
| --- | --- |
| `rsa.GenerateKey(bits)` | Returns PEM private/public keys and an error. Private-key DER is PKCS1; public-key DER is PKIX. |
| `rsa.Encrypt(publicKey, plaintext)` | Parses a PKIX public key, uses RSA-OAEP/SHA-256, and returns binary ciphertext. |
| `rsa.Decrypt(privateKey, ciphertext)` | Parses a PKCS1 private key and decrypts using the same OAEP parameters. |

Although the generated public PEM label is `RSA PUBLIC KEY`, its payload is PKIX rather than PKCS1 public-key DER. Interoperate with external tools according to the actual DER format. The package does not automatically parse arbitrary PKCS8 private keys or certificate PEMs and does not provide signature/verification methods beyond the APIs listed here.

RSA does not split long messages into chunks; inputs exceeding OAEP capacity return an error. Ciphertext is `[]byte`. Choose an explicit encoding such as Base64 for JSON or text transport rather than treating arbitrary binary data as a displayable string. Generated private keys are not passphrase-protected; callers manage persistence and access control.

## Random Utilities

| Method | Behavior |
| --- | --- |
| `GenerateRead(buffer)` | Fills an existing byte slice through `crypto/rand.Read`. Its length determines the number of output bytes; returns an error. |
| `GenerateRandomString(length, allowedChars)` | Selects bytes using cryptographically random indices. Callers supply a nonnegative length and a nonempty byte-oriented character set. |
| `GenerateRandomSecretKey()` | Generates `SecretKeyMinLength` characters from `Charset`, then adds `SecretKeyInsecurePrefix`. This return format does not imply automatic configuration validation or key rotation. |

`allowedChars` is indexed by byte. Multibyte character sets may produce invalid UTF-8; the example uses ASCII. Invalid lengths or character sets are not uniformly converted into errors and may panic. Fixed keys in examples and templates are not deployment credentials.

## Source and Related Guides

[auth/auth.go](../auth/auth.go) · [crypto/aes/aes.go](../crypto/aes/aes.go) · [crypto/rsa/rsa.go](../crypto/rsa/rsa.go) · [crypto/utils.go](../crypto/utils.go)

[Configuration and HTTPS](configuration.en.md) · [Middleware](middleware.en.md) · [HTTP Requests and Responses](http.en.md)
