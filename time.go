package goi

import (
	"time"
)

// 获取当前时区时间
func GetTime() time.Time {
	return time.Now().In(Settings.location)
}
