package parse

import (
	"net/http"
)

type Params map[string]interface{}

type Parsing interface {
	Name() string
	Parse(*http.Request) (Params, error)
}

var defaultParsing = Form

var metaParsing = map[string]Parsing{
	MIME_JSON:              JSON,
	MIME_XML:               XML,
	MIME_XML2:              XML,
	MIME_YAML:              YAML,
	MIME_MultipartPOSTForm: FormMultipart,
}

func RegisterParsing(name string, parsing Parsing) {
	metaParsing[name] = parsing
}

func GetParsing(contentType string) Parsing {
	parsing, ok := metaParsing[contentType]
	if !ok {
		return defaultParsing
	}
	return parsing
}
