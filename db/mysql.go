package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/NeverStopDreamingWang/goi/model/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type MySQLDB struct {
	name        string           // 连接名称
	DB          *sql.DB          // 数据库连接对象
	transaction *sql.Tx          // 事务
	model       mysql.MySQLModel // 模型
	fields      []string         // 模型字段
	field_sql   []string         // 表字段
	where_sql   []string         // 条件语句
	limit_sql   string           // 分页
	group_sql   string           // 分组
	order_sql   string           // 排序
	sql         string           // 执行 sql
	args        []interface{}    // 条件参数
}

// 连接 MySQL 数据库
func MySQLConnect(UseDataBases string) *MySQLDB {
	database, ok := goi.Settings.DATABASES[UseDataBases]
	if !ok {
		databasesNotErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.databases_not_error",
			TemplateData: map[string]interface{}{
				"name": UseDataBases,
			},
		})
		goi.Log.Error(databasesNotErrorMsg)
		panic(databasesNotErrorMsg)
	}
	return &MySQLDB{
		name:      UseDataBases,
		DB:        database.DB(),
		model:     nil,
		fields:    nil,
		field_sql: nil,
		where_sql: nil,
		limit_sql: "",
		group_sql: "",
		order_sql: "",
		sql:       "",
		args:      nil,
	}
}

// 获取数据库别名
func (mysqlDB MySQLDB) Name() string {
	return mysqlDB.name
}

// 获取SQL语句
func (mysqlDB MySQLDB) GetSQL() string {
	return mysqlDB.sql
}

// 根据当前对象副本创建一个事务
func (mysqlDB MySQLDB) WithTransaction(transactionFunc func(mysqlDB *MySQLDB, args ...interface{}) error, args ...interface{}) error {
	var err error
	if mysqlDB.transaction != nil {
		transactionCannotBeNestedErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.transaction_cannot_be_nested_error",
		})
		return errors.New(transactionCannotBeNestedErrorMsg)
	}

	mysqlDB.transaction, err = mysqlDB.DB.Begin()
	if err != nil {
		return err
	}
	err = transactionFunc(&mysqlDB, args...)
	if err != nil {
		_ = mysqlDB.transaction.Rollback()
		return err
	}

	// 提交事务
	err = mysqlDB.transaction.Commit()
	if err != nil {
		_ = mysqlDB.transaction.Rollback()
		return err
	}
	return nil
}

// 执行语句
func (mysqlDB *MySQLDB) Execute(query string, args ...interface{}) (sql.Result, error) {
	if mysqlDB.transaction != nil {
		return mysqlDB.transaction.Exec(query, args...)
	} else {
		return mysqlDB.DB.Exec(query, args...)
	}
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
func (mysqlDB *MySQLDB) Migrate(db_name string, model mysql.MySQLModel) {
	var err error
	modelSettings := model.ModelSet()

	row := mysqlDB.DB.QueryRow("SELECT 1 FROM information_schema.tables WHERE table_schema=? AND table_name=?;", db_name, modelSettings.TABLE_NAME)

	var exists int
	err = row.Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) == false && err != nil {
		selectErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.select_error",
			TemplateData: map[string]interface{}{
				"engine":  "MySQL",
				"db_name": mysqlDB.name,
				"err":     err,
			},
		})
		goi.Log.Error(selectErrorMsg)
		panic(selectErrorMsg)
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

	if exists != 1 { // 创建表
		if modelSettings.MigrationsHandler.BeforeHandler != nil { // 迁移之前处理
			beforeMigrationMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.before_migration",
			})
			goi.Log.Info(beforeMigrationMsg)
			err = modelSettings.MigrationsHandler.BeforeHandler()
			if err != nil {
				beforeMigrationErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "database.before_migration_error",
					TemplateData: map[string]interface{}{
						"err": err,
					},
				})
				goi.Log.Error(beforeMigrationErrorMsg)
				panic(beforeMigrationErrorMsg)
			}
		}
		migrationModelMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.migration",
			TemplateData: map[string]interface{}{
				"engine":  "MySQL",
				"name":    mysqlDB.name,
				"db_name": db_name,
				"tb_name": modelSettings.TABLE_NAME,
			},
		})
		goi.Log.Info(migrationModelMsg)
		_, err = mysqlDB.Execute(createSql)
		if err != nil {
			migrationErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.migration_error",
				TemplateData: map[string]interface{}{
					"err": err,
				},
			})
			goi.Log.Error(migrationErrorMsg)
			panic(migrationErrorMsg)
		}

		if modelSettings.MigrationsHandler.AfterHandler != nil { // 迁移之后处理
			afterMigrationMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.after_migration",
			})
			goi.Log.Info(afterMigrationMsg)
			err = modelSettings.MigrationsHandler.AfterHandler()
			if err != nil {
				afterMigrationErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "database.after_migration_error",
					TemplateData: map[string]interface{}{
						"err": err,
					},
				})
				goi.Log.Error(afterMigrationErrorMsg)
				panic(afterMigrationErrorMsg)
			}
		}

	}
}

