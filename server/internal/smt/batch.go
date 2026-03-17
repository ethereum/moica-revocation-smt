package smt

import (
	"math/big"
	"sync"
)

// BatchAdd inserts multiple key/value pairs into the tree.
// Keys are added sequentially (SMT structure depends on insertion state).
// Holds the lock once for the entire batch to avoid per-entry mutex overhead.
func (t *SMT) BatchAdd(keys []*big.Int, value *big.Int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.growNodes(len(keys))
	for _, key := range keys {
		if err := t.addUnlocked(key, value); err != nil {
			return err
		}
	}
	return nil
}

// BatchAddWithProgress inserts entries and calls the progress callback periodically.
// Holds the lock once for the entire batch to avoid per-entry mutex overhead.
func (t *SMT) BatchAddWithProgress(keys []*big.Int, value *big.Int, batchSize int, onProgress func(done, total int)) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.growNodes(len(keys))
	total := len(keys)
	for i, key := range keys {
		if err := t.addUnlocked(key, value); err != nil {
			return err
		}
		if onProgress != nil && (i+1)%batchSize == 0 {
			onProgress(i+1, total)
		}
	}
	if onProgress != nil {
		onProgress(total, total)
	}
	return nil
}

// PrecomputeLeafHashes computes leaf hashes in parallel.
// Returns a map of key → leaf hash for later tree construction.
func PrecomputeLeafHashes(h Hasher, keys []*big.Int, value *big.Int, workers int) map[string]*big.Int {
	type result struct {
		key  string
		hash *big.Int
	}

	ch := make(chan result, len(keys))
	var wg sync.WaitGroup

	sem := make(chan struct{}, workers)
	one := big.NewInt(1)

	for _, k := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(key *big.Int) {
			defer wg.Done()
			defer func() { <-sem }()
			leafHash := h.Hash3(key, value, one)
			ch <- result{key: key.Text(16), hash: leafHash}
		}(k)
	}

	wg.Wait()
	close(ch)

	hashes := make(map[string]*big.Int, len(keys))
	for r := range ch {
		hashes[r.key] = r.hash
	}
	return hashes
}
