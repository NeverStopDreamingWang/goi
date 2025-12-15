package sqlite3

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/internal/i18n"
	"github.com/NeverStopDreamingWang/goi/utils"
)

const driverName = "sqlite3"

func init() {
	db.Register(driverName, factory)
}

// factory 是 db.Engine 的工厂函数
func factory(UseDataBases string, database goi.DataBase) db.Engine {
	return ConnectDatabase(UseDataBases, database)
}

// ConnectDatabase 连接 SQLite3 数据库
//
// 参数:
//   - UseDataBases: string 数据库配置名称
//   - database: goi.DataBase 数据库连接管理器
//
// 返回:
//   - *Engine: SQLite3 数据库操作实例
//
// 说明:
//   - 如果配置不存在或连接失败则panic
func ConnectDatabase(UseDataBases string, database goi.DataBase) *Engine {
	if database.ENGINE != driverName {
		engineNotMatchErrorMsg := i18n.T("db.engine_type_is_not_match", map[string]any{
			"want_type": driverName,
			"got_type":  database.ENGINE,
		})
		panic(engineNotMatchErrorMsg)
	}
	return &Engine{
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

// Connect 连接 SQLite3 数据库
//
// 参数:
//   - UseDataBases: string 数据库配置名称
//
// 返回:
//   - *Engine: SQLite3 数据库操作实例
//
// 说明:
//   - 从全局配置中获取数据库连接信息
//   - 如果配置不存在或连接失败则panic
func Connect(UseDataBases string) *Engine {
	database, ok := goi.Settings.DATABASES[UseDataBases]
	if !ok {
		databasesNotErrorMsg := i18n.T("db.databases_not_error", map[string]any{
			"name": UseDataBases,
		})
		panic(databasesNotErrorMsg)
	}
	return ConnectDatabase(UseDataBases, *database)
}

// Engine 结构体用于管理 SQLite3 数据库连接和操作
type Engine struct {
	name        string   // 数据库连接名称
	DB          *sql.DB  // 数据库连接对象
	transaction *sql.Tx  // 当前事务对象
	model       Model    // 当前操作的数据模型
	fields      []string // 模型结构体字段名
	field_sql   []string // 数据库表字段名
	where_sql   []string // WHERE 条件语句
	limit_sql   string   // LIMIT 分页语句
	group_sql   string   // GROUP BY 分组语句
	order_sql   string   // ORDER BY 排序语句
	sql         string   // 最终执行的 SQL 语句
	args        []any    // SQL 语句的参数值
}

// Name 获取数据库连接名称
//
// 返回:
//   - string: 当前数据库连接名称
func (engine Engine) Name() string {
	return engine.name
}

// GetSQL 获取最近一次执行的SQL语句
//
// 返回:
//   - string: 最近一次执行的SQL语句
func (engine Engine) GetSQL() string {
	return engine.sql
}

// 事务执行函数
type TransactionFunc func(engine *Engine, args ...any) error

// WithTransaction 在事务中执行指定的函数
//
// 参数:
//   - transactionFunc: TransactionFunc func(engine *Engine, args ...any) error 事务执行函数
//   - args: ...any 传递给事务函数的参数列表
//
// 返回:
//   - error: 事务执行的错误，发生错误时会自动回滚
//
// 说明:
//   - 不支持嵌套事务，如果当前已在事务中则返回错误
//   - 事务函数执行失败时自动回滚
func (engine Engine) WithTransaction(transactionFunc TransactionFunc, args ...any) error {
	if engine.transaction != nil {
		transactionCannotBeNestedErrorMsg := i18n.T("db.transaction_cannot_be_nested_error")
		return errors.New(transactionCannotBeNestedErrorMsg)
	}

	// 开启事务
	var err error
	engine.transaction, err = engine.DB.Begin()
	if err != nil {
		return err
	}

	// 执行事务中的操作
	defer func() {
		if r := recover(); r != nil {
			_ = engine.transaction.Rollback()
			panic(r)
		}
	}()

	// 执行传入的函数
	err = transactionFunc(&engine, args...)
	if err != nil {
		_ = engine.transaction.Rollback()
		return err
	}

	// 提交事务
	return engine.transaction.Commit()
}

// Execute 执行SQL语句
//
// 参数:
//   - query: string SQL语句
//   - args: ...any SQL参数值列表
//
// 返回:
//   - sql.Result: 执行操作的结果
//   - error: 执行过程中的错误
//
// 说明:
//   - 支持在事务中使用
func (engine *Engine) Execute(query string, args ...any) (sql.Result, error) {
	engine.sql = query
	if engine.transaction != nil {
		return engine.transaction.Exec(query, args...)
	}
	return engine.DB.Exec(query, args...)
}

// QueryRow 执行查询SQL语句
//
// 参数:
//   - query: string SQL查询语句
//   - args: ...any SQL参数值列表
//
// 返回:
//   - *sql.Row: 查询结果行
//
// 说明:
//   - 查询失败时返回nil
//   - 支持在事务中使用
func (engine *Engine) QueryRow(query string, args ...any) *sql.Row {
	engine.sql = query
	if engine.transaction != nil {
		return engine.transaction.QueryRow(query, args...)
	}
	return engine.DB.QueryRow(query, args...)
}

// Query 执行查询SQL语句
//
// 参数:
//   - query: string SQL查询语句
//   - args: ...any SQL参数值列表
//
// 返回:
//   - *sql.Rows: 查询结果集
//   - error: 查询过程中的错误
//
// 说明:
//   - 返回的结果集需要调用方手动关闭
//   - 查询失败时返回nil和错误信息
//   - 支持在事务中使用
func (engine *Engine) Query(query string, args ...any) (*sql.Rows, error) {
	engine.sql = query
	if engine.transaction != nil {
		return engine.transaction.Query(query, args...)
	}
	return engine.DB.Query(query, args...)
}

// Migrate 根据模型创建数据库表
//
// 参数:
//   - model: Model 数据模型
//
// 说明:
//   - 检查表是否已存在，存在则跳过创建
//   - 解析模型结构体的字段标签
//   - 支持自定义迁移前后处理函数
//   - 创建失败时会panic
func (engine *Engine) Migrate(model Model) {
	var err error
	settings := model.ModelSet()

	row := engine.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name =?;", settings.TABLE_NAME)

	var exists int
	err = row.Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		selectErrorMsg := i18n.T("db.select_error", map[string]any{
			"engine":  driverName,
			"name":    engine.name,
			"db_name": "",
			"err":     err,
		})
		goi.Log.Error(selectErrorMsg)
		panic(selectErrorMsg)
	}

	if exists == 1 {
		return
	}

	// 构造 CREATE TABLE 语句
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	var columns []string
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldType, ok := field.Tag.Lookup("field_type")
		if !ok {
			continue
		}
		fieldName, ok := field.Tag.Lookup("field_name")
		if !ok {
			fieldName = strings.ToLower(field.Name)
		}

		columns = append(columns, fmt.Sprintf("  `%v` %v", fieldName, fieldType))
	}

	columnsSQL := strings.Join(columns, ",\n")
	createSQL := fmt.Sprintf("CREATE TABLE `%v` (\n%v\n)", settings.TABLE_NAME, columnsSQL)
	if settings.MigrationsHandler.BeforeHandler != nil { // 迁移之前处理
		beforeMigrationMsg := i18n.T("db.before_migration")
		goi.Log.Info(beforeMigrationMsg)
		err = settings.MigrationsHandler.BeforeHandler()
		if err != nil {
			beforeMigrationErrorMsg := i18n.T("db.before_migration_error", map[string]any{"err": err})
			goi.Log.Error(beforeMigrationErrorMsg)
			panic(beforeMigrationErrorMsg)
		}
	}

	migrationModelMsg := i18n.T("db.migration.sqlite3", map[string]any{
		"engine":  driverName,
		"name":    engine.name,
		"tb_name": settings.TABLE_NAME,
	})
	goi.Log.Info(migrationModelMsg)
	_, err = engine.Execute(createSQL)
	if err != nil {
		migrationErrorMsg := i18n.T("db.migration_error", map[string]any{"err": err})
		goi.Log.Error(migrationErrorMsg)
		panic(migrationErrorMsg)
	}

	if settings.MigrationsHandler.AfterHandler != nil { // 迁移之后处理
		afterMigrationMsg := i18n.T("db.after_migration")
		goi.Log.Info(afterMigrationMsg)
		err = settings.MigrationsHandler.AfterHandler()
		if err != nil {
			afterMigrationErrorMsg := i18n.T("db.after_migration_error", map[string]any{"err": err})
			goi.Log.Error(afterMigrationErrorMsg)
			panic(afterMigrationErrorMsg)
		}
	}
}

