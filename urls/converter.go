package urls

var IntConverter = `([0-9]+)`
var StringConverter = `([^/]+)`
var UUIDConverter = `([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
var SlugConverter = `([-a-zA-Z0-9_]+)`
var PathConverter = `(.+)`

var DEFAULT_CONVERTERS = map[string]string{
	"int":  IntConverter,
	"str":  StringConverter,
	"slug": SlugConverter,
	"path": PathConverter,
	"uuid": UUIDConverter,
}

func RegisterConverter(typeName string, converter string) {
	DEFAULT_CONVERTERS[typeName] = converter
}
