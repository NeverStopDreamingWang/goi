package user

import (
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"net/http"
)

// 设置一条数据到缓存
func TestCacheSet(request *goi.Request) interface{} {
	var key string
	var value string
	var exp int
	var err error
	err = request.BodyParams.Get("key", key)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusBadRequest,
				Msg:    "参数错误",
				Data:   nil,
			},
		}
	}
	err = request.BodyParams.Get("value", value)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusBadRequest,
				Msg:    "参数错误",
				Data:   nil,
			},
		}
	}
	err = request.BodyParams.Get("exp", exp)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusBadRequest,
				Msg:    "参数错误",
				Data:   nil,
			},
		}
	}
	fmt.Printf("key: %v\n", key)
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("exp: %v\n", exp)
	goi.Cache.Set(key, value, exp)

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "设置成功！",
			Data:   nil,
		},
	}
}

// 通过 key 获取缓存
func TestCacheGet(request *goi.Request) interface{} {
	var key string
	var err error
	err = request.BodyParams.Get("key", key)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusBadRequest,
				Msg:    "参数错误",
				Data:   nil,
			},
		}
	}

	fmt.Printf("key: %v\n", key)
	value, ok := goi.Cache.Get(key)
	if !ok {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusOK,
				Msg:    "key 值不存在！",
				Data:   nil,
			},
		}
	}

	fmt.Printf("value: %v\n", *value.(*string))
	// if err != nil {
	// 	return goi.Response{
	// 		Status: http.StatusInternalServerError,
	// 		Data:   err.Error(),
	// 	}
	// }
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   value,
		},
	}
}

// 通过 key 删除缓存
func TestCacheDel(request *goi.Request) interface{} {
	var key string
	var err error
	err = request.BodyParams.Get("key", key)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusBadRequest,
				Msg:    "参数错误",
				Data:   nil,
			},
		}
	}

	fmt.Printf("key: %v\n", key)
	goi.Cache.Del(key)
	value, ok := goi.Cache.Get(key)

	if !ok {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusOK,
				Msg:    "删除成功！",
				Data:   nil,
			},
		}
	}
	fmt.Printf("删除失败 value: %v\n", *value.(*string))
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "删除失败！",
			Data:   nil,
		},
	}
}
