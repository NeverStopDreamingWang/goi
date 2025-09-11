package goi

import (
	"errors"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// ParamInfo 路由参数信息
type ParamInfo struct {
	Name      string     // 参数名称
	Converter *Converter // 转换器指针
}

// compilePattern 将路由定义编译为正则
// 支持占位符: <type:name>，type 来自 converter.go 中的注册转换器
func (router *MetaRouter) compilePattern() {
	if router.pattern != "" {
		return
	}

	// 查找占位符 <type:name>
	re := regexp.MustCompile(`<([^/<>]+):([^/<>]+)>`)
	matches := re.FindAllStringSubmatchIndex(router.path, -1)

	paramInfos := make([]ParamInfo, 0, len(matches))
	patternBuilder := strings.Builder{}
	last := 0

	for _, loc := range matches {
		// loc: [fullStart, fullEnd, typeStart, typeEnd, nameStart, nameEnd]
		fullStart, fullEnd := loc[0], loc[1]
		typeStart, typeEnd := loc[2], loc[3]
		nameStart, nameEnd := loc[4], loc[5]

		// 追加占位符之前的字面量
		patternBuilder.WriteString(regexp.QuoteMeta(router.path[last:fullStart]))

		// 解析类型与名称
		typeName := router.path[typeStart:typeEnd]
		paramName := router.path[nameStart:nameEnd]

		converter, ok := GetConverter(typeName)
		if !ok {
			converterIsNotExistsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "router.converter_is_not_exists",
				TemplateData: map[string]interface{}{
					"name": typeName,
				},
			})
			panic(converterIsNotExistsMsg)
		}

		// 使用转换器的正则(其内部已含括号捕获)
		patternBuilder.WriteString(converter.Regex)
		paramInfos = append(paramInfos, ParamInfo{
			Name:      paramName,
			Converter: &converter,
		})

		last = fullEnd
	}

	// 追加尾部字面量
	patternBuilder.WriteString(regexp.QuoteMeta(router.path[last:]))
	router.pattern = patternBuilder.String()

	var rePattern string
	if router.includeRouter == nil { // 不是路由组
		rePattern = "^" + router.pattern + "$"
	} else {
		rePattern = "^" + router.pattern
	}
	router.regex = regexp.MustCompile(rePattern)
	router.paramInfos = paramInfos
}

// match 尝试匹配，返回 参数映射 与 是否匹配
func (router MetaRouter) match(path string) (Params, bool) {
	// 保护：确保已编译正则，避免空指针
	if router.regex == nil {
		(&router).compilePattern()
	}
	loc := router.regex.FindStringSubmatch(path)
	if loc == nil || len(loc)-1 != len(router.paramInfos) {
		return nil, false
	}
	// 第0个是完整匹配，从1开始依次为各参数
	params := make(Params, len(router.paramInfos))
	for i, paramInfo := range router.paramInfos {
		// i 对应第 i+1 个分组
		if i+1 < len(loc) {
			rawValue := loc[i+1]
			convertedValue, err := paramInfo.Converter.ToGo(rawValue)
			if err != nil {
				return nil, false
			}
			params[paramInfo.Name] = convertedValue
		}
	}
	return params, true
}

// resolve
func (router MetaRouter) resolve(Path string) (ViewSet, Params, bool) {
	params, ok := router.match(Path)
	if ok == false {
		return ViewSet{}, Params{}, false
	}
	if router.includeRouter == nil {
		return router.viewSet, params, true
	}
	// 子路由
	for _, itemRouter := range router.includeRouter {
		viewSet, params, isPattern := itemRouter.resolve(Path[len(router.path):])
		if isPattern == true {
			return viewSet, params, isPattern
		}
	}
	return ViewSet{}, Params{}, false
}

// resolveRequest
func (router MetaRouter) resolveRequest(request *Request) (ViewSet, error) {
	// 路由解析
	viewSet, params, isPattern := router.resolve(request.Object.URL.Path)
	if isPattern == false {
		urlNotAllowedMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.url_not_allowed",
			TemplateData: map[string]interface{}{
				"path": request.Object.URL.Path,
			},
		})
		return ViewSet{}, errors.New(urlNotAllowedMsg)
	}
	// 路由参数
	request.PathParams = params
	return viewSet, nil
}
