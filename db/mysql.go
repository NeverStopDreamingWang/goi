package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
	_ "github.com/go-sql-driver/mysql"
	"math"
	"reflect"
	"strings"
)

type MySQLDB struct {
	Name      string
	DB        *sql.DB
	model     model.MySQLModel
	fields    []string
	where_sql []string
	limit_sql string
	order_sql string
	sql       string
	args []interface{}
}

// 连接 MySQL
func MetaMySQLConnect(Database goi.MetaDataBase) (*MySQLDB, error) {
	connectStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", Database.USER, Database.PASSWORD, Database.HOST, Database.PORT, Database.NAME)
	mysqlDB, err := sql.Open(Database.ENGINE, connectStr)
	return &MySQLDB{
		Name:      "",
		DB:        mysqlDB,
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
func (mysqlDB *MySQLDB) Close() error {
	err := mysqlDB.DB.Close()
	return err
}

// 执行语句
func (mysqlDB *MySQLDB) Execute(query string, args ...interface{}) (sql.Result, error) {
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
func (mysqlDB *MySQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
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

	insertValues := make([]interface{}, len(mysqlDB.fields))
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

	mysqlDB.sql = fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)

	return mysqlDB.Execute(mysqlDB.sql, insertValues...)
}

// 设置查询字段
func (mysqlDB *MySQLDB) Fields(fields ...string) *MySQLDB {
	mysqlDB.fields = fields
	return mysqlDB
}

// 查询语句
func (mysqlDB *MySQLDB) Where(query string, args ...interface{}) *MySQLDB {
	mysqlDB.where_sql = append(mysqlDB.where_sql, query)
	mysqlDB.args = append(mysqlDB.args, args...)
	return mysqlDB
}

// 排序
func (mysqlDB *MySQLDB) OrderBy(orders ...string) *MySQLDB {
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
		mysqlDB.order_sql = fmt.Sprintf(" ORDER BY %s", strings.Join(orderFields, ", "))
	} else {
		mysqlDB.order_sql = ""
	}
	return mysqlDB
}

// 排序
func (mysqlDB *MySQLDB) Page(page int, pagesize int) (int, int, error) {
	if page <= 0 {
		page = 1
	}
	if pagesize <= 0 {
		pagesize = 10 // 默认分页大小
	}
	offset := (page - 1) * pagesize
	mysqlDB.limit_sql = fmt.Sprintf(" LIMIT %d OFFSET %d", pagesize, offset)
	total, err := mysqlDB.Count()
	totalPages := int(math.Ceil(float64(total) / float64(pagesize)))
	return total, totalPages, err
}

// 执行查询语句获取数据
func (mysqlDB *MySQLDB) Select(queryResult interface{}) error {
	defer func() {
		mysqlDB.where_sql = nil
		mysqlDB.limit_sql = ""
		mysqlDB.order_sql = ""
		mysqlDB.args = nil
	}()
	if mysqlDB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

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

	queryFields := make([]string, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}
	fieldsSQl := strings.Join(queryFields, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}
	if mysqlDB.order_sql != "" {
		mysqlDB.sql += mysqlDB.order_sql
	}
	if mysqlDB.limit_sql != "" {
		mysqlDB.sql += mysqlDB.limit_sql
	}

	rows, err := mysqlDB.DB.Query(mysqlDB.sql, mysqlDB.args...)
	defer rows.Close()
	if err != nil {
		return err
	} else if rows.Err() != nil {
		return rows.Err()
	}

	for rows.Next() {
		values := make([]interface{}, len(mysqlDB.fields))

		item := reflect.New(resultType).Elem()

		for i, fieldName := range mysqlDB.fields {
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
func (mysqlDB *MySQLDB) First(queryResult interface{}) error {
	defer func() {
		mysqlDB.where_sql = nil
		mysqlDB.limit_sql = ""
		mysqlDB.order_sql = ""
		mysqlDB.args = nil
	}()
	if mysqlDB.model == nil {
		return errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	result := reflect.ValueOf(queryResult)
	if result.Kind() != reflect.Ptr {
		return errors.New("queryResult 不是一个指向结构体的指针")
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Struct {
		return errors.New("queryResult 不是一个结构体")
	}

	queryFields := make([]string, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		queryFields[i] = strings.ToLower(fieldName)
	}
	fieldsSQl := strings.Join(queryFields, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}
	if mysqlDB.order_sql != "" {
		mysqlDB.sql += mysqlDB.order_sql
	}
	if mysqlDB.limit_sql != "" {
		mysqlDB.sql += mysqlDB.limit_sql
	}
	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
	if row == nil {
		return nil
	} else if row.Err() != nil {
		return row.Err()
	}

	values := make([]interface{}, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		values[i] = result.FieldByName(fieldName).Addr().Interface()
	}

	err := row.Scan(values...)
	if err != nil {
		return err
	}
	return nil
}

// 返回查询条数数量
func (mysqlDB *MySQLDB) Count() (int, error) {
	if mysqlDB.model == nil {
		return 0, errors.New("请先设置 SetModel")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	mysqlDB.sql = fmt.Sprintf("SELECT count(*) FROM `%v`", TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}

	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
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
func (mysqlDB *MySQLDB) Update(ModelData model.MySQLModel) (sql.Result, error) {
	defer func() {
		mysqlDB.where_sql = nil
		mysqlDB.args = nil
	}()
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]interface{}, 0)
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

	mysqlDB.sql = fmt.Sprintf("UPDATE `%v` SET %v", TableName, fieldsSQl)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}

	updateValues = append(updateValues, mysqlDB.args...)
	return mysqlDB.Execute(mysqlDB.sql, updateValues...)
}

// 删除数据，返回操作条数
func (mysqlDB *MySQLDB) Delete() (sql.Result, error) {
	defer func() {
		mysqlDB.where_sql = nil
		mysqlDB.args = nil
	}()
	if mysqlDB.model == nil {
		return nil, errors.New("请先设置 SetModel 表")
	}
	TableName := mysqlDB.model.ModelSet().TABLE_NAME
	mysqlDB.sql = fmt.Sprintf("DELETE FROM `%v`", TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}

	return mysqlDB.Execute(mysqlDB.sql, mysqlDB.args...)
}
