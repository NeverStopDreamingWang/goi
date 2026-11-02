package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi/v2"
	"github.com/NeverStopDreamingWang/goi/v2/db"
	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// Engine 管理 PostgreSQL 连接、事务和查询状态
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
	limitSQL          string               // 分页语句
	groupSQL          string               // GROUP BY 分组语句
	orderSQL          string               // ORDER BY 排序语句
	sql               string               // 最终执行的 SQL 语句
	args              []any                // SQL 语句的参数值
}

// Name 获取连接配置名称
//
// 返回:
//   - string: 创建实例时指定的连接配置名称
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

// Execute 执行 SQL
//
// 参数:
//   - query: string SQL 语句
//   - args: ...any SQL 参数
//
// 返回:
//   - sql.Result: 执行结果
//   - error: 执行错误
//
// 说明:
//   - 使用 context.Background() 调用 ExecuteContext
func (engine *Engine) Execute(query string, args ...any) (sql.Result, error) {
	return engine.ExecuteContext(context.Background(), query, args...)
}

// ExecuteContext 使用上下文执行 SQL
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - query: string SQL 语句，使用当前 PlaceholderParser 的占位符（默认 ?）
//   - args: ...any SQL 参数
//
// 返回:
//   - sql.Result: 执行结果
//   - error: 执行错误
//
// 说明:
//   - 执行前通过当前实例的 PlaceholderParser 转换 SQL，并保存供 GetSQL 获取
//   - 有事务时使用当前事务，否则使用连接池
func (engine *Engine) ExecuteContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.ExecContext(ctx, query, args...)
	}
	return engine.DB.ExecContext(ctx, query, args...)
}

// QueryRow 查询单行数据
//
// 参数:
//   - query: string SQL 语句
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Row: 单行结果，错误由 Scan 或 Err 返回
//
// 说明:
//   - 使用 context.Background() 调用 QueryRowContext
func (engine *Engine) QueryRow(query string, args ...any) *sql.Row {
	return engine.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext 使用上下文查询单行数据
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - query: string SQL 语句，使用当前 PlaceholderParser 的占位符（默认 ?）
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Row: 单行结果，错误由 Scan 或 Err 返回
//
// 说明:
//   - 执行前通过当前实例的 PlaceholderParser 转换 SQL，并保存供 GetSQL 获取
//   - 有事务时使用当前事务，否则使用连接池
func (engine *Engine) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	query = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.QueryRowContext(ctx, query, args...)
	}
	return engine.DB.QueryRowContext(ctx, query, args...)
}

// Query 查询多行数据
//
// 参数:
//   - query: string SQL 语句
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Rows: 结果集，需由调用方关闭
//   - error: 查询错误
//
// 说明:
//   - 使用 context.Background() 调用 QueryContext
func (engine *Engine) Query(query string, args ...any) (*sql.Rows, error) {
	return engine.QueryContext(context.Background(), query, args...)
}

// QueryContext 使用上下文查询多行数据
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - query: string SQL 语句，使用当前 PlaceholderParser 的占位符（默认 ?）
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Rows: 结果集，需由调用方关闭
//   - error: 查询错误
//
// 说明:
//   - 执行前通过当前实例的 PlaceholderParser 转换 SQL，并保存供 GetSQL 获取
//   - 有事务时使用当前事务，否则使用连接池
func (engine *Engine) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	query = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.QueryContext(ctx, query, args...)
	}
	return engine.DB.QueryContext(ctx, query, args...)
}

// TransactionFunc 在指定引擎的事务中执行操作
type TransactionFunc func(engine *Engine, args ...any) error

// WithTransaction 在事务中执行回调
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - transactionFunc: TransactionFunc 事务回调
//   - args: ...any 传给回调的参数
//
// 返回:
//   - error: 开启事务、回调或提交时的错误
//
// 说明:
//   - 使用实例副本，原实例的事务状态不变
//   - 嵌套调用复用当前事务，不创建保存点；内层错误需传到最外层才会触发回滚
//   - 最外层回调成功后提交；返回错误或 panic 时回滚，panic 继续向外传播
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

