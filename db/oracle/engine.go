package oracle

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

// Engine 结构体用于管理 Oracle 数据库连接和操作
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

// Name 获取数据库配置名称
//
// 返回:
//   - string: 创建实例时指定的配置名称
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
//   - query: string SQL，使用当前 PlaceholderParser 的占位符（默认 ?）
//   - args: ...any SQL 参数
//
// 返回:
//   - sql.Result: 执行结果
//   - error: 执行错误
//
// 说明:
//   - 执行前解析占位符并保存 SQL；存在事务时使用当前事务执行
func (engine *Engine) ExecuteContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query, args = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.ExecContext(ctx, query, args...)
	}
	return engine.DB.ExecContext(ctx, query, args...)
}

// QueryRow 查询一行数据
//
// 参数:
//   - query: string SQL 查询语句
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Row: 单行结果
//
// 说明:
//   - 使用 context.Background() 调用 QueryRowContext
func (engine *Engine) QueryRow(query string, args ...any) *sql.Row {
	return engine.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext 使用上下文查询一行数据
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - query: string SQL，使用当前 PlaceholderParser 的占位符（默认 ?）
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Row: 单行结果，查询错误通过 Scan 或 Err 获取
//
// 说明:
//   - 查询前解析占位符并保存 SQL；存在事务时使用当前事务查询
//   - 无记录时 Scan 返回 sql.ErrNoRows
func (engine *Engine) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	query, args = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.QueryRowContext(ctx, query, args...)
	}
	return engine.DB.QueryRowContext(ctx, query, args...)
}

// Query 查询多行数据
//
// 参数:
//   - query: string SQL 查询语句
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Rows: 结果集，使用后须关闭
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
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - query: string SQL，使用当前 PlaceholderParser 的占位符（默认 ?）
//   - args: ...any SQL 参数
//
// 返回:
//   - *sql.Rows: 结果集，使用后须关闭并检查 Err
//   - error: 查询错误
//
// 说明:
//   - 查询前解析占位符并保存 SQL；存在事务时使用当前事务查询
func (engine *Engine) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	query, args = engine.placeholderParser.Parse(query, args...)
	engine.sql = query
	if engine.tx != nil {
		return engine.tx.QueryContext(ctx, query, args...)
	}
	return engine.DB.QueryContext(ctx, query, args...)
}

// 事务执行函数
type TransactionFunc func(engine *Engine, args ...any) error

// WithTransaction 在事务中执行回调
//
// 参数:
//   - ctx: context.Context 事务上下文，用于取消和超时控制
//   - transactionFunc: TransactionFunc 事务回调
//   - args: ...any 传给回调的参数
//
// 返回:
//   - error: 回调、开启或提交事务的错误
//
// 说明:
//   - 回调使用 Engine 副本，不改变原实例的事务状态
//   - 最外层回调成功后提交；返回错误或发生 panic 时回滚，panic 继续向上传播
//   - 嵌套调用复用当前事务，不创建保存点，也不单独提交或回滚；内层错误须由外层回调返回
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

