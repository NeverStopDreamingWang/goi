package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// JWT 算法
const (
	AlgHS256 = "HS256" // HMAC using SHA-256
	// 以下算法当前未支持
	// AlgHS384 = "HS384" // HMAC using SHA-384
	// AlgHS512 = "HS512" // HMAC using SHA-512
	// AlgRS256 = "RS256" // RSA using SHA-256
	// AlgRS384 = "RS384" // RSA using SHA-384
	// AlgRS512 = "RS512" // RSA using SHA-512
	// AlgES256 = "ES256" // ECDSA using P-256 and SHA-256
	// AlgES384 = "ES384" // ECDSA using P-384 and SHA-384
	// AlgES512 = "ES512" // ECDSA using P-521 and SHA-512
	// AlgPS256 = "PS256" // RSASSA-PSS using SHA-256 and MGF1
	// AlgPS384 = "PS384" // RSASSA-PSS using SHA-384 and MGF1
	// AlgPS512 = "PS512" // RSASSA-PSS using SHA-512 and MGF1
)

// JWT 类型
const (
	TypJWT = "JWT" // JSON Web Token
)

var (
	// 解码错误
	ErrDecode = errors.New(i18n.T("jwt.decode_error"))
	// 过期错误
	ErrExpiredSignature = errors.New(i18n.T("jwt.expired_signature"))
)

// Header JWT头部结构
// Alg 表示使用的签名算法
// Typ 表示令牌的类型
type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Payloads JWT负载基础结构
// Exp 表示令牌的过期时间
type Payloads struct {
	Exp ExpTime `json:"exp"`
}

// ExpTime 自定义过期时间类型
// 嵌入time.Time以扩展其功能
type ExpTime struct {
	time.Time
}

// 实现 UnmarshalJSON 方法以指定时区
func (expTime *ExpTime) UnmarshalJSON(data []byte) error {
	var expTimeStr string
	var err error

	// 先解码 data 到字符串，避免直接处理原始字节
	err = json.Unmarshal(data, &expTimeStr)
	if err != nil {
		return err
	}
	expTime.Time, err = time.ParseInLocation(time.RFC3339, expTimeStr, goi.GetLocation())
	if err != nil {
		return err
	}
	return nil
}

// NewJWT 生成新的JWT令牌
//
// 参数:
//   - header interface{}: JWT头部信息，通常为 Header 结构体
//   - payload interface{}: JWT负载信息，必须包含 Payloads 结构体
//   - key string: 用于签名的密钥
//
// 返回:
//   - string: 生成的token字符串，格式为: header.payload.signature
//   - error: 生成过程中的错误
func NewJWT(header interface{}, payload interface{}, key string) (string, error) {
	var headerStr string
	var payloadStr string
	var signatureStr string
	var err error
	// header
	headerStr, err = base64Encode(header)
	if err != nil {
		return "", err
	}
	// payload
	payloadStr, err = base64Encode(payload)
	if err != nil {
		return "", err
	}
	// signature
	signatureStr, err = sha256Str(key, headerStr+"."+payloadStr)
	if err != nil {
		return "", err
	}
	signatureStr, err = base64Encode(signatureStr)
	if err != nil {
		return "", err
	}

	token := fmt.Sprintf("%s.%s.%s", headerStr, payloadStr, signatureStr)
	return token, nil
}

// CheckToken 验证JWT令牌的有效性
//
// 参数:
//   - token string: 待验证的JWT令牌字符串
//   - key string: 验证签名用的密钥
//   - payloadsDest interface{}: 用于存储解码后payload的结构体指针，必须包含 Payloads 结构体
//
// 返回:
//   - error: 验证过程中的错误，包括:
//   - 令牌格式错误
//   - 签名验证失败
//   - 令牌已过期
func CheckToken(token string, key string, payloadsDest interface{}) error {
	var err error
	tokenSlice := strings.Split(token, ".")
	if len(tokenSlice) != 3 {
		return ErrDecode
	}
	headerBase64 := tokenSlice[0]
	payloadBase64 := tokenSlice[1]
	signatureBase64 := tokenSlice[2]

	// 检查 signature
	signatureStr, err := sha256Str(key, headerBase64+"."+payloadBase64)
	if err != nil {
		signatureErrMsg := i18n.T("jwt.signature_error", map[string]interface{}{
			"err": err,
		})
		return errors.New(signatureErrMsg)
	}
	signatureStr, err = base64Encode(signatureStr)
	if err != nil {
		return ErrDecode
	}
	if signatureStr != signatureBase64 {
		return ErrDecode
	}
	header := &Header{}
	err = base64Decode(headerBase64, header)
	if err != nil {
		decodeErrMsg := i18n.T("jwt.header_decode_error", map[string]interface{}{
			"err": err,
		})
		return errors.New(decodeErrMsg)
	}
	if header.Alg != AlgHS256 || header.Typ != TypJWT {
		invalidHeaderMsg := i18n.T("jwt.invalid_header", map[string]interface{}{
			"alg": header.Alg,
			"typ": header.Typ,
		})
		return errors.New(invalidHeaderMsg)
	}
	err = base64Decode(payloadBase64, payloadsDest)
	if err != nil {
		payloadDecodeErrMsg := i18n.T("jwt.payload_decode_error", map[string]interface{}{
			"err": err,
		})
		return errors.New(payloadDecodeErrMsg)
	}

	// 检查嵌套的 Payloads 字段并验证 Exp
	err = checkExpField(payloadsDest)
	if err != nil {
		return err
	}
	return nil
}

