package user

import (
	"errors"

	"github.com/NeverStopDreamingWang/goi/auth"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/serializer/mysql"
)

type UserModelSerializer struct {
	mysql.ModelSerializer

	// 实例
	Instance *UserModel
}

func (serializer UserModelSerializer) Validate(mysqlDB *db.MySQLDB, attrs UserModel, partial bool) error {
	var err error
	// 调用 ModelSerializer.Validate
	err = serializer.ModelSerializer.Validate(mysqlDB, attrs, partial)
	if err != nil {
		return err
	}

	// 自定义验证
	mysqlDB.SetModel(attrs)

	if serializer.Instance != nil {
		mysqlDB = mysqlDB.Where("`id` != ?", serializer.Instance.Id)
	}

	if attrs.Username != nil {
		flag, err := mysqlDB.Where("`username` = ?", attrs.Username).Exists()
		if err != nil {
			return errors.New("查询数据库错误")
		}
		if flag == true {
			return errors.New("用户名重复")
		}
	}
	return nil
}

func (serializer UserModelSerializer) Create(mysqlDB *db.MySQLDB, validated_data *UserModel) error {
	var err error
	// 密码加密
	encryptPassword, err := auth.MakePassword(*validated_data.Password)
	if err != nil {
		return errors.New("密码格式错误")
	}
	validated_data.Password = &encryptPassword

	mysqlDB.SetModel(*validated_data)
	result, err := mysqlDB.Insert(*validated_data)
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

func (serializer UserModelSerializer) Update(mysqlDB *db.MySQLDB, validated_data *UserModel) error {
	var err error

	if validated_data.Password != nil {
		// 密码加密
		encryptPassword, err := auth.MakePassword(*validated_data.Password)
		if err != nil {
			return errors.New("密码格式错误")
		}
		validated_data.Password = &encryptPassword
	}

	mysqlDB.SetModel(*validated_data)

	_, err = mysqlDB.Where("`id` = ?", serializer.Instance.Id).Update(validated_data)
	if err != nil {
		return errors.New("修改用户错误")
	}

	// 更新 instance 当前实例
	serializer.ModelSerializer.Update(serializer.Instance, validated_data)
	return nil
}
