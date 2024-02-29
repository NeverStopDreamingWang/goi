package goi

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

type EvictPolicy uint8      // 缓存淘汰策略
type ExpirationPolicy uint8 // 过期策略

// 缓存淘汰策略
const (
	NOEVICTION      EvictPolicy = iota // 直接抛出错误，不淘汰任何已经存在的键
	ALLKEYS_RANDOM                     // 所有的键 随机移除 key
	ALLKEYS_LRU                        // 所有的键 移除最近最少使用的 key
	ALLKEYS_LFU                        // 所有的键 移除最近最不频繁使用的 key
	VOLATILE_RANDOM                    // 所有设置了过期时间的键 随机移除 key
	VOLATILE_LRU                       // 所有设置了过期时间的键 移除最近最少使用的 key
	VOLATILE_LFU                       // 所有设置了过期时间的键 移除最近最不频繁使用的 key
	VOLATILE_TTL                       // 所有设置了过期时间的键 移除快过期的 key
)

var evictPolicyNames = map[EvictPolicy]string{
	NOEVICTION:      "NOEVICTION",
	ALLKEYS_RANDOM:  "ALLKEYS_RANDOM",
	ALLKEYS_LRU:     "ALLKEYS_LRU",
	ALLKEYS_LFU:     "ALLKEYS_LFU",
	VOLATILE_RANDOM: "VOLATILE_RANDOM",
	VOLATILE_LRU:    "VOLATILE_LRU",
	VOLATILE_LFU:    "VOLATILE_LFU",
	VOLATILE_TTL:    "VOLATILE_TTL",
}

// 过期策略
const (
	// 默认惰性删除: 在获取某个 key 的时候，检查这个 key 如果设置了过期时间并且过期了，是的话就删除。
	// 定期删除: 默认每隔 1s 就随机抽取一些设置了过期时间的 key，检查其是否过期，如果有过期就删除。
	// 定时删除: 某个设置了过期时间的 key，到期后立即删除。
	PERIODIC  ExpirationPolicy = iota // 定期删除
	SCHEDULED                         // 定时删除

)

var expirationPolicyNames = map[ExpirationPolicy]string{
	PERIODIC:  "PERIODIC",
	SCHEDULED: "SCHEDULED",
}

type metaCache struct {
	EVICT_POLICY      EvictPolicy      // 缓存淘汰策略
	EXPIRATION_POLICY ExpirationPolicy // 过期策略
	MAX_SIZE          int64
	used_size         int64
	list              *list.List
	dict              map[string]*list.Element
}

type cacheItems struct {
	key       string
	valueType reflect.Type
	byteValue []byte
	expires   time.Time
	mutex     sync.Mutex
	lru       time.Time
	lfu       uint8
}

// 创建缓存
func newCache() *metaCache {
	cache := &metaCache{
		EVICT_POLICY:      NOEVICTION,
		EXPIRATION_POLICY: PERIODIC,
		MAX_SIZE:          0,
		used_size:         0,
		list:              list.New(),
		dict:              make(map[string]*list.Element),
	}
	return cache
}

// 初始化缓存
func (cache *metaCache) initCache() {
	engine.Log.Log(meta, fmt.Sprintf("缓存大小: %v", formatBytes(engine.Cache.MAX_SIZE)))
	engine.Log.Log(meta, fmt.Sprintf("- 淘汰策略: %v", evictPolicyNames[engine.Cache.EVICT_POLICY]))
	engine.Log.Log(meta, fmt.Sprintf("- 过期策略: %v", expirationPolicyNames[engine.Cache.EXPIRATION_POLICY]))

	if cache.EXPIRATION_POLICY == PERIODIC {
		go cache.cachePeriodicDeleteExpires()
	}
}

// Get 查找键的值
func (cache *metaCache) Get(key string) (any, bool) {
	if element, ok := cache.dict[key]; ok {
		cacheItem := element.Value.(*cacheItems)

		nowTime := time.Now().In(Settings.LOCATION)                         // 获取当前时间
		if cacheItem.expires.IsZero() || cacheItem.expires.After(nowTime) { // 判断是否过期
			cacheItem.mutex.Lock()
			// 创建存储类型的新实例
			value := reflect.New(cacheItem.valueType).Interface()

			reader := bytes.NewReader(cacheItem.byteValue)
			decoder := gob.NewDecoder(reader)

			cacheItem.lru = nowTime
			cacheItem.lfu = LFULogIncr(cacheItem.lfu)
			cacheItem.mutex.Unlock()

			// 将字节解码回原始值类型
			err := decoder.Decode(value)
			if err != nil {
				return nil, false
			}
			return value, true
		}
		// 缓存过期删除
		cache.DelExp(key)
		return nil, false
	}
	return nil, false
}

