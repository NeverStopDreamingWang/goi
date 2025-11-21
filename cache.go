package goi

import (
	"bytes"
	"container/list"
	"context"
	"encoding/gob"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// EvictPolicy 缓存淘汰策略类型
type EvictPolicy uint8

// ExpirationPolicy 缓存过期策略类型
type ExpirationPolicy uint8

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

// cacheManager 缓存管理器
//
// 字段:
//   - EVICT_POLICY EvictPolicy: 缓存淘汰策略
//   - EXPIRATION_POLICY ExpirationPolicy: 过期策略
//   - MAX_SIZE int64: 缓存最大容量(字节)
//   - used_size int64: 当前已使用容量
//   - list *list.List: 缓存项双向链表
//   - dict map[string]*list.Element: 缓存键值映射
//   - lock sync.RWMutex: 读写锁
type cacheManager struct {
	EVICT_POLICY      EvictPolicy
	EXPIRATION_POLICY ExpirationPolicy
	MAX_SIZE          int64
	used_size         int64
	list              *list.List
	dict              map[string]*list.Element
	lock              sync.RWMutex
}

// cacheItems 缓存项
//
// 字段:
//   - key string: 缓存键
//   - valueType reflect.Type: 值类型
//   - bytes []byte: 序列化后的值
//   - expires time.Time: 过期时间
//   - mutex sync.Mutex: 互斥锁
//   - lru time.Time: 最近访问时间
//   - lfu uint8: 访问频率计数
type cacheItems struct {
	key       string
	valueType reflect.Type
	bytes     []byte
	expires   time.Time
	mutex     sync.Mutex
	lru       time.Time
	lfu       uint8
}

// newCacheManager 创建新的缓存管理器
//
// 返回:
//   - *cacheManager: 新创建的缓存管理器实例
func newCacheManager() *cacheManager {
	cache := &cacheManager{
		EVICT_POLICY:      NOEVICTION,
		EXPIRATION_POLICY: PERIODIC,
		MAX_SIZE:          0,
		used_size:         0,
		list:              list.New(),
		dict:              make(map[string]*list.Element),
		lock:              sync.RWMutex{},
	}
	return cache
}

// initCache 初始化缓存配置
func (self *cacheManager) initCache() {
	MaxSizeMsg := i18n.T("server.cache.max_size", map[string]interface{}{
		"max_size": FormatBytes(self.MAX_SIZE),
	})
	Log.Log(meta, MaxSizeMsg)
	EvictPolicyMsg := i18n.T("server.cache.evict_policy", map[string]interface{}{
		"evict_policy": evictPolicyNames[self.EVICT_POLICY],
	})
	Log.Log(meta, EvictPolicyMsg)
	ExpirationPolicyMsg := i18n.T("server.cache.expiration_policy", map[string]interface{}{
		"expiration_policy": expirationPolicyNames[self.EXPIRATION_POLICY],
	})
	Log.Log(meta, ExpirationPolicyMsg)
}

// get 获取缓存项
//
// 参数:
//   - key string: 缓存键
//
// 返回:
//   - *list.Element: 缓存项元素
//   - bool: 是否存在该键
func (self *cacheManager) get(key string) (*list.Element, bool) {
	self.lock.RLock()
	element, ok := self.dict[key]
	self.lock.RUnlock()
	return element, ok
}

// set 设置缓存项
//
// 参数:
//   - cacheItem *cacheItems: 要设置的缓存项
func (self *cacheManager) set(cacheItem *cacheItems) {
	self.lock.Lock()
	element := self.list.PushFront(cacheItem)
	self.dict[cacheItem.key] = element
	self.used_size += int64(len(cacheItem.key) + len(cacheItem.bytes))
	self.lock.Unlock()
}

// del 删除缓存项
//
// 参数:
//   - element *list.Element: 要删除的缓存项元素
func (self *cacheManager) del(element *list.Element) {
	cacheItem := element.Value.(*cacheItems)
	self.lock.Lock()
	self.list.Remove(element)
	delete(self.dict, cacheItem.key)
	self.used_size -= int64(len(cacheItem.key) + len(cacheItem.bytes))
	self.lock.Unlock()
}

// Has 检查键是否存在
//
// 参数:
//   - key string: 要检查的键
//
// 返回:
//   - bool: 是否存在该键
func (self *cacheManager) Has(key string) bool {
	self.lock.RLock()
	_, ok := self.dict[key]
	self.lock.RUnlock()
	return ok
}

// Get 获取缓存值
//
// 参数:
//   - key string: 缓存键
//   - value interface{}: 用于存储解码后的值的指针
//
// 返回:
//   - error: 获取过程中的错误信息
func (self *cacheManager) Get(key string, value interface{}) error {
	if element, ok := self.get(key); ok {
		cacheItem := element.Value.(*cacheItems)

		nowTime := GetTime()                                                // 获取当前时间
		if cacheItem.expires.IsZero() || cacheItem.expires.After(nowTime) { // 判断是否过期
			cacheItem.mutex.Lock()
			cacheItem.lru = nowTime
			cacheItem.lfu = LFULogIncr(cacheItem.lfu)
			cacheItem.mutex.Unlock()

			// 将字节解码回原始值类型
			reader := bytes.NewReader(cacheItem.bytes)
			decoder := gob.NewDecoder(reader)
			err := decoder.Decode(value)
			if err != nil {
				return err
			}
			return nil
		}
		// 缓存过期删除
		self.DelExp(key)
		return nil
	}
	return nil
}

// Set 设置缓存键值对
//
// 参数:
//   - key string: 缓存键
//   - value interface{}: 要缓存的值
//   - expires int: 过期时间(秒)，0表示永不过期
//
// 返回:
//   - error: 设置过程中的错误信息
func (self *cacheManager) Set(key string, value interface{}, expires int) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(value)
	if err != nil {
		return err
	}
	var expiresTime time.Time
	if expires > 0 {
		expiresTime = GetTime().Add(time.Second * time.Duration(expires)) // 获取过期时间
		if self.EXPIRATION_POLICY == SCHEDULED {                          // 定时删除
			time.AfterFunc(time.Duration(expires), func() { self.DelExp(key) })
		}
	}
	if element, ok := self.get(key); ok {
		cacheItem := element.Value.(*cacheItems)
		cacheItem.mutex.Lock()
		self.used_size += int64(buffer.Len() - len(cacheItem.bytes))
		cacheItem.valueType = reflect.TypeOf(value)
		cacheItem.bytes = buffer.Bytes()
		cacheItem.expires = expiresTime
		cacheItem.mutex.Unlock()
	} else {
		cacheItem := &cacheItems{
			key:       key,
			valueType: reflect.TypeOf(value),
			bytes:     buffer.Bytes(),
			expires:   expiresTime,
			mutex:     sync.Mutex{},
			lru:       GetTime(),
			lfu:       5,
		}
		self.set(cacheItem)
	}

	for self.MAX_SIZE != 0 && self.MAX_SIZE < self.used_size {
		self.cacheEvict()
	}
	return nil
}

