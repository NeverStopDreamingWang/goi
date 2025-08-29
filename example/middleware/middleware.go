package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"example/example"
	"example/user"
	"example/utils"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/jwt"
)

func init() {
	example.Server.MiddleWare = append(example.Server.MiddleWare, &MyMiddleWare{})
}

type MyMiddleWare struct{}

func (MyMiddleWare) ProcessRequest(request *goi.Request) interface{} {
	// fmt.Println("请求中间件", request.Object.URL)

	var apiList = []string{
		"/api/auth",   // 放行
		"/api/static", // 放行
	}
	for _, api := range apiList {
		// 跳过验证
		if strings.HasPrefix(request.Object.URL.Path, api) {
			return nil
		}
	}

	token := request.Object.Header.Get("Authorization")

	payloads := &utils.Payloads{}
	err := jwt.CkeckToken(token, goi.Settings.SECRET_KEY, payloads)
	if errors.Is(err, jwt.ErrDecode) { // token 解码错误
		return goi.Data{
			Code:    http.StatusUnauthorized,
			Message: "token 解码错误",
			Results: err,
		}
	} else if errors.Is(err, jwt.ErrExpiredSignature) { // token 已过期
		return goi.Data{
			Code:    http.StatusUnauthorized,
			Message: "token 已过期",
			Results: err,
		}
	}
	// 写入请求上下文
	ctx := context.WithValue(request.Object.Context(), "user_id", payloads.User_id)
	request.WithContext(ctx)

	sqlite3DB := db.SQLite3Connect("default")

	userInfo := user.UserModel{}
	sqlite3DB.SetModel(user.UserModel{})
	err = sqlite3DB.Where("`id` = ?", payloads.User_id).First(&userInfo)
	if errors.Is(err, sql.ErrNoRows) {
		return goi.Data{
			Code:    http.StatusBadRequest,
			Message: "用户不存在",
			Results: nil,
		}
	} else if err != nil {
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "查询数据库错误",
			Results: err.Error(),
		}
	}

	// 写入请求参数
	request.Params.Set("user", userInfo)
	return nil
}

func (MyMiddleWare) ProcessView(*goi.Request) interface{} { return nil }

func (MyMiddleWare) ProcessException(*goi.Request, any) interface{} { return nil }

func (MyMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
}
