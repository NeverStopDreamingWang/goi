package redis_db

import (
	"context"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/redis/go-redis/v9"
)

// Redis 配置
type ConfigModel struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

var Config *ConfigModel = &ConfigModel{}

var RedisDB *redis.Client

func Connect() {
	// Redis 连接配置
	RedisDB = redis.NewClient(&redis.Options{
		Addr:            Config.Addr,      // Redis 地址
		Password:        Config.Password,  // Redis 密码（无密码可设置为空字符串 ""）
		DB:              Config.DB,        // Redis 数据库（默认 0）
		PoolSize:        10,               // 连接池大小
		PoolTimeout:     5 * time.Second,  // 获取连接的超时时间
		MinIdleConns:    1,                // 最小空闲连接数
		DialTimeout:     5 * time.Second,  // 连接超时
		ReadTimeout:     3 * time.Second,  // 读超时
		WriteTimeout:    3 * time.Second,  // 写超时
		ConnMaxIdleTime: 10 * time.Minute, // 空闲连接最大存活时间
		ConnMaxLifetime: 30 * time.Minute, // 连接的最大生命周期
	})

	// 使用 Redis 客户端
	ctx := context.Background()
	_, err := RedisDB.Ping(ctx).Result()
	if err != nil {
		msg := fmt.Sprintf("Redis 连接失败: %v", err)
		goi.Log.Error(msg)
		panic(msg)
	}
}

func Close() error {
	err := RedisDB.Close()
	if err != nil {
		return err
	}
	return nil
}