// isSetModel 检查是否已设置数据模型
//
// 说明:
//   - 内部方法，用于验证模型是否已设置
//   - 如果未设置模型则panic
func (engine *Engine) isSetModel() {
	if engine.model == nil {
		notSetModelErrorMsg := i18n.T("db.not_SetModel_error")
		panic(notSetModelErrorMsg)
	}
}

// SetModel 设置当前操作的数据模型
//
// 参数:
//   - model: Model 数据模型实例
//
// 返回:
//   - *Engine: 当前实例指针，支持链式调用
//
// 说明:
//   - 设置后会重置所有查询条件
//   - 解析模型的字段信息用于后续操作
func (engine *Engine) SetModel(model Model) *Engine {
	engine.model = model
	// 获取字段
	engine.fields = nil
	engine.field_sql = nil
	engine.where_sql = nil
	engine.limit_sql = ""
	engine.group_sql = ""
	engine.order_sql = ""
	engine.sql = ""
	engine.args = nil

	modelType := reflect.TypeOf(engine.model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		_, ok := field.Tag.Lookup("field_type")
		if !ok {
			continue
		}
		engine.fields = append(engine.fields, field.Name)
		fieldName, ok := field.Tag.Lookup("field_name")
		if !ok {
			fieldName = strings.ToLower(field.Name)
		}
		engine.field_sql = append(engine.field_sql, fieldName)
	}
	return engine
}

