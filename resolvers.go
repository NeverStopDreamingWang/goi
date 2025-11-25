package goi

import (
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// ParamInfo 路由参数信息
type ParamInfo struct {
	Name      string     // 参数名称
	Converter *Converter // 转换器指针
}

// compilePattern 将路由定义编译为正则
// 支持占位符: <type:name>，type 来自 converter.go 中的注册转换器
func (router *Router) compilePattern() {
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
			converterIsNotExistsMsg := i18n.T("router.converter_is_not_exists", map[string]any{
				"name": typeName,
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
	if router.include == nil { // 不是路由组
		rePattern = "^" + router.pattern + "$"
	} else {
		rePattern = "^" + router.pattern
	}
	router.regex = regexp.MustCompile(rePattern)
	router.paramInfos = paramInfos
}

// match 尝试匹配，返回 参数映射 与 是否匹配
//
// 参数:
//   - path string: 待匹配的URL路径
//   - params Params: 匹配的参数映射
//
// 返回:
//   - MiddleWares: 匹配的路由中间件
//   - bool: 是否匹配成功
func (router Router) match(path string, params Params) (MiddleWares, bool) {
	// 保护：确保已编译正则，避免空指针
	if router.regex == nil {
		(&router).compilePattern()
	}
	loc := router.regex.FindStringSubmatch(path)
	if loc == nil || len(loc)-1 != len(router.paramInfos) {
		return nil, false
	}
	if len(router.paramInfos) > 0 {
		// 第0个是完整匹配，从1开始依次为各参数
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
	}
	return router.middlewares, true
}

// resolve 解析URL路径
//
// 参数:
//   - Path string: 待解析的URL路径
//   - params Params: 匹配的参数映射
//
// 返回:
//   - *ViewSet: 匹配的视图集
//   - MiddleWares: 匹配的路由中间件
//   - bool: 是否匹配成功
func (router Router) resolve(Path string, params Params) (*ViewSet, MiddleWares, bool) {
	middlewares, ok := router.match(Path, params)
	if ok == false {
		return nil, nil, false
	}
	if router.include == nil {
		return &router.viewSet, middlewares, true
	}
	// 子路由
	for _, itemRouter := range router.include {
		viewSet, middlewaresChild, isPattern := itemRouter.resolve(Path[len(router.path):], params)
		if isPattern == true {
			// 合并中间件
			middlewares = append(middlewares, middlewaresChild...)
			return viewSet, middlewares, isPattern
		}
	}
	// 无匹配
	if router.noRoute != nil {
		return router.noRoute, middlewares, true
	}
	return nil, nil, false
}
