package simplykb

import (
	"container/list"
	"sync"
)

type queryEmbeddingCache struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	entries  map[string]*list.Element
}

type queryEmbeddingCacheEntry struct {
	key    string
	vector []float32
}

func newQueryEmbeddingCache(capacity int) *queryEmbeddingCache {
	if capacity <= 0 {
		return nil
	}
	return &queryEmbeddingCache{
		capacity: capacity,
		ll:       list.New(),
		entries:  make(map[string]*list.Element, capacity),
	}
}

func (c *queryEmbeddingCache) Get(key string) ([]float32, bool) {
	if c == nil {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(element)
	entry := element.Value.(*queryEmbeddingCacheEntry)
	return cloneVector(entry.vector), true
}

func (c *queryEmbeddingCache) Put(key string, vector []float32) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.entries[key]; ok {
		c.ll.MoveToFront(element)
		element.Value.(*queryEmbeddingCacheEntry).vector = cloneVector(vector)
		return
	}

	element := c.ll.PushFront(&queryEmbeddingCacheEntry{
		key:    key,
		vector: cloneVector(vector),
	})
	c.entries[key] = element

	if c.ll.Len() <= c.capacity {
		return
	}

	tail := c.ll.Back()
	if tail == nil {
		return
	}
	c.ll.Remove(tail)
	entry := tail.Value.(*queryEmbeddingCacheEntry)
	delete(c.entries, entry.key)
}

func cloneVector(vector []float32) []float32 {
	return append([]float32(nil), vector...)
}
