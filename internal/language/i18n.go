package language

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed all:lang/*
var langFS embed.FS
var bundle *i18n.Bundle
var I18n *i18n.Localizer

func init() {
	bundle = i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	_, _ = bundle.LoadMessageFileFS(langFS, "lang/zh_cn.json")
	_, _ = bundle.LoadMessageFileFS(langFS, "lang/en_us.json")
	I18n = i18n.NewLocalizer(bundle, "zh_cn")
}

// 设置语言
func SetLocalize(langs ...string) {
	I18n = i18n.NewLocalizer(bundle, langs...)
}