// Fields 指定查询要返回的字段
//
// 参数:
//   - fields: ...string 要查询的字段名列表
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名必须是模型结构体中已定义的字段
//   - 只返回带有field_type标签的字段
//   - 字段名大小写不敏感，会自动转换为数据库字段名
//   - 如果字段不存在则会panic
func (engine Engine) Fields(fields ...string) *Engine {
	modelType := reflect.TypeOf(engine.model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	// 获取字段
	engine.fields = nil
	engine.field_sql = nil

	for _, fieldName := range fields {
		field, ok := modelType.FieldByName(fieldName)
		if !ok {
			fieldIsNotErrorMsg := i18n.T("db.field_is_not_error", map[string]any{"name": fieldName})
			goi.Log.Error(fieldIsNotErrorMsg)
			panic(fieldIsNotErrorMsg)
		}

		// 检查字段是否有field_type标签
		if _, ok = field.Tag.Lookup("field_type"); !ok {
			continue
		}
		engine.fields = append(engine.fields, field.Name)

		// 获取数据库字段名，如果没有field_name标签则使用字段名小写
		columnName, ok := field.Tag.Lookup("field_name")
		if !ok {
			columnName = strings.ToLower(field.Name)
		}
		engine.field_sql = append(engine.field_sql, columnName)
	}
	return &engine
}

// Insert 向数据库插入一条记录
//
// 参数:
//   - model: Model 要插入的数据模型实例
//
// 返回:
//   - sql.Result: 插入操作的结果
//   - error: 插入过程中的错误
//
// 说明:
//   - 调用前必须先通过SetModel设置数据模型
//   - 支持指针和非指针类型的字段值
//   - 只插入带有field_type标签的字段
func (engine *Engine) Insert(model Model) (sql.Result, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}

	insertValues := make([]any, len(engine.fields))
	for i, fieldName := range engine.fields {
		field := modelValue.FieldByName(fieldName)
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				insertValues[i] = nil
			} else {
				insertValues[i] = field.Elem().Interface()
			}
		} else {
			insertValues[i] = field.Interface()
		}
	}

	fieldsSQL := strings.Join(engine.field_sql, "`,`")
	valuesSQL := strings.TrimRight(strings.Repeat("?,", len(engine.fields)), ",")

	engine.sql = fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return engine.Execute(engine.sql, insertValues...)
}

