// Package inmem provides an in memory LRU cache with TTL support.
package inmem

import (
	"container/list"
	"sync"
	"time"
)

// Cache of things.
type Cache interface {
	Add(key, value string)
	Get(key string) (string, bool)
	Remove(key string)
	Purge()
	Len() int
}

// cache implements a non-thread safe fixed size cache.
type cache struct {
	refreshAfterAccess bool
	ttl                time.Duration
	size               int
	lru                *list.List
	items              map[string]*list.Element
}

func (c *cache) Purge() {
	c.lru = list.New()
	c.items = make(map[string]*list.Element)
}

// entry in the cache.
type entry struct {
	key       string
	value     string
	expiresAt time.Time
}

// NewUnlocked constructs a new Cache of the given size that is not safe for
// concurrent use. If will panic if size is not a positive integer.
func NewUnlocked(size int, ttl time.Duration, refreshAfterAccess bool) Cache {
	if size <= 0 {
		panic("inmem: must provide a positive size")
	}
	return &cache{
		refreshAfterAccess: refreshAfterAccess,
		ttl:                ttl,
		size:               size,
		lru:                list.New(),
		items:              make(map[string]*list.Element),
	}
}

func (c *cache) Add(key, value string) {
	expiresAt := time.Now().Add(c.ttl)
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

			// adjust expiresAt
			if c.refreshAfterAccess {
				v.expiresAt = time.Now().Add(c.ttl)
			}
			return v.value, true
		}

		// ttl expired
		c.removeElement(ent)
	}
	return nil, false
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

type lockedCache struct {
	c cache
	m sync.Mutex
}

func (l *lockedCache) Purge() {
	l.m.Lock()
	l.c.Purge()
	l.m.Unlock()
}

// NewLocked constructs a new Cache of the given size that is safe for
// concurrent use. If will panic if size is not a positive integer.
func NewLocked(size int, ttl time.Duration, refreshAfterAccess bool) Cache {
	if size <= 0 {
		panic("inmem: must provide a positive size")
	}
	return &lockedCache{
		c: cache{
			refreshAfterAccess: refreshAfterAccess,
			ttl:                ttl,
			size:               size,
			lru:                list.New(),
			items:              make(map[string]*list.Element),
		},
	}
}

func (l *lockedCache) Add(key, value string) {
	l.m.Lock()
	l.c.Add(key, value)
	l.m.Unlock()
}

func (l *lockedCache) Get(key string) (string, bool) {
	l.m.Lock()
	v, f := l.c.Get(key)
	l.m.Unlock()
	return v, f
}

func (l *lockedCache) Remove(key string) {
	l.m.Lock()
	l.c.Remove(key)
	l.m.Unlock()
}

func (l *lockedCache) Len() int {
	l.m.Lock()
	c := l.c.Len()
	l.m.Unlock()
	return c
}
