package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// Engine 结构体用于管理 MySQL 数据库连接和操作
type Engine struct {
	name              string               // 数据库连接名称
	DB                *sql.DB              // 数据库连接对象
	tx                *sql.Tx              // 当前事务对象
	placeholderParser db.PlaceholderParser // 当前实例使用的占位符解析器
	identifierQuoter  db.IdentifierQuoter  // 当前实例使用的标识符引用器
	columnParser      db.ColumnParser      // 当前实例使用的列定义解析器
	model             Model                // 当前操作的数据模型
	tableName         string               // 当前模型对应的完整表名
	fields            []string             // 模型结构体字段名
	fieldSQL          []string             // 数据库表字段名
	whereSQL          []string             // WHERE 条件语句
	limitSQL          string               // LIMIT 分页语句
	groupSQL          string               // GROUP BY 分组语句
	orderSQL          string               // ORDER BY 排序语句
	sql               string               // 最终执行的 SQL 语句
	args              []any                // SQL 语句的参数值
}

// Name 获取数据库连接名称
//
// 返回:
//   - string: 当前数据库连接名称
func (engine Engine) Name() string {
	return engine.name
}

// GetSQL 获取最近一次执行使用的 SQL
//
// 返回:
//   - string: PlaceholderParser 解析后的 SQL，尚未执行时为空；不额外拼接参数值
func (engine Engine) GetSQL() string {
	return engine.sql
}

// Execute 使用默认上下文执行 SQL。
//
// 参数:
//   - query: string SQL 语句
//   - args: ...any SQL 参数值列表
//
// 返回:
//   - sql.Result: 执行结果
//   - error: 执行错误
//
// 说明:
//   - 等同于使用 context.Background() 调用 ExecuteContext
func (engine *Engine) Execute(query string, args ...any) (sql.Result, error) {
	return engine.ExecuteContext(context.Background(), query, args...)
}

// ExecuteContext 使用指定上下文执行 SQL。
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - query: string SQL 语句，使用当前 PlaceholderParser 的 ORM 占位符，默认 ?
//   - args: ...any SQL 参数值列表
//
// 返回:
//   - sql.Result: 执行操作的结果
//   - error: 执行过程中的错误
//
// 说明:
//   - 通过当前实例的 PlaceholderParser 解析 SQL 和参数，默认展开 IN / NOT IN 集合
//   - 保存解析后的 SQL 供 GetSQL 获取
//   - 已开启事务时使用当前事务，否则使用数据库连接
func (engine *Engine) ExecuteContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query, args = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.ExecContext(ctx, query, args...)
	}
	return engine.DB.ExecContext(ctx, query, args...)
}

// QueryRow 使用默认上下文查询单行。
//
// 参数:
//   - query: string SQL 查询语句
//   - args: ...any SQL 参数值列表
//
// 返回:
//   - *sql.Row: 查询结果行
//
// 说明:
//   - 等同于使用 context.Background() 调用 QueryRowContext
func (engine *Engine) QueryRow(query string, args ...any) *sql.Row {
	return engine.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext 使用指定上下文查询单行。
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - query: string 查询语句，使用当前 PlaceholderParser 的 ORM 占位符，默认 ?
//   - args: ...any SQL 参数值列表
//
// 返回:
//   - *sql.Row: 查询结果行
//
// 说明:
//   - 通过当前实例的 PlaceholderParser 解析 SQL 和参数，默认展开 IN / NOT IN 集合
//   - 保存解析后的 SQL 供 GetSQL 获取
//   - 已开启事务时使用当前事务；查询错误由 Row.Err 或 Scan 返回，无记录时 Scan 返回 sql.ErrNoRows
func (engine *Engine) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	query, args = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.QueryRowContext(ctx, query, args...)
	}
	return engine.DB.QueryRowContext(ctx, query, args...)
}

// Query 使用默认上下文查询多行。
//
// 参数:
//   - query: string SQL 查询语句
//   - args: ...any SQL 参数值列表
//
// 返回:
//   - *sql.Rows: 查询结果集
//   - error: 查询过程中的错误
//
// 说明:
//   - 等同于使用 context.Background() 调用 QueryContext
func (engine *Engine) Query(query string, args ...any) (*sql.Rows, error) {
	return engine.QueryContext(context.Background(), query, args...)
}

