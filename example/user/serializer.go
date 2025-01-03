package user

import (
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/auth"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/serializer/mysql"
)

type UserModelSerializer struct {
	mysql.ModelSerializer

	// 实例
	Instance *UserModel
}

func (serializer UserModelSerializer) Validate(mysqlDB *db.MySQLDB, attrs UserModel, partial bool) goi.ValidationError {
	// 调用 ModelSerializer.Validate
	validationErr := serializer.ModelSerializer.Validate(mysqlDB, attrs, partial)
	if validationErr != nil {
		return validationErr
	}

	// 自定义验证
	mysqlDB.SetModel(UserModel{})

	if serializer.Instance != nil {
		mysqlDB = mysqlDB.Where("`id` != ?", serializer.Instance.Id)
	}

	if attrs.Username != nil {
		flag, err := mysqlDB.Where("`username` = ?", attrs.Username).Exists()
		if err != nil {
			return goi.NewValidationError(http.StatusBadRequest, "查询数据库错误")
		}
		if flag == true {
			return goi.NewValidationError(http.StatusBadRequest, "用户名重复")
		}
	}
	return nil
}

func (serializer UserModelSerializer) Create(mysqlDB *db.MySQLDB, validated_data *UserModel) goi.ValidationError {
	// 调用 serializer.Validate
	validationErr := serializer.Validate(mysqlDB, *validated_data, false)
	if validationErr != nil {
		return validationErr
	}

	var err error
	// 密码加密
	encryptPassword, err := auth.MakePassword(*validated_data.Password)
	if err != nil {
		return goi.NewValidationError(http.StatusBadRequest, "密码格式错误")
	}
	validated_data.Password = &encryptPassword

	mysqlDB.SetModel(*validated_data)
	result, err := mysqlDB.Insert(*validated_data)
	if err != nil {
		return goi.NewValidationError(http.StatusInternalServerError, "添加用户错误")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return goi.NewValidationError(http.StatusInternalServerError, "添加用户错误")
	}
	validated_data.Id = &id
	return nil
}

func (serializer UserModelSerializer) Update(mysqlDB *db.MySQLDB, instance *UserModel, validated_data *UserModel) goi.ValidationError {
	var err error

	// 设置当前实例
	serializer.Instance = instance

	// 调用 serializer.Validate
	validationErr := serializer.Validate(mysqlDB, *validated_data, true)
	if validationErr != nil {
		return validationErr
	}

	if validated_data.Password != nil {
		// 密码加密
		encryptPassword, err := auth.MakePassword(*validated_data.Password)
		if err != nil {
			return goi.NewValidationError(http.StatusInternalServerError, "密码格式错误")
		}
		validated_data.Password = &encryptPassword
	}

	mysqlDB.SetModel(UserModel{})

	_, err = mysqlDB.Where("`id` = ?", serializer.Instance.Id).Update(validated_data)
	if err != nil {
		return goi.NewValidationError(http.StatusInternalServerError, "修改用户错误")
	}

	// 更新 instance 当前实例
	serializer.ModelSerializer.Update(instance, validated_data)
	return nil
}
