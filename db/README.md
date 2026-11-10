# 自定义 ORM 开发指南

`db` 是数据库扩展的公共基础包，面向需要实现自定义 ORM、接入数据库方言或复用模型反射与结果扫描能力的开发者。

`db` 不统一规定模型接口、查询构造器、迁移或事务 API。自定义引擎只需实现 `db.Engine` 并注册工厂，就可以通过 `db.Connect` 接入 Goi；其余能力由引擎自行提供。本文以 MySQL 方言为例，说明从实现、注册、配置、执行到关闭的完整流程。

## 一、分层与文件职责

数据库接入包含三个相互独立的层次：

| 层次 | 负责的内容 | 注册或配置入口 |
| --- | --- | --- |
| SQL 驱动 | 实现 `database/sql` 与数据库之间的通信 | 驱动包通过 `sql.Register` 注册，应用通常使用空白导入加载 |
| 连接管理 | 创建、缓存和关闭 `*sql.DB` 连接池 | `goi.Database`、`goi.Settings.Databases` |
| ORM 引擎 | 模型映射、SQL 构建、方言处理和结果写入 | `db.Engine`、`db.Register` |

导入 SQL 驱动不会自动注册 ORM；注册 ORM 也不会自动安装或导入 SQL 驱动。

公共包的各文件可以按需复用，不要求自定义 ORM 使用全部工具：

| 文件 | 对外能力 | 在自定义 ORM 中的用途 |
| --- | --- | --- |
| [engine.go](engine.go) | `Engine`、`TransactionFunc` | 定义 ORM 基础操作接口和事务回调类型 |
| [connect.go](connect.go) | `DBProvider`、`ConnectFunc`、`Register`、`GetConnectFunc`、`Connect`、`ConnectDatabase` | 定义连接池提供者、注册工厂，并根据配置创建具体引擎 |
| [options.go](options.go) | `EngineOptions`、`Merge` | 按实例覆盖标识符、占位符和列解析策略 |
| [model.go](model.go) | `ModelStructValue`、`ValidateOperationModel` | 验证模型结构、取得结构体反射值、检查写入模型类型 |
| [column.go](column.go) | `ColumnParser`、`DefaultColumnParser`、`NewColumnParser`、`SetColumnParser`、`GetColumnParser` | 解析模型列定义，并提供公共默认解析器 |
| [field.go](field.go) | `ModelFields`、`ModelFieldValue` | 提取有序字段列表，读取可交给 SQL 执行入口的模型字段值 |
| [identifier.go](identifier.go) | `IdentifierQuoter`、`DefaultIdentifierQuoter`、`NewIdentifierQuoter` | 按数据库方言引用表名、Schema 名和列名 |
| [placeholder.go](placeholder.go) | `PlaceholderParser`、`DefaultPlaceholderParser`、`NewPlaceholderParser` | 约定 ORM 占位符，并在执行前转换 SQL 和参数 |
| [scanner.go](scanner.go) | `RowScanner`、`ResultWriter`、`PrepareSliceResult`、`PrepareSingleResult` | 校验查询目标，创建逐行扫描与写入回调 |

## 二、引擎接口与注册约定

### 1. 实现最小接口

自定义引擎必须满足以下接口：

```go
type Engine interface {
	Name() string
	Execute(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}
```

`Name` 返回应用的连接配置名称，例如 `"default"`，不是数据库驱动名称。其余三个方法分别负责执行语句、单行查询和多行查询。

`db.Engine` 不要求上下文、事务或 CRUD 方法。引擎可以在保留上述基础方法的同时增加 `ExecuteContext`、`QueryRowContext`、`QueryContext`、`SetModel`、`Select` 等 API。业务代码通过泛型连接入口取得具体类型后，即可调用这些扩展方法。

`db.TransactionFunc` 只是基于 `db.Engine` 的回调类型，不会自动开启或提交事务；具体事务入口和语义仍由引擎实现。

### 2. 注册工厂，而不是注册连接池

工厂签名为：

```go
type ConnectFunc func(name string, database *goi.Database) db.Engine
```

在自定义引擎包的 `init` 中调用 `db.Register("引擎名称", factory)`，应用导入该包后就会完成注册。也可以显式注册，但同一个名称只能注册一次。

- `name`：本次使用的连接配置名称。
- `database`：应用配置的连接管理器；工厂通常通过 `database.DB()` 取得共享连接池。
- 返回值：本次操作使用的新 ORM 实例，而不是全局共享的查询构造对象。

`Register` 在工厂为 nil 或名称重复时 panic。注册表有锁保护，但这不意味着引擎对象、配置 map 或自定义策略自动具备并发安全性。

`GetConnectFunc(name)` 只查找已注册工厂，返回 `(ConnectFunc, bool)`；它不创建引擎，也不建立连接。通常直接使用 `Connect` 或 `ConnectDatabase` 即可。

如果自己的 `ConnectDatabase` 返回 `*Engine`，需要额外的 `factory` 包装成返回 `db.Engine` 的函数：Go 不会把返回具体类型的函数自动转换成返回接口的函数。

内置实现可参考 [mysql/connect.go](mysql/connect.go)，其中集中定义注册、工厂、`Connect` 和 `ConnectDatabase`。

### 3. 区分三种名称

下面的示例使用：

| 名称 | 示例值 | 决定的事情 |
| --- | --- | --- |
| 连接配置名称 | `"default"` | `Settings.Databases` 的键，以及 `Engine.Name()` 的返回值 |
| ORM 注册名称 | `"custom_mysql"` | `db.Register` 的键，必须与 `goi.Database.Engine` 一致 |
| SQL 驱动名称 | `"mysql"` | 传给 `sql.Open`，必须与实际驱动注册名称一致 |

内置 MySQL 的 ORM 注册名称和 SQL 驱动名称恰好都是 `"mysql"`，但自定义 ORM 不必让二者相同。`goi.Database.Connect` 接收的是 `Database.Engine`，因此使用不同名称时，应在连接回调内明确选择实际驱动。

### 4. 连接提供者与实例策略

`db.DBProvider` 只要求提供 `DB() *sql.DB`，`*goi.Database` 已实现它。MySQL 等五个原生方言包的低层 `ConnectDatabase(name, provider, options...)` 可以复用已有连接池，不检查 Engine 名称，也不读取 Config；调用方负责保证 SQL 方言兼容，并显式传入实例选项。

`db.EngineOptions` 包含三个可选策略：

```go
type EngineOptions struct {
	PlaceholderParser db.PlaceholderParser
	IdentifierQuoter  db.IdentifierQuoter
	ColumnParser      db.ColumnParser
}
```

`options.Merge(overrides...)` 返回副本，按顺序覆盖非 nil 字段；nil 项保留原值。方言包先读取有效默认策略，再合并传入选项，保存到新 Engine，不修改默认配置。例如：

```go
options := db.EngineOptions{ColumnParser: db.NewColumnParser("field_name", "field_type")}
orm := mysql.ConnectDatabase("report", database, options,
	db.EngineOptions{IdentifierQuoter: db.NewIdentifierQuoter("`", "`")},
)
```

以 MySQL 为例，各连接入口的职责如下；其它四个原生方言包遵循同样规则：

