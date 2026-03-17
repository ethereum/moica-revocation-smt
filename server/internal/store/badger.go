package store

import (
	badger "github.com/dgraph-io/badger/v4"
)

// BadgerStore wraps BadgerDB as a persistent key-value store.
type BadgerStore struct {
	db *badger.DB
}

// NewBadgerStore opens or creates a BadgerDB at the given path.
func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path).
		WithLogger(nil) // Silence BadgerDB logs
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db}, nil
}

func (s *BadgerStore) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *BadgerStore) Set(key []byte, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *BadgerStore) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *BadgerStore) Has(key []byte) (bool, error) {
	var exists bool
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		exists = true
		return nil
	})
	return exists, err
}

func (s *BadgerStore) NewBatch() Batch {
	return &badgerBatch{db: s.db}
}

func (s *BadgerStore) Close() error {
	return s.db.Close()
}

type badgerBatch struct {
	db  *badger.DB
	ops []batchOp
}

func (b *badgerBatch) Set(key []byte, value []byte) error {
	kcp := make([]byte, len(key))
	copy(kcp, key)
	vcp := make([]byte, len(value))
	copy(vcp, value)
	b.ops = append(b.ops, batchOp{key: kcp, value: vcp})
	return nil
}

func (b *badgerBatch) Delete(key []byte) error {
	kcp := make([]byte, len(key))
	copy(kcp, key)
	b.ops = append(b.ops, batchOp{key: kcp, delete: true})
	return nil
}

func (b *badgerBatch) Flush() error {
	wb := b.db.NewWriteBatch()
	for _, op := range b.ops {
		if op.delete {
			if err := wb.Delete(op.key); err != nil {
				wb.Cancel()
				return err
			}
		} else {
			if err := wb.Set(op.key, op.value); err != nil {
				wb.Cancel()
				return err
			}
		}
	}
	b.ops = nil
	return wb.Flush()
}