// Migrate 根据模型创建表
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - model: Model 待迁移的模型
//
// 说明:
//   - 调用 SetModel 设置模型并重置查询状态；Schema 取 Settings.Schema，默认 public
//   - 表已存在时直接返回，不修改表、不创建 Schema，也不执行迁移回调
//   - 列定义使用当前 ColumnParser；新建表前后执行对应回调
//   - 迁移或回调失败时记录日志并 panic
func (engine *Engine) Migrate(ctx context.Context, model Model) {
	var err error
	engine.SetModel(model)
	settings := engine.model.ModelSet()
	schema := settings.Schema
	placeholder := engine.placeholderParser.Placeholder()

	// schema 为空则使用 public
	if schema == "" {
		schema = "public"
	}

	query := fmt.Sprintf("SELECT 1 FROM information_schema.tables WHERE table_schema = %s AND table_name = %s;", placeholder, placeholder)
	row := engine.QueryRowContext(ctx, query, schema, settings.TableName)

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
		columns = append(columns, fmt.Sprintf("  %s %s", engine.identifierQuoter.QuoteIdentifier(columnName), columnType))
	}

	columnsSQL := strings.Join(columns, ",\n")
	createSQL := fmt.Sprintf("CREATE TABLE %s (\n%s\n)", engine.tableName, columnsSQL)

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

// isSetModel 检查模型是否已设置，未设置时 panic
func (engine *Engine) isSetModel() {
	if engine.model == nil {
		notSetModelErrorMsg := i18n.T("db.not_SetModel_error")
		panic(notSetModelErrorMsg)
	}
}

