package mysql_db

import (
	"database/sql"
)

// MySQL 配置
type ConfigModel struct {
	Uri string `yaml:"uri"`
}

var Config *ConfigModel = &ConfigModel{}

var MySQLDB *sql.DB