// Where 设置查询条件
//
// 参数:
//   - query: string WHERE条件语句
//   - args: ...any 条件参数值列表
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 支持多次调用，条件之间使用AND连接
//   - 支持参数占位符?自动替换
//   - 支持 in 切片类型的参数
func (engine Engine) Where(query string, args ...any) *Engine {
	queryParts := strings.Split(query, "?")
	if len(queryParts)-1 != len(args) {
		whereArgsPlaceholderErrorMsg := i18n.T("db.where_args_placeholder_error")
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

		if (paramValue.Kind() == reflect.Slice || paramValue.Kind() == reflect.Array) && paramValue.Len() != 0 {
			placeholders := "(" + strings.Repeat("?,", paramValue.Len()-1) + "?" + ")"
			queryBuilder.WriteString(placeholders)

			// 将切片/数组元素加入 args
			for j := 0; j < paramValue.Len(); j++ {
				engine.args = append(engine.args, paramValue.Index(j).Interface())
			}
		} else {
			// 普通参数，直接添加占位符
			queryBuilder.WriteString("?")
			engine.args = append(engine.args, param)
		}
		// 添加切割后的下一部分
		queryBuilder.WriteString(queryParts[i+1])
	}
	engine.where_sql = append(engine.where_sql, queryBuilder.String())
	return &engine
}

// GroupBy 设置分组条件
//
// 参数:
//   - groups: ...string 分组字段名列表
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名会自动去除首尾的空格和引号字符
//   - 字段名会自动添加反引号
//   - 空字段名会被忽略
//   - 不传参数时清空分组条件
func (engine Engine) GroupBy(groups ...string) *Engine {
	var groupFields []string
	for _, field := range groups {
		field = strings.Trim(field, " `'\"")
		if field == "" {
			continue
		}
		groupFields = append(groupFields, fmt.Sprintf("`%s`", field))
	}
	if len(groupFields) > 0 {
		engine.group_sql = fmt.Sprintf(" GROUP BY %s", strings.Join(groupFields, ", "))
	} else {
		engine.group_sql = ""
	}
	return &engine
}

// OrderBy 设置排序条件
//
// 参数:
//   - orders: ...string 排序规则列表，字段名前加"-"表示降序，否则为升序
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名会自动去除首尾的空格和引号字符
//   - 字段名会自动添加反引号
//   - 空字段名会被忽略
//   - 不传参数时清空排序条件
func (engine Engine) OrderBy(orders ...string) *Engine {
	var orderFields []string
	for _, order := range orders {
		// 去除字段名首尾的空格和引号
		order = strings.Trim(order, " `'\"")
		if order == "" {
			continue
		}

		var sequence string
		var fieldName string
		// 处理排序方向
		if order[0] == '-' {
			fieldName = order[1:]
			sequence = "DESC"
		} else {
			fieldName = order
			sequence = "ASC"
		}
		orderFields = append(orderFields, fmt.Sprintf("`%s` %s", fieldName, sequence))
	}
	if len(orderFields) > 0 {
		engine.order_sql = fmt.Sprintf(" ORDER BY %s", strings.Join(orderFields, ", "))
	} else {
		engine.order_sql = ""
	}
	return &engine
}

// Page 设置分页参数
//
// 参数:
//   - page: int64 页码，从1开始
//   - pagesize: int64 每页记录数
//
// 返回:
//   - int64: 总记录数
//   - int64: 总页数
//   - error: 查询过程中的错误
//
// 说明:
//   - 页码小于等于0时自动设为1
//   - 每页记录数小于等于0时设为默认值10
//   - 总页数根据总记录数和每页记录数计算
//   - 会自动执行一次COUNT查询获取总记录数
func (engine *Engine) Page(page int64, pageSize int64) (int64, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10 // 默认分页大小
	}
	offset := (page - 1) * pageSize
	engine.limit_sql = fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)
	total, err := engine.Count()
	totalPages := int64(math.Ceil(float64(total) / float64(pageSize)))
	return total, totalPages, err
}