// Migrate 根据模型创建不存在的表
//
// 参数:
//   - ctx: context.Context 迁移上下文，用于取消和超时控制
//   - model: Model 数据模型
//
// 说明:
//   - 会调用 SetModel 并重置查询状态，列定义由 ColumnParser 解析
//   - Schema 取自 Settings.Schema；为空时使用当前 Schema，名称保留大小写
//   - 表存在时直接返回，不修改现有表、不创建 Schema，也不执行回调
//   - 建表时执行迁移前后回调；迁移或回调失败时记录日志并 panic
func (engine *Engine) Migrate(ctx context.Context, model Model) {
	var err error
	engine.SetModel(model)
	settings := engine.model.ModelSet()
	schema := settings.Schema
	placeholder := engine.placeholderParser.Placeholder()

	// 引号标识符保留大小写；空 Schema 使用当前会话的 CURRENT_SCHEMA。
	var row *sql.Row
	if schema == "" {
		query := fmt.Sprintf("SELECT COUNT(1) FROM ALL_TABLES WHERE OWNER = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') AND TABLE_NAME = %s", placeholder)
		row = engine.QueryRowContext(ctx, query, settings.TableName)
	} else {
		query := fmt.Sprintf("SELECT COUNT(1) FROM ALL_TABLES WHERE OWNER = %s AND TABLE_NAME = %s", placeholder, placeholder)
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

	migrationModelMsg := i18n.T("db.migration.oracle", map[string]any{
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
//   - model: Model 结构体或非 nil 结构体指针
//
// 返回:
//   - *Engine: 当前实例，支持链式调用
//
// 说明:
//   - 模型无效时 panic；重置字段、条件、分组、排序、分页和参数
//   - 缓存完整表名；Schema 为空时使用当前 Schema，名称保留大小写
//   - 字段由 ColumnParser 解析；默认要求有效 field_type 标签，field_name 缺省时使用小写字段名
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

// Fields 选择查询、插入或更新使用的字段
//
// 参数:
//   - fields: ...string 模型结构体字段名，区分大小写
//
// 返回:
//   - *Engine: 当前实例的副本，支持链式调用
//
// 说明:
//   - 先调用 SetModel；字段不存在时 panic，ColumnParser 排除的字段会被跳过
//   - 未调用时使用全部映射字段；再次调用覆盖选择，空列表清空选择
//   - 本方法只选择字段；Insert/Update 再按 skipNil 处理 nil
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
//   - 默认 PlaceholderParser 支持 IN ?、NOT IN ? 的集合参数；Where 也会在构建条件时展开
//   - 字节切片（如 []byte）和 nil 指针不展开；其它未匹配的参数交由 database/sql 和驱动处理
//   - IN (?) 等带括号的写法不自动展开，需自行匹配占位符与参数
//   - 空集合（含 nil 切片）展开为 ()，是否合法由数据库处理
//   - 仅按字符串匹配占位符，不解析 SQL 字符串或注释
//   - 执行阶段的 SQL 和参数转换由当前 PlaceholderParser.Parse 决定
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
		inPlaceholder, inArgs, ok := expandInParameter(queryParts[i], placeholder, param)
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
//   - *Engine: 当前实例的副本，支持链式调用
//
// 说明:
//   - 列名由当前 IdentifierQuoter 处理，忽略首尾空格和空字段
//   - 再次调用覆盖原设置，空列表清空分组
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
//   - orders: ...string 数据库列名；默认升序，前缀 - 表示降序，如 -created_at
//
// 返回:
//   - *Engine: 当前实例的副本，支持链式调用
//
// 说明:
//   - 列名由当前 IdentifierQuoter 处理，忽略首尾空格和空字段
//   - 再次调用覆盖原设置，空列表清空排序
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
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - page: int64 页码，小于 1 时按 1 处理
//   - pageSize: int64 每页条数，小于 1 时按 10 处理
//
// 返回:
//   - int64: 总记录数
//   - int64: 总页数
//   - error: 统计错误
//
// 说明:
//   - 先调用 SetModel；修改当前分页设置，并执行一次 Count 查询
//   - 使用 OFFSET/FETCH 分页，要求 Oracle 12c 及以上
func (engine *Engine) Page(ctx context.Context, page int64, pageSize int64) (int64, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10 // 默认分页大小
	}
	offset := (page - 1) * pageSize
	engine.limitSQL = fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, pageSize)
	total, err := engine.Count(ctx)
	totalPages := int64(math.Ceil(float64(total) / float64(pageSize)))
	return total, totalPages, err
}

// Select 查询多行并追加到切片
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - queryResult: any 非 nil 的切片指针，元素支持 struct、*struct 或 map[K]V
//
// 返回:
//   - error: 查询、扫描或结果类型错误
//
// 说明:
//   - 先调用 SetModel；使用当前字段、条件、分组、排序和分页设置
//   - 切片自身可以为 nil，不支持数组；新记录追加到原切片，无记录时返回 nil
//   - 结构体按字段名写入；map 按模型字段类型扫描，以结构体字段名为键，调用方须保证键值类型兼容
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

// First 查询第一条记录
//
// 参数:
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//   - queryResult: any 非 nil 的结构体指针或已初始化的 map[K]V
//
// 返回:
//   - error: 查询、扫描或结果类型错误；无记录时返回 sql.ErrNoRows
//
// 说明:
//   - 先调用 SetModel；使用当前字段、条件、分组、排序和分页设置
//   - 结构体按字段名写入；map 按模型字段类型扫描，以结构体字段名为键，调用方须保证键值类型兼容
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
//   - ctx: context.Context 查询上下文，用于取消和超时控制
//
// 返回:
//   - int64: 记录数，查询失败时为 0
//   - error: 查询错误
//
// 说明:
//   - 先调用 SetModel；仅使用 Where 条件，不受字段选择、分组、排序和分页影响
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
//   - bool: 是否存在记录，无记录时为 false
//   - error: 查询错误，无记录时为 nil
//
// 说明:
//   - 先调用 SetModel；仅使用 Where 条件
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
//   - 先调用 SetModel；未设置 Where 条件时删除整张表的数据，调用方需明确操作范围
func (engine *Engine) Delete(ctx context.Context) (sql.Result, error) {
	engine.isSetModel()

	query := fmt.Sprintf("DELETE FROM %s", engine.tableName)
	if len(engine.whereSQL) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(engine.whereSQL, " AND "))
	}
	return engine.ExecuteContext(ctx, query, engine.args...)
}