// checkExpField 检查payload中的过期时间字段
//
// 参数:
//   - payloadsDest interface{}: 包含 Exp ExpTime 字段的结构体指针
//
// 返回:
//   - error: 检查过程中的错误，如令牌已过期返回 ErrExpiredSignature
//
// 注意:
//   - 如果没有从 payloadsDest 中查找到 Exp ExpTime 字段，则返回 nil 默认验证通过
func checkExpField(payloadsDest interface{}) error {
	payloadsDestValue := reflect.ValueOf(payloadsDest)

	if payloadsDestValue.Kind() == reflect.Ptr {
		payloadsDestValue = payloadsDestValue.Elem()
	}
	if payloadsDestValue.Kind() != reflect.Struct {
		return ErrDecode
	}

	// 先尝试直接访问
	expField := payloadsDestValue.FieldByName("Exp")
	if !expField.IsValid() { // 未找到 Exp 字段
		return nil
	}
	expTime, ok := expField.Interface().(ExpTime)
	if !ok { // 类型不匹配
		return nil
	}
	// 判断过期时间
	if expTime.Time.Before(goi.GetTime()) {
		return ErrExpiredSignature
	}
	return nil // 未找到有效的 Exp 字段
}

// base64Encode 将数据进行base64 URL编码
//
// 参数:
//   - data interface{}: 要编码的数据，支持类型:
//   - string: 字符串类型
//   - []byte: 字节切片类型
//   - 其他可JSON序列化的类型
//
// 返回:
//   - string: base64 URL编码后的字符串
//   - error: 编码过程中的错误
func base64Encode(data interface{}) (string, error) {
	var dataByte []byte
	var err error
	switch value := data.(type) {
	case string:
		dataByte = []byte(value)
	case []byte:
		dataByte = data.([]byte)
	default:
		dataByte, err = json.Marshal(value)
		if err != nil {
			encodeErrMsg := i18n.T("jwt.encode_error", map[string]interface{}{
				"err": err,
			})
			return "", errors.New(encodeErrMsg)
		}
	}
	dataBase64Encode := base64.URLEncoding.EncodeToString(dataByte)
	dataBase64Encode = strings.TrimRight(dataBase64Encode, "=")
	return dataBase64Encode, nil
}

// base64Decode 将base64编码的字符串解码到目标结构体
//
// 参数:
//   - dataStr string: base64编码的字符串
//   - payloadsDest interface{}: 解码目标结构体的指针
//
// 返回:
//   - error: 解码过程中的错误
func base64Decode(dataStr string, payloadsDest interface{}) error {
	// 补全字符串末尾的"="符号
	remainder := len(dataStr) % 4
	if remainder > 0 {
		dataStr += strings.Repeat("=", 4-remainder)
	}
	dataByte, err := base64.URLEncoding.DecodeString(dataStr)
	if err != nil {
		return err
	}
	err = json.Unmarshal(dataByte, payloadsDest)
	if err != nil {
		return err
	}
	return nil
}

// sha256Str 使用HMAC-SHA256算法计算数据的哈希值
//
// 参数:
//   - key string: HMAC密钥
//   - data string: 要计算哈希的数据
//
// 返回:
//   - string: 十六进制格式的哈希字符串
//   - error: 计算过程中的错误
func sha256Str(key string, data string) (string, error) {
	// 创建HS256哈希对象
	hash := hmac.New(sha256.New, []byte(key))
	// 计算哈希值
	_, err := hash.Write([]byte(data))
	if err != nil {
		return "", err
	}
	hashValue := hash.Sum(nil)
	// 将哈希值转换为十六进制字符串
	hashString := hex.EncodeToString(hashValue)

	return hashString, nil
}