// Select 执行查询并将结果扫描到切片中
//
// 参数:
//   - queryResult: []*struct{} 或 []map[string]any 用于接收结果的切片指针
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须传入结构体指针切片或map切片
//   - 支持指针和非指针类型的字段
//   - 自动映射数据库字段到结构体字段
//   - 查询失败时返回错误信息
func (engine *Engine) Select(queryResult any) error {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	var (
		isPtr    bool
		ItemType reflect.Type
		result   = reflect.ValueOf(queryResult)
	)
	if result.Kind() != reflect.Ptr {
		isNotPtrErrorMsg := i18n.T("db.is_not_ptr", map[string]any{"name": "queryResult"})
		return errors.New(isNotPtrErrorMsg)
	}
	result = result.Elem()

	kind := result.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		isNotSlicePtrErrorMsg := i18n.T("db.is_not_slice_or_array", map[string]any{"name": "queryResult"})
		return errors.New(isNotSlicePtrErrorMsg)
	}

	ItemType = result.Type().Elem() // 获取切片中的元素
	if ItemType.Kind() == reflect.Ptr {
		isPtr = true
		ItemType = ItemType.Elem()
	}
	if kind := ItemType.Kind(); kind != reflect.Struct && kind != reflect.Map {
		isNotStructPtrErrorMsg := i18n.T("db.is_not_slice_struct_ptr_or_map", map[string]any{"name": "queryResult"})
		return errors.New(isNotStructPtrErrorMsg)
	}

	fieldsSQL := strings.Join(engine.field_sql, "`,`")

	engine.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQL, TableName)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
	}
	if engine.group_sql != "" {
		engine.sql += engine.group_sql
	}
	if engine.order_sql != "" {
		engine.sql += engine.order_sql
	}
	if engine.limit_sql != "" {
		engine.sql += engine.limit_sql
	}

	rows, err := engine.Query(engine.sql, engine.args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		values := make([]any, len(engine.fields))

		var item reflect.Value
		if ItemType.Kind() == reflect.Struct {
			// 如果是结构体类型，使用 New 创建
			item = reflect.New(ItemType).Elem()

			for i, fieldName := range engine.fields {
				fieldValue := item.FieldByName(fieldName)
				if !fieldValue.IsValid() || !fieldValue.CanAddr() {
					values[i] = new(any)
				} else {
					values[i] = fieldValue.Addr().Interface()
				}
			}
		} else {
			modelType := reflect.TypeOf(engine.model)
			if modelType.Kind() == reflect.Ptr {
				modelType = modelType.Elem()
			}
			for i, fieldName := range engine.fields {
				field, _ := modelType.FieldByName(fieldName)
				val := reflect.New(field.Type) // 获取模型字段类型
				values[i] = val.Interface()
			}
		}

		if err = rows.Scan(values...); err != nil {
			return err
		}

		// 设置 map 值
		if ItemType.Kind() == reflect.Map {
			item = reflect.MakeMap(ItemType)
			for i, fieldName := range engine.field_sql {
				val := reflect.ValueOf(values[i])
				item.SetMapIndex(reflect.ValueOf(fieldName), val.Elem())
			}
		}

		if isPtr {
			result.Set(reflect.Append(result, item.Addr()))
		} else {
			result.Set(reflect.Append(result, item))
		}
	}
	return rows.Err()
}

