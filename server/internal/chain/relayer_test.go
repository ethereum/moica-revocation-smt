package chain

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/moven0831/moica-revocation-smt/server/internal/chain/contract"
)

// testEnv holds a simulated backend with a deployed SMTRootStorage contract.
type testEnv struct {
	backend  *simulated.Backend
	key      *ecdsa.PrivateKey
	relayer  *Relayer
	contract *contract.SMTRootStorage
	addr     common.Address
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	relayerAddr := crypto.PubkeyToAddress(key.PublicKey)

	backend := simulated.NewBackend(types.GenesisAlloc{
		relayerAddr: {Balance: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10))},
	})
	t.Cleanup(func() { backend.Close() })

	client := backend.Client()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		t.Fatal(err)
	}

	addr, _, instance, err := contract.DeploySMTRootStorage(auth, client, relayerAddr)
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()

	r := &Relayer{
		backend:         client,
		privateKey:      key,
		contractAddress: addr,
		chainID:         chainID,
	}

	return &testEnv{
		backend:  backend,
		key:      key,
		relayer:  r,
		contract: instance,
		addr:     addr,
	}
}

// commitPeriodically commits blocks in the background so WaitMined can return.
func commitPeriodically(t *testing.T, backend *simulated.Backend, done <-chan struct{}) {
	t.Helper()
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				backend.Commit()
			}
		}
	}()
}

func TestIssuerIDs(t *testing.T) {
	g2 := crypto.Keccak256Hash([]byte("MOICA-G2"))
	g3 := crypto.Keccak256Hash([]byte("MOICA-G3"))

	if IssuerG2 != g2 {
		t.Errorf("IssuerG2 mismatch: got %x, want %x", IssuerG2, g2)
	}
	if IssuerG3 != g3 {
		t.Errorf("IssuerG3 mismatch: got %x, want %x", IssuerG3, g3)
	}
	if IssuerG2 == IssuerG3 {
		t.Error("IssuerG2 and IssuerG3 should differ")
	}
}

func TestPostRoot(t *testing.T) {
	env := newTestEnv(t)

	done := make(chan struct{})
	defer close(done)
	commitPeriodically(t, env.backend, done)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	root := big.NewInt(12345)
	crlNumber := big.NewInt(100)

	tx, err := env.relayer.PostRoot(ctx, IssuerG2, root, crlNumber)
	if err != nil {
		t.Fatalf("PostRoot failed: %v", err)
	}
	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	// Verify on-chain state.
	stored, err := env.contract.GetRoot(nil, IssuerG2)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Cmp(root) != 0 {
		t.Errorf("stored root = %s, want %s", stored, root)
	}
}

func TestPostRootStaleCRL(t *testing.T) {
	env := newTestEnv(t)

	done := make(chan struct{})
	defer close(done)
	commitPeriodically(t, env.backend, done)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First post succeeds.
	_, err := env.relayer.PostRoot(ctx, IssuerG2, big.NewInt(111), big.NewInt(100))
	if err != nil {
		t.Fatal(err)
	}

	// Second post with same CRL number should fail with "stale CRL".
	_, err = env.relayer.PostRoot(ctx, IssuerG2, big.NewInt(222), big.NewInt(100))
	if err == nil {
		t.Fatal("expected error for stale CRL number, got nil")
	}
	if !strings.Contains(err.Error(), "stale CRL") {
		t.Errorf("expected 'stale CRL' in error, got: %v", err)
	}
}

func TestPostRootMultipleIssuers(t *testing.T) {
	env := newTestEnv(t)

	done := make(chan struct{})
	defer close(done)
	commitPeriodically(t, env.backend, done)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Post G2.
	_, err := env.relayer.PostRoot(ctx, IssuerG2, big.NewInt(111), big.NewInt(10))
	if err != nil {
		t.Fatal(err)
	}

	// Post G3.
	_, err = env.relayer.PostRoot(ctx, IssuerG3, big.NewInt(222), big.NewInt(20))
	if err != nil {
		t.Fatal(err)
	}

	// Verify independently.
	g2Root, _ := env.contract.GetRoot(nil, IssuerG2)
	g3Root, _ := env.contract.GetRoot(nil, IssuerG3)

	if g2Root.Cmp(big.NewInt(111)) != 0 {
		t.Errorf("G2 root = %s, want 111", g2Root)
	}
	if g3Root.Cmp(big.NewInt(222)) != 0 {
		t.Errorf("G3 root = %s, want 222", g3Root)
	}
}
