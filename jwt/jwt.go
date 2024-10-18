package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Jwt 验证错误
var (
	jwtExpiredSignatureError = errors.New(language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "jwt.expired_signature_error",
	}))
	jwtDecodeError = errors.New(language.I18n.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "jwt.decode_error",
	}))
)

// 签名过期
func JwtExpiredSignatureError(err error) bool {
	if err == jwtExpiredSignatureError {
		return true
	}
	return false
}

// Jwt 解码错误
func JwtDecodeError(err error) bool {
	if err == jwtDecodeError {
		return true
	}
	return false
}

// 生成 token
func NewJWT(header map[string]interface{}, payload map[string]interface{}, key string) (string, error) {
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

	token := headerStr + "." + payloadStr + "." + signatureStr
	return token, nil
}

// 验证 token
func CkeckToken(token string, key string) (map[string]interface{}, error) {
	var payloads map[string]interface{}
	var err error
	tokenSlice := strings.Split(token, ".")
	if len(tokenSlice) != 3 {
		return payloads, jwtDecodeError
	}
	headerBase64 := tokenSlice[0]
	payloadBase64 := tokenSlice[1]
	signatureBase64 := tokenSlice[2]

	// 检查 signature
	signatureStr, err := sha256Str(key, headerBase64+"."+payloadBase64)
	if err != nil {
		return payloads, jwtDecodeError
	}
	signatureStr, err = base64Encode(signatureStr)
	if err != nil {
		return payloads, jwtDecodeError
	}
	if signatureStr != signatureBase64 {
		return payloads, jwtDecodeError
	}
	// header, err := base64Decode(headerBase64)
	// if err != nil {
	// 	return false, err
	// }
	payloads, err = base64Decode(payloadBase64)
	if err != nil {
		return payloads, jwtDecodeError
	}

	if expTimeValue, ok := payloads["exp"]; ok {
		expTimeStr := expTimeValue.(string)
		// 解析为time.Time类型
		expTime, err := time.ParseInLocation("2006-01-02 15:04:05", expTimeStr, goi.Settings.GetLocation())
		if err != nil {
			return payloads, jwtDecodeError
		}
		if expTime.Before(time.Now().In(goi.Settings.GetLocation())) {
			return payloads, jwtExpiredSignatureError
		}
	}
	return payloads, nil
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
func base64Decode(dataStr string) (map[string]interface{}, error) {
	var data map[string]interface{}
	// 补全字符串末尾的"="符号
	remainder := len(dataStr) % 4
	if remainder > 0 {
		dataStr += strings.Repeat("=", 4-remainder)
	}
	dataByte, err := base64.URLEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(dataByte, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// hash 256
func sha256Str(key string, data string) (string, error) {
	// 创建SHA-256哈希对象
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
