package v2

import (
	"sync"
)

// ARCCache is a thread-safe fixed size Adaptive Replacement Cache (ARC).
// ARC is an enhancement over the standard LRU cache in that tracks both
// frequency and recency of use. This avoids a burst in access to new
// entries from evicting the frequently used older entries. It adds some
// additional tracking overhead to a standard LRU cache, computationally
// it is roughly 2x the cost, and the extra memory overhead is linear
// with the size of the cache. ARC has been patented by IBM, but is
// similar to the TwoQueueCache (2Q) which requires setting parameters.
type ARCCache struct {
	size int // Size is the total capacity of the cache
	p    int // P is the dynamic preference towards T1 or T2

	t1 LRUCache[string, interface{}] // T1 is the LRU for recently accessed items
	b1 LRUCache[string, struct{}]    // B1 is the LRU for evictions from t1

	t2 LRUCache[string, interface{}] // T2 is the LRU for frequently accessed items
	b2 LRUCache[string, struct{}]    // B2 is the LRU for evictions from t2

	lock sync.RWMutex
}

// NewARC creates an ARC of the given size
func NewARC(size int) (*ARCCache, error) {
	// Create the sub LRUs
	b1, err := NewLRU[string, struct{}](size, nil)
	if err != nil {
		return nil, err
	}
	b2, err := NewLRU[string, struct{}](size, nil)
	if err != nil {
		return nil, err
	}
	t1, err := NewLRU[string, interface{}](size, nil)
	if err != nil {
		return nil, err
	}
	t2, err := NewLRU[string, interface{}](size, nil)
	if err != nil {
		return nil, err
	}

	// Initialize the ARC
	x := &ARCCache{
		size: size,
		p:    0,
		t1:   t1,
		b1:   b1,
		t2:   t2,
		b2:   b2,
	}
	return x, nil
}

// Get looks up a key's value from the cache.
func (x *ARCCache) Get(key string) (value interface{}, ok bool) {
	x.lock.Lock()
	defer x.lock.Unlock()

	// If the value is contained in T1 (recent), then
	// promote it to T2 (frequent)
	if val, ok := x.t1.Peek(key); ok {
		x.t1.Remove(key)
		x.t2.Add(key, val)
		return val, ok
	}

	// Check if the value is contained in T2 (frequent)
	if val, ok := x.t2.Get(key); ok {
		return val, ok
	}

	// No hit
	return
}

// Add adds a value to the cache.
func (x *ARCCache) Add(key string, value interface{}) {
	x.lock.Lock()
	defer x.lock.Unlock()

	// Check if the value is contained in T1 (recent), and potentially
	// promote it to frequent T2
	if x.t1.Contains(key) {
		x.t1.Remove(key)
		x.t2.Add(key, value)
		return
	}

	// Check if the value is already in T2 (frequent) and update it
	if x.t2.Contains(key) {
		x.t2.Add(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// recently used list
	if x.b1.Contains(key) {
		// T1 set is too small, increase P appropriately
		delta := 1
		b1Len := x.b1.Len()
		b2Len := x.b2.Len()
		if b2Len > b1Len {
			delta = b2Len / b1Len
		}
		if x.p+delta >= x.size {
			x.p = x.size
		} else {
			x.p += delta
		}

		// Potentially need to make room in the cache
		if x.t1.Len()+x.t2.Len() >= x.size {
			x.replace(false)
		}

		// Remove from B1
		x.b1.Remove(key)

		// Add the key to the frequently used list
		x.t2.Add(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// frequently used list
	if x.b2.Contains(key) {
		// T2 set is too small, decrease P appropriately
		delta := 1
		b1Len := x.b1.Len()
		b2Len := x.b2.Len()
		if b1Len > b2Len {
			delta = b1Len / b2Len
		}
		if delta >= x.p {
			x.p = 0
		} else {
			x.p -= delta
		}

		// Potentially need to make room in the cache
		if x.t1.Len()+x.t2.Len() >= x.size {
			x.replace(true)
		}

		// Remove from B2
		x.b2.Remove(key)

		// Add the key to the frequently used list
		x.t2.Add(key, value)
		return
	}

	// Potentially need to make room in the cache
	if x.t1.Len()+x.t2.Len() >= x.size {
		x.replace(false)
	}

	// Keep the size of the ghost buffers trim
	if x.b1.Len() > x.size-x.p {
		x.b1.RemoveOldest()
	}
	if x.b2.Len() > x.p {
		x.b2.RemoveOldest()
	}

	// Add to the recently seen list
	x.t1.Add(key, value)
}

// replace is used to adaptively evict from either T1 or T2
// based on the current learned value of P
func (x *ARCCache) replace(b2ContainsKey bool) {
	t1Len := x.t1.Len()
	if t1Len > 0 && (t1Len > x.p || (t1Len == x.p && b2ContainsKey)) {
		k, _, ok := x.t1.RemoveOldest()
		if ok {
			x.b1.Add(k, struct{}{})
		}
	} else {
		k, _, ok := x.t2.RemoveOldest()
		if ok {
			x.b2.Add(k, struct{}{})
		}
	}
}

// Len returns the number of cached entries
func (x *ARCCache) Len() int {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.t1.Len() + x.t2.Len()
}

// Cap returns the capacity of the cache
func (x *ARCCache) Cap() int {
	return x.size
}

// Keys returns all the cached keys
func (x *ARCCache) Keys() []string {
	x.lock.RLock()
	defer x.lock.RUnlock()
	k1 := x.t1.Keys()
	k2 := x.t2.Keys()
	return append(k1, k2...)
}

// Values returns all the cached values
func (x *ARCCache) Values() []interface{} {
	x.lock.RLock()
	defer x.lock.RUnlock()
	v1 := x.t1.Values()
	v2 := x.t2.Values()
	return append(v1, v2...)
}

// Remove is used to purge a key from the cache
func (x *ARCCache) Remove(key string) {
	x.lock.Lock()
	defer x.lock.Unlock()
	if x.t1.Remove(key) {
		return
	}
	if x.t2.Remove(key) {
		return
	}
	if x.b1.Remove(key) {
		return
	}
	if x.b2.Remove(key) {
		return
	}
}

// Purge is used to clear the cache
func (x *ARCCache) Purge() {
	x.lock.Lock()
	defer x.lock.Unlock()
	x.t1.Purge()
	x.t2.Purge()
	x.b1.Purge()
	x.b2.Purge()
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (x *ARCCache) Contains(key string) bool {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.t1.Contains(key) || x.t2.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (x *ARCCache) Peek(key string) (value interface{}, ok bool) {
	x.lock.RLock()
	defer x.lock.RUnlock()
	if val, ok := x.t1.Peek(key); ok {
		return val, ok
	}
	return x.t2.Peek(key)
}