// First 获取查询结果的第一条记录
//
// 参数:
//   - queryResult: *struct{} 或 map[string]any 用于接收结果的结构体指针或map
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须传入结构体指针或map
//   - 自动映射数据库字段到结构体字段
//   - 无记录时返回sql.ErrNoRows
//   - 查询失败时返回错误信息
func (engine *Engine) First(queryResult any) error {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	result := reflect.ValueOf(queryResult)
	if result.Kind() != reflect.Map {
		if result.Kind() != reflect.Ptr {
			isNotPtrErrorMsg := i18n.T("db.is_not_ptr", map[string]any{"name": "queryResult"})
			return errors.New(isNotPtrErrorMsg)
		}
		result = result.Elem()
	}
	if kind := result.Kind(); kind != reflect.Struct && kind != reflect.Map {
		isNotStructPtrErrorMsg := i18n.T("db.is_not_struct_ptr_or_map", map[string]any{"name": "queryResult"})
		return errors.New(isNotStructPtrErrorMsg)
	}

	fieldsSQL := strings.Join(engine.field_sql, "`,`")
	engine.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQL, TableName)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
	}
	if engine.group_sql != "" {
		engine.sql += engine.group_sql
	}
	if engine.order_sql != "" {
		engine.sql += engine.order_sql
	}
	if engine.limit_sql != "" {
		engine.sql += engine.limit_sql
	}

	row := engine.QueryRow(engine.sql, engine.args...)
	if err := row.Err(); err != nil {
		return err
	}

	values := make([]any, len(engine.fields))
	if result.Kind() == reflect.Struct {
		for i, fieldName := range engine.fields {
			fieldValue := result.FieldByName(fieldName)
			if !fieldValue.IsValid() || !fieldValue.CanAddr() {
				values[i] = new(any)
			} else {
				values[i] = fieldValue.Addr().Interface()
			}
		}
	} else {
		modelType := reflect.TypeOf(engine.model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		for i, fieldName := range engine.fields {
			field, _ := modelType.FieldByName(fieldName)
			val := reflect.New(field.Type) // 获取模型字段类型
			values[i] = val.Interface()
		}
	}

	if err := row.Scan(values...); err != nil {
		return err
	}

	// 更新实际扫描到的值
	if result.Kind() == reflect.Map {
		for i, fieldName := range engine.field_sql {
			val := reflect.ValueOf(values[i])
			result.SetMapIndex(reflect.ValueOf(fieldName), val.Elem())
		}
	}

	return nil
}

// Count 获取符合当前条件的记录总数
//
// 返回:
//   - int64: 记录总数
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须先通过SetModel设置数据模型
//   - 会考虑当前设置的WHERE条件
//   - 查询失败时返回0和错误信息
func (engine *Engine) Count() (int64, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	engine.sql = fmt.Sprintf("SELECT count(*) FROM `%v`", TableName)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
	}

	row := engine.QueryRow(engine.sql, engine.args...)
	if err := row.Err(); err != nil {
		return 0, err
	}

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists 检查是否存在符合条件的记录
//
// 返回:
//   - bool: 是否存在记录
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须先通过SetModel设置数据模型
//   - 会考虑当前设置的WHERE条件
//   - 使用SELECT 1优化查询性能
//   - 查询出错时返回false和错误信息
//   - 无记录时返回false和nil
func (engine *Engine) Exists() (bool, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	engine.sql = fmt.Sprintf("SELECT 1 FROM `%v`", TableName)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
	}

	row := engine.QueryRow(engine.sql, engine.args...)
	if err := row.Err(); err != nil {
		return false, err
	}

	var exists int
	if err := row.Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}

// Update 更新符合条件的记录
//
// 参数:
//   - model: Model 包含更新数据的模型实例
//
// 返回:
//   - sql.Result: 更新操作的结果
//   - error: 更新过程中的错误
//
// 说明:
//   - 调用前必须先通过SetModel设置数据模型
//   - 只更新非空字段
//   - 支持指针和非指针类型的字段值
//   - 会考虑当前设置的WHERE条件
func (engine *Engine) Update(model Model) (sql.Result, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]any, 0)
	utils.Zip(engine.fields, engine.field_sql, func(fieldName, fieldSQL string) {
		field := modelValue.FieldByName(fieldName)
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		if field.IsValid() {
			updateFields = append(updateFields, fmt.Sprintf("`%v`=?", fieldSQL))
			updateValues = append(updateValues, field.Interface())
		}
	})
	fieldsSQL := strings.Join(updateFields, ",")

	engine.sql = fmt.Sprintf("UPDATE `%v` SET %v", TableName, fieldsSQL)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
		updateValues = append(updateValues, engine.args...)
	}
	return engine.Execute(engine.sql, updateValues...)
}

// Delete 删除符合条件的记录
//
// 返回:
//   - sql.Result: 删除操作的结果
//   - error: 删除过程中的错误
//
// 说明:
//   - 调用前必须先通过SetModel设置数据模型
//   - 根据当前设置的WHERE条件删除记录
func (engine *Engine) Delete() (sql.Result, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME
	engine.sql = fmt.Sprintf("DELETE FROM `%v`", TableName)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
	}
	return engine.Execute(engine.sql, engine.args...)
}

// Close 关闭数据库连接
func (engine *Engine) Close() error {
	return engine.DB.Close()
}
