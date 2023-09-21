package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"github.com/NeverStopDreamingWang/hgee/model"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
	"strings"
)

type MysqlDB struct {
	Name   string
	DB     *sql.DB
	model  model.MysqlModel
	fields []string
	sql    string
	args   []any
}

// 连接 Mysql
func MetaMysqlConnect(Database hgee.MetaDataBase) (*MysqlDB, error) {
	connectStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", Database.USER, Database.PASSWORD, Database.HOST, Database.PORT, Database.NAME)
	mysqlDB, err := sql.Open(Database.ENGINE, connectStr)
	return &MysqlDB{
		Name:   "",
		DB:     mysqlDB,
		model:  nil,
		fields: nil,
		sql:    "",
		args:   nil,
	}, err
}

// 关闭连接
func (mysqlDB *MysqlDB) Close() error {
	err := mysqlDB.DB.Close()
	return err
}

// 执行语句
func (mysqlDB *MysqlDB) Execute(query string, args ...any) error {
	// 开启事务
	transaction, err := mysqlDB.DB.Begin()
	if err != nil {
		return err
	}

	// 执行SQL语句
	_, err = transaction.Exec(query, args...)
	if err != nil {
		transaction.Rollback() // 回滚事务
		return err
	}

	// 提交事务
	err = transaction.Commit()
	if err != nil {
		return err
	}

	return err
}

// 查询语句
func (mysqlDB *MysqlDB) Query(query string, args ...any) (*sql.Rows, error) {
	rows, err := mysqlDB.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, err
}

// 设置使用模型
func (mysqlDB *MysqlDB) SetModel(model model.MysqlModel) *MysqlDB {
	mysqlDB.model = model
	return mysqlDB
}

// 设置查询字段
func (mysqlDB *MysqlDB) Fields(fields ...string) *MysqlDB {
	mysqlDB.fields = fields
	return mysqlDB
}

// 插入数据库
func (mysqlDB *MysqlDB) Insert(ModelData model.MysqlModel) (sql.Result, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TableName

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}
	ModelType := ModelValue.Type()

	// 获取字段
	if len(mysqlDB.fields) == 0 {
		mysqlDB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelValue.NumField(); i++ {
			mysqlDB.fields[i] = ModelType.Field(i).Name
		}
	}

	insertValues := make([]any, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		field := ModelValue.FieldByName(fieldName)
		if field.Elem().IsValid() {
			insertValues[i] = field.Elem().Interface()
		} else {
			insertValues[i] = field.Interface()
		}
	}

	fieldsSQL := strings.Join(mysqlDB.fields, "`,`")

	tempValues := make([]string, len(mysqlDB.fields))
	for i := 0; i < len(mysqlDB.fields); i++ {
		tempValues[i] = "?"
	}
	valuesSQL := strings.Join(tempValues, ",")

	insertSQL := fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return mysqlDB.DB.Exec(insertSQL, insertValues...)
}

// 查询语句
func (mysqlDB *MysqlDB) Where(query string, args ...any) *MysqlDB {
	mysqlDB.sql = query
	mysqlDB.args = args
	return mysqlDB
}

// 执行查询语句获取所有
func (mysqlDB *MysqlDB) Select(ModelSlice []any) error {
	if mysqlDB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TableName

	ModelType := reflect.TypeOf(mysqlDB.model)
	if ModelType.Kind() == reflect.Ptr {
		ModelType = ModelType.Elem()
	}

	if len(mysqlDB.fields) == 0 {
		mysqlDB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelType.NumField(); i++ {
			mysqlDB.fields[i] = ModelType.Field(i).Name
		}
	}

	queryFields := make([]string, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}

	fieldsSQl := strings.Join(queryFields, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v` WHERE %v", fieldsSQl, TableName, mysqlDB.sql)

	rows, err := mysqlDB.DB.Query(mysqlDB.sql, mysqlDB.args...)
	if err != nil {
		return err
	} else if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()

	idx := 0
	for rows.Next() {
		if len(ModelSlice) == idx {
			break
		}
		scanValues := make([]any, len(queryFields))
		ModelData := reflect.New(ModelType).Interface()
		ModelSlice[idx] = ModelData

		ModelValue := reflect.ValueOf(ModelData)
		if ModelValue.Kind() == reflect.Ptr {
			ModelValue = ModelValue.Elem()
		}
		for i, fieldName := range mysqlDB.fields {
			field := ModelValue.FieldByName(fieldName)
			scanValues[i] = field.Addr().Interface()
		}

		err = rows.Scan(scanValues...)
		if err != nil {
			return err
		}
		idx += 1
	}
	for len(ModelSlice) != idx {
		ModelData := reflect.New(ModelType).Interface()
		ModelSlice[idx] = ModelData
		idx += 1
	}
	return nil
}

// 返回第一条数据
func (mysqlDB *MysqlDB) First(ModelData model.MysqlModel) error {
	if mysqlDB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TableName

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}
	ModelType := ModelValue.Type()

	if len(mysqlDB.fields) == 0 {
		mysqlDB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelType.NumField(); i++ {
			mysqlDB.fields[i] = ModelType.Field(i).Name
		}
	}

	queryFields := make([]string, len(mysqlDB.fields))
	scanValues := make([]any, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		queryFields[i] = strings.ToLower(fieldName)
		field := ModelValue.FieldByName(fieldName)
		scanValues[i] = field.Addr().Interface()
	}

	fieldsSQl := strings.Join(queryFields, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v` WHERE %v", fieldsSQl, TableName, mysqlDB.sql)

	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
	if row == nil {
		return nil
	} else if row.Err() != nil {
		return row.Err()
	}

	err := row.Scan(scanValues...)
	if err != nil {
		return err
	}
	return nil
}

// 更新数据，返回操作条数
func (mysqlDB *MysqlDB) Update(ModelData model.MysqlModel) (sql.Result, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	if mysqlDB.sql == "" {
		return nil, errors.New("请先设置 Where 条件")
	}
	TableName := mysqlDB.model.ModelSet().TableName

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}
	ModelType := ModelValue.Type()

	// 获取字段
	if len(mysqlDB.fields) == 0 {
		mysqlDB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelValue.NumField(); i++ {
			mysqlDB.fields[i] = ModelType.Field(i).Name
		}
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]any, 0)
	for _, fieldName := range mysqlDB.fields {
		field := ModelValue.FieldByName(fieldName)
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		if field.IsValid() {
			fieldValue := field.Interface()
			updateFields = append(updateFields, fmt.Sprintf("`%v`=?", strings.ToLower(fieldName)))
			updateValues = append(updateValues, fieldValue)
		}
	}
	fieldsSQl := strings.Join(updateFields, ",")

	mysqlDB.sql = fmt.Sprintf("UPDATE `%v` SET %v WHERE %v", TableName, fieldsSQl, mysqlDB.sql)

	updateValues = append(updateValues, mysqlDB.args...)
	return mysqlDB.DB.Exec(mysqlDB.sql, updateValues...)
}

// 删除数据，返回操作条数
func (mysqlDB *MysqlDB) Delete() (sql.Result, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	if mysqlDB.sql == "" {
		return nil, errors.New("请先设置 Where 条件")
	}
	TableName := mysqlDB.model.ModelSet().TableName

	mysqlDB.sql = fmt.Sprintf("DELETE FROM `%v` WHERE %v", TableName, mysqlDB.sql)

	return mysqlDB.DB.Exec(mysqlDB.sql, mysqlDB.args...)
}