// QueryContext 使用指定上下文查询多行。
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - query: string 查询语句，使用当前 PlaceholderParser 的 ORM 占位符，默认 ?
//   - args: ...any SQL 参数值列表
//
// 返回:
//   - *sql.Rows: 查询结果集
//   - error: 查询过程中的错误
//
// 说明:
//   - 通过当前实例的 PlaceholderParser 解析 SQL 和参数，默认展开 IN / NOT IN 集合
//   - 保存解析后的 SQL 供 GetSQL 获取
//   - 已开启事务时使用当前事务；调用方须关闭结果集并检查 rows.Err()
func (engine *Engine) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	query, args = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.QueryContext(ctx, query, args...)
	}
	return engine.DB.QueryContext(ctx, query, args...)
}

// TransactionFunc 在指定 Engine 的事务中执行操作，返回错误表示执行失败。
type TransactionFunc func(engine *Engine, args ...any) error

// WithTransaction 在事务中执行回调。
//
// 参数:
//   - ctx: context.Context 事务上下文，用于取消和超时控制
//   - transactionFunc: TransactionFunc 事务回调，接收当前 Engine 的副本
//   - args: ...any 传递给事务函数的参数列表
//
// 返回:
//   - error: 开启事务、回调或提交时的错误
//
// 说明:
//   - 原实例的事务状态不变；最外层回调成功后提交，出错或 panic 时尝试回滚
//   - panic 在回滚后继续向上传播，不在此处恢复
//   - 嵌套调用复用当前事务，不创建保存点或单独提交、回滚；内层错误须向外返回才能使外层回滚
func (engine Engine) WithTransaction(ctx context.Context, transactionFunc TransactionFunc, args ...any) (err error) {
	if engine.tx != nil {
		err = transactionFunc(&engine, args...)
		if err != nil {
			return err
		}
		return nil
	}

	// 开启事务
	engine.tx, err = engine.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// 执行事务中的操作
	panicked := true
	defer func() {
		if panicked || err != nil {
			_ = engine.tx.Rollback()
		}
	}()

	// 执行传入的函数
	err = transactionFunc(&engine, args...)
	if err != nil {
		return err
	}

	panicked = false
	// 提交事务
	return engine.tx.Commit()
}

// Migrate 根据模型创建尚不存在的表。
//
// 参数:
//   - ctx: context.Context 迁移上下文，用于取消和超时控制
//   - model: Model 数据模型
//
// 说明:
//   - 会调用 SetModel 切换当前模型；表已存在时跳过，不修改已有表或创建数据库
//   - Settings.Schema 指定数据库，留空时使用当前连接选择的数据库
//   - 列定义由当前 ColumnParser 解析，表选项取自模型 Settings
//   - 支持迁移前后回调；查询、建表或回调失败时记录日志并 panic
func (engine *Engine) Migrate(ctx context.Context, model Model) {
	var err error
	engine.SetModel(model)
	settings := engine.model.ModelSet()
	schema := settings.Schema
	placeholder := engine.placeholderParser.Placeholder()

	var row *sql.Row
	if schema == "" {
		query := fmt.Sprintf("SELECT 1 FROM `information_schema`.`tables` WHERE `table_schema` = DATABASE() AND `table_name` = %s;", placeholder)
		row = engine.QueryRowContext(ctx, query, settings.TableName)
	} else {
		query := fmt.Sprintf("SELECT 1 FROM `information_schema`.`tables` WHERE `table_schema` = %s AND `table_name` = %s;", placeholder, placeholder)
		row = engine.QueryRowContext(ctx, query, schema, settings.TableName)
	}

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

	if exists > 0 {
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
		columnName, columnType, ok := engine.columnParser.Parse(field)
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("  %s %v", engine.identifierQuoter.QuoteIdentifier(columnName), columnType))
	}

	columnsSQL := strings.Join(columns, ",\n")
	createSQL := fmt.Sprintf("CREATE TABLE %s (\n%v\n)", engine.tableName, columnsSQL)
	if settings.Engine != "" { // 设置存储引擎
		createSQL += fmt.Sprintf(" Engine=%v", settings.Engine)
	}
	if settings.AUTO_INCREMENT != 0 { // 设置自增长起始值
		createSQL += fmt.Sprintf(" AUTO_INCREMENT=%v", settings.AUTO_INCREMENT)
	}
	if settings.DEFAULT_CHARSET != "" { // 设置默认字符集
		createSQL += fmt.Sprintf(" DEFAULT CHARSET=%v", settings.DEFAULT_CHARSET)
	}
	if settings.COLLATE != "" { // 设置校对规则
		createSQL += fmt.Sprintf(" COLLATE=%v", settings.COLLATE)
	}
	if settings.MIN_ROWS != 0 { // 设置最小行数
		createSQL += fmt.Sprintf(" MIN_ROWS=%v", settings.MIN_ROWS)
	}
	if settings.MAX_ROWS != 0 { // 设置最大行数
		createSQL += fmt.Sprintf(" MAX_ROWS=%v", settings.MAX_ROWS)
	}
	if settings.CHECKSUM != 0 { // 表格的校验和算法
		createSQL += fmt.Sprintf(" CHECKSUM=%v", settings.CHECKSUM)
	}
	if settings.DELAY_KEY_WRITE != 0 { // 控制非唯一索引的写延迟
		createSQL += fmt.Sprintf(" DELAY_KEY_WRITE=%v", settings.DELAY_KEY_WRITE)
	}
	if settings.ROW_FORMAT != "" { // 设置行的存储格式
		createSQL += fmt.Sprintf(" ROW_FORMAT=%v", settings.ROW_FORMAT)
	}
	if settings.DATA_DIRECTORY != "" { // 设置数据存储目录
		createSQL += fmt.Sprintf(" DATA DIRECTORY='%v'", settings.DATA_DIRECTORY)
	}
	if settings.INDEX_DIRECTORY != "" { // 设置索引存储目录
		createSQL += fmt.Sprintf(" INDEX DIRECTORY='%v'", settings.INDEX_DIRECTORY)
	}
	if settings.PARTITION_BY != "" { // 定义分区方式
		createSQL += fmt.Sprintf(" PARTITION_BY %v", settings.PARTITION_BY)
	}
	if settings.COMMENT != "" { // 设置表注释
		createSQL += fmt.Sprintf(" COMMENT='%v'", settings.COMMENT)
	}

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

	migrationModelMsg := i18n.T("db.migration.mysql", map[string]any{
		"engine":  driverName,
		"name":    engine.name,
		"db_name": schema,
		"tb_name": settings.TableName,
	})
	goi.Log.Info(migrationModelMsg)
	_, err = engine.ExecuteContext(ctx, createSQL)
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