| 入口 | 连接来源 | 工厂与配置读取 |
| --- | --- | --- |
| `mysql.Connect(name)` | `goi.Settings.Databases[name]` | 检查 Engine 为 mysql，再调用本包 factory 读取 Config |
| `db.Connect[*mysql.Engine](name)` | `goi.Settings.Databases[name]` | 按 Engine 查找已注册工厂，由工厂读取 Config |
| `db.ConnectDatabase[*mysql.Engine](name, database)` | 显式传入的 `*goi.Database` | 按 Engine 查找已注册工厂，由工厂读取 Config；无需写入全局配置 |
| `mysql.ConnectDatabase(name, provider, options...)` | 只需实现 `db.DBProvider` 的对象 | 不查注册表、不读取 Config，只合并显式 options |

原生方言包把 `GetConfig` 集中在 factory 中，每次创建 Engine 只读取一次。`Connect` 校验连接名称和 Engine 后复用 factory，不再增加配置读取辅助层。泛型入口会校验工厂返回值是否符合指定的 Engine 类型，不会根据泛型参数自动切换方言。

Kingbase 和 SeaBoxSQL 是模式路由入口，不属于上表的五个原生方言包。它们的 `ConnectDatabase(name, database)` 接收 `*goi.Database`，读取各自的 Config，再调用所选方言包的低层构造方法，详见文末对应说明。

### 5. 内置方言包的连接级配置

五个原生方言包都提供实现 `goi.DatabaseConfig` 的 Config，值和指针均可赋给 `goi.Database.Config`，也可使用其它能向对应目标写入配置的实现：

| 配置类型 | GetConfig 的写入目标 |
| --- | --- |
| `mysql.Config` | `*mysql.Config` |
| `postgres.Config` | `*postgres.Config` |
| `oracle.Config` | `*oracle.Config` |
| `sqlserver.Config` | `*sqlserver.Config` |
| `sqlite3.Config` | `*sqlite3.Config` |

这些 Config 都只有显式字段 `EngineOptions db.EngineOptions`，不使用匿名嵌入：

```go
type Config struct {
	EngineOptions db.EngineOptions
}
```

`GetConfig` 接收非 nil 的本包 `*Config`，按值复制配置，策略对象仍使用原引用；目标无效时返回错误。nil 配置或 nil 配置指针保留默认策略；factory 读取失败时会在获取连接池前原样 panic。配置类型应与使用的 ORM 包对应，不能把 `postgres.Config` 当作 `mysql.Config` 使用。

例如，已导入 MySQL SQL 驱动并准备好 `dsn` 后，为一个连接指定列标签：

```go
goi.Settings.Databases["report"] = &goi.Database{
	Engine: "mysql",
	Config: mysql.Config{
		EngineOptions: db.EngineOptions{
			ColumnParser: db.NewColumnParser("column", "sql_type"),
		},
	},
	Connect: func(_ string) *sql.DB {
		pool, err := sql.Open("mysql", dsn)
		if err != nil {
			panic(err)
		}
		return pool
	},
}
orm := db.Connect[*mysql.Engine]("report")
```

该连接的模型使用 `column`、`sql_type` 标签；未指定的其它策略沿用默认值。最后一行也可使用 `mysql.Connect("report")`，两种入口都由 factory 读取 Config。直接调用 `mysql.ConnectDatabase` 则仍需显式传入 `EngineOptions`，即使提供者是带 Config 的 `*goi.Database`。

## 三、用 MySQL 方言实现一个最小 ORM

下面创建应用模块 `example.com/dbdemo`，在其中编写 `custommysql` 包。它演示注册、策略保存、基础 SQL 执行和多行模型查询；文件可以直接组合使用。为便于阅读，最小示例将连接入口和操作放在同一个文件，完整 ORM 可按后文职责拆分。

### 1. 实现引擎与基础 SQL 方法

文件：`custommysql/engine.go`。

```go
package custommysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
)

const EngineName = "custom_mysql"

// 默认策略只在配置阶段设置；每个 Engine 创建时保存自己的策略对象。
var (
	identifierQuoter  db.IdentifierQuoter  = db.NewIdentifierQuoter("`", "`")
	placeholderParser db.PlaceholderParser = db.NewPlaceholderParser("?")
)

type Engine struct {
	name              string
	DB                *sql.DB
	identifierQuoter  db.IdentifierQuoter
	placeholderParser db.PlaceholderParser
	columnParser      db.ColumnParser
}

type Config struct {
	EngineOptions db.EngineOptions
}

func (config Config) GetConfig(target any) error {
	value, ok := target.(*Config)
	if !ok || value == nil {
		return fmt.Errorf("custommysql: target must be a non-nil *Config, got %T", target)
	}
	*value = config
	return nil
}

var _ db.Engine = (*Engine)(nil)
var _ goi.DatabaseConfig = Config{}

func init() {
	db.Register(EngineName, factory)
}

func factory(name string, database *goi.Database) db.Engine {
	var config Config
	if err := database.GetConfig(&config); err != nil {
		panic(err)
	}
	return ConnectDatabase(name, database, config.EngineOptions)
}

func ConnectDatabase(name string, database db.DBProvider, options ...db.EngineOptions) *Engine {
	strategies := (db.EngineOptions{
		IdentifierQuoter:  identifierQuoter,
		PlaceholderParser: placeholderParser,
		ColumnParser:      db.GetColumnParser(),
	}).Merge(options...)
	return &Engine{
		name:              name,
		DB:                database.DB(),
		identifierQuoter:  strategies.IdentifierQuoter,
		placeholderParser: strategies.PlaceholderParser,
		columnParser:      strategies.ColumnParser,
	}
}

func Connect(name string) *Engine {
	database, ok := goi.Settings.Databases[name]
	if !ok {
		panic(fmt.Errorf("database configuration %q does not exist", name))
	}
	if database.Engine != EngineName {
		panic(fmt.Errorf("expected ORM %q, got %q", EngineName, database.Engine))
	}
	return factory(name, database).(*Engine)
}

func (engine *Engine) Name() string {
	return engine.name
}

func (engine *Engine) Execute(query string, args ...any) (sql.Result, error) {
	return engine.ExecuteContext(context.Background(), query, args...)
}

func (engine *Engine) ExecuteContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query, args = engine.placeholderParser.Parse(query, args...)
	return engine.DB.ExecContext(ctx, query, args...)
}

func (engine *Engine) QueryRow(query string, args ...any) *sql.Row {
	return engine.QueryRowContext(context.Background(), query, args...)
}

func (engine *Engine) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	query, args = engine.placeholderParser.Parse(query, args...)
	return engine.DB.QueryRowContext(ctx, query, args...)
}

func (engine *Engine) Query(query string, args ...any) (*sql.Rows, error) {
	return engine.QueryContext(context.Background(), query, args...)
}

func (engine *Engine) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	query, args = engine.placeholderParser.Parse(query, args...)
	return engine.DB.QueryContext(ctx, query, args...)
}
```

工厂不直接调用 `sql.Open`，连接创建统一交给 `goi.Database`；这样不同 ORM 实例可以复用同一个连接池。基础方法统一转入上下文方法，所有最终执行入口都使用 `PlaceholderParser.Parse` 返回的 SQL 和参数，避免某一类查询遗漏方言转换。

需要对外开放策略配置时，可以参考 [mysql/identifier.go](mysql/identifier.go)、[mysql/placeholder.go](mysql/placeholder.go)、[mysql/column.go](mysql/column.go) 提供 Set/Get 方法。配置应在创建引擎前完成，不应与连接或查询并发修改。

#### 配置接口与策略覆盖

