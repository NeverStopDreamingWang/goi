package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLDB struct {
	DB           *sql.DB
	name         string
	metaDatabase goi.MetaDataBase
	model        model.MySQLModel
	fields       []string
	where_sql    []string
	limit_sql    string
	order_sql    string
	sql          string
	args         []interface{}
}

// 连接 MySQL 数据库
func MySQLConnect(UseDataBases ...string) (*MySQLDB, error) {
	if len(UseDataBases) == 0 { // 使用默认数据库
		UseDataBases = append(UseDataBases, "default")
	}

	for _, name := range UseDataBases {
		database, ok := goi.Settings.DATABASES[name]
		if ok == true && strings.ToLower(database.ENGINE) == "mysql" {
			databaseObject, err := MetaMySQLConnect(name, database)
			if err != nil {
				continue
			}
			return databaseObject, err
		}
	}
	return nil, errors.New("没有连接成功的 MySQL 数据库！")
}

// 连接 MySQL
func MetaMySQLConnect(name string, database goi.MetaDataBase) (*MySQLDB, error) {
	connectStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", database.USER, database.PASSWORD, database.HOST, database.PORT, database.NAME)
	mysqlDB, err := sql.Open(database.ENGINE, connectStr)
	return &MySQLDB{
		DB:           mysqlDB,
		name:         name,
		metaDatabase: database,
		model:        nil,
		fields:       nil,
		where_sql:    nil,
		limit_sql:    "",
		order_sql:    "",
		sql:          "",
		args:         nil,
	}, err
}

// 关闭连接
func (mysqlDB *MySQLDB) Close() error {
	err := mysqlDB.DB.Close()
	return err
}

// 获取数据库别名
func (mysqlDB *MySQLDB) Name() string {
	return mysqlDB.name
}

// 获取数据库配置
func (mysqlDB *MySQLDB) MetaDatabase() goi.MetaDataBase {
	return mysqlDB.metaDatabase
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

// 模型迁移
func (mysqlDB *MySQLDB) Migrate(model model.MySQLModel) {
	var err error
	modelSettings := model.ModelSet()

	row := mysqlDB.DB.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name=?;", mysqlDB.metaDatabase.NAME, modelSettings.TABLE_NAME)
	if row.Err() != nil {
		panic(fmt.Sprintf("MySQL [%v] 数据库 查询错误错误: %v", mysqlDB.name, row.Err()))
	}
	var count int
	err = row.Scan(&count)
	if err != nil {
		panic(fmt.Sprintf("MySQL [%v] 数据库 查询错误错误: %v", mysqlDB.name, err))
	}

	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	tampFieldMap := make(map[string]bool)
	fieldSqlSlice := make([]string, modelType.NumField(), modelType.NumField())
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldName, ok := field.Tag.Lookup("field_name")
		if !ok {
			fieldName = strings.ToLower(field.Name)
		}
		fieldType, ok := field.Tag.Lookup("field_type")
		tampFieldMap[fieldName] = true
		fieldSqlSlice[i] = fmt.Sprintf("  `%v` %v", fieldName, fieldType)
	}
	createSql := fmt.Sprintf("CREATE TABLE `%v` (\n%v\n)", modelSettings.TABLE_NAME, strings.Join(fieldSqlSlice, ",\n"))
	if modelSettings.ENGINE != "" { // 设置存储引擎
		createSql += fmt.Sprintf(" ENGINE=%v", modelSettings.ENGINE)
	}
	if modelSettings.AUTO_INCREMENT != 0 { // 设置自增长起始值
		createSql += fmt.Sprintf(" AUTO_INCREMENT=%v", modelSettings.AUTO_INCREMENT)
	}
	if modelSettings.DEFAULT_CHARSET != "" { // 设置默认字符集
		createSql += fmt.Sprintf(" DEFAULT CHARSET=%v", modelSettings.DEFAULT_CHARSET)
	}
	if modelSettings.COLLATE != "" { // 设置校对规则
		createSql += fmt.Sprintf(" COLLATE=%v", modelSettings.COLLATE)
	}
	if modelSettings.MIN_ROWS != 0 { // 设置最小行数
		createSql += fmt.Sprintf(" MIN_ROWS=%v", modelSettings.MIN_ROWS)
	}
	if modelSettings.MAX_ROWS != 0 { // 设置最大行数
		createSql += fmt.Sprintf(" MAX_ROWS=%v", modelSettings.MAX_ROWS)
	}
	if modelSettings.CHECKSUM != 0 { // 表格的校验和算法
		createSql += fmt.Sprintf(" CHECKSUM=%v", modelSettings.CHECKSUM)
	}
	if modelSettings.DELAY_KEY_WRITE != 0 { // 控制非唯一索引的写延迟
		createSql += fmt.Sprintf(" DELAY_KEY_WRITE=%v", modelSettings.DELAY_KEY_WRITE)
	}
	if modelSettings.ROW_FORMAT != "" { // 设置行的存储格式
		createSql += fmt.Sprintf(" ROW_FORMAT=%v", modelSettings.ROW_FORMAT)
	}
	if modelSettings.DATA_DIRECTORY != "" { // 设置数据存储目录
		createSql += fmt.Sprintf(" DATA DIRECTORY='%v'", modelSettings.DATA_DIRECTORY)
	}
	if modelSettings.INDEX_DIRECTORY != "" { // 设置索引存储目录
		createSql += fmt.Sprintf(" INDEX DIRECTORY='%v'", modelSettings.INDEX_DIRECTORY)
	}
	if modelSettings.PARTITION_BY != "" { // 定义分区方式
		createSql += fmt.Sprintf(" PARTITION_BY %v", modelSettings.PARTITION_BY)
	}
	if modelSettings.COMMENT != "" { // 设置表注释
		createSql += fmt.Sprintf(" COMMENT='%v'", modelSettings.COMMENT)
	}

	if count == 0 { // 创建表
		if modelSettings.MigrationsHandler.BeforeFunc != nil { // 迁移之前处理
			goi.Log.Info("迁移之前处理...")
			err = modelSettings.MigrationsHandler.BeforeFunc()
			if err != nil {
				panic(fmt.Sprintf("迁移之前处理错误: %v", err))
			}
		}

		goi.Log.Info(fmt.Sprintf("正在迁移 MySQL: %v 数据库: %v 表: %v ...", mysqlDB.name, mysqlDB.metaDatabase.NAME, modelSettings.TABLE_NAME))
		_, err = mysqlDB.Execute(createSql)
		if err != nil {
			panic(fmt.Sprintf("迁移错误: %v", err))
		}

		if modelSettings.MigrationsHandler.AfterFunc != nil { // 迁移之后处理
			goi.Log.Info("迁移之后处理...")
			err = modelSettings.MigrationsHandler.AfterFunc()
			if err != nil {
				panic(fmt.Sprintf("迁移之后处理错误: %v", err))
			}
		}

	}
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
		isPtr    bool
		ItemType reflect.Type
		result   = reflect.ValueOf(queryResult)
	)
	if result.Kind() != reflect.Ptr {
		return errors.New("queryResult 不是一个指向结构体的指针")
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Slice {
		return errors.New("queryResult 不是一个切片")
	}

	ItemType = result.Type().Elem() // 获取切片中的元素
	if ItemType.Kind() == reflect.Ptr {
		isPtr = true
		ItemType = ItemType.Elem()
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

		item := reflect.New(ItemType).Elem()

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
