package example

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 注册验证器

	// 手机号
	goi.RegisterValidator("phone", phoneValidate)
}

// phone 类型
func phoneValidate(value string) goi.ValidationError {
	var IntRe = `^(1[3456789]\d{9})$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return goi.NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}
