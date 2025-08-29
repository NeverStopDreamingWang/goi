package goi

import (
	"testing"
	"time"
)

// TestGetLocation_Default 确认当 Settings 为 nil 时：
// - GetLocation 返回 time.Local
// - GetTime 返回 GetLocation() 的“有感知”时间（与系统本地时区一致）
func TestGetLocation_Default(t *testing.T) {
	prev := Settings
	defer func() { Settings = prev }()

	Settings = nil
	loc := GetLocation()
	if loc != time.Local {
		t.Fatalf("expected time.Local, got %v", loc)
	}

	now := GetTime()
	if now.Location() != loc {
		t.Fatalf("expected GetTime to use GetLocation, got %s", now.Location())
	}
	zone, offset := now.Zone()
	t.Logf("[default] unix=%d time=%s loc=%s zone=%s offset=%d",
		now.Unix(), now.Format(time.RFC3339), now.Location().String(), zone, offset)
}

// TestGetTime_USE_TZ 验证 USE_TZ 对 GetTime 的影响
// - USE_TZ=true：返回当前配置时区的“有感知”时间（Location=GetLocation）
// - USE_TZ=false：返回“无感知”时间（Location=UTC），墙钟等于配置时区的当前墙钟
func TestGetTime_USE_TZ(t *testing.T) {
	prev := Settings
	defer func() { Settings = prev }()

	Settings = newSettings()
	_ = Settings.SetTimeZone("Asia/Shanghai")

	tNow := time.Now()
	zone, offset := tNow.Zone()
	t.Logf("[now] unix=%d time=%s loc=%s zone=%s offset=%d",
		tNow.Unix(), tNow.Format(time.RFC3339), tNow.Location().String(), zone, offset)

	// 1) USE_TZ=true -> aware 本地时间（带 +08:00）
	Settings.USE_TZ = true
	tAware := GetTime()
	if tAware.Location().String() != "Asia/Shanghai" {
		t.Fatalf("expected Asia/Shanghai, got %s", tAware.Location())
	}
	if _, off := tAware.Zone(); off != 8*3600 {
		t.Fatalf("expected +08:00 offset, got %d", off)
	}
	// 2) USE_TZ=false -> naive（Location=UTC），墙钟应等于 Asia/Shanghai 当前墙钟
	Settings.USE_TZ = false
	tNaive := GetTime()
	if tNaive.Location().String() != "UTC" {
		t.Fatalf("expected UTC location for naive, got %s", tNaive.Location())
	}
	local := time.Now().In(GetLocation())
	if !(tNaive.Year() == local.Year() && tNaive.Month() == local.Month() && tNaive.Day() == local.Day() && tNaive.Hour() == local.Hour()) {
		t.Fatalf("naive wall-clock mismatch: naive=%v local=%v", tNaive, local)
	}
}