// SetModel 设置当前操作的模型
//
// 参数:
//   - model: Model 结构体模型或非 nil 结构体指针
//
// 返回:
//   - *Engine: 当前实例，可继续链式调用
//
// 说明:
//   - 模型无效时 panic；表名取 ModelSet，Schema 默认 public
//   - 重置字段选择、条件、参数、分组、排序和分页，不重置连接或事务
//   - 通过当前 ColumnParser 获取字段；默认读取 field_type、field_name 标签
func (engine *Engine) SetModel(model Model) *Engine {
	modelValue, err := db.ModelStructValue(model)
	if err != nil {
		panic(err)
	}
	engine.model = model
	settings := model.ModelSet()
	schema := settings.Schema
	if schema == "" {
		schema = "public"
	}
	engine.tableName = engine.identifierQuoter.QuoteIdentifier(schema) + "." + engine.identifierQuoter.QuoteIdentifier(settings.TableName)
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

// Fields 选择查询、插入或更新的字段
//
// 参数:
//   - fields: ...string 模型结构体字段名，区分大小写
//
// 返回:
//   - *Engine: 当前实例的副本，可继续链式调用
//
// 说明:
//   - 先调用 SetModel 设置模型
//   - 未调用时使用全部映射字段；多次调用覆盖选择，空参数清空选择
//   - 不存在的字段会记录日志并 panic；ColumnParser 不接受的字段会跳过
//   - Insert/Update 在选中字段内继续按 skipNil 处理 nil
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
//   - query: string 条件语句，使用当前 PlaceholderParser 的占位符，默认 ?
//   - args: ...any 与占位符一一对应的参数
//
// 返回:
//   - *Engine: 条件独立的实例副本，支持链式调用
//
// 说明:
//   - 多次调用以 AND 连接；占位符与参数数量不一致时 panic
//   - IN ?、NOT IN ? 对应的切片或数组会展开，支持集合指针；[]byte 不展开
//   - IN (?) 等带括号的写法须手动提供占位符和参数；空集合展开为 ()，由数据库处理
//   - 仅按字符串匹配占位符，不解析 SQL 字符串或注释；不匹配 IN 规则的参数不展开
func (engine Engine) Where(query string, args ...any) *Engine {
	// 值接收者只会复制切片头，whereSQL 与 args 仍可能和原 Engine 共享底层数组。
	// 当切片容量大于长度时，append 会复用共享数组并覆盖其他查询分支追加的数据。
	// 因此在追加新条件前复制这两个切片，使当前查询分支拥有独立的底层数组。
	engine.whereSQL = append([]string(nil), engine.whereSQL...)
	engine.args = append([]any(nil), engine.args...)
	placeholder := engine.placeholderParser.Placeholder()
	queryParts := strings.Split(query, placeholder)
	if len(queryParts)-1 != len(args) {
		whereArgsPlaceholderErrorMsg := i18n.T("db.where_args_placeholder_error")
		goi.Log.Error(whereArgsPlaceholderErrorMsg)
		panic(whereArgsPlaceholderErrorMsg)
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(queryParts[0])
	for i, param := range args {
		inPlaceholder, inArgs, ok := whereIn(queryParts[i], placeholder, param)
		if ok {
			queryBuilder.WriteString(inPlaceholder)
			engine.args = append(engine.args, inArgs...)
		} else {
			queryBuilder.WriteString(placeholder)
			engine.args = append(engine.args, param)
		}
		queryBuilder.WriteString(queryParts[i+1])
	}
	engine.whereSQL = append(engine.whereSQL, queryBuilder.String())
	return &engine
}

// GroupBy 设置分组字段
//
// 参数:
//   - groups: ...string 数据库列名
//
// 返回:
//   - *Engine: 当前实例的副本，可继续链式调用
//
// 说明:
//   - 字段由当前 IdentifierQuoter 引用，默认使用双引号
//   - 多次调用覆盖原设置；默认引用器会忽略空字段，空参数清除分组
func (engine Engine) GroupBy(groups ...string) *Engine {
	groupFields := engine.identifierQuoter.QuoteIdentifiers(groups)
	if groupFields != "" {
		engine.groupSQL = " GROUP BY " + groupFields
	} else {
		engine.groupSQL = ""
	}
	return &engine
}

// OrderBy 设置排序字段
//
// 参数:
//   - orders: ...string 数据库列名，默认升序；前缀 "-" 表示降序，例如 "-created_at"
//
// 返回:
//   - *Engine: 当前实例的副本，可继续链式调用
//
// 说明:
//   - 字段去除首尾空格后交给当前 IdentifierQuoter 引用，默认使用双引号
//   - 多次调用覆盖原设置；空字段会跳过，空参数清除排序
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

// Page 设置分页并统计总数
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - page: int64 页码，从 1 开始；小于等于 0 时使用 1
//   - pageSize: int64 每页条数；小于等于 0 时使用 10
//
// 返回:
//   - int64: 总记录数
//   - int64: 总页数
//   - error: 统计错误
//
// 说明:
//   - 先调用 SetModel；分页修改当前实例，使用 LIMIT pageSize OFFSET offset
//   - 调用 Count 按当前 WHERE 条件统计，不按分组、排序或分页统计
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

// Select 查询并将结果追加到切片
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - queryResult: any 非 nil 切片指针，元素支持结构体、结构体指针或 map
//
// 返回:
//   - error: 目标无效、查询或扫描错误
//
// 说明:
//   - 先调用 SetModel；Fields 控制查询列，分组、排序和分页沿用当前设置
//   - 切片可以为 nil，但不支持数组；无记录时返回 nil，已有切片内容不清空
//   - 结构体按同名字段写入；map 按模型字段类型扫描，以结构体字段名为键，调用方需保证类型可写入
//   - 自动关闭结果集，并返回迭代错误
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

// First 查询并写入第一条记录
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//   - queryResult: any 非 nil 结构体指针或已初始化的 map
//
// 返回:
//   - error: 目标无效、查询或扫描错误；无记录时为 sql.ErrNoRows
//
// 说明:
//   - 先调用 SetModel；Fields 控制查询列，分组、排序和分页沿用当前设置
//   - 结构体按同名字段写入；map 按模型字段类型扫描，以结构体字段名为键，调用方需保证类型可写入
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

// Count 统计符合条件的记录数
//
// 参数:
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//
// 返回:
//   - int64: 记录数，出错时为 0
//   - error: 查询错误
//
// 说明:
//   - 先调用 SetModel；仅使用当前 WHERE 条件，忽略分组、排序和分页
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
//   - ctx: context.Context 操作上下文，用于取消和超时控制
//
// 返回:
//   - bool: 是否至少存在一条记录
//   - error: 查询错误
//
// 说明:
//   - 先调用 SetModel；仅使用当前 WHERE 条件
//   - 无记录时返回 false, nil，其他错误原样返回
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
//   - sql.Result: 删除结果
//   - error: 删除错误
//
// 说明:
//   - 先调用 SetModel；根据当前 Where 条件删除
//   - 未设置 Where 条件时删除整张表的数据，调用方需明确操作范围
func (engine *Engine) Delete(ctx context.Context) (sql.Result, error) {
	engine.isSetModel()

	query := fmt.Sprintf("DELETE FROM %s", engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(engine.whereSQL, " AND "))
	}
	return engine.ExecuteContext(ctx, query, engine.args...)
}
