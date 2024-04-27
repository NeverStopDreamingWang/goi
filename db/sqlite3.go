package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
	_ "github.com/mattn/go-sqlite3"
	"math"
	"os"
	"path"
	"reflect"
	"strings"
)

type SQLite3DB struct {
	Name      string
	DB        *sql.DB
	model     model.SQLite3Model
	fields    []string
	where_sql []string
	limit_sql string
	order_sql string
	sql       string
	args      []interface{}
}

// 连接 SQLite3
func MetaSQLite3Connect(Database goi.MetaDataBase) (*SQLite3DB, error) {
	dir := path.Dir(Database.NAME)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建数据库目录错误: ", err))
		}
	}
	sqlite3DB, err := sql.Open(Database.ENGINE, Database.NAME)
	return &SQLite3DB{
		Name:      "",
		DB:        sqlite3DB,
		model:     nil,
		fields:    nil,
		where_sql: nil,
		limit_sql: "",
		order_sql: "",
		sql:       "",
		args:      nil,
	}, err
}

// 关闭连接
func (sqlite3DB *SQLite3DB) Close() error {
	err := sqlite3DB.DB.Close()
	return err
}

// 执行语句
func (sqlite3DB *SQLite3DB) Execute(query string, args ...interface{}) (sql.Result, error) {
	// 开启事务
	transaction, err := sqlite3DB.DB.Begin()
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
func (sqlite3DB *SQLite3DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := sqlite3DB.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, err
}

// 设置使用模型
func (sqlite3DB *SQLite3DB) SetModel(model model.SQLite3Model) *SQLite3DB {
	sqlite3DB.model = model
	ModelType := reflect.TypeOf(model)
	// 获取字段
	sqlite3DB.fields = make([]string, ModelType.NumField())
	for i := 0; i < ModelType.NumField(); i++ {
		sqlite3DB.fields[i] = ModelType.Field(i).Name
	}
	return sqlite3DB
}

// 插入数据库
func (sqlite3DB *SQLite3DB) Insert(ModelData model.SQLite3Model) (sql.Result, error) {
	if sqlite3DB.model == nil {
		return nil, errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}

	insertValues := make([]interface{}, len(sqlite3DB.fields))
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

	sqlite3DB.sql = fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return sqlite3DB.Execute(sqlite3DB.sql, insertValues...)
}

// 设置查询字段
func (sqlite3DB *SQLite3DB) Fields(fields ...string) *SQLite3DB {
	sqlite3DB.fields = fields
	return sqlite3DB
}

// 查询语句
func (sqlite3DB *SQLite3DB) Where(query string, args ...interface{}) *SQLite3DB {
	sqlite3DB.where_sql = append(sqlite3DB.where_sql, query)
	sqlite3DB.args = append(sqlite3DB.args, args...)
	return sqlite3DB
}

// 排序
func (sqlite3DB *SQLite3DB) OrderBy(orders ...string) *SQLite3DB {
	var orderFields []string
	for _, order := range orders {
		order = strings.Trim(order, " `'\"")
		if order == "" {
			continue
		}

		var sequence string
		var orderSQL string
		if order[0] == '-' {
			orderSQL = order[1:]
			sequence = "DESC"
		} else {
			orderSQL = order
			sequence = "ASC"
		}
		orderFields = append(orderFields, fmt.Sprintf("`%s` %s", orderSQL, sequence))
	}
	if len(orderFields) > 0 {
		sqlite3DB.order_sql = fmt.Sprintf(" ORDER BY %s", strings.Join(orderFields, ", "))
	} else {
		sqlite3DB.order_sql = ""
	}
	return sqlite3DB
}

// 排序
func (sqlite3DB *SQLite3DB) Page(page int, pagesize int) (int, int, error) {
	if page <= 0 {
		page = 1
	}
	if pagesize <= 0 {
		pagesize = 10 // 默认分页大小
	}
	offset := (page - 1) * pagesize
	sqlite3DB.limit_sql = fmt.Sprintf(" LIMIT %d OFFSET %d", pagesize, offset)
	total, err := sqlite3DB.Count()
	totalPages := int(math.Ceil(float64(total) / float64(pagesize)))
	return total, totalPages, err
}

// 执行查询语句获取数据
func (sqlite3DB *SQLite3DB) Select(queryResult interface{}) error {
	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.limit_sql = ""
		sqlite3DB.order_sql = ""
		sqlite3DB.args = nil
	}()
	if sqlite3DB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	var (
		isPtr      bool
		resultType reflect.Type
		result     = reflect.ValueOf(queryResult)
	)
	if result.Kind() != reflect.Ptr {
		return errors.New("queryResult 不是一个指向结构体的指针")
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Slice {
		return errors.New("queryResult 不是一个切片")
	}

	resultType = result.Type().Elem()
	if resultType.Kind() == reflect.Ptr {
		isPtr = true
		resultType = resultType.Elem()
	}

	queryFields := make([]string, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}
	fieldsSQl := strings.Join(queryFields, "`,`")

	sqlite3DB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	if sqlite3DB.order_sql != "" {
		sqlite3DB.sql += sqlite3DB.order_sql
	}
	if sqlite3DB.limit_sql != "" {
		sqlite3DB.sql += sqlite3DB.limit_sql
	}
	rows, err := sqlite3DB.DB.Query(sqlite3DB.sql, sqlite3DB.args...)
	defer rows.Close()
	if err != nil {
		return err
	} else if rows.Err() != nil {
		return rows.Err()
	}

	for rows.Next() {
		values := make([]interface{}, len(sqlite3DB.fields))

		item := reflect.New(resultType).Elem()

		for i, fieldName := range sqlite3DB.fields {
			values[i] = item.FieldByName(fieldName).Addr().Interface()
		}

		err = rows.Scan(values...)
		if err != nil {
			return err
		}
		if isPtr {
			result.Set(reflect.Append(result, item.Addr()))
		} else {
			result.Set(reflect.Append(result, item))
		}
	}
	return nil
}

// 返回第一条数据
func (sqlite3DB *SQLite3DB) First(queryResult interface{}) error {
	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.limit_sql = ""
		sqlite3DB.order_sql = ""
		sqlite3DB.args = nil
	}()
	if sqlite3DB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	result := reflect.ValueOf(queryResult)
	if result.Kind() != reflect.Ptr {
		return errors.New("queryResult 不是一个指向结构体的指针")
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Struct {
		return errors.New("queryResult 不是一个结构体")
	}

	queryFields := make([]string, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}
	fieldsSQl := strings.Join(queryFields, "`,`")

	sqlite3DB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	if sqlite3DB.order_sql != "" {
		sqlite3DB.sql += sqlite3DB.order_sql
	}
	if sqlite3DB.limit_sql != "" {
		sqlite3DB.sql += sqlite3DB.limit_sql
	}
	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)
	if row == nil {
		return nil
	} else if row.Err() != nil {
		return row.Err()
	}

	values := make([]interface{}, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		values[i] = result.FieldByName(fieldName).Addr().Interface()
	}

	err := row.Scan(values...)
	if err != nil {
		return err
	}
	return nil
}

// 返回查询条数数量
func (sqlite3DB *SQLite3DB) Count() (int, error) {
	if sqlite3DB.model == nil {
		return 0, errors.New("请先设置 SetModel")
	}
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	sqlite3DB.sql = fmt.Sprintf("SELECT count(*) FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)
	if row == nil {
		return 0, nil
	} else if row.Err() != nil {
		return 0, row.Err()
	}

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 更新数据，返回操作条数
func (sqlite3DB *SQLite3DB) Update(ModelData model.SQLite3Model) (sql.Result, error) {
	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.args = nil
	}()
	if sqlite3DB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]interface{}, 0)
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

	sqlite3DB.sql = fmt.Sprintf("UPDATE `%v` SET %v", TableName, fieldsSQl)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	updateValues = append(updateValues, sqlite3DB.args...)
	return sqlite3DB.Execute(sqlite3DB.sql, updateValues...)
}

// 删除数据，返回操作条数
func (sqlite3DB *SQLite3DB) Delete() (sql.Result, error) {
	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.args = nil
	}()
	if sqlite3DB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME
	sqlite3DB.sql = fmt.Sprintf("DELETE FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	return sqlite3DB.Execute(sqlite3DB.sql, sqlite3DB.args...)
}