// isSetModel 在尚未设置模型时 panic。
func (engine *Engine) isSetModel() {
	if engine.model == nil {
		notSetModelErrorMsg := i18n.T("db.not_SetModel_error")
		panic(notSetModelErrorMsg)
	}
}

// SetModel 设置当前模型并重置查询状态。
//
// 参数:
//   - model: Model 要设置的数据模型实例
//
// 返回:
//   - *Engine: 当前实例，支持链式调用
//
// 说明:
//   - 模型须为结构体或非 nil 的结构体指针，否则 panic
//   - 按当前 ColumnParser 解析字段并缓存完整表名，清空条件、分组、排序、分页及参数
//   - 默认按 field_type 和 field_name 标签解析，具体规则见 GetColumnParser
func (engine *Engine) SetModel(model Model) *Engine {
	modelValue, err := db.ModelStructValue(model)
	if err != nil {
		panic(err)
	}
	engine.model = model
	settings := model.ModelSet()
	if settings.Schema == "" {
		engine.tableName = engine.identifierQuoter.QuoteIdentifier(settings.TableName)
	} else {
		engine.tableName = engine.identifierQuoter.QuoteIdentifier(settings.Schema) + "." + engine.identifierQuoter.QuoteIdentifier(settings.TableName)
	}
	// 获取字段
	engine.fields = nil
	engine.fieldSQL = nil
	engine.whereSQL = nil
	engine.limitSQL = ""
	engine.groupSQL = ""
	engine.orderSQL = ""
	engine.args = nil
	engine.fields, engine.fieldSQL = db.ModelFields(modelValue, engine.columnParser)
	return engine
}