`goi.Database.Config` 的类型为 `goi.DatabaseConfig`，默认为 nil；配置对象须实现 `GetConfig(target any) error`，不能直接赋入未实现该方法的普通结构体。`Database.GetConfig(target)` 委托配置对象写入目标；配置为 nil 或 nil 配置指针时，不修改目标已有的默认值。`Database.DB()` 不参与配置读取。

上面的 Config 实现以下接口，由配置对象校验目标类型并写入数据：

```go
type DatabaseConfig interface {
	GetConfig(target any) error
}
```

`DBProvider` 只提供 `DB()`，没有配置读取能力。因此 factory 先调用 `database.GetConfig(&config)`，再把 `config.EngineOptions` 传给低层 `ConnectDatabase`。`Connect` 直接复用本包 factory；该 factory 的正常返回值固定为本包 `*Engine`。

应用在创建连接或 ORM 实例前设置配置；不设置 Config 时沿用默认规则：

```go
database.Config = custommysql.Config{
	EngineOptions: db.EngineOptions{
		ColumnParser: db.NewColumnParser("field_name", "field_type"),
	},
}
```

`target` 使用 any，但目标类型和写入规则由具体配置实现决定。本例只接受非 nil 的 `*Config`，读取错误会在获取连接池前 panic；额外约束也应由实现校验并返回错误。通过通用连接入口会调用注册工厂；直接调用此示例的 `custommysql.ConnectDatabase` 时，应显式传入 `EngineOptions`。

配置应在连接或创建 Engine 前完成，不要在运行期间修改共享对象。核心不反射复制配置字段；本例只复制 Config 结构体，不深复制其中的接口、指针或 map，构造函数保存的策略引用仍由实现者管理。只有调用了 `GetConfig` 的 ORM 才会使用该配置。

### 2. 复用公共工具完成模型查询

继续添加到同一个 `custommysql/engine.go`：

```go
func (engine *Engine) Select(
	ctx context.Context,
	model any,
	schema string,
	table string,
	queryResult any,
) error {
	modelValue, err := db.ModelStructValue(model)
	if err != nil {
		return err
	}
	fields, columns := db.ModelFields(modelValue, engine.columnParser)

	// 在发出 SQL 之前校验目标，并只准备一次逐行写入回调。
	writeRow, err := db.PrepareSliceResult(model, queryResult, fields)
	if err != nil {
		return err
	}

	// Schema 和表名分别引用，不能把 schema.table 当作单个标识符。
	tableName := engine.identifierQuoter.QuoteIdentifier(table)
	if schema != "" {
		tableName = engine.identifierQuoter.QuoteIdentifier(schema) + "." + tableName
	}
	query := fmt.Sprintf(
		"SELECT %s FROM %s",
		engine.identifierQuoter.QuoteIdentifiers(columns),
		tableName,
	)

	rows, err := engine.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := writeRow(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}
```

这里把 Schema 和表名作为方法参数，只是为了集中展示公共工具的衔接。`db` 不规定这些配置应放在哪里；完整 ORM 可以像 [mysql/model.go](mysql/model.go) 一样定义自己的模型接口和 `Settings`，再在 `SetModel` 中解析并保存表名、字段列表等信息。

模型、表名、列类型和查询结构属于开发者控制的元数据；查询值应通过参数传递，不应把不可信输入直接拼入 SQL。

### 3. 应用加载驱动并配置连接

在应用模块中添加 MySQL SQL 驱动依赖：

```sh
go get github.com/go-sql-driver/mysql
```

文件：`main.go`。将 `example.com/dbdemo/custommysql` 替换成自己的模块路径。

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"example.com/dbdemo/custommysql"
	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID   int64  `field_name:"id" field_type:"BIGINT"`
	Name string `field_name:"name" field_type:"VARCHAR(64)"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		return fmt.Errorf("MYSQL_DSN is required")
	}

	database := &goi.Database{
		Engine: custommysql.EngineName,
		Connect: func(_ string) *sql.DB {
			// 此处回调参数为 ORM 名称 custom_mysql，实际 SQL 驱动仍是 mysql。
			pool, err := sql.Open("mysql", dsn)
			if err != nil {
				panic(err)
			}
			pool.SetMaxOpenConns(10)
			pool.SetMaxIdleConns(5)
			pool.SetConnMaxLifetime(time.Hour)
			return pool
		},
	}
	goi.Settings.Databases["default"] = database

	// 独立程序负责关闭连接管理器；不是每次查询后关闭共享连接池。
	defer func() {
		if err := database.Close(); err != nil {
			log.Print(err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orm := db.Connect[*custommysql.Engine]("default")
	if err := orm.DB.PingContext(ctx); err != nil {
		return err
	}

	var users []User
	if err := orm.Select(ctx, User{}, "", "users", &users); err != nil {
		return err
	}
	fmt.Println(orm.Name(), len(users))
	return nil
}
```

配置 `MYSQL_DSN` 时选择已有数据库，例如 `user:password@tcp(127.0.0.1:3306)/app?charset=utf8mb4&parseTime=true&loc=Local`。示例查询已有的 `users` 表，`ID` 和 `Name` 的标签分别对应 `id`、`name` 列。

完整调用链为：

```text
导入 SQL 驱动 ──> database/sql 注册驱动
导入 custommysql ──> init ──> db.Register("custom_mysql", factory)

db.Connect[*custommysql.Engine]("default")
  └─ 读取 goi.Settings.Databases["default"]
     └─ db.ConnectDatabase：按 Database.Engine 查找工厂
        └─ factory：GetConfig 读取一次连接配置
           └─ custommysql.ConnectDatabase：合并默认策略与连接选项
              ├─ database.DB()：首次创建、以后复用 *sql.DB 连接池
              └─ 返回新的 Engine，保存本次选定的策略引用

orm.Select(ctx, ...)：解析模型 → 准备结果回调 → 构建 SQL → 执行 → 扫描 → 关闭 Rows
```

已有 `*goi.Database` 且不需要全局配置时，使用 `db.ConnectDatabase[*custommysql.Engine]("name", database)`。它仍然根据 `Database.Engine` 查找已注册工厂，不会跳过注册流程。直接调用自定义包的 `ConnectDatabase` 则由该包自己完成构造。

`db.Connect[T]` 和 `db.ConnectDatabase[T]` 都会把工厂结果断言为泛型参数指定的类型。配置不存在、ORM 未注册或返回类型不匹配时会 panic，因此注册、配置与泛型类型需要对应。

## 四、模型解析与写入工具

### 1. 模型验证

`ModelStructValue(model)` 接受结构体或非 nil 的单层结构体指针，返回解引用后的 `reflect.Value`；nil、其它类型和多层指针返回错误。它不要求模型实现某个 ORM 专属接口，也不解析表名。

`ValidateOperationModel(configuredModel, operationModel)` 先验证两个模型，再检查解引用后的结构体类型是否完全一致，返回操作模型的结构体值。可以用它保证 `Insert`、`Update` 接收到的数据与当前设置的模型一致；它不执行数据库约束或字段内容校验。

### 2. 有序字段列表

`ModelFields(modelValue, columnParser)` 返回两个平行列表：

| 返回值 | 示例 | 用途 |
| --- | --- | --- |
| 结构体字段名 | `[]string{"ID", "Name"}` | 从 Go 模型读取值、向查询结果写入值 |
| 数据库列名 | `[]string{"id", "name"}` | 通过标识符引用器构建 SQL |