func (mysqlDB *MySQLDB) isSetModel() {
	if mysqlDB.model == nil {
		notSetModelErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.not_SetModel_error",
		})
		goi.Log.Error(notSetModelErrorMsg)
		panic(notSetModelErrorMsg)
	}
}

// 设置使用模型  return 当前 *MySQLDB 本身
func (mysqlDB *MySQLDB) SetModel(model mysql.MySQLModel) *MySQLDB {
	mysqlDB.model = model
	ModelType := reflect.TypeOf(mysqlDB.model)
	// 获取字段
	mysqlDB.fields = make([]string, ModelType.NumField())
	mysqlDB.field_sql = make([]string, ModelType.NumField())
	for i := 0; i < ModelType.NumField(); i++ {
		field := ModelType.Field(i)
		mysqlDB.fields[i] = field.Name

		field_name, ok := field.Tag.Lookup("field_name")
		if !ok {
			field_name = strings.ToLower(field.Name)
		}
		mysqlDB.field_sql[i] = field_name
	}
	mysqlDB.where_sql = nil
	mysqlDB.limit_sql = ""
	mysqlDB.group_sql = ""
	mysqlDB.order_sql = ""
	mysqlDB.sql = ""
	mysqlDB.args = nil
	return mysqlDB
}

// 设置查询结果字段  return 当前 *MySQLDB 的副本指针
func (mysqlDB MySQLDB) Fields(fields ...string) *MySQLDB {
	ModelType := reflect.TypeOf(mysqlDB.model)
	// 获取字段
	mysqlDB.fields = make([]string, len(fields))
	mysqlDB.field_sql = make([]string, len(fields))
	for i, fieldName := range fields {
		field, ok := ModelType.FieldByName(fieldName)
		if !ok {
			fieldIsNotErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.field_is_not_error",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			goi.Log.Error(fieldIsNotErrorMsg)
			panic(fieldIsNotErrorMsg)
		}
		mysqlDB.fields[i] = field.Name

		field_name, ok := field.Tag.Lookup("field_name")
		if !ok {
			field_name = strings.ToLower(field.Name)
		}
		mysqlDB.field_sql[i] = field_name
	}
	return &mysqlDB
}

// 插入到数据库
func (mysqlDB *MySQLDB) Insert(ModelData mysql.MySQLModel) (sql.Result, error) {
	mysqlDB.isSetModel()

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

	fieldsSQL := strings.Join(mysqlDB.field_sql, "`,`")

	tempValues := make([]string, len(mysqlDB.fields))
	for i := 0; i < len(mysqlDB.fields); i++ {
		tempValues[i] = "?"
	}
	valuesSQL := strings.Join(tempValues, ",")

	mysqlDB.sql = fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return mysqlDB.Execute(mysqlDB.sql, insertValues...)
}

// 设置条件查询语句  return 当前 *MySQLDB 的副本指针
func (mysqlDB MySQLDB) Where(query string, args ...interface{}) *MySQLDB {
	queryParts := strings.Split(query, "?")
	if len(queryParts)-1 != len(args) {
		whereArgsPlaceholderErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.where_args_placeholder_error",
		})
		goi.Log.Error(whereArgsPlaceholderErrorMsg)
		panic(whereArgsPlaceholderErrorMsg)
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(queryParts[0])
	for i, param := range args {
		// 检查是否是切片
		paramValue := reflect.ValueOf(param)
		if paramValue.Kind() == reflect.Ptr {
			paramValue = paramValue.Elem()
		}

		if paramValue.Kind() == reflect.Slice {
			placeholders := "(" + strings.Repeat("?,", paramValue.Len()-1) + "?" + ")"
			queryBuilder.WriteString(placeholders)

			// 将切片元素加入 args
			for j := 0; j < paramValue.Len(); j++ {
				mysqlDB.args = append(mysqlDB.args, paramValue.Index(j).Interface())
			}
		} else {
			// 普通参数，直接添加占位符
			queryBuilder.WriteString("?")
			mysqlDB.args = append(mysqlDB.args, param)
		}
		// 添加切割后的下一部分
		queryBuilder.WriteString(queryParts[i+1])
	}
	mysqlDB.where_sql = append(mysqlDB.where_sql, queryBuilder.String())
	return &mysqlDB
}

