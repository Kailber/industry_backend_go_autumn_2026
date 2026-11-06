package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return value, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return value, false
	}

	c.ll.MoveToFront(element)

	item := element.Value.(*entry[K, V])
	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		item := element.Value.(*entry[K, V])
		item.value = value
		return
	}

	if c.ll.Len() >= c.capacity {
		oldestElement := c.ll.Back()
		if oldestElement != nil {
			oldestItem := oldestElement.Value.(*entry[K, V])
			delete(c.items, oldestItem.key)
			c.ll.Remove(oldestElement)
		}
	}

	newItem := &entry[K, V]{
		key:   key,
		value: value,
	}

	element := c.ll.PushFront(newItem)
	c.items[key] = element
}
