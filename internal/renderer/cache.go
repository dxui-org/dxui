package renderer

// CacheStats reports exact configured-accounting bytes, entries, and
// evictions. Native driver overhead is measured separately.
type CacheStats struct {
	BudgetBytes int
	Bytes       int
	Entries     int
	Evictions   uint64
}

type cacheEntry[V any] struct {
	value V
	bytes int
	refs  int
	stamp uint64
}

// ByteCache is a byte-bounded LRU. Only zero-reference entries are evicted
// during normal operation. Clear releases all values during renderer teardown.
type ByteCache[K comparable, V any] struct {
	budget  int
	bytes   int
	stamp   uint64
	evicted uint64
	values  map[K]cacheEntry[V]
	release func(V)
}

// NewByteCache creates a cache. A negative budget is normalized to zero.
func NewByteCache[K comparable, V any](budget int, release func(V)) *ByteCache[K, V] {
	if budget < 0 {
		budget = 0
	}
	return &ByteCache[K, V]{budget: budget, release: release}
}

// Get returns and marks a value most-recently used without retaining it.
func (cache *ByteCache[K, V]) Get(key K) (V, bool) {
	entry, ok := cache.values[key]
	if !ok {
		var zero V
		return zero, false
	}
	cache.stamp++
	entry.stamp = cache.stamp
	cache.values[key] = entry
	return entry.value, true
}

// Put inserts a zero-reference value. Values larger than the total budget are
// immediately released and not cached.
func (cache *ByteCache[K, V]) Put(key K, value V, bytes int) bool {
	return cache.put(key, value, bytes, 0)
}

// PutRetained inserts a value with one active reference. Active working-set
// values may exceed the zero-reference cache budget; Release removes them as
// soon as they cannot fit after becoming unreferenced.
func (cache *ByteCache[K, V]) PutRetained(key K, value V, bytes int) bool {
	return cache.put(key, value, bytes, 1)
}

func (cache *ByteCache[K, V]) put(key K, value V, bytes, refs int) bool {
	if bytes < 0 || refs == 0 && (bytes > cache.budget || cache.budget == 0) {
		cache.releaseValue(value)
		return false
	}
	if old, exists := cache.values[key]; exists {
		if old.refs != 0 {
			cache.releaseValue(value)
			return false
		}
		cache.remove(key, old)
	}
	cache.makeRoom(bytes)
	if cache.bytes+bytes > cache.budget && refs == 0 {
		cache.releaseValue(value)
		return false
	}
	if cache.values == nil {
		cache.values = make(map[K]cacheEntry[V])
	}
	cache.stamp++
	cache.values[key] = cacheEntry[V]{value: value, bytes: bytes, refs: refs, stamp: cache.stamp}
	cache.bytes += bytes
	return true
}

// Retain pins an existing value against capacity eviction.
func (cache *ByteCache[K, V]) Retain(key K) bool {
	entry, ok := cache.values[key]
	if !ok {
		return false
	}
	entry.refs++
	cache.stamp++
	entry.stamp = cache.stamp
	cache.values[key] = entry
	return true
}

// Release drops one reference. Extra releases are ignored deterministically.
func (cache *ByteCache[K, V]) Release(key K) {
	entry, ok := cache.values[key]
	if !ok || entry.refs == 0 {
		return
	}
	entry.refs--
	cache.values[key] = entry
	cache.makeRoom(0)
}

// Stats returns current capacity accounting.
func (cache *ByteCache[K, V]) Stats() CacheStats {
	return CacheStats{BudgetBytes: cache.budget, Bytes: cache.bytes, Entries: len(cache.values), Evictions: cache.evicted}
}

// Clear releases every value, including pinned values, during owner teardown.
func (cache *ByteCache[K, V]) Clear() {
	for key, entry := range cache.values {
		cache.remove(key, entry)
	}
}

func (cache *ByteCache[K, V]) makeRoom(incoming int) {
	for cache.bytes+incoming > cache.budget {
		var oldestKey K
		var oldest cacheEntry[V]
		found := false
		for key, entry := range cache.values {
			if entry.refs == 0 && (!found || entry.stamp < oldest.stamp) {
				oldestKey, oldest, found = key, entry, true
			}
		}
		if !found {
			return
		}
		cache.remove(oldestKey, oldest)
	}
}

func (cache *ByteCache[K, V]) remove(key K, entry cacheEntry[V]) {
	delete(cache.values, key)
	cache.bytes -= entry.bytes
	cache.evicted++
	cache.releaseValue(entry.value)
}

func (cache *ByteCache[K, V]) releaseValue(value V) {
	if cache.release != nil {
		cache.release(value)
	}
}
