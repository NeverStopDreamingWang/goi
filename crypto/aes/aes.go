package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// adjustKeyLength 调整密钥到有效的 AES 密钥长度（16、24 或 32 字节）
//
// 参数:
//   - key []byte: 原始密钥字节切片
//
// 返回:
//   - []byte: 调整后的密钥字节切片
func adjustKeyLength(key []byte) []byte {
	validLengths := []int{16, 24, 32}
	keyLength := len(key)

	// 如果已经是有效长度，直接返回
	for _, length := range validLengths {
		if keyLength == length {
			return key
		}
	}

	// 找到需要扩展到的目标长度
	targetLength := validLengths[len(validLengths)-1] // 初始设为最大长度
	for i := len(validLengths) - 1; i >= 0; i-- {
		if keyLength <= validLengths[i] {
			targetLength = validLengths[i]
		}
	}

	// 创建新的密钥切片
	adjustedKey := make([]byte, targetLength)

	if keyLength > targetLength {
		// 如果原始密钥过长，截断
		copy(adjustedKey, key[:targetLength])
	} else {
		// 如果原始密钥过短，用0填充
		copy(adjustedKey, key)
		for i := keyLength; i < targetLength; i++ {
			adjustedKey[i] = '0'
		}
	}

	return adjustedKey
}

// pkcs7Padding 使用 PKCS7 标准填充
// PKCS7 填充规则：
// 1. 如果数据长度小于块大小，填充到块大小
// 2. 如果数据长度是块大小的整数倍，填充一个完整块
// 3. 填充的字节值等于填充的长度
//
// 参数:
//   - data []byte: 需要填充的原始数据字节切片
//   - blockSize int: 加密块大小，AES 固定为 16 字节
//
// 返回:
//   - []byte: 填充后的数据字节切片
func pkcs7Padding(data []byte, blockSize int) []byte {
	// 计算需要填充的字节数（范围是1到blockSize）
	padding := blockSize - (len(data) % blockSize)

	// 创建填充切片：重复 padding 个值为 padding 的字节
	padText := bytes.Repeat([]byte{byte(padding)}, padding)

	// 将填充附加到原数据后
	return append(data, padText...)
}

// pkcs7UnPadding 去除 PKCS7 标准填充
//
// 参数:
//   - data []byte: 需要去除填充的数据字节切片
//   - blockSize int: 加密块大小，AES 固定为 16 字节
//
// 返回:
//   - []byte: 去除填充后的原始数据字节切片
//   - error: 如果填充无效则返回错误
//
// 错误:
//   - "数据长度为0": 输入数据为空
//   - "填充长度无效": 填充长度不在有效范围内
//   - "填充无效": 填充字节值不一致
func pkcs7UnPadding(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		errMsg := i18n.T("crypto.aes.empty_data")
		return nil, errors.New(errMsg)
	}

	// 最后一个字节的值就是填充长度
	padding := int(data[length-1])

	// 验证填充值的有效性
	if padding == 0 || padding > blockSize {
		errMsg := i18n.T("crypto.aes.invalid_padding_length")
		return nil, errors.New(errMsg)
	}

	// 验证所有填充字节是否相同
	for i := 0; i < padding; i++ {
		if int(data[length-1-i]) != padding {
			errMsg := i18n.T("crypto.aes.invalid_padding")
			return nil, errors.New(errMsg)
		}
	}

	// 移除填充
	return data[:length-padding], nil
}

// Encrypt 使用 AES CBC 模式加密数据
//
// 参数:
//   - plainText string: 需要加密的原文字符串
//   - keyBytes []byte: 加密密钥字节切片（会自动调整到有效的AES密钥长度）
//
// 返回:
//   - string: base64编码的加密结果，包含IV和密文
//   - error: 加密过程中的错误
//
// 加密过程:
//  1. 将密钥调整到有效长度（16、24或32字节）
//  2. 对明文进行PKCS7填充
//  3. 生成随机IV
//  4. 使用CBC模式加密
//  5. 将IV和密文组合并base64编码
func Encrypt(plainText string, keyBytes []byte) (string, error) {
	// 将字符串转换为字节数组
	plainTextBytes := []byte(plainText)

	// 调整密钥长度
	keyBytes = adjustKeyLength(keyBytes)

	// 创建 AES 加密块
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	// 使用 PKCS7 填充
	blockSize := block.BlockSize()
	plainTextBytes = pkcs7Padding(plainTextBytes, blockSize)

	// 生成 IV (初始化向量)
	iv := make([]byte, blockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// 创建 CBC 加密器
	mode := cipher.NewCBCEncrypter(block, iv)

	// 执行加密
	cipherText := make([]byte, len(plainTextBytes))
	mode.CryptBlocks(cipherText, plainTextBytes)

	// 合并 IV 和密文
	finalCipherText := append(iv, cipherText...)

	// 返回 base64 编码的密文
	return base64.StdEncoding.EncodeToString(finalCipherText), nil
}

// Decrypt 使用 AES CBC 模式解密数据
//
// 参数:
//   - cipherTextBase64 string: base64编码的密文字符串（包含IV）
//   - keyBytes []byte: 解密密钥字节切片（需要与加密时使用的密钥相同）
//
// 返回:
//   - string: 解密后的原文字符串
//   - error: 解密过程中的错误
//
// 解密过程:
//  1. 将密钥调整到有效长度
//  2. 对密文进行base64解码
//  3. 提取IV和密文
//  4. 使用CBC模式解密
//  5. 去除PKCS7填充
func Decrypt(cipherTextBase64 string, keyBytes []byte) (string, error) {
	// 调整密钥长度
	keyBytes = adjustKeyLength(keyBytes)

	// 解码 base64 密文
	cipherText, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		errMsg := i18n.T("crypto.aes.base64_decode_error", map[string]any{
			"err": err,
		})
		return "", errors.New(errMsg)
	}

	// 创建 AES 加密块
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		errMsg := i18n.T("crypto.aes.new_cipher_error", map[string]any{
			"err": err,
		})
		return "", errors.New(errMsg)
	}

	// 检查密文长度
	blockSize := block.BlockSize()
	if len(cipherText) < blockSize {
		errMsg := i18n.T("crypto.aes.invalid_ciphertext_length")
		return "", errors.New(errMsg)
	}

	// 提取 IV
	iv := cipherText[:blockSize]
	cipherText = cipherText[blockSize:]

	// 创建 CBC 解密器
	mode := cipher.NewCBCDecrypter(block, iv)

	// 执行解密
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// 去除 PKCS7 填充
	plainText, err = pkcs7UnPadding(plainText, blockSize)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
