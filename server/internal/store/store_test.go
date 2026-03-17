package store

import (
	"os"
	"testing"
)

// storeTests runs a common test suite against any Store implementation.
func storeTests(t *testing.T, s Store) {
	t.Helper()

	// Set and Get
	err := s.Set([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatal(err)
	}
	val, err := s.Get([]byte("key1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "value1" {
		t.Errorf("Get: got %q, want %q", val, "value1")
	}

	// Has
	exists, err := s.Has([]byte("key1"))
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Has: expected true")
	}
	exists, err = s.Has([]byte("nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Has: expected false for nonexistent")
	}

	// Get nonexistent
	_, err = s.Get([]byte("nonexistent"))
	if err == nil {
		t.Error("expected error for nonexistent key")
	}

	// Delete
	err = s.Delete([]byte("key1"))
	if err != nil {
		t.Fatal(err)
	}
	exists, err = s.Has([]byte("key1"))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Has after delete: expected false")
	}

	// Batch operations
	batch := s.NewBatch()
	batch.Set([]byte("b1"), []byte("v1"))
	batch.Set([]byte("b2"), []byte("v2"))
	batch.Set([]byte("b3"), []byte("v3"))
	if err := batch.Flush(); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"b1", "b2", "b3"} {
		exists, err := s.Has([]byte(k))
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("batch Set: %s not found after flush", k)
		}
	}

	// Batch delete
	batch2 := s.NewBatch()
	batch2.Delete([]byte("b2"))
	if err := batch2.Flush(); err != nil {
		t.Fatal(err)
	}
	exists, err = s.Has([]byte("b2"))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("batch Delete: b2 should not exist after flush")
	}
}

func TestMemoryStore(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()
	storeTests(t, s)
}

func TestBadgerStore(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	s, err := NewBadgerStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	storeTests(t, s)
}