两者顺序一致。`ModelFields` 自身不处理标识符引用，构建 SQL 时应由引用器处理列名。选择字段子集或调整顺序时，应同步维护两个列表。扫描使用结构体字段名，不能把数据库列名直接当作反射字段名。

该方法按模型的顶层结构体字段调用解析器，不递归展开嵌入结构体，也不自动判断业务主键、自动时间戳或字段默认值。模型应使用可供反射读写的导出字段，忽略规则由 `ColumnParser` 决定。

### 3. 字段值与 SQL NULL

`ModelFieldValue(modelValue, fieldName)` 按 Go 字段名读取值，返回 `(any, bool)`。`bool` 仅表示字段是否存在且可读取，不表示值是否非空：

| 字段状态 | 返回值 | 内置 ORM 的写入行为 |
| --- | --- | --- |
| 不存在或不可读取 | `nil, false` | 同时跳过该列及其参数 |
| 合法的 nil 指针 | `nil, true` | `skipNil=false` 时写 NULL，`true` 时跳过 |
| 非 nil 指针 | 原指针、`true` | 交给 `database/sql` 转换，保留指针接收者的 `driver.Valuer` |
| 普通值，包括零值 | 原值、`true` | 写入对应值，不自动省略零值 |

该方法先检查反射字段是否有效、是否可转换为接口值，再把 nil 指针统一为真正的 nil 参数。它不手动解引用非 nil 指针，也不提前调用 `driver.Valuer.Value`。自定义写入逻辑可以这样复用：

```go
func operationValues(configuredModel any, operationModel any, fields []string) ([]any, error) {
	modelValue, err := db.ValidateOperationModel(configuredModel, operationModel)
	if err != nil {
		return nil, err
	}
	args := make([]any, 0, len(fields))
	for _, fieldName := range fields {
		value, ok := db.ModelFieldValue(modelValue, fieldName)
		if !ok {
			continue
		}
		args = append(args, value)
	}
	return args, nil
}
```

上面的辅助方法只收集参数；构建 SQL 时，遇到不可读取的字段还应同步跳过对应列名，保持列与参数一一对应。`ModelFieldValue` 不决定字段是否被选中、不进行字段内容校验，也不执行 SQL。

### 4. 使用 Fields 和 skipNil 控制写入

内置 ORM 的写入入口为 `Insert(ctx, model, skipNil)` 和 `Update(ctx, model, skipNil)`。`skipNil bool` 是必填参数，没有隐式默认值。

使用内置 ORM 的模型查询或 CRUD 前，先调用 `SetModel(UserModel{})`；直接执行基础 SQL 不需要设置模型。`SetModel` 原地修改当前 Engine，重新选择全部映射字段，并清空已有 WHERE、分组、排序、分页和参数。因此应先设置模型，再添加筛选条件，不要在更新或删除前意外清掉 WHERE。

`SetModel` 默认选中全部映射字段；`Fields("Name", "Status")` 使用 Go 结构体字段名重新指定查询、插入或更新的字段。`Fields` 返回副本，应接收返回值或继续链式调用。写入时先在该字段集合内取值，再应用 `skipNil`：

| 字段情况 | `skipNil=true` | `skipNil=false` |
| --- | --- | --- |
| 未被 Fields 选中 | 不写入 | 不写入 |
| 被选中且取值为 nil | 省略该列；Update 不修改原值 | 包含该列并写入 SQL NULL |
| 被选中的非 nil 值、零值 | 写入对应值 | 写入对应值 |

`skipNil` 判断的是 `ModelFieldValue` 返回的 Go 值是否等于 nil，不会提前调用 `driver.Valuer.Value`。`0`、`false`、空字符串不会被跳过；非 nil 的 `sql.Null*` 或自定义 Valuer 即使最终转换为 SQL NULL，也仍会参与写入。

例如，`orm.Fields("Name").Where("id = ?", id).Update(ctx, UserModel{Name: nil}, false)` 只把 Name 列置为 NULL；把最后一个参数改为 `true` 则跳过 Name。若跳过后没有任何可更新字段，`Update` 返回无可更新字段错误，不执行 SQL。

插入时可使用 `orm.Fields("Name", "Status").Insert(ctx, model, true)`：ID 未被选中，Name、Status 中的 nil 值也会被省略，交由数据库处理对应列的默认值、自动生成值或约束。省略列与显式传入 NULL 不等价，需根据目标选择 `Fields` 和 `skipNil`。

未调用 `Fields` 时使用全部映射字段：`skipNil=false` 会写入其中的 nil 和零值；`true` 只排除 nil，不自动识别哪些零值是“未提供”。`Fields()` 表示空集合，不是选择全部字段；空更新返回错误，空插入仍按统一规则构建 SQL。

`Validate(model, partial)` 是独立的字段校验入口，不会被 `Insert`、`Update` 自动调用。`partial=true` 只跳过 nil 指针的校验，不等于写入时的 `skipNil=true`。需要校验显式 NULL 时，不应依赖这种跳过规则；数据库约束仍由数据库检查。

### 5. 可选 WHERE 与全表操作

内置 ORM 的 `Update`、`Delete` 允许不设置 WHERE 条件。它们先构建基础 SQL，只有条件列表非空时才追加 `WHERE`：

- `orm.Fields("Status").Update(ctx, model, true)` 没有 WHERE，会更新整张表中选中的 Status 列；仍按 `skipNil` 决定是否写入 nil。
- `orm.Where("id = ?", id).Fields("Status").Update(ctx, model, false)` 只更新符合条件的记录。
- `orm.Delete(ctx)` 删除当前模型表的全部记录；`orm.Where("id = ?", id).Delete(ctx)` 只删除符合条件的记录。

框架不自动阻止无条件更新或删除，调用方应明确是否允许全表操作。无 WHERE 的更新仍遵循 `Fields`、`skipNil` 和无可更新字段时返回错误的规则，不会执行空 SET 语句。

### 6. Where 与集合参数

`Where(query, args...)` 使用当前 `PlaceholderParser.Placeholder()` 返回的 ORM 占位符，默认是 `?`。多个 `Where` 以 AND 连接；每次调用复制条件和参数切片，避免查询分支追加时相互覆盖，但不深拷贝参数对象。需接收返回值或继续链式调用。

```go
query := orm.Where("id IN ?", []int64{1, 2})
// 执行时条件展开为 id IN (?,?)，参数展开为 1、2

query = query.Where("status IN (?,?)", "enabled", "disabled")
// 已写好括号和占位符时，逐个提供参数，不自动展开集合
```

集合展开只处理占位符前匹配 IN 或 NOT IN 的切片、数组及其非 nil 指针，不限制元素类型；`[]byte`、`[]uint8` 和命名枚举切片也会展开。若要把字节切片作为单个二进制值传入，请使用 `payload = ?` 或手动带括号的 `payload IN (?)`。空切片、nil 切片和空数组生成 `IN ()`，是否合法交由数据库判断；nil 指针及其它不匹配的参数原样交给驱动。

MySQL 的 `Where` 只保存条件与参数，在执行时由默认占位符解析器的 `Parse` 展开集合；其它方言的 `Where` 仍在构建条件时展开集合。各原生方言包的默认解析器均自行实现集合展开，因此 `ExecuteContext`、`QueryContext`、`QueryRowContext` 及无 Context 的三个基础方法默认也支持集合参数。基础执行入口只调用当前解析器，并将返回的 SQL 和参数交给 `sql.DB` 或 `sql.Tx`。