// Fields 指定查询、插入或更新使用的字段。
//
// 参数:
//   - fields: ...string 模型结构体字段名，区分大小写，不是数据库列名
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 先调用 SetModel；字段不存在时 panic，当前 ColumnParser 排除的字段会被忽略
//   - 未调用时使用全部映射字段；每次调用替换之前的选择，不传字段时选中空集合
//   - Insert/Update 在选中字段内按 skipNil 处理 nil 值
func (engine Engine) Fields(fields ...string) *Engine {
	modelType := reflect.TypeOf(engine.model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	// 获取字段
	engine.fields = nil
	engine.fieldSQL = nil

	for _, fieldName := range fields {
		field, ok := modelType.FieldByName(fieldName)
		if !ok {
			fieldIsNotErrorMsg := i18n.T("db.field_is_not_error", map[string]any{"name": fieldName})
			goi.Log.Error(fieldIsNotErrorMsg)
			panic(fieldIsNotErrorMsg)
		}

		columnName, _, ok := engine.columnParser.Parse(field)
		if !ok {
			continue
		}
		engine.fields = append(engine.fields, field.Name)
		engine.fieldSQL = append(engine.fieldSQL, columnName)
	}
	return &engine
}

// Insert 向数据库插入一条记录
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - model: Model 待插入的模型实例
//   - skipNil: bool nil 值的处理方式；true 时省略该列，false 时写入 SQL NULL
//
// 返回:
//   - sql.Result: 插入结果
//   - error: 插入错误
//
// 说明:
//   - 先调用 SetModel 设置模型；Fields 可选插入字段，默认使用全部映射字段
//   - 支持普通值和指针；nil 指针按 nil 处理，0、false、空字符串正常写入
//   - 省略的列由数据库按默认值、自动生成规则及约束处理
//   - sql.Null、driver.Valuer 由 database/sql 转换，skipNil 不检查转换结果
//   - 字段列表为空时仍执行 SQL，由数据库处理
func (engine *Engine) Insert(ctx context.Context, model Model, skipNil bool) (sql.Result, error) {
	engine.isSetModel()

	modelValue, err := db.ValidateOperationModel(engine.model, model)
	if err != nil {
		return nil, err
	}

	insertFields := make([]string, 0, len(engine.fields))
	insertValues := make([]any, 0, len(engine.fields))
	for i, fieldName := range engine.fields {
		value, ok := db.ModelFieldValue(modelValue, fieldName)
		if !ok || skipNil && value == nil {
			continue
		}
		insertFields = append(insertFields, engine.fieldSQL[i])
		insertValues = append(insertValues, value)
	}

	placeholder := engine.placeholderParser.Placeholder()
	valuesSQL := strings.TrimRight(strings.Repeat(placeholder+",", len(insertFields)), ",")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", engine.tableName, engine.identifierQuoter.QuoteIdentifiers(insertFields), valuesSQL)
	return engine.ExecuteContext(ctx, query, insertValues...)
}

// Where 追加查询条件
//
// 参数:
//   - query: string 条件语句，使用当前 PlaceholderParser 的 ORM 占位符，默认 ?
//   - args: ...any 与占位符一一对应的 SQL 参数；IN 集合可传切片、数组及其非 nil 指针
//
// 返回:
//   - *Engine: 条件独立的实例副本，支持链式调用
//
// 说明:
//   - 多次调用以 AND 连接；占位符与参数数量不一致时 panic
//   - 默认 PlaceholderParser 在执行时展开 IN ?、NOT IN ? 的集合参数
//   - 字节切片（如 []byte）和 nil 指针不展开；其它未匹配的参数交由 database/sql 和驱动处理
//   - IN (?) 等带括号的写法不自动展开，需自行匹配占位符与参数
//   - 空集合（含 nil 切片）展开为 ()，是否合法由数据库处理
//   - 仅按字符串匹配占位符，不解析 SQL 字符串或注释
//   - 自定义 PlaceholderParser 时，集合展开及 SQL、参数转换由其 Parse 方法决定
func (engine Engine) Where(query string, args ...any) *Engine {
	// 复制条件和参数，避免不同查询分支相互影响。
	engine.whereSQL = append([]string(nil), engine.whereSQL...)
	engine.args = append([]any(nil), engine.args...)
	placeholder := engine.placeholderParser.Placeholder()
	if strings.Count(query, placeholder) != len(args) {
		whereArgsPlaceholderErrorMsg := i18n.T("db.where_args_placeholder_error")
		goi.Log.Error(whereArgsPlaceholderErrorMsg)
		panic(whereArgsPlaceholderErrorMsg)
	}

	engine.whereSQL = append(engine.whereSQL, query)
	engine.args = append(engine.args, args...)
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
//   - 字段交由当前 IdentifierQuoter 处理，默认使用反引号并忽略空字段
//   - 每次调用替换之前的分组，不传有效字段时清空分组
func (engine Engine) GroupBy(groups ...string) *Engine {
	groupFields := engine.identifierQuoter.QuoteIdentifiers(groups)
	if groupFields != "" {
		engine.groupSQL = " GROUP BY " + groupFields
	} else {
		engine.groupSQL = ""
	}
	return &engine
}

// OrderBy 设置排序条件
//
// 参数:
//   - orders: ...string 排序字段；默认升序，前缀 - 表示降序，例如 name、-created_at
//
// 返回:
//   - *Engine: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 去除字段首尾空格后交由当前 IdentifierQuoter 处理，默认使用反引号
//   - 每次调用替换之前的排序，空字段会被忽略；不传有效字段时清空排序
func (engine Engine) OrderBy(orders ...string) *Engine {
	var orderFields []string
	for _, order := range orders {
		order = strings.TrimSpace(order)
		if order == "" {
			continue
		}

		var sequence string
		var fieldName string
		// 处理排序方向
		if order[0] == '-' {
			fieldName = strings.TrimSpace(order[1:])
			sequence = "DESC"
		} else {
			fieldName = order
			sequence = "ASC"
		}
		if fieldName == "" {
			continue
		}
		orderFields = append(orderFields, fmt.Sprintf("%s %s", engine.identifierQuoter.QuoteIdentifier(fieldName), sequence))
	}
	if len(orderFields) > 0 {
		engine.orderSQL = fmt.Sprintf(" ORDER BY %s", strings.Join(orderFields, ", "))
	} else {
		engine.orderSQL = ""
	}
	return &engine
}

// Page 设置当前实例的分页并查询总记录数。
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - page: int64 页码，从 1 开始
//   - pageSize: int64 每页记录数
//
// 返回:
//   - int64: 总记录数
//   - int64: 总页数
//   - error: 查询过程中的错误
//
// 说明:
//   - page 小于等于 0 时按 1 处理，pageSize 小于等于 0 时按 10 处理
//   - 使用 LIMIT/OFFSET 分页，并调用 Count；总数仅受 WHERE 影响，不受分组和分页影响
func (engine *Engine) Page(ctx context.Context, page int64, pageSize int64) (int64, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10 // 默认分页大小
	}
	offset := (page - 1) * pageSize
	engine.limitSQL = fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)
	total, err := engine.Count(ctx)
	totalPages := int64(math.Ceil(float64(total) / float64(pageSize)))
	return total, totalPages, err
}

