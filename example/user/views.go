package user

import (
	"github.com/NeverStopDreamingWang/hgee"
	"net/http"
)

func UserTest(request *hgee.Request) any {

	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   nil,
		},
	}
}