// Set 设置键值
func (cache *metaCache) Set(key string, value any, expires int) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(value)
	if err != nil {
		return
	}
	var expiresTime time.Time
	if expires > 0 {
		expiresTime = time.Now().In(Settings.LOCATION).Add(time.Second * time.Duration(expires)) // 获取过期时间

		if cache.EXPIRATION_POLICY == SCHEDULED { // 定时删除
			time.AfterFunc(time.Duration(expires), func() { cache.DelExp(key) })
		}
	}
	if element, ok := cache.dict[key]; ok {
		cacheItem := element.Value.(*cacheItems)
		cacheItem.mutex.Lock()

		cache.used_size += int64(buffer.Len() - len(cacheItem.byteValue))
		cacheItem.valueType = reflect.TypeOf(value)
		cacheItem.byteValue = buffer.Bytes()
		cacheItem.expires = expiresTime
		cacheItem.mutex.Unlock()
	} else {
		cacheItem := &cacheItems{
			key:       key,
			valueType: reflect.TypeOf(value),
			byteValue: buffer.Bytes(),
			expires:   expiresTime,
			mutex:     sync.Mutex{},
			lru:       time.Now().In(Settings.LOCATION),
			lfu:       5,
		}
		element = cache.list.PushFront(cacheItem)
		cache.used_size += int64(len(key) + buffer.Len())
		cache.dict[key] = element
	}

	for cache.MAX_SIZE != 0 && cache.MAX_SIZE < cache.used_size {
		cache.cacheEvict()
	}
}

// Del 删除键值
func (cache *metaCache) Del(key string) {
	if element, ok := cache.dict[key]; ok {
		cacheItem := element.Value.(*cacheItems)

		cache.list.Remove(element)
		delete(cache.dict, key)
		cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
	}
}

// DelExp 删除过期的 key， 如果 key 已过期则删除返回 true，未过期则不会删除返回 false
func (cache *metaCache) DelExp(key string) bool {
	if element, ok := cache.dict[key]; ok {
		cacheItem := element.Value.(*cacheItems)

		cache.list.Remove(element)
		nowTime := time.Now().In(Settings.LOCATION) // 获取当前时间
		if !cacheItem.expires.IsZero() && cacheItem.expires.Before(nowTime) {
			delete(cache.dict, key)
			cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
			return true
		}
	}
	return false
}

