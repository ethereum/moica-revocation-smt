package store

import (
	"fmt"
	"sync"
)

// MemoryStore is an in-memory key-value store for testing.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string][]byte)}
}

func (m *MemoryStore) Get(key []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (m *MemoryStore) Set(key []byte, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(value))
	copy(cp, value)
	m.data[string(key)] = cp
	return nil
}

func (m *MemoryStore) Delete(key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, string(key))
	return nil
}

func (m *MemoryStore) Has(key []byte) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[string(key)]
	return ok, nil
}

func (m *MemoryStore) NewBatch() Batch {
	return &memoryBatch{store: m}
}

func (m *MemoryStore) Close() error {
	return nil
}

// Len returns the number of entries (for testing).
func (m *MemoryStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

type memoryBatch struct {
	store *MemoryStore
	ops   []batchOp
}

type batchOp struct {
	key    []byte
	value  []byte
	delete bool
}

func (b *memoryBatch) Set(key []byte, value []byte) error {
	b.ops = append(b.ops, batchOp{key: key, value: value})
	return nil
}

func (b *memoryBatch) Delete(key []byte) error {
	b.ops = append(b.ops, batchOp{key: key, delete: true})
	return nil
}

func (b *memoryBatch) Flush() error {
	b.store.mu.Lock()
	defer b.store.mu.Unlock()
	for _, op := range b.ops {
		if op.delete {
			delete(b.store.data, string(op.key))
		} else {
			cp := make([]byte, len(op.value))
			copy(cp, op.value)
			b.store.data[string(op.key)] = cp
		}
	}
	b.ops = nil
	return nil
}
