package user

import (
	"errors"
	"time"

	"example/utils"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/auth"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/sqlite3"
)

func (self UserModel) Validate() error {
	// 自定义验证
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)

	if self.Id != nil {
		sqlite3DB = sqlite3DB.Where("`id` != ?", self.Id)
	}

	if self.Username != nil {
		flag, err := sqlite3DB.Where("`username` = ?", self.Username).Exists()
		if err != nil {
			return errors.New("查询数据库错误")
		}
		if flag == true {
			return errors.New("用户名重复")
		}
	}
	return nil
}

func (self *UserModel) Create() error {
	if self.Create_time == nil {
		Create_time := goi.GetTime().Format(time.DateTime)
		self.Create_time = &Create_time
	}

	// 密码加密
	encryptPassword, err := auth.MakePassword(*self.Password)
	if err != nil {
		return errors.New("密码格式错误")
	}
	self.Password = &encryptPassword

	err = sqlite3.Validate(self, true)
	if err != nil {
		return err
	}

	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	result, err := sqlite3DB.Insert(self)
	if err != nil {
		return errors.New("添加用户错误")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return errors.New("添加用户错误")
	}
	self.Id = &id
	return nil
}

func (self *UserModel) Update(validated_data *UserModel) error {
	if validated_data.Password != nil {
		// 密码加密
		encryptPassword, err := auth.MakePassword(*validated_data.Password)
		if err != nil {
			return errors.New("密码格式错误")
		}
		validated_data.Password = &encryptPassword
	}

	Update_time := goi.GetTime().Format(time.DateTime)
	validated_data.Update_time = &Update_time

	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	_, err := sqlite3DB.Where("`id` = ?", self.Id).Update(validated_data)
	if err != nil {
		return errors.New("修改用户错误")
	}

	utils.Update(self, validated_data)
	return nil
}

func (self UserModel) Delete() error {
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	_, err := sqlite3DB.Where("`id` = ?", self.Id).Delete()
	if err != nil {
		return errors.New("删除用户错误")
	}
	return nil
}
