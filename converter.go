package goi

var intConverter = `([0-9]+)`
var stringConverter = `([^/]+)`
var slugConverter = `([-a-zA-Z0-9_]+)`
var pathConverter = `(.+)`
var uuidConverter = `([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`

var metaConverter = map[string]string{
	"int":    intConverter,
	"string": stringConverter,
	"slug":   slugConverter,
	"path":   pathConverter,
	"uuid":   uuidConverter,
}

func RegisterConverter(name string, converter string) {
	metaConverter[name] = converter
}
