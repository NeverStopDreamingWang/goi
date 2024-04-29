package user

import (
	"fmt"
	"net/http"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/jwt"
)

func UserTest(request *goi.Request) interface{} {

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   nil,
		},
	}
}

// 参数验证
type testloginParams struct {
	Username string `name:"username" required:"string"`
	Password string `name:"password" required:"string"`
}

func Testlogin(request *goi.Request) interface{} {
	var params testloginParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()
	}
	fmt.Println("username", params.Username)
	fmt.Println("password", params.Password)

	header := map[string]interface{}{
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

	payloads := map[string]interface{}{
		"user_id":  1,
		"username": "wjh123",
		"exp":      expTime,
	}
	token, err := jwt.NewJWT(header, payloads, "#wrehta)a^x&ichxfrut&wdl8g&q&u2b#yh%^@1+1(bsyn498y")
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusInternalServerError,
				Msg:    "ok",
				Data:   err,
			},
		}
	}
	goi.Log.Info(token)

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   token,
		},
	}
}

func TestAuth(request *goi.Request) interface{} {
	token := request.Object.Header.Get("Authorization")

	payloads, err := jwt.CkeckToken(token, "#wrehta)a^x&ichxfrut&wdl8g&q&u2b#yh%^@1+1(bsyn498y")
	if jwt.JwtDecodeError(err) { // token 解码错误！
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusUnauthorized,
				Msg:    "token 解码错误！",
				Data:   err,
			},
		}
	} else if jwt.JwtExpiredSignatureError(err) { // token 已过期！
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusUnauthorized,
				Msg:    "token 已过期！",
				Data:   err,
			},
		}
	}

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   payloads,
		},
	}
}