MySQL 传入的集合对象不深拷贝，执行前修改集合会影响实际参数；默认解析器不会改写已保存的条件与参数，可重复执行。注册自定义 `PlaceholderParser` 后，执行阶段的 SQL 和参数转换完全由自定义实现负责，不会自动继承原默认解析器的规则；但非 MySQL 方言在 `Where` 构建时已经完成的集合展开不受此替换影响。

```go
rows, err := orm.QueryContext(ctx, "SELECT id FROM roles WHERE id IN ?", []int64{1, 2})
// 传给驱动的 SQL 为 SELECT id FROM roles WHERE id IN (?,?)，参数为 1、2
// 使用完 rows 后应关闭，并检查 rows.Err()。
```

`Where` 按占位符字符串核对参数数量，数量不一致时 panic；各方言的基础 SQL 入口不新增数量校验，交由解析器或驱动处理。默认集合展开与 Where 数量检查都不解析 SQL 字符串或注释。自定义 `Parse` 可以处理复杂 SQL，但不能修复 Where 阶段已经发生的错误计数；复杂条件可使用不冲突的 ORM 占位符。

## 五、三个可替换的策略接口

标识符与占位符默认由各方言包维护；列解析器另有 db 包级公共默认。创建 Engine 时选取最终策略并保存到实例，不在每次查询时重新读取默认值。

`SetIdentifierQuoter(nil)`、`SetPlaceholderParser(nil)` 和 `SetColumnParser(nil)` 均直接返回，保留当前配置。

### 1. ColumnParser：列定义解析

接口方法为 `Parse(field reflect.StructField) (columnName string, columnType string, ok bool)`：

- `columnName`：数据库列名，由调用方处理标识符引用；默认解析器原样返回标签中的列名。
- `columnType`：列类型及约束定义，可用于迁移和字段校验。
- `ok`：是否参与数据库操作；false 表示忽略该字段。

`db.NewColumnParser("field_name", "field_type")` 的默认规则：

- 列类型标签缺失、为空白或去除首尾空白后为 `"-"` 时忽略字段。
- 列名标签未设置时使用 Go 字段名的小写形式；不会自动转成下划线命名。
- 显式设置的列名标签按原值返回；有效列类型也保留标签原始文本。
- 不负责类型有效性校验、标识符引用或字段值校验。

如需其它标签键，可创建 `db.NewColumnParser("column", "sql_type")`；如需复合标签、命名约定或额外的忽略规则，可以自行实现 `db.ColumnParser`。同一个 Engine 内，模型选列、迁移和校验逻辑应使用一致的解析规则。

列解析器按以下优先级选择：连接配置的 `Config.EngineOptions.ColumnParser`（或低层构造入口的显式选项）→ 对应数据库包的 `SetColumnParser` → `db.SetColumnParser` / 内置规则。`db.GetColumnParser()` 初始使用 `field_name`、`field_type`；原生数据库包未设置局部解析器时，`GetColumnParser()` 回退到公共默认。db 和各原生包的 `SetColumnParser(nil)` 均直接返回，保留当前配置，不用于清除或重置。

例如，初始化时分别设置公共、MySQL 包级和单个实例的规则：

```go
db.SetColumnParser(db.NewColumnParser("column", "sql_type"))
mysql.SetColumnParser(db.NewColumnParser("mysql_column", "mysql_type"))
orm := mysql.ConnectDatabase("report", database, db.EngineOptions{
	ColumnParser: db.NewColumnParser("field_name", "field_type"),
})
```

本次创建的 `orm` 使用实例规则，后续替换默认配置不会改变它。自定义 ORM 可像上面的最小示例一样，在构造函数中读取 `db.GetColumnParser()`，再合并实例选项。

### 2. IdentifierQuoter：标识符引用

接口包含 `QuoteIdentifier(identifier string) string` 和 `QuoteIdentifiers(identifiers []string) string`。

`db.NewIdentifierQuoter(left, right)` 可以指定相同或不同的左右定界符。默认实现先去除首尾空白，再通过 `strings.TrimLeft`、`strings.TrimRight` 清理首尾定界符字符，最后添加一层左右定界符。批量方法按顺序引用并以逗号连接，跳过空白项。

需要注意：

- 它处理的是单个标识符，不是 `schema.table`、`COUNT(*)` 或包含别名的 SQL 表达式。
- 左右定界符按字符集合清理，不是完整字符串匹配；不会保留名称首尾恰好出现的定界符字符。
- 不进行定界符转义或 SQL 语法分析；特殊命名规则应自行实现该接口。
- 参数占位符不能替代表名、列名等标识符。

### 3. PlaceholderParser：占位符与执行前解析

接口包含：

- `Placeholder() string`：返回 ORM 构建 SQL 时使用的占位符。
- `Parse(query string, args ...any) (string, []any)`：返回最终传给 `sql.DB` 或 `sql.Tx` 的 SQL 和参数。

`db.NewPlaceholderParser("?")` 只指定 ORM 占位符，其 `Parse` 原样返回 SQL 和参数，不展开集合，也不转换编号格式。各原生方言包注册自己的默认解析器，在各自的 `Parse` 中展开 `IN ?` / `NOT IN ?` 的集合参数，再按驱动要求转换占位符；Kingbase 等路由入口可复用已有方言解析器。

上面的最小 `custommysql` 示例使用公共透传解析器，因此不自动支持集合展开。需要复用内置 MySQL 的解析规则时，可将其默认策略设为 `mysql.GetPlaceholderParser()`；其它方言也可复用对应包的解析器。

`Placeholder()` 约定的是 ORM 输入标记，不是驱动输出格式。内置方言解析器匹配该标记后，分别写入固定的 `?`、`$n`、`:n` 或 `@pn`，集合展开也沿用同一编号序列。需要其它规则时可以实现自己的解析器，并在全部最终执行入口调用；需要正确识别复杂 SQL 时，还应区分真实占位符与字符串、注释等内容。

`Parse` 可以返回调整后的参数列表，例如展开集合、重排或替换参数；调用方必须同时使用返回的 SQL 和参数，保证占位符与参数对应，不应把参数值插入 SQL。不需要转换时返回 `query, args` 即可。此接口不返回 error。

自定义解析器会整体替换执行阶段的默认实现，不会自动调用它；原生 SQL 入口所需的集合展开和占位符转换均由自定义实现负责。`Where` 的提前处理差异见上文。

### 4. 策略对象的生命周期

所有默认策略都应在初始化阶段设置，不要与连接构造、校验或查询并发修改。Engine 保存创建时选中的策略引用，不受后续公共或包级默认替换影响。

`mysql.Validate` 等独立校验函数每次调用时取得对应包的有效默认列解析器，不读取已有 Engine 的实例级覆盖。调用方需保证校验和写入使用一致的字段规则。

保存接口值不等于深复制对象。如果自定义策略内部持有可变指针或 map，其不可变性与并发安全仍由实现者保证。策略注册或配置之后不再修改内部状态，可以避免同一查询前后使用不同规则。

## 六、查询结果扫描

`RowScanner` 只要求实现 `Scan(dest ...any) error`，因此 `*sql.Rows`、`*sql.Row` 或自定义扫描对象都能传入。`ResultWriter` 是 `func(scanner RowScanner) error`，每次调用负责读取一行并写入目标。

### 1. 多行结果

`PrepareSliceResult(model, queryResult, fields)` 在查询前验证模型和结果目标，返回可重复调用的逐行写入回调。

