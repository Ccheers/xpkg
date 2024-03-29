package group

import "fmt"

type Counter struct {
	Value int
}

func (c *Counter) Incr() {
	c.Value++
}

func ExampleGroup_Get() {
	new := func() *Counter {
		fmt.Println("Only Once")
		return &Counter{}
	}
	group := NewGroup(new)

	// Create a new Counter
	group.Get("pass").Incr()

	// Get the created Counter again.
	group.Get("pass").Incr()
	// Output:
	// Only Once
}

func ExampleGroup_Reset() {
	new := func() *Counter {
		return &Counter{}
	}
	group := NewGroup(new)

	newV2 := func() *Counter {
		fmt.Println("New V2")
		return &Counter{}
	}
	// Reset the new function and clear all created objects.
	group.Reset(newV2)

	// Create a new Counter
	group.Get("pass").Incr()
	// Output:
	// New V2
}
