package parser

import (
	"net/http"

	"gopkg.in/yaml.v3"
)

const MIME_YAML = "application/x-yaml"

var YAML yamlParser

type yamlParser struct{}

func (yamlParser) Name() string {
	return "yaml"
}

func (yamlParser) Parse(request *http.Request) (Params, error) {
	var err error
	params := make(Params)

	decoder := yaml.NewDecoder(request.Body)
	err = decoder.Decode(&params)
	if err != nil {
		return nil, err
	}
	return params, nil
}