- 支持 `*[]T`、`*[]*T`，其中 T 为结构体；也支持 `*[]map[K]V`。
- 可以传入 `var list []T` 的 `&list`；它是非 nil 的切片指针，切片本身为 nil 不影响追加。
- 不接受直接切片值、数组、nil 切片指针或指针类型的 map 元素。
- 每次成功扫描后向切片追加一项，不会先清空已有内容。
- ORM 负责 `rows.Next()`、调用回调、检查 `rows.Err()` 并关闭 `Rows`。查询无结果时不会调用回调，也就不会追加数据。

### 2. 单行结果

`PrepareSingleResult(model, queryResult, fields)` 支持结构体指针或已初始化的直接 map，不接受 `*map`。先准备回调，再把 `QueryRowContext` 返回的 `*sql.Row` 交给它：

```go
func (engine *Engine) ScanOne(
	ctx context.Context,
	model any,
	queryResult any,
	fields []string,
	query string,
	args ...any,
) error {
	writeRow, err := db.PrepareSingleResult(model, queryResult, fields)
	if err != nil {
		return err
	}
	return writeRow(engine.QueryRowContext(ctx, query, args...))
}
```

`sql.Row` 的查询和无记录错误会在 `Scan` 时返回，例如 `sql.ErrNoRows`。公共扫描工具不负责 SQL 执行，也不替引擎决定 SQL 中是否添加单行限制。

### 3. 字段顺序与 map 规则

`fields` 必须与实际 SELECT 列的数量、顺序一致，并使用 Go 结构体字段名。字段子集查询和顺序调整应把对应列表传给准备方法，不能始终使用完整模型字段列表。

结构体结果按字段名写入；目标结构体没有对应可寻址字段时，该列会被扫描后丢弃。因此可使用仅包含部分字段的结果结构体。

map 使用模型字段类型作为扫描目标，并以 Go 字段名作为键。例如模型字段 `Name *string` 会产生键 `"Name"` 和类型为 `*string` 的值，而不是使用列名或 JSON 标签。`map[string]any` 可以容纳不同字段类型；其它 map 类型由调用方保证键和值可以直接赋值，工具不做额外转换。

map 分支要求 `fields` 对应模型中真实存在的字段；目标 map 的键值类型不匹配时，反射写入可能 panic，不会统一转换成 error。使用命名键类型或固定值类型的 map 时，应自行确保它们能直接接收字段名字符串和模型字段值。

切片扫描中，某一行失败不会追加该行，但此前成功追加的行会保留；单行结构体直接作为扫描目标，不保证失败时恢复原值。

## 七、连接、操作与停机生命周期

### 1. 初始化阶段

导入驱动和 ORM 包，完成策略设置，再写入 `goi.Settings.Databases`。如果在启动任务中使用数据库，连接配置应在任务启动前完成。

`Settings.Databases` 和示例中的策略变量属于应用配置，不应在业务查询期间无同步地修改。ORM 注册表的锁只保护工厂注册和查找。

### 2. 获取引擎与连接池

`db.Connect` 本身不缓存 ORM 实例；每次都会调用已注册的工厂。示例工厂为每次调用创建新 Engine，但通过 `database.DB()` 复用连接池。

同一个 `*goi.Database` 首次调用 `DB()` 时执行 `Connect(Database.Engine)`，之后复用返回的 `*sql.DB`。它是连接池，不是一条固定物理连接；`sql.Open` 也不等于验证数据库可连接，需要时由应用或连接回调调用 `PingContext`。

不应让 ORM 工厂每次都自行 `sql.Open`，否则会绕过已有连接池的缓存与关闭管理。

### 3. 查询与事务阶段

`*sql.DB` 可以被多个 ORM 实例共享；带有当前模型、条件、参数、事务或最后执行 SQL 的 ORM 对象，不能因此视为并发安全。建议独立操作流程创建独立 Engine，避免共享可变查询状态。

实现链式查询时，值接收者只复制切片描述符，不复制底层数组；分支追加可能覆盖其它分支的数据，需要按实际修改的切片隔离状态。可参考 [mysql/engine.go](mysql/engine.go) 的 `Where` 实现。

事务属于具体 ORM 的责任。完整实现通常需要：

1. 通过 `BeginTx` 开启事务，并让事务内的全部基础 SQL 方法分派到同一个 `*sql.Tx`。
2. 把绑定事务的 Engine 交给回调；返回错误时回滚，成功时提交。
3. 明确定义嵌套事务策略，并在退出时释放事务资源。

内置 MySQL 通过 `WithTransaction` 实现上述流程：嵌套调用复用外层事务，不创建保存点，也不独立提交；错误需要向外传播，由最外层决定回滚。回调发生 panic 时先尝试回滚，再保持原 panic 向外传播；回滚错误不另行返回。最小示例没有事务状态，扩展时不能只修改 `ExecuteContext` 而遗漏另外两个查询入口。

### 4. 释放结果集与关闭连接池

每次多行查询结束，由发起查询的一方关闭 `Rows`。普通 ORM 实例不拥有共享连接池，不应在一次查询、请求或事务结束时调用 `DB.Close()`。

Goi 服务的正常停机顺序为：停止 HTTP 监听并等待在途请求、停止后台任务、执行停机回调，最后关闭 `Settings.Databases` 中的连接管理器。自定义 ORM 通过这些管理器取得连接池即可参与该关闭流程，不需要为每个 ORM 实例额外注册一次关闭回调。

独立程序，或只通过 `ConnectDatabase` 使用且没有放入服务配置的连接管理器，由使用方调用 `Database.Close()`。关闭前应先停止仍可能使用它的查询、后台任务和事务。

直接传入自有 `DBProvider` 时，连接池由提供者的所有者关闭；`DBProvider` 接口和普通 Engine 都不会自动接管其生命周期。

`Database.Close()` 会关闭已缓存的池并清空缓存；没有创建池时直接返回 nil。之后再次调用 `DB()` 可以创建新池，但已有 Engine 仍然指向旧的 `*sql.DB`，不会自动切换，应重新创建 Engine。

## 八、从最小引擎扩展为完整 ORM

最小示例已经覆盖 Goi 接入流程。继续实现完整 ORM 时，可以按以下职责拆分，并参考内置 MySQL：

| 扩展点 | 参考文件 | 实现重点 |
| --- | --- | --- |
| 模型与表配置 | [mysql/model.go](mysql/model.go) | 定义自己的模型接口、Schema、表名和迁移选项；这些不是 `db.Engine` 的要求 |
| 连接级配置 | [mysql/config.go](mysql/config.go) | 定义 Config，实现 GetConfig 将配置写入指定目标 |
| 注册与连接构造 | [mysql/connect.go](mysql/connect.go) | 工厂统一读取 Config；Connect 检查名称后复用工厂；ConnectDatabase 获取连接池并合并显式策略 |
| ORM 操作与事务 | [mysql/engine.go](mysql/engine.go) | 管理 Engine 状态；解析模型、构建 SQL；统一执行入口和事务分派 |
| 三个策略的默认值与配置入口 | [mysql/column.go](mysql/column.go)、[mysql/identifier.go](mysql/identifier.go)、[mysql/placeholder.go](mysql/placeholder.go) | 包级默认配置与实例级策略对象分离 |
| 字段验证 | [mysql/validator.go](mysql/validator.go) | 根据列类型和业务规则校验模型；由 ORM 或调用方决定何时执行 |
| 查询构建帮助方法 | [mysql/utils.go](mysql/utils.go) | 分离集合参数展开等内部工具，与数据库方言规则配合 |

