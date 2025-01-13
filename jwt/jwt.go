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

// Jwt 验证错误
var (
	jwtExpiredSignatureError = errors.New("Expired Signature Error")
	jwtDecodeError           = errors.New("Decode Error")
)

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type Payloads struct {
	Exp ExpTime `json:"exp"`
}

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
	expTime.Time, err = time.ParseInLocation(time.RFC3339, expTimeStr, goi.Settings.GetLocation())
	if err != nil {
		return err
	}
	return nil
}

// 签名过期
func JwtExpiredSignatureError(err error) bool {
	return errors.Is(err, jwtExpiredSignatureError)
}

// Jwt 解码错误
func JwtDecodeError(err error) bool {
	return errors.Is(err, jwtDecodeError)
}

// 生成 token
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

// 验证 token
func CkeckToken(token string, key string, payloadsDest interface{}) error {
	var err error
	tokenSlice := strings.Split(token, ".")
	if len(tokenSlice) != 3 {
		return jwtDecodeError
	}
	headerBase64 := tokenSlice[0]
	payloadBase64 := tokenSlice[1]
	signatureBase64 := tokenSlice[2]

	// 检查 signature
	signatureStr, err := sha256Str(key, headerBase64+"."+payloadBase64)
	if err != nil {
		return jwtDecodeError
	}
	signatureStr, err = base64Encode(signatureStr)
	if err != nil {
		return jwtDecodeError
	}
	if signatureStr != signatureBase64 {
		return jwtDecodeError
	}
	header := &Header{}
	err = base64Decode(headerBase64, header)
	if err != nil {
		return err
	}
	if header.Alg != AlgHS256 || header.Typ != TypJWT {
		return jwtDecodeError
	}
	err = base64Decode(payloadBase64, payloadsDest)
	if err != nil {
		return jwtDecodeError
	}

	// 检查嵌套的 Payloads 字段并验证 Exp
	err = checkExpField(payloadsDest)
	if err != nil {
		return err
	}
	return nil
}

// checkExpField 用于检查 payload 中的 Exp 字段
func checkExpField(payloadsDest interface{}) error {
	payloadsDestValue := reflect.ValueOf(payloadsDest)

	if payloadsDestValue.Kind() == reflect.Ptr {
		payloadsDestValue = payloadsDestValue.Elem()
	}
	if payloadsDestValue.Kind() != reflect.Struct {
		return jwtDecodeError
	}
	// 获取 Payloads 类型
	PayloadsType := reflect.TypeOf(Payloads{})

	// 查找 Payloads 类型字段
	for i := 0; i < payloadsDestValue.NumField(); i++ {
		field := payloadsDestValue.Field(i)

		// 查找 Payloads 类型字段
		if field.Type() != PayloadsType {
			continue
		}

		// 直接获取嵌套的 Exp 字段
		expField := field.FieldByName("Exp")
		if expField.IsValid() == false {
			return jwtDecodeError
		}
		expTime, ok := expField.Interface().(ExpTime)
		if ok == false {
			return jwtDecodeError
		}

		// 判断过期时间
		if expTime.Time.Before(goi.GetTime()) {
			return jwtExpiredSignatureError
		}
		return nil // Exp 时间有效
	}
	return jwtDecodeError // 未找到有效的 Exp 字段
}

// base64 编码
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
			return "", err
		}
	}
	dataBase64Encode := base64.URLEncoding.EncodeToString(dataByte)
	dataBase64Encode = strings.TrimRight(dataBase64Encode, "=")
	return dataBase64Encode, nil
}

// base64 解码
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

// HS256
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
