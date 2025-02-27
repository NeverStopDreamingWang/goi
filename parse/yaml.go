package parse

import (
	"net/http"

	"gopkg.in/yaml.v3"
)

const MIME_YAML = "application/x-yaml"

var YAML yamlParsing

type yamlParsing struct{}

func (yamlParsing) Name() string {
	return "yaml"
}

func (yamlParsing) Parse(request *http.Request) (Params, error) {
	var params Params
	var err error

	decoder := yaml.NewDecoder(request.Body)
	err = decoder.Decode(&params)
	if err != nil {
		return nil, err
	}
	return params, nil
}