// Del 删除缓存键
//
// 参数:
//   - key string: 要删除的缓存键
func (self *cacheManager) Del(key string) {
	if element, ok := self.get(key); ok {
		self.del(element)
	}
}

// DelExp 删除过期的缓存键
//
// 参数:
//   - key string: 要检查的缓存键
//
// 返回:
//   - bool: 如果键已过期并被删除返回true，否则返回false
func (self *cacheManager) DelExp(key string) bool {
	if element, ok := self.get(key); ok {
		cacheItem := element.Value.(*cacheItems)

		nowTime := GetTime() // 获取当前时间
		if !cacheItem.expires.IsZero() && cacheItem.expires.Before(nowTime) {
			self.del(element)
			return true
		}
	}
	return false
}

// cacheEvict 执行缓存淘汰
func (self *cacheManager) cacheEvict() {
	if len(self.dict) == 0 {
		return
	}

	switch self.EVICT_POLICY {
	case NOEVICTION: // 直接返回错误，不淘汰任何已经存在的键
		NoEvictionMsg := i18n.T("server.cache.noeviction", map[string]interface{}{
			"max_size": FormatBytes(self.MAX_SIZE),
		})
		Log.Error(NoEvictionMsg)
		panic(NoEvictionMsg)
	case ALLKEYS_RANDOM: // 所有的键 随机移除 key
		var elementRemove []*list.Element
		self.lock.RLock()
		for _, element := range self.dict {
			if self.MAX_SIZE > self.used_size {
				break
			}
			elementRemove = append(elementRemove, element)
		}
		self.lock.RUnlock()
		for _, element := range elementRemove {
			self.del(element)
		}
	case ALLKEYS_LRU: // 所有的键 移除最近最少使用的 key
		var averageUnix int64
		var num int64 = 100
		for self.MAX_SIZE < self.used_size {
			var totalUnix int64
			var i int64 = 0

			self.lock.RLock()
			for _, element := range self.dict {
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
			var elementRemove []*list.Element
			for _, element := range self.dict {
				if self.MAX_SIZE > self.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)

				if averageUnix > cacheItem.lru.Unix() {
					elementRemove = append(elementRemove, element)
				}
				i++
				if i >= num {
					break
				}
			}
			self.lock.RUnlock()
			for _, element := range elementRemove {
				self.del(element)
			}
		}

	case ALLKEYS_LFU: // 所有的键 移除最近最不频繁使用的 key
		var averageCounter uint8
		var num int64 = 100
		for self.MAX_SIZE < self.used_size {
			var totalCounter int64
			var i int64 = 0

			self.lock.RLock()
			for _, element := range self.dict {
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
			var elementRemove []*list.Element
			for _, element := range self.dict {
				if self.MAX_SIZE > self.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)

				if averageCounter > cacheItem.lfu {
					elementRemove = append(elementRemove, element)
				}
				i++
				if i >= num {
					break
				}
			}
			self.lock.RUnlock()
			for _, element := range elementRemove {
				self.del(element)
			}
		}
	case VOLATILE_RANDOM: // 所有设置了过期时间的键 随机移除 key
		var elementRemove []*list.Element
		self.lock.RLock()
		for _, element := range self.dict {
			if self.MAX_SIZE > self.used_size {
				break
			}
			cacheItem := element.Value.(*cacheItems)
			if cacheItem.expires.IsZero() {
				continue
			}
			elementRemove = append(elementRemove, element)
		}
		self.lock.RUnlock()
		for _, element := range elementRemove {
			self.del(element)
		}
	case VOLATILE_LRU: // 所有设置了过期时间的键 移除最近最少使用的 key
		var averageUnix int64
		var num int64 = 100
		for self.MAX_SIZE < self.used_size {
			var totalUnix int64
			var i int64 = 0

			self.lock.RLock()
			for _, element := range self.dict {
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
			var elementRemove []*list.Element
			for _, element := range self.dict {
				if self.MAX_SIZE > self.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}

				if averageUnix > cacheItem.lru.Unix() {
					elementRemove = append(elementRemove, element)
				}
				i++
				if i >= num {
					break
				}
			}
			self.lock.RUnlock()
			for _, element := range elementRemove {
				self.del(element)
			}
		}
	case VOLATILE_LFU: // 所有设置了过期时间的键 移除最近最不频繁使用的 key
		var averageCounter uint8
		var num int64 = 100
		for self.MAX_SIZE < self.used_size {
			var totalCounter int64
			var i int64 = 0

			self.lock.RLock()
			for _, element := range self.dict {
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
			var elementRemove []*list.Element
			for _, element := range self.dict {
				if self.MAX_SIZE > self.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}

				if averageCounter > cacheItem.lfu {
					elementRemove = append(elementRemove, element)
				}
				i++
				if i >= num {
					break
				}
			}
			self.lock.RUnlock()
			for _, element := range elementRemove {
				self.del(element)
			}
		}
	case VOLATILE_TTL: // 所有设置了过期时间的键 移除快过期的 key
		var averageUnix int64
		var num int64 = 200
		for self.MAX_SIZE < self.used_size {
			var totalUnix int64
			var i int64 = 0

			self.lock.RLock()
			for _, element := range self.dict {
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
			var elementRemove []*list.Element
			for _, element := range self.dict {
				if self.MAX_SIZE > self.used_size {
					break
				}
				cacheItem := element.Value.(*cacheItems)
				if cacheItem.expires.IsZero() {
					continue
				}

				if averageUnix > cacheItem.lru.Unix() {
					elementRemove = append(elementRemove, element)
				}
				i++
				if i >= num {
					break
				}
			}
			self.lock.RUnlock()
			for _, element := range elementRemove {
				self.del(element)
			}
		}
	}
}

// cachePeriodicDeleteExpires 定期删除过期缓存项
//
// 参数:
//   - ctx context.Context: 上下文对象，用于控制goroutine退出
//   - wg *sync.WaitGroup: 等待组，用于同步goroutine
func (cache *metaCache) cachePeriodicDeleteExpires(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if cache.EXPIRATION_POLICY != PERIODIC {
				return
			}
			cache.lock.RLock()
			cacheCount := len(cache.dict)
			if cacheCount <= 0 {
				cache.lock.RUnlock()
				time.Sleep(3 * time.Second)
				continue
			}
			count := rand.Intn(cacheCount)
			var elementRemove []*list.Element
			for _, element := range cache.dict {
				if count <= 0 {
					break
				}
				cacheItem := element.Value.(*cacheItems)

				nowTime := GetTime() // 获取当前时间
				if !cacheItem.expires.IsZero() && cacheItem.expires.Before(nowTime) {
					elementRemove = append(elementRemove, element)
				}
				count--
			}
			cache.lock.RUnlock()
			for _, element := range elementRemove {
				cache.del(element)
			}
			time.Sleep(3 * time.Second)
		}
	}
}

const (
	LFUInitValue    = 5   // LFU初始化值
	LFULogFactor    = 0.1 // LFU对数增长因子
	MaxCounterValue = 255 // 计数器最大值
)

// LFULogIncr 计算LFU访问频率的递增值
//
// 参数:
//   - counter uint8: 当前计数值
//
// 返回:
//   - uint8: 更新后的计数值
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
