package sqlite3_db

import (
	"database/sql"
	"path/filepath"

	"github.com/NeverStopDreamingWang/goi"
)

// SQLite3 配置
type ConfigModel struct {
	Uri string `yaml:"uri"`
}

var Config = &ConfigModel{}

var SQLite3DB *sql.DB

func Connect(ENGINE string) *sql.DB {
	var err error
	DataSourceName := filepath.Join(goi.Settings.BASE_DIR, Config.Uri)
	SQLite3DB, err = sql.Open(ENGINE, DataSourceName)
	if err != nil {
		goi.Log.Error(err)
		panic(err)
	}
	return SQLite3DB
}
