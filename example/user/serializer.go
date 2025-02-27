package user

import (
	"errors"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/auth"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/serializer/sqlite3"
)

type UserModelSerializer struct {
	sqlite3.ModelSerializer

	// 实例
	Instance *UserModel
}

func (serializer UserModelSerializer) Validate(sqlite3DB *db.SQLite3DB, attrs UserModel, partial bool) error {
	var err error
	// 调用 ModelSerializer.Validate
	err = serializer.ModelSerializer.Validate(sqlite3DB, attrs, partial)
	if err != nil {
		return err
	}

	// 自定义验证
	sqlite3DB.SetModel(attrs)

	if serializer.Instance != nil {
		sqlite3DB = sqlite3DB.Where("`id` != ?", serializer.Instance.Id)
	}

	if attrs.Username != nil {
		flag, err := sqlite3DB.Where("`username` = ?", attrs.Username).Exists()
		if err != nil {
			return errors.New("查询数据库错误")
		}
		if flag == true {
			return errors.New("用户名重复")
		}
	}
	return nil
}

func (serializer UserModelSerializer) Create(sqlite3DB *db.SQLite3DB, validated_data *UserModel) error {
	var err error

	if validated_data.Create_time == nil {
		Create_time := goi.GetTime().Format("2006-01-02 15:04:05")
		validated_data.Create_time = &Create_time
	}

	// 密码加密
	encryptPassword, err := auth.MakePassword(*validated_data.Password)
	if err != nil {
		return errors.New("密码格式错误")
	}
	validated_data.Password = &encryptPassword

	sqlite3DB.SetModel(*validated_data)
	result, err := sqlite3DB.Insert(*validated_data)
	if err != nil {
		return errors.New("添加用户错误")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return errors.New("添加用户错误")
	}
	validated_data.Id = &id
	return nil
}

func (serializer UserModelSerializer) Update(sqlite3DB *db.SQLite3DB, validated_data *UserModel) error {
	var err error

	Update_time := goi.GetTime().Format("2006-01-02 15:04:05")
	validated_data.Update_time = &Update_time

	if validated_data.Password != nil {
		// 密码加密
		encryptPassword, err := auth.MakePassword(*validated_data.Password)
		if err != nil {
			return errors.New("密码格式错误")
		}
		validated_data.Password = &encryptPassword
	}

	sqlite3DB.SetModel(*validated_data)

	_, err = sqlite3DB.Where("`id` = ?", serializer.Instance.Id).Update(validated_data)
	if err != nil {
		return errors.New("修改用户错误")
	}

	// 更新 instance 当前实例
	serializer.ModelSerializer.Update(serializer.Instance, validated_data)
	return nil
}
