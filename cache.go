package hgee

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"reflect"
	"sync"
)

type cachePolicy uint8

const (
	NOEVICTION      cachePolicy = iota // 直接返回错误，不淘汰任何已经存在的键
	ALLKEYS_RANDOM                     // 所有的键 随机淘汰键
	ALLKEYS_LRU                        // 所有的键 使用 LRU 算法淘汰
	ALLKEYS_LFU                        // 所有的键 使用 LFU 算法淘汰
	VOLATILE_RANDOM                    // 所有设置了过期时间的键 随机淘汰键
	VOLATILE_LRU                       // 所有设置了过期时间的键 使用 LRU 算法淘汰
	VOLATILE_LFU                       // 所有设置了过期时间的键 使用 LFU 算法淘汰
)

type MetaCache struct {
	POLICY    cachePolicy
	MAX_SIZE  int64
	used_size int64
	mutex     sync.Mutex
	data      *list.List
	cache     map[string]*list.Element
}

type CacheData struct {
	key       string
	valueType reflect.Type
	byteValue []byte
}

// 初始化缓存
func newCache() *MetaCache {
	return &MetaCache{
		POLICY:    NOEVICTION,
		MAX_SIZE:  0,
		used_size: 0,
		data:      list.New(),
		cache:     make(map[string]*list.Element),
	}
}

// Get 查找键的值
func (cache *MetaCache) Get(key string) (any, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if element, ok := cache.cache[key]; ok {
		cache.data.MoveToBack(element)
		cacheData := element.Value.(*CacheData)

		// Create a new instance of the stored type
		value := reflect.New(cacheData.valueType).Interface()

		reader := bytes.NewReader(cacheData.byteValue)
		decoder := gob.NewDecoder(reader)

		// Decode the bytes back to the original value type
		err := decoder.Decode(value)
		if err != nil {
			return nil, false
		}
		return value, true
	}
	return nil, false
}

// Set 设置键值
func (cache *MetaCache) Set(key string, value any) {
	cache.mutex.Lock()

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(value)
	if err != nil {
		return
	}

	if element, ok := cache.cache[key]; ok {
		cache.data.MoveToBack(element)
		cacheData := element.Value.(*CacheData)
		cache.used_size += int64(buffer.Len() - len(cacheData.byteValue))
		cacheData.valueType = reflect.TypeOf(value)
		cacheData.byteValue = buffer.Bytes()
	} else {
		cacheData := &CacheData{
			key:       key,
			valueType: reflect.TypeOf(value),
			byteValue: buffer.Bytes(),
		}
		element = cache.data.PushBack(cacheData)
		cache.used_size += int64(len(key) + buffer.Len())
		cache.cache[key] = element
	}
	cache.mutex.Unlock()

	for cache.MAX_SIZE != 0 && cache.MAX_SIZE < cache.used_size {
		cache.Elimination()
	}
}

// Del 删除键值
func (cache *MetaCache) Del(key string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if element, ok := cache.cache[key]; ok {
		cache.data.Remove(element)
		cacheData := element.Value.(*CacheData)
		delete(cache.cache, key)
		cache.used_size -= int64(len(cacheData.key) + len(cacheData.byteValue))
	}
}

// 内存淘汰
func (cache *MetaCache) Elimination() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	switch cache.POLICY {
	case NOEVICTION:
		panic("超出设置最大缓存！")
	case ALLKEYS_LRU:
		element := cache.data.Front()
		if element != nil {
			cache.data.Remove(element)
			cacheData := element.Value.(*CacheData)
			delete(cache.cache, cacheData.key)
			cache.used_size -= int64(len(cacheData.key) + len(cacheData.byteValue))
		}
	}
}
