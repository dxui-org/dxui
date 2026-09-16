package renderer

import "testing"

type trackedCacheValue struct{ released *int }

func TestByteCacheAllocatesBackingMapOnlyOnFirstRetainedEntry(t *testing.T) {
	cache := NewByteCache[string, int](16, nil)
	if cache.values != nil || cache.Stats().Entries != 0 {
		t.Fatal("empty cache allocated a backing map")
	}
	if !cache.Put("shadow", 1, 8) || cache.values == nil {
		t.Fatal("first retained entry did not allocate the backing map")
	}
}

func TestTextureCacheCapacityLRUAndPinnedResourceRelease(t *testing.T) {
	released := 0
	cache := NewByteCache[string, trackedCacheValue](20, func(value trackedCacheValue) { *value.released++ })
	value := trackedCacheValue{released: &released}
	if !cache.Put("old", value, 10) || !cache.Put("pinned", value, 10) || !cache.Retain("pinned") {
		t.Fatal("failed to seed cache")
	}
	if !cache.Put("new", value, 10) {
		t.Fatal("zero-reference LRU was not evicted")
	}
	if _, ok := cache.Get("old"); ok {
		t.Fatal("oldest entry survived capacity eviction")
	}
	if _, ok := cache.Get("pinned"); !ok {
		t.Fatal("pinned entry was evicted")
	}
	stats := cache.Stats()
	if stats.Bytes > stats.BudgetBytes || stats.Entries != 2 || released != 1 {
		t.Fatalf("stats/released = %+v/%d", stats, released)
	}
	cache.Release("pinned")
	cache.Clear()
	if cache.Stats().Entries != 0 || released != 3 {
		t.Fatalf("clear did not release resources: stats=%+v released=%d", cache.Stats(), released)
	}
}

func TestByteCacheRejectsOversizedValueAndReleasesIt(t *testing.T) {
	released := 0
	cache := NewByteCache[int, trackedCacheValue](8, func(value trackedCacheValue) { *value.released++ })
	if cache.Put(1, trackedCacheValue{released: &released}, 9) || released != 1 || cache.Stats().Bytes != 0 {
		t.Fatalf("oversized insert = stats=%+v released=%d", cache.Stats(), released)
	}
}

func TestByteCacheRetainsActiveOversizedValueUntilRelease(t *testing.T) {
	released := 0
	cache := NewByteCache[int, trackedCacheValue](8, func(value trackedCacheValue) { *value.released++ })
	if !cache.PutRetained(1, trackedCacheValue{released: &released}, 9) {
		t.Fatal("active oversized insert rejected")
	}
	if stats := cache.Stats(); stats.Entries != 1 || stats.Bytes != 9 || released != 0 {
		t.Fatalf("active oversized stats=%+v released=%d", stats, released)
	}
	cache.Release(1)
	if stats := cache.Stats(); stats.Entries != 0 || stats.Bytes != 0 || released != 1 {
		t.Fatalf("released oversized stats=%+v released=%d", stats, released)
	}
}
