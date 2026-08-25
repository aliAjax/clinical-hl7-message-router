package infrastructure

import "sync"

type SchemaRegistry struct {
	mu       sync.RWMutex
	versions map[string][]byte
}

func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{versions: make(map[string][]byte)}
}

func (r *SchemaRegistry) Put(version string, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.versions[version] = append([]byte(nil), data...)
}

func (r *SchemaRegistry) Get(version string) ([]byte, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, ok := r.versions[version]
	return append([]byte(nil), data...), ok
}
