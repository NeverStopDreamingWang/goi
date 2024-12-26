package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
)

// 参数验证
type userSqliteListParams struct {
	Page     int `name:"page" required:"int"`
	Pagesize int `name:"pagesize" required:"int"`
}

// 读取多条数据到 Model
func UserSqliteList(request *goi.Request) interface{} {
	var params userSqliteListParams
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	sqlite3DB := db.SQLite3Connect("sqlite")

	// sqlite3 数据库
	var userSlice []UserSqliteModel
	sqlite3DB.SetModel(UserSqliteModel{})
	sqlite3DB.Fields("Id", "Username", "Password").Where("id>?", 1).OrderBy("-id")
	total, page_number, err := sqlite3DB.Page(params.Page, params.Pagesize)
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	err = sqlite3DB.Select(&userSlice)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	for i, user := range userSlice {
		// *user.Username = "test"
		fmt.Println(i)
		if user.Id != nil {
			fmt.Println("Id", *user.Id)
		}
		if user.Username != nil {
			fmt.Println("Username", *user.Username)
		}
		if user.Create_time != nil {
			fmt.Println("Create_time", *user.Create_time)
		}
		if user.Update_time != nil {
			fmt.Println("Update_time", *user.Update_time)
		}
	}

	var user_data = map[string]interface{}{
		"total":       total,
		"page_number": page_number,
		"user":        userSlice,
	}
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: "ok",
			Results: user_data,
		},
	}
}

// 参数验证
type userSqliteRetrieveParams struct {
	User_id int `name:"user_id" required:"int"`
}

// 读取第一条数据到 Model
func UserSqliteRetrieve(request *goi.Request) interface{} {
	var params userSqliteRetrieveParams
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	if params.User_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	sqlite3DB := db.SQLite3Connect("sqlite")

	// sqlite3 数据库
	user := UserSqliteModel{}
	sqlite3DB.SetModel(UserSqliteModel{})
	err := sqlite3DB.Fields("Id", "Username").Where("id=?", params.User_id).First(&user)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if user.Id != nil {
		fmt.Println("Id", *user.Id)
	}
	if user.Username != nil {
		fmt.Println("Username", *user.Username)
	}
	if user.Create_time != nil {
		fmt.Println("Create_time", *user.Create_time)
	}
	if user.Update_time != nil {
		fmt.Println("Update_time", *user.Update_time)
	}

	return goi.Response{
		Status: http.StatusOK,
		Data:   user,
	}
}

// 参数验证
type userSqliteCreateParams struct {
	Username string `name:"username" required:"string"`
	Password string `name:"password" required:"string"`
}

// 添加一条数据到 Model
func UserSqliteCreate(request *goi.Request) interface{} {
	var params userSqliteCreateParams
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	sqlite3DB := db.SQLite3Connect("sqlite")

	create_time := goi.GetTime().Format("2006-01-02 15:04:05")
	// sqlite3 数据库
	user := &UserSqliteModel{
		Username:    &params.Username,
		Password:    &params.Password,
		Create_time: &create_time,
	}
	sqlite3DB.SetModel(UserSqliteModel{})
	// result, err := sqlite3DB.Insert(user)
	result, err := sqlite3DB.Fields("Id", "Password", "Create_time").Insert(user)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	id, err := result.LastInsertId()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	fmt.Println("id", id)

	user.Id = &id

	if user.Username != nil {
		fmt.Println("Username", *user.Username)
	}
	if user.Create_time != nil {
		fmt.Println("Create_time", *user.Create_time)
	}
	if user.Update_time != nil {
		fmt.Println("Update_time", *user.Update_time)
	}

	// jsonData, err := json.Marshal(user)
	// if err != nil {
	// 	return goi.Response{
	// 		Status: http.StatusInternalServerError,
	// 		Data:   err.Error(),
	// 	}
	// }
	return goi.Response{
		Status: http.StatusOK,
		Data:   user,
	}
}

// 参数验证
type userSqliteUpdateParams struct {
	User_id  int    `name:"user_id" required:"int"`
	Username string `name:"username" optional:"string"`
	Password string `name:"password" optional:"string"`
}

// 修改 Model
func UserSqliteUpdate(request *goi.Request) interface{} {
	var params userSqliteUpdateParams
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams()
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	if params.User_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	sqlite3DB := db.SQLite3Connect("sqlite")

	update_time := goi.GetTime().Format("2006-01-02 15:04:05")
	// sqlite3 数据库
	update_user := &UserSqliteModel{
		Username:    nil,
		Password:    nil,
		Update_time: &update_time,
	}
	if params.Username != "" {
		update_user.Username = &params.Username
	}
	if params.Password != "" {
		update_user.Password = &params.Password
	}
	sqlite3DB.SetModel(UserSqliteModel{})
	result, err := sqlite3DB.Where("id=?", params.User_id).Update(update_user)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	rowNum, err := result.RowsAffected()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if rowNum == 0 {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Code:    http.StatusInternalServerError,
				Message: "修改失败",
				Results: nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)

	// sqlite3 数据库
	user := UserSqliteModel{}
	sqlite3DB.SetModel(UserSqliteModel{})
	err = sqlite3DB.Where("id=?", params.User_id).First(&user)
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if user.Id != nil {
		fmt.Println("Id", *user.Id)
	}
	if user.Username != nil {
		fmt.Println("Username", *user.Username)
	}
	if user.Create_time != nil {
		fmt.Println("Create_time", *user.Create_time)
	}
	if user.Update_time != nil {
		fmt.Println("Update_time", *user.Update_time)
	}

	jsonData, err := json.Marshal(user)
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	return goi.Response{
		Status: http.StatusOK,
		Data:   jsonData,
	}
}

// 删除 Model
func UserSqliteDelete(request *goi.Request) interface{} {
	var user_id int
	var validationErr goi.ValidationError
	var err error
	validationErr = request.PathParams.Get("user_id", &user_id)
	if validationErr != nil {
		return validationErr.Response()
	}

	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	sqlite3DB := db.SQLite3Connect("sqlite")

	// sqlite3 数据库
	sqlite3DB.SetModel(UserSqliteModel{})
	result, err := sqlite3DB.Where("id=?", user_id).Delete()

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	rowNum, err := result.RowsAffected()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if rowNum == 0 {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Code:    http.StatusOK,
				Message: "已删除",
				Results: nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: "删除成功",
			Results: nil,
		},
	}
}
