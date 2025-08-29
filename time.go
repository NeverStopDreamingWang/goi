package goi

import (
	"time"
)

// GetLocation 获取当前时区
//
// 返回:
//   - *time.Location: 时区对象，如果Settings已初始化则返回配置的时区，否则返回系统本地时区
func GetLocation() *time.Location {
	if Settings == nil {
		return time.Local
	}
	return Settings.GetLocation()
}

// GetTime 获取当前时间
//   - Settings.USE_TZ=true：返回 GetLocation() 时区的时间 “有感知时区”
//   - Settings.USE_TZ=false：返回 GetLocation() 时区的时间，但时区标注为 UTC “无感知时区”，避免任何时区换算，直存直取
func GetTime() time.Time {
	now := time.Now().In(GetLocation())
	if Settings != nil && Settings.USE_TZ == false {
		// 无感知：墙钟=loc 的当前墙钟，但 Location 固定为 UTC
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
	}
	return now
}

// Localize 将给定时间转换为当前配置时区的“表示”（不改变时间瞬间，仅改变 Location）
func Localize(t time.Time) time.Time {
	return t.In(GetLocation())
}
