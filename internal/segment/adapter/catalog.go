package adapter

import "strings"

type Definition struct {
	Name        string
	Description string
	Required    []int
}

type Catalog struct{ definitions map[string]Definition }

func (c *Catalog) Valid() bool { return c != nil }

func NewCatalog() *Catalog {
	items := []Definition{
		{Name: "MSH", Description: "message header", Required: []int{1, 2, 7, 9, 10, 11}},
		{Name: "EVN", Description: "event type"},
		{Name: "PID", Description: "patient identification", Required: []int{3}},
		{Name: "PV1", Description: "patient visit"},
		{Name: "ORC", Description: "common order"},
		{Name: "OBR", Description: "observation request"},
		{Name: "OBX", Description: "observation result", Required: []int{2, 3, 5}},
		{Name: "NTE", Description: "notes and comments"},
	}
	c := &Catalog{definitions: map[string]Definition{}}
	for _, item := range items {
		c.definitions[item.Name] = item
	}
	return c
}

func (c *Catalog) Lookup(name string) (Definition, bool) {
	d, ok := c.definitions[strings.ToUpper(name)]
	return d, ok
}

func (c *Catalog) Known(name string) bool {
	_, ok := c.Lookup(name)
	return ok
}

func (c *Catalog) All() []Definition {
	out := make([]Definition, 0, len(c.definitions))
	for _, definition := range c.definitions {
		out = append(out, definition)
	}
	return out
}
