package i18n

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
	bundle = i18n.NewBundle(language.SimplifiedChinese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	_, _ = bundle.LoadMessageFileFS(langFS, "lang/zh-CN.json")
	_, _ = bundle.LoadMessageFileFS(langFS, "lang/en-US.json")
	I18n = i18n.NewLocalizer(bundle)
}

// 设置语言
func SetLocalize(langs ...string) {
	I18n = i18n.NewLocalizer(bundle, langs...)
}

func T(messageID string, data ...map[string]interface{}) string {
	localizeConfig := &i18n.LocalizeConfig{
		MessageID: messageID,
	}
	if len(data) > 0 && data[0] != nil {
		localizeConfig.TemplateData = data[0]
	}
	return I18n.MustLocalize(localizeConfig)
}
