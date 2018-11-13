package cache

import (
	"container/list"
	"encoding/binary"
	"net/http"
	"sync"
	"time"
)

// LRUCache is a typical LRU cache implementation.  If the cache
// reaches the capacity, the least recently used item is deleted from
// the cache. Note the capacity is not the number of items, but the
// total sum of the Size() of each item.
type LRUCache struct {
	mu sync.Mutex

	// list & table of *entry objects
	list  *list.List
	table map[string]*list.Element

	// Our current size. Obviously a gross simplification and
	// low-grade approximation.
	size int64

	// How much we are limiting the cache to.
	capacity int64

	expiry time.Duration

	// Stop garbage collection routine, stops any running GC routine.
	stopGC chan struct{}

	OnEviction func(key string)
}

type CachedResponse struct {
	Resp *http.Response
}

func (cp *CachedResponse) Size() int {
	return binary.Size(cp.Resp)
}

// Item is what is stored in the cache
type Item struct {
	Key   string
	Value CachedResponse
}

type entry struct {
	key          string
	value        *CachedResponse
	size         int64
	timeAccessed time.Time
}

// NewLRUCache creates a new empty cache with the given capacity.
func NewLRUCache(capacity int64, expiry time.Duration) *LRUCache {
	cache :=  &LRUCache{
		list:     list.New(),
		table:    make(map[string]*list.Element),
		capacity: capacity,
		expiry:            expiry,
	}


	// We have expiry start the janitor routine.
	if expiry > 0 {
		// Initialize a new stop GC channel.
		cache.stopGC = make(chan struct{})

		// Start garbage collection routine to expire objects.
		cache.StartGC()
	}

	return cache
}

// Get returns a value from the cache, and marks the entry as most
// recently used.
func (lru *LRUCache) Get(key string) (v *CachedResponse, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[key]
	if element == nil {
		return nil, false
	}
	lru.moveToFront(element)
	return element.Value.(*entry).value, true
}

// Set sets a value in the cache.
func (lru *LRUCache) Set(key string, value *CachedResponse) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[key]; element != nil {
		lru.updateInplace(element, value)
	} else {
		lru.addNew(key, value)
	}
}

// SetIfAbsent will set the value in the cache if not present. If the
// value exists in the cache, we don't set it.
func (lru *LRUCache) SetIfAbsent(key string, value *CachedResponse) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[key]; element != nil {
		lru.moveToFront(element)
	} else {
		lru.addNew(key, value)
	}
}

// Delete removes an entry from the cache, and returns if the entry existed.
func (lru *LRUCache) Delete(key string) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[key]
	if element == nil {
		return false
	}

	lru.list.Remove(element)
	delete(lru.table, key)
	lru.size -= element.Value.(*entry).size
	return true
}

// Stats returns a few stats on the cache.
func (lru *LRUCache) Stats() (length, size, capacity int64, oldest time.Time) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if lastElem := lru.list.Back(); lastElem != nil {
		oldest = lastElem.Value.(*entry).timeAccessed
	}
	return int64(lru.list.Len()), lru.size, lru.capacity, oldest
}

// Length returns how many elements are in the cache
func (lru *LRUCache) Length() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return int64(lru.list.Len())
}

// Size returns the sum of the objects' Size() method.
func (lru *LRUCache) Size() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.size
}

// Capacity returns the cache maximum capacity.
func (lru *LRUCache) Capacity() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.capacity
}

func (lru *LRUCache) updateInplace(element *list.Element, value *CachedResponse) {
	valueSize := int64(value.Size())
	sizeDiff := valueSize - element.Value.(*entry).size
	element.Value.(*entry).value = value
	element.Value.(*entry).size = valueSize
	lru.size += sizeDiff
	lru.moveToFront(element)
	lru.checkCapacity()
}

func (lru *LRUCache) moveToFront(element *list.Element) {
	lru.list.MoveToFront(element)
	element.Value.(*entry).timeAccessed = time.Now()
}

func (lru *LRUCache) addNew(key string, value *CachedResponse) {
	newEntry := &entry{key, value, int64(value.Size()), time.Now()}
	element := lru.list.PushFront(newEntry)
	lru.table[key] = element
	lru.size += newEntry.size
	lru.checkCapacity()
}

func (lru *LRUCache) checkCapacity() {
	// Partially duplicated from Delete
	for lru.size > lru.capacity {
		delElem := lru.list.Back()
		delValue := delElem.Value.(*entry)
		lru.list.Remove(delElem)
		delete(lru.table, delValue.key)
		lru.size -= delValue.size
	}
}

// gc garbage collect all the expired entries from the cache.
func (lru *LRUCache) gc() {
	var evictedEntries []string
	lru.mu.Lock()
	for k, elem := range lru.table {
		if lru.expiry > 0 && time.Now().UTC().Sub(elem.Value.(*entry).timeAccessed) > lru.expiry {
			evictedEntries = append(evictedEntries, k)
		}
	}
	lru.mu.Unlock()
	for _, k := range evictedEntries {
		if lru.OnEviction != nil {
			lru.OnEviction(k)
		}
	}
}

// rest deletes all the entries from the cache.
func (lru *LRUCache) Reset() {
	for k := range lru.table {
		lru.Delete(k)
	}
}

// StopGC sends a message to the expiry routine to stop
// expiring cached entries. NOTE: once this is called, cached
// entries will not be expired, be careful if you are using this.
func (c *LRUCache) StopGC() {
	if c.stopGC != nil {
		c.stopGC <- struct{}{}
	}
}

// StartGC starts running a routine ticking at expiry interval,
// on each interval this routine does a sweep across the cache
// entries and garbage collects all the expired entries.
func (c *LRUCache) StartGC() {
	go func() {
		for {
			select {
			// Wait till cleanup interval and initiate delete expired entries.
			case <-time.After(c.expiry / 4):
				c.gc()
				// Stop the routine, usually called by the user of object cache during cleanup.
			case <-c.stopGC:
				return
			}
		}
	}()
}