// 分组  return 当前 *MySQLDB 的副本指针
func (mysqlDB MySQLDB) GroupBy(groups ...string) *MySQLDB {
	var groupFields []string
	for _, group := range groups {
		group = strings.Trim(group, " `'\"")
		if group == "" {
			continue
		}
		groupFields = append(groupFields, fmt.Sprintf("`%s`", group))
	}
	if len(groups) > 0 {
		mysqlDB.group_sql = fmt.Sprintf(" GROUP BY %s", strings.Join(groups, ", "))
	} else {
		mysqlDB.group_sql = ""
	}
	return &mysqlDB
}

// 排序  return 当前 *MySQLDB 的副本指针
func (mysqlDB MySQLDB) OrderBy(orders ...string) *MySQLDB {
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
	return &mysqlDB
}

// 分页
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
	mysqlDB.isSetModel()

	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	var (
		isPtr    bool
		ItemType reflect.Type
		result   = reflect.ValueOf(queryResult)
	)
	if result.Kind() != reflect.Ptr {
		isNotPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotPtrErrorMsg)
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Slice {
		isNotSlicePtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_slice_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotSlicePtrErrorMsg)
	}

	ItemType = result.Type().Elem() // 获取切片中的元素
	if ItemType.Kind() == reflect.Ptr {
		isPtr = true
		ItemType = ItemType.Elem()
	}

	fieldsSQl := strings.Join(mysqlDB.field_sql, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}
	if mysqlDB.group_sql != "" {
		mysqlDB.sql += mysqlDB.group_sql
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
			fieldValue := item.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				values[i] = new(interface{})
			} else {
				values[i] = fieldValue.Addr().Interface()
			}
		}

		err = rows.Scan(values...)
		if err != nil {
			return err
		}
		if isPtr == true {
			result.Set(reflect.Append(result, item.Addr()))
		} else {
			result.Set(reflect.Append(result, item))
		}
	}
	return nil
}

// 返回第一条数据
func (mysqlDB *MySQLDB) First(queryResult interface{}) error {
	mysqlDB.isSetModel()

	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	result := reflect.ValueOf(queryResult)
	if result.Kind() != reflect.Ptr {
		isNotPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotPtrErrorMsg)
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Struct {
		isNotStructPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_struct_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotStructPtrErrorMsg)
	}

	fieldsSQl := strings.Join(mysqlDB.field_sql, "`,`")

	mysqlDB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}
	if mysqlDB.group_sql != "" {
		mysqlDB.sql += mysqlDB.group_sql
	}
	if mysqlDB.order_sql != "" {
		mysqlDB.sql += mysqlDB.order_sql
	}
	if mysqlDB.limit_sql != "" {
		mysqlDB.sql += mysqlDB.limit_sql
	}
	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
	err := row.Err()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(mysqlDB.fields))
	for i, fieldName := range mysqlDB.fields {
		fieldValue := result.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			values[i] = new(interface{})
		} else {
			values[i] = fieldValue.Addr().Interface()
		}
	}

	err = row.Scan(values...)
	if err != nil {
		return err
	}
	return nil
}

// 返回查询条数数量
func (mysqlDB *MySQLDB) Count() (int, error) {
	mysqlDB.isSetModel()

	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	mysqlDB.sql = fmt.Sprintf("SELECT count(*) FROM `%v`", TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}

	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
	err := row.Err()
	if err != nil {
		return 0, err
	}

	var count int
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 是否存在
func (mysqlDB *MySQLDB) Exists() (bool, error) {
	mysqlDB.isSetModel()

	TableName := mysqlDB.model.ModelSet().TABLE_NAME

	mysqlDB.sql = fmt.Sprintf("SELECT 1 FROM `%v`", TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}

	row := mysqlDB.DB.QueryRow(mysqlDB.sql, mysqlDB.args...)
	err := row.Err()
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	var exists int
	err = row.Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// 更新数据，返回操作条数
func (mysqlDB *MySQLDB) Update(ModelData mysql.MySQLModel) (sql.Result, error) {
	mysqlDB.isSetModel()

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
	mysqlDB.isSetModel()

	TableName := mysqlDB.model.ModelSet().TABLE_NAME
	mysqlDB.sql = fmt.Sprintf("DELETE FROM `%v`", TableName)
	if len(mysqlDB.where_sql) > 0 {
		mysqlDB.sql += fmt.Sprintf(" WHERE %v", strings.Join(mysqlDB.where_sql, " AND "))
	}

	return mysqlDB.Execute(mysqlDB.sql, mysqlDB.args...)
}
