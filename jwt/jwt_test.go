package jwt_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/jwt"
)

// 自定义负载结构体
type CustomPayloads struct {
	jwt.Payloads        // 嵌入基础 Payloads 结构体
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
}

func ExampleNewJWT() {
	// 创建头部信息
	header := jwt.Header{
		Alg: jwt.AlgHS256,
		Typ: jwt.TypJWT,
	}

	// 创建自定义负载
	payload := CustomPayloads{
		Payloads: jwt.Payloads{
			// 设置过期时间为2小时后
			Exp: jwt.ExpTime{Time: goi.GetTime().Add(time.Hour * 2)},
		},
		UserID:   1,
		Username: "test_user",
	}

	// 签名密钥
	key := "your-secret-key"

	// 生成 JWT 令牌
	token, err := jwt.NewJWT(header, payload, key)
	if err != nil {
		fmt.Println("生成令牌错误:", err)
		return
	}
	fmt.Println("生成的令牌成功")

	// 验证并解析令牌
	var decodedPayload CustomPayloads
	err = jwt.CkeckToken(token, key, &decodedPayload)
	if err != nil {
		fmt.Println("验证令牌错误:", err)
		return
	}

	fmt.Println("解析的用户ID:", decodedPayload.UserID)
	fmt.Println("解析的用户名:", decodedPayload.Username)

	// Output:
	// 生成的令牌成功
	// 解析的用户ID: 1
	// 解析的用户名: test_user
}

func ExampleCkeckToken() {
	// 签名密钥
	key := "your-secret-key"

	// 创建一个已过期的令牌
	header := jwt.Header{
		Alg: jwt.AlgHS256,
		Typ: jwt.TypJWT,
	}

	payload := CustomPayloads{
		Payloads: jwt.Payloads{
			// 设置过期时间为1小时前
			Exp: jwt.ExpTime{Time: goi.GetTime().Add(-time.Hour)},
		},
		UserID:   1,
		Username: "test_user",
	}

	// 生成过期的令牌
	expiredToken, _ := jwt.NewJWT(header, payload, key)

	// 尝试验证过期的令牌
	var decodedPayload CustomPayloads
	err := jwt.CkeckToken(expiredToken, key, &decodedPayload)
	if errors.Is(err, jwt.ErrExpiredSignature) {
		fmt.Println("令牌已过期")
	} else if err == jwt.ErrDecode {
		fmt.Println("令牌格式错误")
	} else if err != nil {
		fmt.Println("其他错误:", err)
	}

	// 使用错误的密钥验证令牌
	wrongKey := "wrong-secret-key"
	err = jwt.CkeckToken(expiredToken, wrongKey, &decodedPayload)
	if err == jwt.ErrDecode {
		fmt.Println("签名验证失败")
	}

	// Output:
	// 令牌已过期
	// 签名验证失败
}
