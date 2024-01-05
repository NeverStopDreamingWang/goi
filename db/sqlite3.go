package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path"
	"reflect"
	"strings"
)

type Sqlite3DB struct {
	Name   string
	DB     *sql.DB
	model  model.Sqlite3Model
	fields []string
	sql    string
	args   []any
}

// 连接 Sqlite3
func MetaSqlite3Connect(Database goi.MetaDataBase) (*Sqlite3DB, error) {
	dir := path.Dir(Database.NAME)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0644)
		if err != nil {
			panic(fmt.Sprintf("创建数据库目录错误: ", err))
		}
	}
	sqlite3DB, err := sql.Open(Database.ENGINE, Database.NAME)
	return &Sqlite3DB{
		Name:   "",
		DB:     sqlite3DB,
		model:  nil,
		fields: nil,
		sql:    "",
		args:   nil,
	}, err
}

// 关闭连接
func (sqlite3DB *Sqlite3DB) Close() error {
	err := sqlite3DB.DB.Close()
	return err
}

// 执行语句
func (sqlite3DB *Sqlite3DB) Execute(query string, args ...any) (sql.Result, error) {
	// 开启事务
	transaction, err := sqlite3DB.DB.Begin()
	if err != nil {
		return nil, err
	}

	// 执行SQL语句
	result, err := transaction.Exec(query, args...)
	if err != nil {
		err = transaction.Rollback() // 回滚事务
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	// 提交事务
	err = transaction.Commit()
	if err != nil {
		return nil, err
	}

	return result, err
}

// 查询语句
func (sqlite3DB *Sqlite3DB) Query(query string, args ...any) (*sql.Rows, error) {
	rows, err := sqlite3DB.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, err
}

// 设置使用模型
func (sqlite3DB *Sqlite3DB) SetModel(model model.Sqlite3Model) *Sqlite3DB {
	sqlite3DB.model = model
	return sqlite3DB
}

// 设置查询字段
func (sqlite3DB *Sqlite3DB) Fields(fields ...string) *Sqlite3DB {
	sqlite3DB.fields = fields
	return sqlite3DB
}

// 插入数据库
func (sqlite3DB *Sqlite3DB) Insert(ModelData model.Sqlite3Model) (sql.Result, error) {
	if sqlite3DB.model == nil {
		return nil, errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TableName

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}
	ModelType := ModelValue.Type()

	// 获取字段
	if len(sqlite3DB.fields) == 0 {
		sqlite3DB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelValue.NumField(); i++ {
			sqlite3DB.fields[i] = ModelType.Field(i).Name
		}
	}

	insertValues := make([]any, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		field := ModelValue.FieldByName(fieldName)
		if field.Elem().IsValid() {
			insertValues[i] = field.Elem().Interface()
		} else {
			insertValues[i] = field.Interface()
		}
	}

	fieldsSQL := strings.Join(sqlite3DB.fields, "`,`")

	tempValues := make([]string, len(sqlite3DB.fields))
	for i := 0; i < len(sqlite3DB.fields); i++ {
		tempValues[i] = "?"
	}
	valuesSQL := strings.Join(tempValues, ",")

	insertSQL := fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return sqlite3DB.Execute(insertSQL, insertValues...)
}

// 查询语句
func (sqlite3DB *Sqlite3DB) Where(query string, args ...any) *Sqlite3DB {
	sqlite3DB.sql = query
	sqlite3DB.args = args
	return sqlite3DB
}

// 执行查询语句获取数据
func (sqlite3DB *Sqlite3DB) Select(ModelSlice []any) error {
	if sqlite3DB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TableName

	ModelType := reflect.TypeOf(sqlite3DB.model)
	if ModelType.Kind() == reflect.Ptr {
		ModelType = ModelType.Elem()
	}

	if len(sqlite3DB.fields) == 0 {
		sqlite3DB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelType.NumField(); i++ {
			sqlite3DB.fields[i] = ModelType.Field(i).Name
		}
	}

	queryFields := make([]string, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}

	fieldsSQl := strings.Join(queryFields, "`,`")

	sqlite3DB.sql = fmt.Sprintf("SELECT `%v` FROM `%v` WHERE %v", fieldsSQl, TableName, sqlite3DB.sql)

	rows, err := sqlite3DB.DB.Query(sqlite3DB.sql, sqlite3DB.args...)
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
		for i, fieldName := range sqlite3DB.fields {
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
func (sqlite3DB *Sqlite3DB) First(ModelData model.Sqlite3Model) error {
	if sqlite3DB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TableName

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}
	ModelType := ModelValue.Type()

	if len(sqlite3DB.fields) == 0 {
		sqlite3DB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelType.NumField(); i++ {
			sqlite3DB.fields[i] = ModelType.Field(i).Name
		}
	}

	queryFields := make([]string, len(sqlite3DB.fields))
	scanValues := make([]any, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		queryFields[i] = strings.ToLower(fieldName)
		field := ModelValue.FieldByName(fieldName)
		scanValues[i] = field.Addr().Interface()
	}

	fieldsSQl := strings.Join(queryFields, "`,`")

	sqlite3DB.sql = fmt.Sprintf("SELECT `%v` FROM `%v` WHERE %v", fieldsSQl, TableName, sqlite3DB.sql)

	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)
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
func (sqlite3DB *Sqlite3DB) Update(ModelData model.Sqlite3Model) (sql.Result, error) {
	if sqlite3DB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	if sqlite3DB.sql == "" {
		return nil, errors.New("请先设置 Where 条件")
	}
	TableName := sqlite3DB.model.ModelSet().TableName

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}
	ModelType := ModelValue.Type()

	// 获取字段
	if len(sqlite3DB.fields) == 0 {
		sqlite3DB.fields = make([]string, ModelType.NumField())
		for i := 0; i < ModelValue.NumField(); i++ {
			sqlite3DB.fields[i] = ModelType.Field(i).Name
		}
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]any, 0)
	for _, fieldName := range sqlite3DB.fields {
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

	sqlite3DB.sql = fmt.Sprintf("UPDATE `%v` SET %v WHERE %v", TableName, fieldsSQl, sqlite3DB.sql)

	updateValues = append(updateValues, sqlite3DB.args...)
	return sqlite3DB.Execute(sqlite3DB.sql, updateValues...)
}

// 删除数据，返回操作条数
func (sqlite3DB *Sqlite3DB) Delete() (sql.Result, error) {
	if sqlite3DB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	if sqlite3DB.sql == "" {
		return nil, errors.New("请先设置 Where 条件")
	}
	TableName := sqlite3DB.model.ModelSet().TableName

	sqlite3DB.sql = fmt.Sprintf("DELETE FROM `%v` WHERE %v", TableName, sqlite3DB.sql)

	return sqlite3DB.Execute(sqlite3DB.sql, sqlite3DB.args...)
}