// Select 执行查询并将结果扫描到切片中
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - queryResult: any 非 nil 的切片指针，元素可为结构体、结构体指针或 map
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 先调用 SetModel；按 Fields 选择的字段扫描，默认使用全部映射字段
//   - 切片本身可为 nil，不支持数组；每行成功扫描后追加到切片，不清空已有元素
//   - 结构体按字段名匹配；map 按模型字段类型读取，以结构体字段名为键，调用方须保证键值类型兼容
//   - 无记录时不修改切片并返回 nil；查询、扫描和迭代错误会返回，已追加的结果会保留
func (engine *Engine) Select(ctx context.Context, queryResult any) error {
	engine.isSetModel()

	writeResult, err := db.PrepareSliceResult(engine.model, queryResult, engine.fields)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", engine.identifierQuoter.QuoteIdentifiers(engine.fieldSQL), engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %v", strings.Join(engine.whereSQL, " AND "))
	}
	if engine.groupSQL != "" {
		query += engine.groupSQL
	}
	if engine.orderSQL != "" {
		query += engine.orderSQL
	}
	if engine.limitSQL != "" {
		query += engine.limitSQL
	}

	rows, err := engine.QueryContext(ctx, query, engine.args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err = writeResult(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// First 获取查询结果的第一条记录
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - queryResult: any 非 nil 的结构体指针或已初始化的 map
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 先调用 SetModel；按 Fields 选择的字段扫描，默认使用全部映射字段
//   - 结构体按字段名匹配；map 按模型字段类型读取，以结构体字段名为键，调用方须保证键值类型兼容
//   - 使用当前条件、分组、排序及分页获取第一行；不自动按主键排序
//   - 无记录时返回 sql.ErrNoRows，其他查询或扫描错误原样返回
func (engine *Engine) First(ctx context.Context, queryResult any) error {
	engine.isSetModel()

	writeResult, err := db.PrepareSingleResult(engine.model, queryResult, engine.fields)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", engine.identifierQuoter.QuoteIdentifiers(engine.fieldSQL), engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %v", strings.Join(engine.whereSQL, " AND "))
	}
	if engine.groupSQL != "" {
		query += engine.groupSQL
	}
	if engine.orderSQL != "" {
		query += engine.orderSQL
	}
	if engine.limitSQL != "" {
		query += engine.limitSQL
	}

	row := engine.QueryRowContext(ctx, query, engine.args...)
	return writeResult(row)
}