`mysql`、`postgres`、`sqlite3`、`oracle`、`sqlserver` 统一采用上述 `connect.go` / `engine.go` 拆分。`kingbase` 的 [connect.go](kingbase/connect.go) 和 `seaboxsql` 的 [connect.go](seaboxsql/connect.go) 只负责按 Mode 路由，不定义自己的 Engine、Model 或 Validate，连接配置均放在各包的 `config.go`。两个路由包直接继承所选方言的默认策略，不单独实现占位符解析器。

迁移、字段验证、主键处理、分页、是否限制全表操作、默认值与 NULL 处理等行为，应由自定义 ORM 明确定义。公共 `db` 工具负责可复用的基础工作，不会自动替引擎实现这些策略。

## 九、内置数据库方言参考

各方言引擎在创建时保存标识符、占位符和列定义策略。基础 SQL 方法提供无上下文及 `...Context` 两组入口；迁移、CRUD、计数、分页和事务方法直接接收 `context.Context`。模型的 `Settings.Schema` 用于缓存完整表名，并与迁移的命名空间保持一致。`kingbase` 和 `seaboxsql` 不定义独立模型和引擎，而是按 Mode 创建下表中的对应方言引擎。

| ORM 包 | 默认标识符 | ORM 占位符 → 驱动占位符 | Schema 为空时 | 分页 |
| --- | --- | --- | --- | --- |
| `mysql` | 反引号 | `?` → `?` | 连接当前数据库 | `LIMIT ... OFFSET ...` |
| `postgres` | 双引号 | `?` → `$1, $2, ...` | `public` | `LIMIT ... OFFSET ...` |
| `kingbase` | 继承 Mode 对应方言 | 继承 Mode 对应方言，可按实际驱动覆盖 | 继承 Mode 对应方言 | 继承 Mode 对应方言 |
| `seaboxsql` | 继承 Mode 对应方言 | 继承 Mode 对应方言，可按实际驱动覆盖 | 继承 Mode 对应方言 | 继承 Mode 对应方言 |
| `sqlite3` | 双引号 | `?` → `?` | 不限定数据库；也可指定 `main`、`temp` 或已附加的数据库 | `LIMIT ... OFFSET ...` |
| `oracle` | 双引号 | `?` → `:1, :2, ...` | 当前 Schema | `OFFSET ... ROWS FETCH NEXT ... ROWS ONLY`，要求 Oracle 12c 及以上 |
| `sqlserver` | 方括号 | `?` → `@p1, @p2, ...` | `dbo` | `OFFSET ... ROWS FETCH NEXT ... ROWS ONLY`，需先设置排序 |

默认编号占位符解析器沿用顺序替换规则，不做完整 SQL 词法分析；含同名运算符、字符串字面量或注释的复杂 SQL，可配置自定义解析器。使用原生编号占位符、且 SQL 中没有 `?` 时，默认解析器保留原语句。

`Migrate(ctx, model)` 会调用 `SetModel` 并重置当前查询状态，只创建尚不存在的表；表已存在时不修改结构，也不执行迁移前后回调。它不提供已有表的字段变更或版本迁移能力。

`Page(ctx, page, pageSize)` 修改当前实例的分页设置，并执行一次 `Count`。总记录数只按当前 WHERE 统计，不受 Fields、分组、排序或分页影响；需要分组统计总数时，应自行编写对应查询。

`Insert` 的实际列列表为空时（例如调用 `Fields()`，或 `skipNil=true` 后所有字段都被跳过），框架不额外处理，仍将 `INSERT INTO ... () VALUES ()` 交给数据库执行；该语句是否合法由数据库判断。`skipNil=false` 时，选中的 nil 指针会分别绑定为 NULL。需要执行默认值行插入时，请通过 `ExecuteContext` 自行编写对应方言 SQL。相关模拟驱动单元测试只检查框架转交的 SQL，不表示各数据库接受该语法。

字段验证保持方言语义：PostgreSQL 的 VARCHAR 按字符计数；Oracle 区分 VARCHAR2 的 BYTE/CHAR 与 NVARCHAR2；SQL Server 区分 VARCHAR 字节数与 NVARCHAR 的 UTF-16 代码单元；SQLite 不额外强制 VARCHAR 长度。Schema 不由迁移自动创建，SQLite 的附加数据库也需要调用方预先建立。SQLite 的 ATTACH 作用于物理连接，使用连接池时还需保证执行相关查询的连接已经附加该数据库。

### Kingbase 按兼容模式选择方言

Kingbase 的 `Connect` 和注册工厂均调用 `kingbase.ConnectDatabase(name, database)`，由这个路由入口通过 `database.GetConfig(&config)` 读取一次配置。它接收 `*goi.Database`，不同于原生方言包只使用 DBProvider 和显式选项的低层构造入口。

`kingbase.Config` 值和指针均实现 `goi.DatabaseConfig`；也可使用其它能向 `*kingbase.Config` 写入配置的实现。读取错误会在获取连接池前 panic；nil 配置、nil 配置指针和空 Mode 均按 PostgreSQL 模式处理。Mode 描述服务端已启用的模式，不负责切换服务端设置。

Mode 的字符串值为 `postgres`、`mysql`、`oracle`、`sqlserver`，与对应数据库包的 `driverName` 一致，不直接作为服务端参数使用。

| Mode | 返回类型 | 模型设置与字段校验 |
| --- | --- | --- |
| `ModePostgreSQL`（默认） | `*postgres.Engine` | `postgres.Model`、`postgres.Settings`、`postgres.Validate` |
| `ModeMySQL` | `*mysql.Engine` | `mysql.Model`、`mysql.Settings`、`mysql.Validate` |
| `ModeOracle` | `*oracle.Engine` | `oracle.Model`、`oracle.Settings`、`oracle.Validate` |
| `ModeSQLServer` | `*sqlserver.Engine` | `sqlserver.Model`、`sqlserver.Settings`、`sqlserver.Validate` |

例如，使用官方 Gokb 驱动连接 MySQL 兼容模式的 Kingbase。驱动需按实际交付包安装，导入路径为 `kingbase.com/gokb`；底层仍连接 Kingbase，不改用 MySQL 驱动：

```go
import (
	"database/sql"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/kingbase"
	"github.com/NeverStopDreamingWang/goi/v3/db/mysql"
	_ "kingbase.com/gokb"
)

func connectKingbase(dsn string) *mysql.Engine {
	goi.Settings.Databases["kes_mysql"] = &goi.Database{
		Engine: "kingbase",
		Config: kingbase.Config{
			Mode: kingbase.ModeMySQL,
			EngineOptions: db.EngineOptions{
				ColumnParser: db.NewColumnParser("field_name", "field_type"),
			},
		},
		Connect: func(_ string) *sql.DB {
			pool, err := sql.Open("kingbase", dsn)
			if err != nil {
				panic(err)
			}
			return pool
		},
	}
	return db.Connect[*mysql.Engine]("kes_mysql")
}
```

`kingbase.Config.EngineOptions` 是显式字段，配置字面量通过 `EngineOptions: db.EngineOptions{...}` 填写。Kingbase 只读取一次自己的 Config，再把选项传给所选方言包的低层 `ConnectDatabase`，不会要求原生包重复解析 Kingbase 配置。

