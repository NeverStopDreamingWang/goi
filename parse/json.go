package parse

import (
	"encoding/json"
	"net/http"
)

const MIME_JSON = "application/json"

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// any as a Number instead of as a float64.
var EnableDecoderUseNumber = false

// EnableDecoderDisallowUnknownFields is used to call the DisallowUnknownFields method
// on the JSON Decoder instance. DisallowUnknownFields causes the Decoder to
// return an error when the destination is a struct and the input contains object
// keys which do not match any non-ignored, exported fields in the destination.
var EnableDecoderDisallowUnknownFields = false

var JSON jsonParsing

type jsonParsing struct{}

func (jsonParsing) Name() string {
	return "json"
}

func (jsonParsing) Parse(request *http.Request) (Params, error) {
	var err error
	params := make(Params)

	decoder := json.NewDecoder(request.Body)
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	err = decoder.Decode(&params)
	if err != nil {
		return nil, err
	}
	return params, nil
}
