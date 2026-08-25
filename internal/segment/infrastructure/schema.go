package infrastructure

import (
	"sync"

	"github.com/example/hl7v2-message-router/internal/segment/adapter"
)

type Schema interface {
	Valid() bool
	Lookup(string) (adapter.Definition, bool)
}

type SchemaRegistry struct {
	mu       sync.RWMutex
	versions map[string]Schema
}

func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{versions: make(map[string]Schema)}
}

func (r *SchemaRegistry) Put(version string, schema Schema) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.versions[version] = schema
}

func (r *SchemaRegistry) Get(version string) (Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schema, ok := r.versions[version]
	return schema, ok
}
