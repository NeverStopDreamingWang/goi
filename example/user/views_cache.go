package user

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"net/http"
	"strconv"
)

// 设置一条数据到缓存
func TestCacheSet(request *hgee.Request) any {
	Key := request.BodyParams.Get("key")
	Value := request.BodyParams.Get("value")
	Exp := request.BodyParams.Get("exp")
	if Key == nil {
		return hgee.Response{
			Status: http.StatusBadRequest,
			Data:   "缺少参数 key",
		}
	}
	if Value == nil {
		return hgee.Response{
			Status: http.StatusBadRequest,
			Data:   "缺少参数 value",
		}
	}

	exp := "0"
	if Exp != nil {
		exp = Exp.(string)
	}
	expTime, err := strconv.Atoi(exp)
	if err != nil {
		return hgee.Response{
			Status: http.StatusBadRequest,
			Data:   "exp 参数错误！",
		}
	}
	key := Key.(string)
	value := Value.(string)
	fmt.Printf("key: %v\n", key)
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("exp: %v\n", expTime)
	hgee.Cache.Set(key, value, expTime)

	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "设置成功！",
			Data:   nil,
		},
	}
}

// 通过 key 获取缓存
func TestCacheGet(request *hgee.Request) any {
	key := request.BodyParams.Get("key").(string)

	fmt.Printf("key: %v\n", key)
	value, ok := hgee.Cache.Get(key)
	if !ok {
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusOK,
				Msg:    "key 值不存在！",
				Data:   nil,
			},
		}
	}

	fmt.Printf("value: %v\n", *value.(*string))
	// if err != nil {
	// 	return hgee.Response{
	// 		Status: http.StatusInternalServerError,
	// 		Data:   err.Error(),
	// 	}
	// }
	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   value,
		},
	}
}

// 通过 key 删除缓存
func TestCacheDel(request *hgee.Request) any {
	key := request.BodyParams.Get("key").(string)

	fmt.Printf("key: %v\n", key)
	hgee.Cache.Del(key)
	value, ok := hgee.Cache.Get(key)

	if !ok {
		return hgee.Response{
			Status: http.StatusOK,
			Data: hgee.Data{
				Status: http.StatusOK,
				Msg:    "删除成功！",
				Data:   nil,
			},
		}
	}
	fmt.Printf("删除失败 value: %v\n", *value.(*string))
	return hgee.Response{
		Status: http.StatusOK,
		Data: hgee.Data{
			Status: http.StatusOK,
			Msg:    "删除失败！",
			Data:   nil,
		},
	}
}
