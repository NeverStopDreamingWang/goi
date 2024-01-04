package hgee

var intConverter = `([0-9]+)`
var stringConverter = `([^/]+)`
var slugConverter = `([-a-zA-Z0-9_]+)`
var pathConverter = `(.+)`
var uuidConverter = `([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`

var metaConverter = map[string]string{
	"int":  intConverter,
	"str":  stringConverter,
	"slug": slugConverter,
	"path": pathConverter,
	"uuid": uuidConverter,
}

func RegisterConverter(typeName string, converter string) {
	metaConverter[typeName] = converter
}
