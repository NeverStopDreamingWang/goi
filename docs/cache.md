# 内存缓存

**简体中文** | [English](cache.en.md) · [文档目录](README.md)

`goi.Cache` 是当前进程内的缓存管理器；`NewHTTPServer()` 的 `Server.Cache` 指向同一个包级对象。它不是 Redis 客户端，没有跨进程共享、持久化或分布式锁。配置应在启动服务前完成，不要把多个 Engine 当成相互隔离的缓存实例。

## 基本读写

```go
package main

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3"
)

type Profile struct {
	Name string
}

func main() {
	cache := goi.Cache
	cache.MaxSize = 0 // 不启用容量淘汰
	cache.ExpirationPolicy = goi.Periodic
	if err := cache.Set("profile:1", Profile{Name: "Alice"}, 60); err != nil {
		panic(err)
	}
	var result Profile
	if err := cache.Get("profile:1", &result); err != nil {
		panic(err)
	}
	fmt.Println(result.Name)
	cache.Del("profile:1")
}
```

值通过 `encoding/gob` 编码保存，读取时再解码到目标指针，不是直接保存和返回原始对象引用。结构体字段应导出；目标类型需与编码内容兼容。`any` 参数不表示所有 Go 值都可编码，编码或解码失败通过 `error` 返回。

| 方法 | 当前行为 |
| --- | --- |
| `Set(key, value, expires)` | 写入或覆盖值；`expires` 单位为秒。大于 0 设置 TTL，0 或负数不设置过期时间。 |
| `Get(key, &value)` | 命中未过期项时解码；缺失或已过期也返回 nil，且不重置目标变量。返回的 error 主要表示解码失败。 |
| `Has(key)` | 只检查键是否还在内部字典，不检查 TTL；过期但未清理的键也可能返回 true。 |
| `Del(key)` | 删除键，缺失时不做处理，无返回值。 |
| `DelExp(key)` | 仅在键已设置过期时间且已过期时删除，删除后返回 true；其余情况返回 false。 |

当前没有 `(value, found)`、`SetNX`、原子递增或缓存加载回调等接口。不要将 `Get` 的 nil error 当成命中证明，也不要将 `Has` 后接 `Get` 视为原子操作。需要区分“未命中”和“零值”的业务，应自行设计结果封装或使用具有明确命中语义的缓存实现。

## TTL 与后台清理

读取时始终检查 TTL，已过期值不会被解码给调用方。这与“何时从内存中清理”是两件事。

| `ExpirationPolicy` | 当前实现 |
| --- | --- |
| `Periodic`（默认） | `RunServer` 初始化缓存时自动把缓存管理器注册为启动任务；任务每 3 秒随机抽取部分项检查过期，停机时响应取消并退出。不是每次扫描全部键。 |
| `Scheduled` | `Set` 为 TTL 项创建计时回调，但当前实现有时间单位问题，不能依赖它在 TTL 到期时及时删除键，见下方说明。 |

独立调用 `Set`、`Get` 不会自行启动 Periodic 清理协程；上面的短程序仅依赖读取时的惰性删除。正常服务使用 `RunServer` 即会注册该内部任务，不需要重复调用 `RegisterOnStartup(goi.Cache)`。

当前实现边界：

- Periodic 的抽样数来自 `rand.Intn(cacheCount)`。只有一个键时抽样数固定为 0，因此不会被这一后台扫描主动删除；后续 `Get` 或 `DelExp` 仍可清理。
- Scheduled 在过期时间计算中使用秒，但 `time.AfterFunc(time.Duration(expires), ...)` 没有乘 `time.Second`。回调通常过早触发，此时 `DelExp` 判定尚未过期而不删除，也不会再次安排计时。
- 因此不要将任一模式用作精确定时任务或到期即释放内存的保证。本指南示例使用默认 Periodic，但同样不承诺精确清理时间。

## 容量与淘汰策略

`MaxSize` 单位为字节，默认 0 表示不限制。内部 `usedSize` 只累计键字符串与 gob 数据的长度，不包括 map、链表、锁和 Go 对象等开销，因此不是进程堆内存上限。

| `EvictPolicy` | 策略名称表达的目标 |
| --- | --- |
| `NoEviction`（默认） | 不淘汰已有项；超限时当前实现会记录日志并 panic，而不是由 `Set` 返回容量错误。写入已经发生，不会回滚。 |
| `AllKeysRandom` | 从所有键中移除项。 |
| `AllKeysLRU` / `AllKeysLFU` | 根据访问时间 / 访问计数的抽样平均值选择项。 |
| `VolatileRandom` | 仅移除设置了过期时间的项。 |
| `VolatileLRU` / `VolatileLFU` | 仅在设置了过期时间的项中按访问时间 / 计数选择。 |
| `VolatileTTL` | 名称表示优先移除临近过期项，但当前代码实际按访问时间 `lru` 取样比较，不是按 `expires` 排序。 |

这些名称不代表严格实现 Redis 或其他缓存的同名算法。当前容量淘汰存在以下限制，启用有限容量前必须针对实际数据验证：

- Random 策略先收集再删除，可能一次删除整批甚至全部候选项，不保证仅淘汰一项。
- LRU/LFU 的比较使用“小于平均值”，候选值相同时可能没有进展；当前循环没有统一的无进展退出处理。
- Volatile 策略没有可淘汰项时，可能无法降低用量；部分平均值计算还可能因候选数为 0 而 panic。
- 容量淘汰并非“返回 error 即可恢复”的 API。超限、写入结果和后续可用性不能按事务式语义理解。

如需确定性的容量限制、严格淘汰或持久服务能力，应先解决这些实现边界或采用满足要求的独立缓存方案。本次文档说明不改变实现。

## 并发使用边界

内部使用了读写锁与项级锁，但配置字段、容量统计和部分值读取没有统一的同步边界；不能据此承诺任意并发读写都无数据竞争。不要在服务运行时修改策略或容量配置；多步读改写不是原子操作。高并发访问应先对实际调用路径进行 race 和压力验证，不要把它用于必须具备强一致性的计数、锁或限流凭据。

## 源码与关联指南

- [cache.go](../cache.go)：缓存值、TTL、容量与淘汰实现。
- [server.go](../server.go)、[startup.go](../startup.go)：内部清理任务的注册与停止。
- [启动任务与优雅停机](lifecycle.md)、[时间与配置](configuration.md)：生命周期和时间来源。
