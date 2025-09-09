package user

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
)

// 参数验证
type listValidParams struct {
	Page     int    `name:"page" type:"int" required:"true"`
	Pagesize int    `name:"pagesize" type:"int" required:"true"`
	Username string `name:"username" type:"string"`
}

func listView(request *goi.Request) interface{} {
	var params listValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError

	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	sqlite3DB := db.SQLite3Connect("default")

	sqlite3DB.SetModel(UserModel{}) // 设置操作表

	if params.Username != "" {
		sqlite3DB = sqlite3DB.Where("`username` like ?", "%"+params.Username+"%")
	}

	sqlite3DB = sqlite3DB.Where("id>?", 1).OrderBy("-id")
	total, page_number, err := sqlite3DB.Page(params.Page, params.Pagesize)
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	var userList []UserModel
	err = sqlite3DB.Fields("Id", "Username", "Password").Select(&userList)
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "ok",
		Results: map[string]interface{}{
			"total":    total,
			"page_sum": page_number,
			"list":     userList,
		},
	}
}

// 参数验证
type createValidParams struct {
	Username string `name:"username" type:"string" required:"true"`
	Password string `name:"password" type:"string" required:"true"`
}

func createView(request *goi.Request) interface{} {
	var params createValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError

	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	user := &UserModel{
		Username: &params.Username,
		Password: &params.Password,
	}

	// 参数验证
	err := user.Validate()
	if err != nil {
		return goi.Data{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Results: nil,
		}
	}
	// 添加
	err = user.Create()
	if err != nil {
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Results: nil,
		}
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "添加用户",
		Results: user,
	}
}

func retrieveView(request *goi.Request) interface{} {
	var pk int64
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	sqlite3DB := db.SQLite3Connect("default")

	user := UserModel{}
	sqlite3DB.SetModel(UserModel{})
	err := sqlite3DB.Where("`id` = ?", pk).First(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return goi.Data{
				Code:    http.StatusBadRequest,
				Message: "用户不存在",
				Results: nil,
			}
		}
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "查询用户失败",
			Results: nil,
		}
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "",
		Results: user,
	}
}

// 参数验证
type updateValidParams struct {
	Username *string `name:"username" type:"string"`
	Password *string `name:"password" type:"string"`
}

func updateView(request *goi.Request) interface{} {
	var pk int64
	var params updateValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError

	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	sqlite3DB := db.SQLite3Connect("default")

	instance := &UserModel{}
	sqlite3DB.SetModel(UserModel{})
	err := sqlite3DB.Where("`id` = ?", pk).First(instance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return goi.Data{
				Code:    http.StatusBadRequest,
				Message: "用户不存在",
				Results: nil,
			}
		}
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "查询用户失败",
			Results: nil,
		}
	}

	user := &UserModel{
		Username: params.Username,
		Password: params.Password,
	}
	// 参数验证
	err = user.Validate()
	if err != nil {
		return goi.Data{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Results: nil,
		}
	}
	// 更新
	err = instance.Update(user)
	if err != nil {
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Results: nil,
		}
	}
	return goi.Data{
		Code:    http.StatusOK,
		Message: "修改成功",
		Results: instance,
	}
}

func deleteView(request *goi.Request) interface{} {
	var pk int64
	var validationErr goi.ValidationError

	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	sqlite3DB := db.SQLite3Connect("default")

	instance := &UserModel{}
	sqlite3DB.SetModel(UserModel{})
	err := sqlite3DB.Where("`id` = ?", pk).First(instance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return goi.Data{
				Code:    http.StatusBadRequest,
				Message: "用户不存在",
				Results: nil,
			}
		}
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "查询用户失败",
			Results: nil,
		}
	}

	err = instance.Delete()
	if err != nil {
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: "删除失败",
		}
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "删除成功",
		Results: nil,
	}
}

// 使用事务
func TestWithTransaction(request *goi.Request) interface{} {
	var pk int64
	var params createValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError
	var err error

	validationErr = request.PathParams.Get("pk", &pk)
	if validationErr != nil {
		return validationErr.Response()
	}

	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	sqlite3DB := db.SQLite3Connect("default")

	// 返回错误事务自动回滚
	err = sqlite3DB.WithTransaction(func(sqlite3DB *db.SQLite3DB, args ...interface{}) error {
		create_time := goi.GetTime().Format(time.DateTime)
		// mysql 数据库
		user := &UserModel{
			Username:    &params.Username,
			Password:    &params.Password,
			Create_time: &create_time,
		}

		sqlite3DB.SetModel(UserModel{})
		_, err = sqlite3DB.Insert(user)
		if err != nil {
			return err
		}

		sqlite3DB.SetModel(UserModel{})
		_, err = sqlite3DB.Where("id=?", pk).Delete()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return goi.Data{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: "执行成功",
			Results: nil,
		},
	}
}
