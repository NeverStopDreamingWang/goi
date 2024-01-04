package user

import (
	"encoding/json"
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"github.com/NeverStopDreamingWang/hgee/db"
	"net/http"
	"time"
)

// 读取多条数据到 Model
func TestModelList(request *hgee.Request) any {
	// mysqlObj, err := db.MysqlConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	Sqlite3DB, err := db.Sqlite3Connect("sqlite_1")
	defer Sqlite3DB.Close()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	userSlice := make([]any, 5) // 切片的长度，决定取多少行数据

	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})                                                         // 设置操作表
	// err = mysqlObj.Fields("Id", "Username", "Password").Where("id>?", 1).Select(userSlice) // 将数据读到 user 中

	// sqlite3 数据库
	Sqlite3DB.SetModel(UserSqliteModel{})
	err = Sqlite3DB.Fields("Id", "Username", "Password").Where("id>?", 1).Select(userSlice)

	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	for i, model := range userSlice {
		if model == nil {
			break
		}
		// user := model.(*UserModel) // mysql 数据库
		user := model.(*UserSqliteModel) // sqlite3 数据库
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

	jsonData, err := json.Marshal(userSlice)
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	return hgee.Response{
		Status: http.StatusOK,
		Data:   jsonData,
	}
}

// 读取第一条数据到 Model
func TestModelRetrieve(request *hgee.Request) any {
	user_id := request.PathParams.Get("user_id").(int)

	if user_id == 0 {
		// 返回 hgee.Response
		return hgee.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	// mysqlObj, err := db.MysqlConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	Sqlite3DB, err := db.Sqlite3Connect("sqlite_1")
	defer Sqlite3DB.Close()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// mysqlObj.SetModel(UserModel{}).Where().First()

	// mysql 数据库
	// user := &UserModel{}
	// mysqlObj.SetModel(UserModel{})                                       // 设置操作表
	// err = mysqlObj.Fields("Id", "Username").Where("id=?", user_id).First(user) // 将数据读到 user 中
	// if err != nil {
	// 	return hgee.Response{
	// 		Status: http.StatusInternalServerError,
	// 		Data:   err.Error(),
	// 	}
	// }

	// sqlite3 数据库
	user := &UserSqliteModel{}
	Sqlite3DB.SetModel(UserSqliteModel{})
	err = Sqlite3DB.Fields("Id", "Username").Where("id=?", user_id).First(user)
	if err != nil {
		return hgee.Response{
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
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	return hgee.Response{
		Status: http.StatusOK,
		Data:   jsonData,
	}
}

// 添加一条数据到 Model
func TestModelCreate(request *hgee.Request) any {
	// mysqlObj, err := db.MysqlConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	Sqlite3DB, err := db.Sqlite3Connect("sqlite_1")
	defer Sqlite3DB.Close()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	username := request.BodyParams.Get("username").(string)
	password := request.BodyParams.Get("password").(string)
	create_time := time.Now().In(hgee.Settings.LOCATION).Format("2006-01-02 15:04:05")
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
	Sqlite3DB.SetModel(UserSqliteModel{})
	// result, err := Sqlite3DB.Insert(user)
	result, err := Sqlite3DB.Fields("Id", "Password", "Create_time").Insert(user)

	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	id, err := result.LastInsertId()
	if err != nil {
		return hgee.Response{
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
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	return hgee.Response{
		Status: http.StatusOK,
		Data:   jsonData,
	}
}

// 修改 Model
func TestModelUpdate(request *hgee.Request) any {
	user_id := request.PathParams.Get("user_id").(int)
	username := request.BodyParams.Get("username").(string)

	if user_id == 0 {
		// 返回 hgee.Response
		return hgee.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	// mysqlObj, err := db.MysqlConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	Sqlite3DB, err := db.Sqlite3Connect("sqlite_1")
	defer Sqlite3DB.Close()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	update_time := time.Now().In(hgee.Settings.LOCATION).Format("2006-01-02 15:04:05")
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
	Sqlite3DB.SetModel(UserSqliteModel{})
	result, err := Sqlite3DB.Where("id=?", user_id).Update(update_user)

	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	rowNum, err := result.RowsAffected()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if rowNum == 0 {
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusInternalServerError,
				Msg:    "修改失败！",
				Data:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)

	// sqlite3 数据库
	user := &UserSqliteModel{}
	Sqlite3DB.SetModel(UserSqliteModel{})
	err = Sqlite3DB.Where("id=?", user_id).First(user)
	if err != nil {
		return hgee.Response{
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
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	return hgee.Response{
		Status: http.StatusOK,
		Data:   jsonData,
	}
}

// 删除 Model
func TestModelDelete(request *hgee.Request) any {
	user_id := request.PathParams.Get("user_id").(int)

	if user_id == 0 {
		// 返回 hgee.Response
		return hgee.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	// mysqlObj, err := db.MysqlConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	Sqlite3DB, err := db.Sqlite3Connect("sqlite_1")
	defer Sqlite3DB.Close()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Where("id=?", user_id).Delete()

	// sqlite3 数据库
	Sqlite3DB.SetModel(UserSqliteModel{})
	result, err := Sqlite3DB.Where("id=?", user_id).Delete()

	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	rowNum, err := result.RowsAffected()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if rowNum == 0 {
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusOK,
				Msg:    "已删除！",
				Data:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)

	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "删除成功！",
			Data:   nil,
		},
	}
}
