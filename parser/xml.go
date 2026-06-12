package parser

import (
	"encoding/xml"
	"net/http"
)

const MIMEXML = "application/xml"
const MIMETextXML = "text/xml"

var XML xmlParser

type xmlParser struct{}

func (xmlParser) Name() string {
	return "xml"
}

func (xmlParser) Parse(request *http.Request) (Params, error) {
	var err error
	params := make(Params)

	decoder := xml.NewDecoder(request.Body)
	err = decoder.Decode(&params)
	if err != nil {
		return nil, err
	}
	return params, nil
}
