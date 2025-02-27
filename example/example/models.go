package example

import (
	"example/utils/mongo_db"
	"example/utils/mysql_db"
	"example/utils/redis_db"
	"example/utils/sqlite3_db"
)

// 配置
type ConfigModel struct {
	SQLite3Config *sqlite3_db.ConfigModel `yaml:"sqlite3"`
	MySQLConfig   *mysql_db.ConfigModel   `yaml:"mysql"`
	RedisConfig   *redis_db.ConfigModel   `yaml:"redis"`
	MongoDBConfig *mongo_db.ConfigModel   `yaml:"mongodb"`
}
