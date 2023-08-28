package hgee

var intConverter = `([0-9]+)`
var stringConverter = `([^/]+)`
var uuidConverter = `([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
var slugConverter = `([-a-zA-Z0-9_]+)`
var pathConverter = `(.+)`

var settingConverter = map[string]string{
	"int":  intConverter,
	"str":  stringConverter,
	"slug": uuidConverter,
	"path": slugConverter,
	"uuid": pathConverter,
}

func RegisterConverter(typeName string, converter string) {
	settingConverter[typeName] = converter
}
