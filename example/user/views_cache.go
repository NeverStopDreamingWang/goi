package user

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"net/http"
)

// 设置一条数据到缓存
func TestCacheSet(request *hgee.Request) any {
	key := request.BodyParams.Get("key").(string)
	Value := request.BodyParams.Get("value").(string)
	fmt.Printf("key: %v\n", key)
	fmt.Printf("Value: %v\n", Value)
	hgee.Cache.Set(key, Value)

	return hgee.Response{
		Status: http.StatusOK,
		Data:   "设置成功！",
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
			Data:   "key 值不存在！",
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
		Data:   value,
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
			Data:   "删除成功！",
		}
	}
	fmt.Printf("删除失败 value: %v\n", *value.(*string))
	return hgee.Response{
		Status: http.StatusOK,
		Data:   "删除失败！",
	}
}
