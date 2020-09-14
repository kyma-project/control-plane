package storage

import (
	"sync"
)

type memoryStorage struct {
	sync.RWMutex
	items map[string]interface{}
}

var _ Storage = &memoryStorage{}

// Put sets the value for a key.
func (c *memoryStorage) Put(key string, obj interface{}) {
	c.Lock()
	c.items[key] = obj
	c.Unlock()
}

// Get returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (c *memoryStorage) Get(key string) (obj interface{}, exists bool) {
	c.RLock()
	result, ok := c.items[key]
	c.RUnlock()

	return result, ok
}

// Update simply calls `Put`.
func (c *memoryStorage) Update(key string, obj interface{}) {
	c.Put(key, obj)
}

// Delete deletes the value for a key.
func (c *memoryStorage) Delete(key string) {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
}

// List returns a list of all the objects.
func (c *memoryStorage) List() []interface{} {
	result := make([]interface{}, 0)

	c.RLock()
	for _, v := range c.items {
		result = append(result, v)
	}
	c.RUnlock()

	return result
}

// ListKeys returns a list of all the keys associated with objects.
func (c *memoryStorage) ListKeys() []string {
	result := make([]string, 0)

	c.RLock()
	for k := range c.items {
		result = append(result, k)
	}
	c.RUnlock()

	return result
}

// NewMemoryStorage returns an in-memory storage.
func NewMemoryStorage() Storage {
	return &memoryStorage{
		items: make(map[string]interface{}),
	}
}
