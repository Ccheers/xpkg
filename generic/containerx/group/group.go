// Package group provides a sample lazy load container.
// The group only creating a new object not until the object is needed by user.
// And it will cache all the objects to reduce the creation of object.
package group

import "sync"

// Group is a lazy load container.
type Group[T any] struct {
	new  func() T
	objs map[string]T
	sync.RWMutex
}

// NewGroup news a group container.
func NewGroup[T any](new func() T) *Group[T] {
	if new == nil {
		panic("container.group: can't assign a nil to the new function")
	}
	return &Group[T]{
		new:  new,
		objs: make(map[string]T),
	}
}

// Get gets the object by the given key.
func (g *Group[T]) Get(key string) T {
	g.RLock()
	obj, ok := g.objs[key]
	if ok {
		g.RUnlock()
		return obj
	}
	g.RUnlock()

	// double check
	g.Lock()
	defer g.Unlock()
	obj, ok = g.objs[key]
	if ok {
		return obj
	}
	obj = g.new()
	g.objs[key] = obj
	return obj
}

// Reset resets the new function and deletes all existing objects.
func (g *Group[T]) Reset(new func() T) {
	if new == nil {
		panic("container.group: can't assign a nil to the new function")
	}
	g.Lock()
	g.new = new
	g.Unlock()
	g.Clear()
}

// Clear deletes all objects.
func (g *Group[T]) Clear() {
	g.Lock()
	g.objs = make(map[string]T)
	g.Unlock()
}
