package user

import (
	"encoding/json"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"net/http"
	"time"
)

// 读取多条数据到 Model
func TestModelList(request *goi.Request) any {
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})                                                              // 设置操作表
	// resultSlice, err := mysqlObj.Fields("Id", "Username", "Password").Where("id>?", 1).Select()

	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	resultSlice, err := SQLite3DB.Fields("Id", "Username", "Password").Where("id>? limit 1, 2", 1).Select()

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	for i, item := range resultSlice {
		// user := item.(UserModel) // mysql 数据库
		user := item.(UserSqliteModel) // sqlite3 数据库
		// *user.Username = "test"
		fmt.Println(i)
		if user.Id != nil {
			fmt.Println("Id", user.Id)
		}
		if user.Username != nil {
			fmt.Println("Username", user.Username)
		}
		if user.Create_time != nil {
			fmt.Println("Create_time", user.Create_time)
		}
		if user.Update_time != nil {
			fmt.Println("Update_time", user.Update_time)
		}
	}

	jsonData, err := json.Marshal(resultSlice)
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

// 读取第一条数据到 Model
func TestModelRetrieve(request *goi.Request) any {
	user_id := request.PathParams.Get("user_id").(int)

	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})                                                  // 设置操作表
	// result, err := mysqlObj.Fields("Id", "Username").Where("id=?", user_id).First()

	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	result, err := SQLite3DB.Fields("Id", "Username").Where("id=?", user_id).First()

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// user := result.(UserModel)
	user := result.(UserSqliteModel)
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

// 添加一条数据到 Model
func TestModelCreate(request *goi.Request) any {
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	username := request.BodyParams.Get("username").(string)
	password := request.BodyParams.Get("password").(string)
	create_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
	// mysql 数据库
	// user := &UserModel{
	// 	Username:    &username,
	// 	Password:    &password,
	// 	Create_time: &create_time,
	// }
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Insert(user)
	// result, err := mysqlObj.Fields("Id", "Password").Insert(user) // 指定插入字段

	// // sqlite3 数据库
	user := &UserSqliteModel{
		Username:    &username,
		Password:    &password,
		Create_time: &create_time,
	}
	SQLite3DB.SetModel(UserSqliteModel{})
	// result, err := SQLite3DB.Insert(user)
	result, err := SQLite3DB.Fields("Id", "Password", "Create_time").Insert(user)

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

// 修改 Model
func TestModelUpdate(request *goi.Request) any {
	user_id := request.PathParams.Get("user_id").(int)
	username := request.BodyParams.Get("username").(string)

	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	update_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
	// mysql 数据库
	// update_user := &UserModel{
	// 	Username:    &username,
	// 	Update_time: &update_time,
	// }
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Where("id=?", user_id).Update(update_user)
	// result, err := mysqlObj.Fields("Username").Where("id=?", user_id).Update(update_user) // 更新指定字段

	// sqlite3 数据库
	update_user := &UserSqliteModel{
		Username:    &username,
		Update_time: &update_time,
	}
	SQLite3DB.SetModel(UserSqliteModel{})
	result, err := SQLite3DB.Where("id=?", user_id).Update(update_user)

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
				Status: http.StatusInternalServerError,
				Msg:    "修改失败！",
				Data:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)

	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	queryResult, err := SQLite3DB.Where("id=?", user_id).First()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	user := queryResult.(UserSqliteModel)

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
func TestModelDelete(request *goi.Request) any {
	user_id := request.PathParams.Get("user_id").(int)

	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Where("id=?", user_id).Delete()

	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	result, err := SQLite3DB.Where("id=?", user_id).Delete()

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
				Status: http.StatusOK,
				Msg:    "已删除！",
				Data:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "删除成功！",
			Data:   nil,
		},
	}
}