// 缓存淘汰
func (cache *metaCache) cacheEvict() {
	if len(cache.dict) == 0 {
		return
	}

	switch cache.EVICT_POLICY {
	case NOEVICTION: // 直接返回错误，不淘汰任何已经存在的键
		panic("超出设置最大缓存！")
	case ALLKEYS_RANDOM: // 所有的键 随机移除 key
		for key, _ := range cache.dict {
			if cache.MAX_SIZE > cache.used_size {
				break
			}
			cache.Del(key)
		}
	case ALLKEYS_LRU: // 所有的键 移除最近最少使用的 key
		var averageUnix int64
		var num int64 = 100
		for cache.MAX_SIZE < cache.used_size {
			var totalUnix int64
			var i int64 = 0
			for _, element := range cache.dict {
				cacheItem := element.Value.(*cacheItems)
				totalUnix += cacheItem.lru.Unix()
				i++
				if i >= num {
					break
				}
			}
			if averageUnix != 0 { // 合并两个平均值
				averageUnix = (averageUnix + (totalUnix / i)) / 2
			} else {
				averageUnix = totalUnix / i
			}

			i = 0
			for _, element := range cache.dict {
				if cache.MAX_SIZE > cache.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)

				if averageUnix > cacheItem.lru.Unix() {
					cache.list.Remove(element)
					delete(cache.dict, cacheItem.key)
					cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
				}
				i++
				if i >= num {
					break
				}
			}
		}

	case ALLKEYS_LFU: // 所有的键 移除最近最不频繁使用的 key
		var averageCounter uint8
		var num int64 = 100
		for cache.MAX_SIZE < cache.used_size {
			var totalCounter int64
			var i int64 = 0
			for _, element := range cache.dict {
				cacheItem := element.Value.(*cacheItems)
				totalCounter += int64(cacheItem.lfu)
				i++
				if i >= num {
					break
				}
			}
			if averageCounter != 0 { // 合并两个平均值
				averageCounter = (averageCounter + uint8(totalCounter/i)) / 2
			} else {
				averageCounter = uint8(totalCounter / i)
			}

			i = 0
			for _, element := range cache.dict {
				if cache.MAX_SIZE > cache.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)

				if averageCounter > cacheItem.lfu {
					cache.list.Remove(element)
					delete(cache.dict, cacheItem.key)
					cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
				}
				i++
				if i >= num {
					break
				}
			}
		}
	case VOLATILE_RANDOM: // 所有设置了过期时间的键 随机移除 key
		for key, element := range cache.dict {
			if cache.MAX_SIZE > cache.used_size {
				break
			}
			if cacheItem := element.Value.(*cacheItems); cacheItem.expires.IsZero() {
				continue
			}
			cache.Del(key)
		}
	case VOLATILE_LRU: // 所有设置了过期时间的键 移除最近最少使用的 key
		var averageUnix int64
		var num int64 = 100
		for cache.MAX_SIZE < cache.used_size {
			var totalUnix int64
			var i int64 = 0
			for _, element := range cache.dict {
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}
				totalUnix += cacheItem.lru.Unix()
				i++
				if i >= num {
					break
				}
			}
			if averageUnix != 0 { // 合并两个平均值
				averageUnix = (averageUnix + (totalUnix / i)) / 2
			} else {
				averageUnix = totalUnix / i
			}

			i = 0
			for _, element := range cache.dict {
				if cache.MAX_SIZE > cache.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}

				if averageUnix > cacheItem.lru.Unix() {
					cache.list.Remove(element)
					delete(cache.dict, cacheItem.key)
					cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
				}
				i++
				if i >= num {
					break
				}
			}
		}
	case VOLATILE_LFU: // 所有设置了过期时间的键 移除最近最不频繁使用的 key
		var averageCounter uint8
		var num int64 = 100
		for cache.MAX_SIZE < cache.used_size {
			var totalCounter int64
			var i int64 = 0
			for _, element := range cache.dict {
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}
				totalCounter += int64(cacheItem.lfu)
				i++
				if i >= num {
					break
				}
			}
			if averageCounter != 0 { // 合并两个平均值
				averageCounter = (averageCounter + uint8(totalCounter/i)) / 2
			} else {
				averageCounter = uint8(totalCounter / i)
			}

			i = 0
			for _, element := range cache.dict {
				if cache.MAX_SIZE > cache.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}

				if averageCounter > cacheItem.lfu {
					cache.list.Remove(element)
					delete(cache.dict, cacheItem.key)
					cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
				}
				i++
				if i >= num {
					break
				}
			}
		}
	case VOLATILE_TTL: // 所有设置了过期时间的键 移除快过期的 key
		var averageUnix int64
		var num int64 = 200
		for cache.MAX_SIZE < cache.used_size {
			var totalUnix int64
			var i int64 = 0
			for _, element := range cache.dict {
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}
				totalUnix += cacheItem.lru.Unix()
				i++
				if i >= num {
					break
				}
			}
			if averageUnix != 0 { // 合并两个平均值
				averageUnix = (averageUnix + (totalUnix / i)) / 2
			} else {
				averageUnix = totalUnix / i
			}

			i = 0
			for _, element := range cache.dict {
				if cache.MAX_SIZE > cache.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}

				if averageUnix > cacheItem.lru.Unix() {
					cache.list.Remove(element)
					delete(cache.dict, cacheItem.key)
					cache.used_size -= int64(len(cacheItem.key) + len(cacheItem.byteValue))
				}
				i++
				if i >= num {
					break
				}
			}
		}
	}
}

// 定期删除
func (cache *metaCache) cachePeriodicDeleteExpires() {
	for cache.EXPIRATION_POLICY == PERIODIC {
		if len(cache.dict) == 0 {
			time.Sleep(3 * time.Second)
			continue
		}
		count := rand.Intn(len(cache.dict))
		for key, _ := range cache.dict {
			if count <= 0 {
				break
			}
			cache.DelExp(key)
			count--
		}
		time.Sleep(1 * time.Second)
	}
}

const (
	LFUInitValue    = 5   // LFU初始化值
	LFULogFactor    = 0.1 // LFU对数增长因子
	MaxCounterValue = 255 // 计数器最大值
)

// 计算 LFU 递增值
func LFULogIncr(counter uint8) uint8 {
	if counter == MaxCounterValue {
		return MaxCounterValue
	}

	r := rand.Float64() // 生成[0.0, 1.0)范围内的随机浮点数
	baseVal := float64(counter - LFUInitValue)
	if baseVal < 0 {
		baseVal = 0
	}

	p := 1.0 / (baseVal*LFULogFactor + 1)
	if r < p {
		counter++
	}
	return counter
}
