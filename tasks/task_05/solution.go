package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]V),
	}
}

func (cache *Cache[K, V]) Get(key K) (V, bool) {
	value, found := cache.items[key]
	return value, found
}

func (cache *Cache[K, V]) Set(key K, value V) bool {
	_, exists := cache.items[key]
	if !exists && len(cache.items) >= cache.capacity {
		return false
	}

	cache.items[key] = value
	return true
}
