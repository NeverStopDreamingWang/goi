package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
	"strings"
)

type MySQLDB struct {
	Name   string
	DB     *sql.DB
	model  model.MySQLModel
	fields []string
	sql    string
	args   []any
}

// 连接 MySQL
func MetaMySQLConnect(Database goi.MetaDataBase) (*MySQLDB, error) {
	connectStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", Database.USER, Database.PASSWORD, Database.HOST, Database.PORT, Database.NAME)
	mysqlDB, err := sql.Open(Database.ENGINE, connectStr)
	return &MySQLDB{
		Name:   "",
		DB:     mysqlDB,
		model:  nil,
		fields: nil,
		sql:    "",
		args:   nil,
	}, err
}

// 关闭连接
func (mysqlDB *MySQLDB) Close() error {
	err := mysqlDB.DB.Close()
	return err
}

// 执行语句
func (mysqlDB *MySQLDB) Execute(query string, args ...any) (sql.Result, error) {
	// 开启事务
	transaction, err := mysqlDB.DB.Begin()
	if err != nil {
		return nil, err
	}

	// 执行SQL语句
	result, err := transaction.Exec(query, args...)
	if err != nil {
		rollErr := transaction.Rollback() // 回滚事务
		if rollErr != nil {
			return result, rollErr
		}
		return result, err
	}

	// 提交事务
	err = transaction.Commit()
	if err != nil {
		return result, err
	}

	return result, err
}

// 查询语句
func (mysqlDB *MySQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	rows, err := mysqlDB.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, err
}

// 设置使用模型
func (mysqlDB *MySQLDB) SetModel(model model.MySQLModel) *MySQLDB {
	mysqlDB.model = model
	ModelType := reflect.TypeOf(model)
	// 获取字段
	mysqlDB.fields = make([]string, ModelType.NumField())
	for i := 0; i < ModelType.NumField(); i++ {
		mysqlDB.fields[i] = ModelType.Field(i).Name
	}
	return mysqlDB
}

// 设置查询字段
func (mysqlDB *MySQLDB) Fields(fields ...string) *MySQLDB {
	mysqlDB.fields = fields
	return mysqlDB
}

// 插入数据库
func (mysqlDB *MySQLDB) Insert(ModelData model.MySQLModel) (sql.Result, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
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

	return mysqlDB.Execute(insertSQL, insertValues...)
}

// 查询语句
func (mysqlDB *MySQLDB) Where(query string, args ...any) *MySQLDB {
	mysqlDB.sql = query
	mysqlDB.args = args
	return mysqlDB
}

// 执行查询语句获取数据
func (mysqlDB *MySQLDB) Select() (any, error) {
	var queryResult []model.MySQLModel

	if mysqlDB.model == nil {
		return queryResult, errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	ModelType := reflect.TypeOf(mysqlDB.model)
	if ModelType.Kind() == reflect.Ptr {
		ModelType = ModelType.Elem()
	}

	queryFields := make([]string, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}

	fieldsSQl := strings.Join(queryFields, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v` WHERE %v", fieldsSQl, TableName, mysqlDB.sql)

	rows, err := mysqlDB.DB.Query(mysqlDB.sql, mysqlDB.args...)
	defer rows.Close()
	if err != nil {
		return queryResult, err
	} else if rows.Err() != nil {
		return queryResult, rows.Err()
	}

	for rows.Next() {
		scanValues := make([]any, len(queryFields))
		ModelItem := reflect.New(ModelType)

		if ModelItem.Kind() == reflect.Ptr {
			ModelItem = ModelItem.Elem()
		}
		for i, fieldName := range mysqlDB.fields {
			field := ModelItem.FieldByName(fieldName)
			scanValues[i] = field.Addr().Interface()
		}

		err = rows.Scan(scanValues...)
		if err != nil {
			return queryResult, err
		}
		queryResult = append(queryResult, ModelItem.Interface().(model.MySQLModel))
	}
	return queryResult, nil
}

// 返回第一条数据
func (mysqlDB *MySQLDB) First() (any, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	ModelType := reflect.TypeOf(mysqlDB.model)
	if ModelType.Kind() == reflect.Ptr {
		ModelType = ModelType.Elem()
	}

	queryResult := reflect.New(ModelType)
	if queryResult.Kind() == reflect.Ptr {
		queryResult = queryResult.Elem()
	}

	queryFields := make([]string, len(mysqlDB.fields))
	scanValues := make([]any, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		queryFields[i] = strings.ToLower(fieldName)
		field := queryResult.FieldByName(fieldName)
		scanValues[i] = field.Addr().Interface()
	}

	fieldsSQl := strings.Join(queryFields, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v` WHERE %v", fieldsSQl, TableName, mysqlDB.sql)

	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
	if row == nil {
		return nil, nil
	} else if row.Err() != nil {
		return nil, row.Err()
	}

	err := row.Scan(scanValues...)
	if err != nil {
		return nil, err
	}
	return queryResult.Interface().(model.MySQLModel), nil
}

// 更新数据，返回操作条数
func (mysqlDB *MySQLDB) Update(ModelData model.MySQLModel) (sql.Result, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	if mysqlDB.sql == "" {
		return nil, errors.New("请先设置 Where 条件")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
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
	return mysqlDB.Execute(mysqlDB.sql, updateValues...)
}

// 删除数据，返回操作条数
func (mysqlDB *MySQLDB) Delete() (sql.Result, error) {
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	if mysqlDB.sql == "" {
		return nil, errors.New("请先设置 Where 条件")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	mysqlDB.sql = fmt.Sprintf("DELETE FROM `%v` WHERE %v", TableName, mysqlDB.sql)

	return mysqlDB.Execute(mysqlDB.sql, mysqlDB.args...)
}