各 Mode 直接把 `Config.EngineOptions` 传给对应方言的 `ConnectDatabase`，由方言包按“包级有效默认值 → 连接级非 nil 选项”合并。占位符、标识符和列解析器均继承所选方言，不预先设置跨方言覆盖；默认驱动占位符分别为 PostgreSQL 的 `$n`、MySQL 的 `?`、Oracle 的 `:n`、SQL Server 的 `@pn`。包级配置替换只影响后续创建的实例，已有实例保留创建时的策略。

这是一套默认选择规则，不代表各 Gokb 版本都已验证支持全部格式。[MySQL 兼容版 V9R3C11 的 Gokb 指南](https://help.kingbase.com.cn/v9.3.11/PDF/KingbaseES%E5%AE%A2%E6%88%B7%E7%AB%AF%E7%BC%96%E7%A8%8B%E6%8E%A5%E5%8F%A3%E6%8C%87%E5%8D%97-Gokb.pdf)列出 `$N`、`?`、`:NAME`；[SQLServer 兼容版 V9R4C12 指南](https://help.kingbase.com.cn/v9.4.12/development/application-develop-guide/application_development/client-interfaces/go/go-3.html)的常规绑定说明也不能直接证明普通 DML 支持 `@p1`，命名参数 `:NAME` 也不能等同于位置参数 `:1`。未确认的格式按所选方言默认值处理，调用方应针对实际驱动版本验证。

实际驱动要求不同格式时，通过 `Config.EngineOptions.PlaceholderParser` 显式覆盖。例如已确认需要 `$n` 时，可设置 `PlaceholderParser: postgres.GetPlaceholderParser()`，无需更改 Model 所属的数据库方言；连接级配置始终优先。

使用泛型入口时，类型必须与 Mode 对应；例如上述模式应取得 `*mysql.Engine`，模型的 `ModelSet` 返回 `*mysql.Settings`，字段校验调用 `mysql.Validate`。`kingbase.Connect` 返回公共 `db.Engine`，仅暴露基础 SQL 方法；不要通过 `mysql.Connect("kes_mysql")` 读取 Engine 为 kingbase 的配置。

旧的 `kingbase.Engine`、`kingbase.Model`、`kingbase.Settings`、`kingbase.Validate` 及三种策略的包级 Set/Get 入口已移除。调用点需迁移到所选方言包，原策略配置改放 `kingbase.Config.EngineOptions`。改变 Mode 时，泛型类型和模型配置也要同步，不是只改一个配置值就能保持所有旧调用不变。

该路由复用方言实现，不代表四种模式与原生数据库完全等价。模拟驱动测试只验证分派、策略和 SQL 转交，实际 Gokb/Kingbase 版本仍需集成验证。特别是 Oracle 模式的 VARCHAR2 BYTE/CHAR 语义，应使模型声明、校验规则与目标实例配置保持一致。

### SeaBoxSQL 按兼容模式选择方言

SeaBoxSQL 的 `Connect` 和注册工厂均调用 `seaboxsql.ConnectDatabase(name, database)`，通过 `database.GetConfig(&config)` 读取一次配置，再创建所选方言的 Engine。该入口接收 `*goi.Database`，不直接接收 DBProvider 或实例选项。

`seaboxsql.Config` 值和指针均实现 `goi.DatabaseConfig`；也可使用其它能向 `*seaboxsql.Config` 写入配置的实现。读取错误会在获取连接池前原样 panic；nil 配置、nil 配置指针和空 Mode 均默认选择 PostgreSQL 方言。

Mode 只选择 ORM 方言，不修改服务端兼容模式，也不选择 SQL 驱动。应根据目标实例已启用的模式选择 `postgres`、`mysql` 或 `oracle`，不支持 `sqlserver`。

| Mode | 返回类型 | 模型设置与字段校验 |
| --- | --- | --- |
| `ModePostgreSQL`（默认） | `*postgres.Engine` | `postgres.Model`、`postgres.Settings`、`postgres.Validate` |
| `ModeMySQL` | `*mysql.Engine` | `mysql.Model`、`mysql.Settings`、`mysql.Validate` |
| `ModeOracle` | `*oracle.Engine` | `oracle.Model`、`oracle.Settings`、`oracle.Validate` |

例如，为已启用 MySQL 兼容模式的 SeaBoxSQL 配置 MySQL 方言，并复用应用提供的连接回调：

```go
import (
	"database/sql"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/mysql"
	"github.com/NeverStopDreamingWang/goi/v3/db/seaboxsql"
)

func connectSeaBoxSQL(connector func(string) *sql.DB) *mysql.Engine {
	goi.Settings.Databases["seabox_mysql"] = &goi.Database{
		Engine: "seaboxsql",
		Config: seaboxsql.Config{
			Mode: seaboxsql.ModeMySQL,
			EngineOptions: db.EngineOptions{
				ColumnParser: db.NewColumnParser("field_name", "field_type"),
			},
		},
		Connect: connector,
	}
	return db.Connect[*mysql.Engine]("seabox_mysql")
}
```

连接回调接收 ORM 注册名 `"seaboxsql"`，由调用方选择实际驱动、DSN 和连接池设置，不应直接把该参数当成 SQL 驱动名。导入 `seaboxsql` 只注册 ORM 工厂，不加载物理驱动；示例也不表示应改用 MySQL 驱动连接 SeaBoxSQL。

`seaboxsql.Config.EngineOptions` 是显式字段，各 Mode 直接把它传给对应方言，由方言包合并包级默认值和连接级非 nil 选项。占位符、标识符及列解析器默认均继承所选方言；SeaBoxSQL 不额外指定解析器，因此 PostgreSQL、MySQL、Oracle 模式的默认驱动占位符分别为 `$n`、`?`、`:n`。已有 Engine 保存创建时的策略引用，后续默认配置替换不改变它。

[官方兼容模式说明](https://www.seaboxdata.com/seabox-handbook/source_files/overview/mpp/feature_intro/compatibility_mode/)列出的是 SQL 语义开关，不能据此确认 Go 驱动的参数绑定规则。[服务端 PREPARE](https://seaboxdata.com/seabox-handbook/source_files/ref_guide/sql_commands/PREPARE/) 使用 `$n`，而 [JDBC PreparedStatement](https://seaboxdata.com/seabox-handbook/source_files/server-programming/JDBC/query/) 使用 `?`；二者也不能代替 Go 驱动在各 Mode 下的验证。缺少完整映射依据时，框架采用所选方言默认值，不声明所有格式都已兼容。

如果实际采用的驱动需要 PostgreSQL 风格的参数绑定，可在 `Config.EngineOptions` 中设置 `PlaceholderParser: postgres.GetPlaceholderParser()`；这只覆盖占位符规则，不改变所选 Model、标识符和其它方言行为。

模型接口、Settings 和 Validate 使用 Mode 对应的原生包，泛型类型也必须匹配。`seaboxsql.Connect` 返回公共 `db.Engine`，需要 CRUD 等扩展方法时使用对应具体类型的泛型入口；不要调用 `mysql.Connect("seabox_mysql")` 读取 Engine 为 seaboxsql 的配置。该包不提供独立的 Engine、Model、Settings、Validate 或包级策略 Set/Get，实例策略统一放在 Config 中。

路由成功不代表与对应原生数据库完整兼容。模拟驱动测试不能证明服务端接受生成的全部 SQL；驱动协议、迁移元数据、类型、Schema、分页等需按目标版本和模式做集成验证。
