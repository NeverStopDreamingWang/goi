package mysql

import (
	"github.com/NeverStopDreamingWang/goi/v2"
)

// MigrationsHandler 定义建表前后的回调。
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}

// Settings 定义 MySQL 表、迁移及自定义配置。
type Settings struct {
	MigrationsHandler // 迁移处理函数

	Schema          string // 设置数据库名称；MySQL 中 Schema 与 Database 含义相同
	TableName       string // 设置表名
	Engine          string // 设置存储引擎，默认: InnoDB
	AUTO_INCREMENT  int    // 设置自增长起始值
	DEFAULT_CHARSET string // 设置默认字符集，如: utf8mb4
	COLLATE         string // 设置校对规则，如 utf8mb4_0900_ai_ci;
	MIN_ROWS        int    // 设置最小行数
	MAX_ROWS        int    // 设置最大行数
	CHECKSUM        int    // 表格的校验和算法，如 1 开启校验和
	DELAY_KEY_WRITE int    // 控制非唯一索引的写延迟，如 1
	ROW_FORMAT      string // 设置行的存储格式，如 DYNAMIC, COMPACT, FULL.
	DATA_DIRECTORY  string // 设置数据存储目录
	INDEX_DIRECTORY string // 设置索引存储目录
	PARTITION_BY    string // 定义分区方式，如 RANGE、HASH、LIST
	COMMENT         string // 设置表注释
	// 自定义配置
	Settings goi.Params
}

// Model 提供 MySQL 模型的表及迁移配置。
type Model interface {
	// ModelSet 返回当前模型的配置。
	ModelSet() *Settings
}
