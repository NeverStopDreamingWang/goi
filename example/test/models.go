package test

type UserModel struct {
	Id          int64  `json:"id,omitempty"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Create_time string `json:"create_time"`
	Update_time string `json:"update_Time"`
}
