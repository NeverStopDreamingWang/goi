package postgresql

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

// driverName 与 goi.Settings.DATABASES[*].ENGINE 保持一致
const driverName = "postgres"

func init() {
	// 注册 PostgreSQL ORM 引擎
	db.Register(driverName, factory)
}

// factory 是 db.Engine 的工厂函数
func factory(UseDataBases string, database goi.DataBase) db.Engine {
	return ConnectDatabase(UseDataBases, database)
}

// ConnectDatabase 连接 PostgreSQL 数据库
//
// 参数:
//   - UseDataBases: string 数据库配置名称
//   - database: goi.DataBase 数据库连接管理器
//
// 返回:
//   - *Engine: PostgreSQL 数据库操作实例
//
// 说明:
//   - 如果 ENGINE 不匹配则 panic
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

// Connect 连接 PostgreSQL 数据库
//
// 参数:
//   - UseDataBases: string 数据库配置名称
//
// 返回:
//   - *Engine: PostgreSQL 数据库操作实例
//
// 说明:
//   - 从全局配置中获取数据库连接信息
//   - 如果 ENGINE 不匹配则 panic
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

// Engine 结构体用于管理 PostgreSQL 数据库连接和操作
type Engine struct {
	name        string   // 数据库连接名称
	DB          *sql.DB  // 数据库连接对象
	transaction *sql.Tx  // 当前事务对象
	model       Model    // 当前操作的数据模型
	fields      []string // 模型结构体字段名
	field_sql   []string // 数据库表字段名
	where_sql   []string // WHERE 条件语句
	limit_sql   string   // LIMIT/OFFSET 分页语句
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

// GetSQL 获取最近一次执行的 SQL 语句
//
// 返回:
//   - string: 最近一次执行的 SQL 语句
func (engine Engine) GetSQL() string {
	return engine.sql
}

// 事务执行函数
type TransactionFunc func(engine *Engine, args ...any) error

// WithTransaction 使用事务执行函数
//
// 参数:
//   - transactionFunc: TransactionFunc func(engine *Engine, args ...any) error 事务执行函数
//   - args: ...any 传递给事务函数的参数列表
//
// 返回:
//   - error: 事务执行的错误，发生错误时会自动回滚
//
// 说明:
//   - 使用值接收者，在事务内部会基于当前 Engine 复制一个副本，避免事务内外状态互相干扰
//   - 不支持嵌套事务，如果当前已在事务中则返回错误
//   - 事务函数执行失败时自动回滚，支持内部 panic，panic 会在回滚后重新抛出
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

// Execute 执行 SQL 语句
//
// 参数:
//   - query: string SQL语句，使用 ? 作为占位符，例如: "id = ? AND status IN (?)"
//   - args: ...any SQL参数值列表
//
// 返回:
//   - sql.Result: 执行操作的结果
//   - error: 执行过程中的错误
//
// 说明:
//   - 使用 ? 作为统一占位符，最终会被转换为 PostgreSQL 的 $1,$2,... 格式
//   - 支持在事务中使用
func (engine *Engine) Execute(query string, args ...any) (sql.Result, error) {
	query = convertPlaceholders(query, len(args))
	engine.sql = query
	if engine.transaction != nil {
		return engine.transaction.Exec(query, args...)
	}
	return engine.DB.Exec(query, args...)
}

// QueryRow 执行查询 SQL 语句
//
// 参数:
//   - query: string SQL查询语句，使用 ? 作为占位符，例如: "id = ? AND status IN (?)"
//   - args: ...any SQL参数值列表
//
// 返回:
//   - *sql.Row: 查询结果行
//
// 说明:
//   - 使用 ? 作为统一占位符，最终会被转换为 PostgreSQL 的 $1,$2,... 格式
//   - 查询失败时返回nil
//   - 支持在事务中使用
func (engine *Engine) QueryRow(query string, args ...any) *sql.Row {
	query = convertPlaceholders(query, len(args))
	engine.sql = query
	if engine.transaction != nil {
		return engine.transaction.QueryRow(query, args...)
	}
	return engine.DB.QueryRow(query, args...)
}

// Query 执行查询 SQL 语句
//
// 参数:
//   - query: string SQL查询语句，使用 ? 作为占位符，例如: "id = ? AND status IN (?)"
//   - args: ...any SQL参数值列表
//
// 返回:
//   - *sql.Rows: 查询结果集
//   - error: 查询过程中的错误
//
// 说明:
// - 使用 ? 作为统一占位符，最终会被转换为 PostgreSQL 的 $1,$2,... 格式
// - 返回的结果集需要调用方手动关闭
// - 查询失败时返回nil和错误信息
// - 支持在事务中使用
func (engine *Engine) Query(query string, args ...any) (*sql.Rows, error) {
	query = convertPlaceholders(query, len(args))
	engine.sql = query
	if engine.transaction != nil {
		return engine.transaction.Query(query, args...)
	}
	return engine.DB.Query(query, args...)
}

// Migrate 根据模型创建数据库表
//
// 参数:
//   - schema: string 数据库名称，如果连接字符串中已指定数据库则可以传入空字符串
//   - model: Model 数据模型
//
// 说明:
//   - PostgreSQL 不区分数据库名，通常通过 schema 管理命名空间
//   - 检查表是否已存在，存在则跳过创建
//   - 解析模型结构体的字段标签
//   - 支持自定义迁移前后处理函数
//   - 创建失败时会panic
func (engine *Engine) Migrate(schema string, model Model) {
	var err error
	settings := model.ModelSet()

	// schema 为空则使用 public
	if schema == "" {
		schema = "public"
	}

	row := engine.QueryRow("SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?;", schema, settings.TABLE_NAME)

	var exists int
	err = row.Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		selectErrorMsg := i18n.T("db.select_error", map[string]any{
			"engine":  driverName,
			"name":    engine.name,
			"db_name": schema,
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
		columns = append(columns, fmt.Sprintf("  \"%v\" %v", fieldName, fieldType))
	}

	columnsSQL := strings.Join(columns, ",\n")
	createSQL := fmt.Sprintf("CREATE TABLE \"%v\".\"%v\" (\n%v\n)", schema, settings.TABLE_NAME, columnsSQL)

	// 创建表
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

	migrationModelMsg := i18n.T("db.migration.postgresql", map[string]any{
		"engine":  driverName,
		"name":    engine.name,
		"db_name": schema,
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
//   - 如果未设置模型则 panic
func (engine *Engine) isSetModel() {
	if engine.model == nil {
		notSetModelErrorMsg := i18n.T("db.not_SetModel_error")
		panic(notSetModelErrorMsg)
	}
}

// SetModel 设置当前操作的数据模型
//
// 参数:
//   - model: Model 要设置的数据模型实例
//
// 返回:
//   - *Engine: 当前 Engine 指针，支持链式调用
//
// 说明:
//   - 会重置内部的字段缓存、条件、排序、分组、分页等状态
//   - 只解析带有 field_type 标签的结构体字段
//   - 数据库字段名默认使用字段名的小写形式，或由 field_name 标签显式指定
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
//   - fields: ...string 要查询的字段名列表（模型结构体中的字段名）
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名必须是模型结构体中已定义的字段
//   - 只返回带有 field_type 标签的字段，未标记的字段会被忽略
//   - 字段名大小写不敏感，会自动转换为数据库字段名
//   - 数据库字段名默认使用字段名的小写形式，或由 field_name 标签显式指定
//   - 如果字段不存在则会 panic
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
//   - 调用前必须先通过 SetModel 设置数据模型
//   - 支持指针和非指针类型的字段值
//   - 只插入带有 field_type 标签的字段
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

	fieldsSQL := strings.Join(engine.field_sql, "\",\"")
	valuesSQL := strings.TrimRight(strings.Repeat("?,", len(engine.fields)), ",")

	engine.sql = fmt.Sprintf("INSERT INTO \"%v\" (\"%v\") VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return engine.Execute(engine.sql, insertValues...)
}

// Where 构造 WHERE 条件
//
// 参数:
//   - query: string 条件语句，使用 ? 作为占位符，例如: "id = ? AND status IN (?)"
//   - args: ...any 对应占位符的参数列表，支持基本类型和切片/数组
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 支持多次调用，条件之间使用 AND 连接
//   - 使用 ? 作为统一占位符，最终会在 Execute/Query 中被转换为 PostgreSQL 的 $1,$2,... 格式
//   - 对切片/数组参数会自动展开为 IN (...) 形式，占位符数量与元素个数一致
//   - 空切片/数组会被当作普通参数处理，此时生成的 SQL 可能不是预期的 IN () 语义，应在业务层避免传入空集合
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
//   - 字段名会被自动去除首尾空格和引号
//   - 字段名最终会使用 PostgreSQL 的双引号引用: "column"
//   - 空字段名会被忽略
//   - 多次调用会覆盖之前的分组设置
func (engine Engine) GroupBy(groups ...string) *Engine {
	var groupFields []string
	for _, field := range groups {
		field = strings.Trim(field, " `'\"")
		if field == "" {
			continue
		}
		groupFields = append(groupFields, fmt.Sprintf("\"%s\"", field))
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
//   - orders: ...string 排序字段列表，支持前缀 "-" 表示倒序，例如: "name", "-created_at"
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名会被自动去除首尾空格和引号
//   - 不带前缀时使用 ASC 升序，带 "-" 前缀时使用 DESC 降序
//   - 字段名最终会使用 PostgreSQL 的双引号引用: "column" ASC/DESC
//   - 多次调用会覆盖之前的排序设置
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
		orderFields = append(orderFields, fmt.Sprintf("\"%s\" %s", fieldName, sequence))
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
//   - page: int64 页码，从 1 开始
//   - pageSize: int64 每页记录数
//
// 返回:
//   - int64: 总记录数
//   - int64: 总页数
//   - error: 查询过程中的错误
//
// 说明:
//   - 页码小于等于 0 时自动设为 1
//   - 每页记录数小于等于 0 时设为默认值 10
//   - 总页数根据总记录数和每页记录数计算
//   - 会自动执行一次 Count 查询获取总记录数
//   - PostgreSQL 使用 LIMIT pageSize OFFSET offset 实现分页
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
//   - queryResult: *[]T 或 []*T，用于接收结果的切片指针，其中 T 可以是 struct 或 map 类型
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须传入切片指针，且元素类型为结构体或 map
//   - 支持指针和非指针类型的结构体元素（[]T / []*T）
//   - 会根据 SetModel/Fields 解析的字段集合，自动将数据库字段映射到结构体字段或 map 键上
//   - 查询结束后会检查 rows.Err()，不会静默吞掉迭代过程中的错误
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

	fieldsSQL := strings.Join(engine.field_sql, "\",\"")
	engine.sql = fmt.Sprintf("SELECT \"%v\" FROM \"%v\"", fieldsSQL, TableName)
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
//   - queryResult: *struct{} 或 map[string]any / *map[string]any，用于接收结果的结构体指针或 map
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须传入结构体指针或 map/map 指针
//   - 自动映射数据库字段到结构体字段或 map 键
//   - 无记录时返回 nil（不视为错误），调用方可通过业务逻辑判断是否为空
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

	fieldsSQL := strings.Join(engine.field_sql, "\",\"")
	engine.sql = fmt.Sprintf("SELECT \"%v\" FROM \"%v\"", fieldsSQL, TableName)
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

// Count 统计符合当前条件的记记录数
//
// 返回:
//   - int64: 满足当前 WHERE 条件的记录总数
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须先通过 SetModel 设置数据模型
//   - 会根据当前设置的 WHERE 条件生成 COUNT 语句
//   - 查询失败时返回0和错误信息
func (engine *Engine) Count() (int64, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	engine.sql = fmt.Sprintf("SELECT count(*) FROM \"%v\"", TableName)
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

// Exists 判断是否存在符合条件的记录
//
// 返回:
//   - bool: 是否存在至少一条满足条件的记录
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须先通过 SetModel 设置数据模型
//   - 会根据当前 WHERE 条件生成 SELECT 1 FROM "table" ...
//   - 使用 QueryRow 执行，若返回 sql.ErrNoRows 则视为不存在（false, nil）
//   - 其他错误会原样返回
func (engine *Engine) Exists() (bool, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME

	engine.sql = fmt.Sprintf("SELECT 1 FROM \"%v\"", TableName)
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
//   - model: Model 要更新的数据模型实例
//
// 返回:
//   - sql.Result: 更新操作的结果
//   - error: 更新过程中的错误
//
// 说明:
//   - 调用前必须先通过 SetModel 设置数据模型
//   - 只更新非空字段
//   - 支持指针和非指针类型的字段值
//   - 会根据当前设置的 WHERE 条件生成 UPDATE 语句
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
			updateFields = append(updateFields, fmt.Sprintf("\"%v\"= ?", fieldSQL))
			updateValues = append(updateValues, field.Interface())
		}
	})
	fieldsSQL := strings.Join(updateFields, ",")

	engine.sql = fmt.Sprintf("UPDATE \"%v\" SET %v", TableName, fieldsSQL)
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
//   - 调用前必须先通过 SetModel 设置数据模型
//   - 会根据当前设置的 WHERE 条件生成 DELETE 语句
//   - 未设置 WHERE 条件时将删除整张表中的所有记录，使用时需谨慎
func (engine *Engine) Delete() (sql.Result, error) {
	engine.isSetModel()

	TableName := engine.model.ModelSet().TABLE_NAME
	engine.sql = fmt.Sprintf("DELETE FROM \"%v\"", TableName)
	if len(engine.where_sql) > 0 {
		engine.sql += fmt.Sprintf(" WHERE %v", strings.Join(engine.where_sql, " AND "))
	}
	return engine.Execute(engine.sql, engine.args...)
}

// Close 关闭数据库连接
//
// 返回:
//   - error: 关闭连接时发生的错误
//
// 说明:
//   - 内部调用 sql.DB.Close()
//   - 一般在应用进程退出前框架自动调用注册过的数据库连接
func (engine *Engine) Close() error {
	return engine.DB.Close()
}
