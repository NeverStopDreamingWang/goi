package auth

import (
	"github.com/NeverStopDreamingWang/goi/jwt"
)

type Payloads struct {
	jwt.Payloads
	User_id  int64  `json:"user_id"`
	Username string `json:"username"`
}
