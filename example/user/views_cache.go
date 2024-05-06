package user

import (
	"fmt"
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
)

// 参数验证
type testCacheSetParams struct {
	Key   string `name:"key" required:"string"`
	Value string `name:"value" required:"string"`
	Exp   int    `name:"exp" required:"int"`
}

// 设置一条数据到缓存
func TestCacheSet(request *goi.Request) interface{} {
	var params testCacheSetParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	fmt.Printf("key: %v\n", params.Key)
	fmt.Printf("Value: %v\n", params.Value)
	fmt.Printf("exp: %v\n", params.Exp)
	goi.Cache.Set(params.Key, params.Value, params.Exp)

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "设置成功！",
			Data:   nil,
		},
	}
}

// 参数验证
type cacheKeyParams struct {
	Key string `name:"key" required:"string"`
}

// 通过 key 获取缓存
func TestCacheGet(request *goi.Request) interface{} {
	var params cacheKeyParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()
	}

	fmt.Printf("key: %v\n", params.Key)
	var value string
	err := goi.Cache.Get(params.Key, &value)
	if err != nil {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusOK,
				Msg:    "key 值不存在！",
				Data:   nil,
			},
		}
	}

	fmt.Printf("value: %v\n", value)
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
	var params cacheKeyParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()
	}

	fmt.Printf("key: %v\n", params.Key)
	goi.Cache.Del(params.Key)
	ok := goi.Cache.Has(params.Key)

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
	fmt.Println("删除失败!")
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "删除失败！",
			Data:   nil,
		},
	}
}
