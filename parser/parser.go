package parser

import (
	"net/http"
	"sync"
)

type Params map[string]any

// Parser 解析器接口
type Parser interface {
	Name() string
	Parse(*http.Request) (Params, error)
}

var defaultParser = Form

// parserMu 保护 parsers
var parserMu sync.RWMutex

// parsers MIME 类型到解析器的映射
var parsers = map[string]Parser{
	MIMEJSON:              JSON,
	MIMEXML:               XML,
	MIMETextXML:           XML,
	MIMEYAML:              YAML,
	MIMEMultipartPostForm: FormMultipart,
}

// RegisterParser 注册自定义解析器
func RegisterParser(name string, p Parser) {
	parserMu.Lock()
	defer parserMu.Unlock()
	parsers[name] = p
}

// GetParser 根据 Content-Type 获取解析器
func GetParser(contentType string) Parser {
	parserMu.RLock()
	defer parserMu.RUnlock()
	p, ok := parsers[contentType]
	if !ok {
		return defaultParser
	}
	return p
}
