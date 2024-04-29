package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
	_ "github.com/mattn/go-sqlite3"
)

type SQLite3DB struct {
	DB           *sql.DB
	name         string
	metaDatabase goi.MetaDataBase
	model        model.SQLite3Model
	fields       []string
	where_sql    []string
	limit_sql    string
	order_sql    string
	sql          string
	args         []interface{}
}

// 连接 SQLite3 数据库
func SQLite3Connect(UseDataBases ...string) (*SQLite3DB, error) {
	if len(UseDataBases) == 0 { // 使用默认数据库
		UseDataBases = append(UseDataBases, "default")
	}

	for _, name := range UseDataBases {
		database, ok := goi.Settings.DATABASES[name]
		if ok == true && strings.ToLower(database.ENGINE) == "sqlite3" {
			databaseObject, err := MetaSQLite3Connect(name, database)
			if err != nil {
				continue
			}
			return databaseObject, err
		}
	}
	return nil, errors.New("没有连接成功的 SQLite3 数据库！")
}

// 连接 SQLite3
func MetaSQLite3Connect(name string, database goi.MetaDataBase) (*SQLite3DB, error) {
	dir := path.Dir(database.NAME)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(fmt.Sprintf("创建数据库目录错误: ", err))
		}
	}
	sqlite3DB, err := sql.Open(database.ENGINE, database.NAME)
	return &SQLite3DB{
		DB:           sqlite3DB,
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
func (sqlite3DB *SQLite3DB) Close() error {
	err := sqlite3DB.DB.Close()
	return err
}

// 获取数据库别名
func (sqlite3DB *SQLite3DB) Name() string {
	return sqlite3DB.name
}

// 获取数据库配置
func (sqlite3DB *SQLite3DB) MetaDatabase() goi.MetaDataBase {
	return sqlite3DB.metaDatabase
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

// 模型迁移
func (sqlite3DB *SQLite3DB) Migrate(model model.SQLite3Model) {
	var err error
	modelSettings := model.ModelSet()

	row := sqlite3DB.DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name =?;", modelSettings.TABLE_NAME)
	if row.Err() != nil {
		panic(fmt.Sprintf("SQLite3 [%v] 数据库 查询错误错误: %v", sqlite3DB.name, row.Err()))
	}
	var count int
	err = row.Scan(&count)
	if err != nil {
		panic(fmt.Sprintf("SQLite3 [%v] 数据库 查询错误错误: %v", sqlite3DB.name, err))
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

	if count == 0 { // 创建表
		if modelSettings.MigrationsHandler.BeforeFunc != nil { // 迁移之前处理
			goi.Log.Info("迁移之前处理...")
			err = modelSettings.MigrationsHandler.BeforeFunc()
			if err != nil {
				panic(fmt.Sprintf("迁移之前处理错误: %v", err))
			}
		}

		goi.Log.Info(fmt.Sprintf("正在迁移 SQLite3: %v 数据库: %v 表: %v ...", sqlite3DB.name, sqlite3DB.metaDatabase.NAME, modelSettings.TABLE_NAME))
		_, err = sqlite3DB.Execute(createSql)
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

	} else { // 自动更新表
		var rows *sql.Rows
		rows, err = sqlite3DB.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name=?;", modelSettings.TABLE_NAME)
		if err != nil {
			panic(fmt.Sprintf("SQLite3 [%v] 数据库 查询错误: %v", sqlite3DB.name, err))
		}
		var old_createSql string
		for rows.Next() {
			err = rows.Scan(&old_createSql)
			if err != nil {
				panic(fmt.Sprintf("SQLite3 [%v] 数据库 查询错误: %v", sqlite3DB.name, err))
			}
		}
		if err = rows.Close(); err != nil {
			panic(fmt.Sprintf("关闭 Rows 错误: %v", err))
		}
		if createSql != old_createSql { // 创建表语句不一样
			goi.Log.Info(fmt.Sprintf("正在更新 SQLite3: %v 数据库: %v 表: %v ", sqlite3DB.name, sqlite3DB.metaDatabase.NAME, modelSettings.TABLE_NAME))

			oldTableName := fmt.Sprintf("%v_%v", modelSettings.TABLE_NAME, goi.Version())
			goi.Log.Info(fmt.Sprintf("- 重命名旧表：%v...", oldTableName))

			_, err = sqlite3DB.Execute(fmt.Sprintf("ALTER TABLE `%v` RENAME TO `%v`;", modelSettings.TABLE_NAME, oldTableName))
			if err != nil {
				panic(fmt.Sprintf("重命名旧表错误: %v", err))
			}

			goi.Log.Info(fmt.Sprintf("- 创建新表：%v...", modelSettings.TABLE_NAME))
			_, err = sqlite3DB.Execute(createSql)
			if err != nil {
				panic(fmt.Sprintf("创建新表错误: %v", err))
			}

			TabelFieldSlice := make([]string, 0)
			rows, err = sqlite3DB.Query(fmt.Sprintf("PRAGMA table_info(`%v`);", oldTableName))
			if err != nil {
				panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", sqlite3DB.name, err))
			}
			for rows.Next() {
				var (
					cid        int
					name       string
					dataType   string
					notNull    int
					dflt_value *string
					pk         int
				)
				err = rows.Scan(&cid, &name, &dataType, &notNull, &dflt_value, &pk)
				if err != nil {
					panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", sqlite3DB.name, err))
				}
				if _, exists := tampFieldMap[name]; exists {
					TabelFieldSlice = append(TabelFieldSlice, name)
				}
			}
			if err = rows.Close(); err != nil {
				panic(fmt.Sprintf("关闭 Rows 错误: %v", err))
			}

			goi.Log.Info("- 迁移数据...")
			migrateFieldSql := strings.Join(TabelFieldSlice, ",")
			migrateSql := fmt.Sprintf("INSERT INTO `%v` (%v) SELECT %v FROM `%v`;", modelSettings.TABLE_NAME, migrateFieldSql, migrateFieldSql, oldTableName)
			_, err = sqlite3DB.Execute(migrateSql)
			if err != nil {
				panic(fmt.Sprintf("迁移表数据错误: %v", err))
			}
		}
	}
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

		item := reflect.New(ItemType).Elem()

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