// Count 统计符合当前 WHERE 条件的记录数。
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//
// 返回:
//   - int64: 满足当前 WHERE 条件的记录总数
//   - error: 查询过程中的错误
//
// 说明:
//   - 先调用 SetModel；未设置 WHERE 时统计全表，不受 Fields、分组、排序或分页影响
//   - 查询失败时返回 0 和错误
func (engine *Engine) Count(ctx context.Context) (int64, error) {
	engine.isSetModel()

	query := fmt.Sprintf("SELECT count(*) FROM %s", engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %v", strings.Join(engine.whereSQL, " AND "))
	}

	row := engine.QueryRowContext(ctx, query, engine.args...)
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
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//
// 返回:
//   - bool: 是否存在至少一条满足条件的记录
//   - error: 查询过程中的错误
//
// 说明:
//   - 先调用 SetModel；仅使用当前 WHERE 条件，未设置时检查全表
//   - 无记录时返回 false, nil，其他查询错误原样返回
func (engine *Engine) Exists(ctx context.Context) (bool, error) {
	engine.isSetModel()

	query := fmt.Sprintf("SELECT 1 FROM %s", engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %v", strings.Join(engine.whereSQL, " AND "))
	}

	row := engine.QueryRowContext(ctx, query, engine.args...)
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
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - model: Model 待更新的模型实例
//   - skipNil: bool nil 值的处理方式；true 时保留原值，false 时更新为 SQL NULL
//
// 返回:
//   - sql.Result: 更新结果
//   - error: 更新错误
//
// 说明:
//   - 先调用 SetModel 设置模型；Fields 可选更新字段，默认使用全部映射字段
//   - 未选中或跳过的字段保持原值
//   - 支持普通值和指针；nil 指针按 nil 处理，0、false、空字符串正常写入
//   - sql.Null、driver.Valuer 由 database/sql 转换，skipNil 不检查转换结果
//   - 没有可更新字段时返回错误，不执行 SQL
//   - 未设置 Where 条件时更新整张表，调用方需明确操作范围
func (engine *Engine) Update(ctx context.Context, model Model, skipNil bool) (sql.Result, error) {
	engine.isSetModel()

	modelValue, err := db.ValidateOperationModel(engine.model, model)
	if err != nil {
		return nil, err
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]any, 0)
	placeholder := engine.placeholderParser.Placeholder()
	for i, fieldName := range engine.fields {
		value, ok := db.ModelFieldValue(modelValue, fieldName)
		if !ok || skipNil && value == nil {
			continue
		}
		updateFields = append(updateFields, fmt.Sprintf("%s = %s", engine.identifierQuoter.QuoteIdentifier(engine.fieldSQL[i]), placeholder))
		updateValues = append(updateValues, value)
	}
	if len(updateFields) == 0 {
		updateFieldsIsEmptyErrorMsg := i18n.T("db.update_fields_is_empty")
		return nil, errors.New(updateFieldsIsEmptyErrorMsg)
	}
	fieldsSQL := strings.Join(updateFields, ",")

	query := fmt.Sprintf("UPDATE %s SET %s", engine.tableName, fieldsSQL)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(engine.whereSQL, " AND "))
		updateValues = append(updateValues, engine.args...)
	}
	return engine.ExecuteContext(ctx, query, updateValues...)
}

// Delete 删除符合条件的记录
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//
// 返回:
//   - sql.Result: 删除操作的结果
//   - error: 删除过程中的错误
//
// 说明:
//   - 先调用 SetModel；仅使用当前 WHERE 条件，不受 Fields、分组、排序或分页影响
//   - 未设置 Where 条件时删除整张表，调用方需明确操作范围
func (engine *Engine) Delete(ctx context.Context) (sql.Result, error) {
	engine.isSetModel()

	query := fmt.Sprintf("DELETE FROM %s", engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(engine.whereSQL, " AND "))
	}
	return engine.ExecuteContext(ctx, query, engine.args...)
}
