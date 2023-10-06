package hgee

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"reflect"
	"sync"
	"time"
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
	list      *list.List
	cache     map[string]*list.Element
}

type CacheItems struct {
	key       string
	valueType reflect.Type
	byteValue []byte
	exp       time.Time
}

// 初始化缓存
func newCache() *MetaCache {
	cache := &MetaCache{
		POLICY:    NOEVICTION,
		MAX_SIZE:  0,
		used_size: 0,
		list:      list.New(),
		cache:     make(map[string]*list.Element),
	}
	go cache.crontabDeleteExpired()
	return cache
}

// Get 查找键的值
func (cache *MetaCache) Get(key string) (any, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if element, ok := cache.cache[key]; ok {
		cache.list.MoveToBack(element)
		cacheItem := element.Value.(*CacheItems)

		nowTime := time.Now().In(Settings.LOCATION)                 // 获取当前时间
		if cacheItem.exp.IsZero() || cacheItem.exp.After(nowTime) { // 判断是否过期
			// 创建存储类型的新实例
			value := reflect.New(cacheItem.valueType).Interface()

			reader := bytes.NewReader(cacheItem.byteValue)
			decoder := gob.NewDecoder(reader)

			// 将字节解码回原始值类型
			err := decoder.Decode(value)
			if err != nil {
				return nil, false
			}
			return value, true
		} else { // 缓存过期删除
			cache.DelExp(key)
			return nil, false
		}
	}
	return nil, false
}

// Set 设置键值
func (cache *MetaCache) Set(key string, value any, exp int) {
	cache.mutex.Lock()

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(value)
	if err != nil {
		return
	}
	var expTime time.Time
	if exp > 0 {
		expTime = time.Now().In(Settings.LOCATION).Add(time.Second * time.Duration(exp)) // 获取过期时间
		time.AfterFunc(time.Duration(exp), func() {
			cache.DelExp(key)
		})
	}
	if element, ok := cache.cache[key]; ok {
		cache.list.MoveToBack(element)
		cacheItem := element.Value.(*CacheItems)
		cache.used_size += int64(buffer.Len() - len(cacheItem.byteValue))
		cacheItem.valueType = reflect.TypeOf(value)
		cacheItem.byteValue = buffer.Bytes()
		cacheItem.exp = expTime
	} else {
		cacheItem := &CacheItems{
			key:       key,
			valueType: reflect.TypeOf(value),
			byteValue: buffer.Bytes(),
			exp:       expTime,
		}
		element = cache.list.PushBack(cacheItem)
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
		cache.list.Remove(element)
		cacheItem := element.Value.(*CacheItems)
		delete(cache.cache, key)
		cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
	}
}

// DelExp 删除过期的 key， 如果 key 已过期则删除返回 true，未过期则不会删除返回 false
func (cache *MetaCache) DelExp(key string) bool {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if element, ok := cache.cache[key]; ok {
		cache.list.Remove(element)
		cacheItem := element.Value.(*CacheItems)
		nowTime := time.Now().In(Settings.LOCATION) // 获取当前时间
		if !cacheItem.exp.IsZero() && cacheItem.exp.Before(nowTime) {
			delete(cache.cache, key)
			cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
			return true
		}
	}
	return false
}

// 内存淘汰
func (cache *MetaCache) Elimination() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	switch cache.POLICY {
	case NOEVICTION:
		panic("超出设置最大缓存！")
	case ALLKEYS_LRU:
		element := cache.list.Front()
		if element != nil {
			cache.list.Remove(element)
			cacheItem := element.Value.(*CacheItems)
			delete(cache.cache, cacheItem.key)
			cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
		}
	}
}

// 定期删除
func (cache *MetaCache) crontabDeleteExpired() {
	for true {
		if cache.list == nil || cache.used_size == 0 {
			time.Sleep(10 * time.Second)
			continue
		}
		count := 20
		expired := 0
		for key, _ := range cache.cache {
			if count <= 0 {
				break
			}
			if ok := cache.DelExp(key); ok {
				expired++
			}
			count--
		}
		if expired < 5 {
			time.Sleep(10 * time.Second)
		}
	}
}
