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
			Status:  http.StatusOK,
			Message: "ok",
			Results: nil,
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
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()
	}
	fmt.Println("username", params.Username)
	fmt.Println("password", params.Password)

	header := jwt.Header{
		Alg: jwt.AlgHS256,
		Typ: jwt.TypJWT,
	}

	// 设置过期时间
	// 获取当前时间
	now := time.Now()
	// 增加两小时
	twoHoursLater := now.Add(2 * time.Hour)
	// twoHoursLater := now.Add(10 * time.Second)

	payloads := example.Payloads{
		Payloads: jwt.Payloads{
			Exp: jwt.ExpTime{twoHoursLater},
		},
		User_id:  1,
		Username: "admin",
	}
	token, err := jwt.NewJWT(header, payloads, goi.Settings.SECRET_KEY)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status:  http.StatusInternalServerError,
				Message: "ok",
				Results: err,
			},
		}
	}
	goi.Log.Info(token)

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status:  http.StatusOK,
			Message: "ok",
			Results: token,
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
				Status:  http.StatusUnauthorized,
				Message: "token 解码错误！",
				Results: err,
			},
		}
	} else if jwt.JwtExpiredSignatureError(err) { // token 已过期！
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status:  http.StatusUnauthorized,
				Message: "token 已过期！",
				Results: err,
			},
		}
	}

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status:  http.StatusOK,
			Message: "ok",
			Results: payloads,
		},
	}
}
