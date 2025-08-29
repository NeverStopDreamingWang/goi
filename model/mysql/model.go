package mysql

import (
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
)

// MySQL 模型设置
type MySQLSettings struct {
	model.MigrationsHandler // 迁移处理函数

	TABLE_NAME      string // 设置表名
	ENGINE          string // 设置存储引擎，默认: InnoDB
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

// 模型类
type MySQLModel interface {
	ModelSet() *MySQLSettings
}

// MySQL 创建迁移
type MySQLMakeMigrations struct {
	DATABASES []string
	MODELS    []MySQLModel
}
