package store

// Store is the interface for persistent key-value storage.
type Store interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	Has(key []byte) (bool, error)
	NewBatch() Batch
	Close() error
}

// Batch supports atomic batch writes.
type Batch interface {
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	Flush() error
}

// Key prefixes for the storage layout.
const (
	PrefixNode     = "n:" // node hash → encoded node
	PrefixMetaRoot = "m:root"
	PrefixMetaCount = "m:count"
	PrefixMetaCRL  = "m:crl:" // + issuerID
)
