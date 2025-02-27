package sqlite3_db

import (
	"database/sql"
)

// SQLite3 配置
type ConfigModel struct {
	Uri string `yaml:"uri"`
}

var Config *ConfigModel = &ConfigModel{}

var SQLite3DB *sql.DB
