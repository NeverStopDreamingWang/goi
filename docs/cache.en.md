# In-process Cache

[简体中文](cache.md) | **English** · [Documentation](README.en.md)

`goi.Cache` belongs to the current process. `Server.Cache` from `NewHTTPServer()` references the same package-level object. It is not a Redis client and provides no cross-process sharing, persistence, or distributed locks. Configure it before serving; multiple Engines do not create isolated caches.

## Basic Operations

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
	cache.MaxSize = 0 // Disable capacity eviction.
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

Values are encoded with `encoding/gob` and decoded into destination pointers, rather than retained and returned as original object references. Struct fields should be exported and destination types must be compatible. An `any` parameter does not mean every Go value is encodable; encoding and decoding failures return errors.

| Method | Current behavior |
| --- | --- |
| `Set(key, value, expires)` | Inserts or replaces a value. `expires` is in seconds; positive values set TTL, while zero or negative values do not expire. |
| `Get(key, &value)` | Decodes a live hit. Missing and expired entries also return nil without resetting the destination. Returned errors mainly indicate decoding failure. |
| `Has(key)` | Checks only the internal dictionary, not TTL. An expired entry awaiting cleanup can still return true. |
| `Del(key)` | Removes an entry; a missing key is a no-op. No return value. |
| `DelExp(key)` | Removes an entry only if it has expired, returning true on deletion and false otherwise. |

There is no `(value, found)` result, `SetNX`, atomic increment, or cache loader callback. A nil `Get` error does not prove a hit, and `Has` followed by `Get` is not atomic. Applications that must distinguish misses from zero values need their own result design or a cache implementation with explicit hit semantics.

## TTL and Background Cleanup

Reads always check TTL and do not decode expired values into the caller's destination. This is separate from when an entry is removed from memory.

| `ExpirationPolicy` | Current implementation |
| --- | --- |
| `Periodic` (default) | Cache initialization in `RunServer` registers an internal startup task. It checks a random subset every 3 seconds and exits on shutdown cancellation. It does not scan every key on each tick. |
| `Scheduled` | `Set` schedules callbacks for TTL entries, but a duration-unit issue currently prevents reliable cleanup at the TTL deadline; see below. |

Calling `Set` or `Get` independently does not start the Periodic goroutine. The short program above relies only on lazy expiration during reads. A regular server registers the internal task through `RunServer`; do not register `goi.Cache` again with `RegisterOnStartup`.

Current limitations:

- Periodic obtains the sample count from `rand.Intn(cacheCount)`. With a single entry, the count is always zero, so that background scan never actively removes it. A later `Get` or `DelExp` can still clean it up.
- Scheduled computes deadlines in seconds but passes `time.Duration(expires)` to `time.AfterFunc` without multiplying by `time.Second`. The callback usually runs too early; `DelExp` then finds a live entry and does not schedule another check.
- Do not use either policy as a precise timer or a guarantee of immediate memory reclamation. The examples use the default Periodic policy without promising exact cleanup timing.

## Capacity and Eviction

`MaxSize` is in bytes; zero means unlimited. Internal `usedSize` counts only key strings and gob data, excluding maps, list nodes, locks, and Go object overhead. It is not a process heap limit.

| `EvictPolicy` | Goal expressed by its name |
| --- | --- |
| `NoEviction` (default) | Retains existing entries. The current implementation logs and panics on overflow rather than returning a capacity error from `Set`. The write has already happened and is not rolled back. |
| `AllKeysRandom` | Removes candidates from all keys. |
| `AllKeysLRU` / `AllKeysLFU` | Selects candidates using sampled average access times / access counters. |
| `VolatileRandom` | Removes only entries with expiration times. |
| `VolatileLRU` / `VolatileLFU` | Selects by access time / counters among entries with expiration times. |
| `VolatileTTL` | Its name implies expiration priority, but the current code samples access time (`lru`), not sorted expiration deadlines (`expires`). |

These names do not promise strict equivalents of Redis or other cache algorithms. Verify these limitations against real data before enabling finite capacity:

- Random policies collect candidates before deleting them and may remove a whole batch or all eligible entries, not just one.
- LRU/LFU comparisons use values below the average. Equal candidates may make no progress; there is no uniform no-progress exit condition.
- Volatile policies may fail to reduce usage if no eligible entries exist. Some average calculations may panic when the candidate count is zero.
- Eviction is not an API that can always recover through a returned error. Writes and overflow handling are not transactional.

Deterministic capacity limits, strict eviction, and durable service requirements require resolving these boundaries or choosing another implementation. This documentation does not change the code.

## Concurrency Boundaries

The implementation uses structural and item locks, but configuration, usage accounting, and some value reads do not share a complete synchronization boundary. This is not a guarantee that arbitrary concurrent reads and writes are race-free. Do not change capacity or policies while serving. Multi-step read/modify/write operations are not atomic. Run race and load checks for actual usage before relying on it in high-concurrency code; do not treat it as a strongly consistent counter, lock, or rate-limit credential store.

## Source and Related Guides

- [cache.go](../cache.go): values, TTL, capacity, and eviction.
- [server.go](../server.go), [startup.go](../startup.go): registration and cancellation of cleanup tasks.
- [Startup and Graceful Shutdown](lifecycle.en.md), [Configuration and Time](configuration.en.md): lifecycle and clock configuration.
