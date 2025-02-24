package goi

import (
	"time"
)

// 获取当前时区
func GetLocation() *time.Location {
	if Settings != nil {
		return Settings.GetLocation()
	}
	return time.Local
}

// 获取当前时区时间
func GetTime() time.Time {
	return time.Now().In(GetLocation())
}
