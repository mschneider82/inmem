// Package inmem copies the Cache interface and and a modified cache
// type from github.com/facebookgo/inmem.
package inmem

import (
	"container/list"
	"sync"
	"time"
)

// Cache of things.
type Cache interface {
	Add(key, value string, expiresAt time.Time)
	Get(key string) (string, bool)
	Remove(key string)
	Len() int
}

// cache implements a non-thread safe fixed size string cache.
type cache struct {
	size  int
	lru   *list.List
	items map[string]*list.Element
}

// entry is a string key/value entry in the cache.
type entry struct {
	key       string
	value     string
	expiresAt time.Time
}

// NewUnlocked constructs a new Cache of the given size that is not safe for
// concurrent use. If will panic if size is not a positive integer.
func NewUnlocked(size int) Cache {
	if size <= 0 {
		panic("cache: must provide a positive size")
	}
	return &cache{
		size:  size,
		lru:   list.New(),
		items: make(map[string]*list.Element),
	}
}

func (c *cache) Add(key, value string, expiresAt time.Time) {
	if ent, ok := c.items[key]; ok {
		// update existing entry
		c.lru.MoveToFront(ent)
		v := ent.Value.(*entry)
		v.value = value
		v.expiresAt = expiresAt
		return
	}

	// add new entry
	c.items[key] = c.lru.PushFront(&entry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	})

	// remove oldest
	if c.lru.Len() > c.size {
		ent := c.lru.Back()
		if ent != nil {
			c.removeElement(ent)
		}
	}
}

func (c *cache) Get(key string) (string, bool) {
	if ent, ok := c.items[key]; ok {
		v := ent.Value.(*entry)

		if v.expiresAt.After(time.Now()) {
			// found good entry
			c.lru.MoveToFront(ent)
			return v.value, true
		}

		// ttl expired
		c.removeElement(ent)
	}
	return "", false
}

func (c *cache) Remove(key string) {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}

func (c *cache) Len() int {
	return c.lru.Len()
}

// removeElement is used to remove a given list element from the cache
func (c *cache) removeElement(e *list.Element) {
	c.lru.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
}

type lockedStringCache struct {
	c cache
	m sync.Mutex
}

// NewLockedString constructs a new Cache of the given size that is safe for
// concurrent use. It will panic if size is not a positive integer.
func NewLockedString(size int) Cache {
	if size <= 0 {
		panic("cache: must provide a positive size")
	}
	return &lockedStringCache{
		c: cache{
			size:  size,
			lru:   list.New(),
			items: make(map[string]*list.Element),
		},
	}
}

func (ls *lockedStringCache) Add(key, value string, expiresAt time.Time) {
	ls.m.Lock()
	ls.c.Add(key, value, expiresAt)
	ls.m.Unlock()
}

func (ls *lockedStringCache) Get(key string) (string, bool) {
	ls.m.Lock()
	v, f := ls.c.Get(key)
	ls.m.Unlock()
	return v, f
}

func (ls *lockedStringCache) Remove(key string) {
	ls.m.Lock()
	ls.c.Remove(key)
	ls.m.Unlock()
}

func (ls *lockedStringCache) Len() int {
	ls.m.Lock()
	c := ls.c.Len()
	ls.m.Unlock()
	return c
}
