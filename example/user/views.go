package user

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"github.com/NeverStopDreamingWang/hgee/jwt"
	"net/http"
	"time"
)

func UserTest(request *hgee.Request) any {

	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   nil,
		},
	}
}

func Testlogin(request *hgee.Request) any {
	username := request.BodyParams.Get("username")
	password := request.BodyParams.Get("password")
	fmt.Println("username", username)
	fmt.Println("password", password)

	header := map[string]any{
		"alg": "SHA256",
		"typ": "JWT",
	}

	// 设置过期时间
	// 获取当前时间
	now := time.Now()
	// 增加两小时
	// twoHoursLater := now.Add(2 * time.Hour)
	twoHoursLater := now.Add(10 * time.Second)
	// 格式化为字符串
	expTime := twoHoursLater.Format("2006-01-02 15:04:05")
	fmt.Println(expTime)

	payloads := map[string]any{
		"user_id":  1,
		"username": "wjh123",
		"exp":      expTime,
	}
	token, err := jwt.NewJWT(header, payloads, "#wrehta)a^x&ichxfrut&wdl8g&q&u2b#yh%^@1+1(bsyn498y")
	if err != nil {
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusInternalServerError,
				Msg:    "ok",
				Data:   err,
			},
		}
	}
	hgee.Log.Info(token)

	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   token,
		},
	}
}

func TestAuth(request *hgee.Request) any {
	token := request.Object.Header.Get("Authorization")

	payloads, err := jwt.CkeckToken(token, "#wrehta)a^x&ichxfrut&wdl8g&q&u2b#yh%^@1+1(bsyn498y")
	if jwt.JwtDecodeError(err) { // token 解码错误！
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusUnauthorized,
				Msg:    "token 解码错误！",
				Data:   err,
			},
		}
	} else if jwt.JwtExpiredSignatureError(err) { // token 已过期！
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusUnauthorized,
				Msg:    "token 已过期！",
				Data:   err,
			},
		}
	}

	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   payloads,
		},
	}
}
