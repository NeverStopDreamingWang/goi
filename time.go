package goi

import (
	"time"
)

// GetLocation 获取当前时区
//
// 返回:
//   - *time.Location: 时区对象，如果Settings已初始化则返回配置的时区，否则返回系统本地时区
func GetLocation() *time.Location {
	if Settings != nil {
		return Settings.GetLocation()
	}
	return time.Local
}

// GetTime 获取当前时区的时间
//
// 返回:
//   - time.Time: 当前时区的时间对象
func GetTime() time.Time {
	return time.Now().In(GetLocation())
}
