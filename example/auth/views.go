package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"example/user"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/auth"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/jwt"
)

type loginParams struct {
	Username *string `name:"username" type:"string"`
	Password string  `name:"password" type:"string" required:"true"`
}

func loginView(request *goi.Request) interface{} {
	var params loginParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError
	var err error

	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	sqlite3DB := db.SQLite3Connect("default")

	userInfo := user.UserModel{}
	sqlite3DB.SetModel(user.UserModel{})
	if params.Username != nil && *params.Username != "" {
		err = sqlite3DB.Where("username=?", params.Username).First(&userInfo)
	} else {
		return goi.Data{
			Code:    http.StatusBadRequest,
			Message: "请输入账号或邮箱",
			Results: nil,
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return goi.Data{
			Code:    http.StatusBadRequest,
			Message: "账号不存在",
			Results: nil,
		}
	} else if err != nil {
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "查询数据库错误",
			Results: nil,
		}
	}

	if auth.CheckPassword(params.Password, *userInfo.Password) == false {
		return goi.Data{
			Code:    http.StatusBadRequest,
			Message: "账号或密码错误",
			Results: nil,
		}
	}

	header := jwt.Header{
		Alg: jwt.AlgHS256,
		Typ: jwt.TypJWT,
	}

	// 设置过期时间
	twoHoursLater := goi.GetTime().Add(24 * 15 * time.Hour)

	payloads := Payloads{ // 包含 jwt.Payloads
		Payloads: jwt.Payloads{
			Exp: jwt.ExpTime{twoHoursLater},
		},
		User_id:  *userInfo.Id,
		Username: *userInfo.Username,
	}
	token, err := jwt.NewJWT(header, payloads, goi.Settings.SECRET_KEY)
	if err != nil {
		goi.Log.Error("生成 Token 错误", err.Error())
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "生成 Token 错误",
			Results: err,
		}
	}

	data := map[string]interface{}{
		"user":  userInfo,
		"token": token,
	}
	return goi.Data{
		Code:    http.StatusOK,
		Message: "登录成功",
		Results: data,
	}
}
