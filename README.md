# Moica Revocation SMT

Pipeline that fetches Taiwan MOICA Certificate Revocation Lists (CRLs), builds a Sparse Merkle Tree (SMT) from revoked serial numbers, serves ZK-friendly membership/non-membership proofs via REST and gRPC, and posts roots on-chain.

## Architecture

```
MOICA CRL (DER)
       │
  CRL Fetcher/Parser (internal/crl)
       │
       ▼
  TreeManager (internal/manager) ── per-issuer SMTs (g2, g3)
       │                                    │
       ▼                                    ▼
  REST API (chi)  +  gRPC API         Chain Relayer
  GET /proof/{issuerId}/{sn}          posts root on-chain
  GET /status                         via SMTRootStorage.sol
```

## Quick Start

```bash
cd server
make build    # → bin/smtserver
make run      # starts REST + gRPC servers

# or run tests
make test

# integration tests (API E2E ~10s + live CRL fetch ~30min)
make test-integration
```

## Snapshots

Pre-built SMT snapshots are published as GitHub Release assets (`snapshot-latest` tag), updated twice daily.

When the server starts, it automatically:
1. Loads local snapshots from `$DATA_DIR/{issuerID}/tree-snapshot.json.gz`
2. Falls back to downloading from the GitHub release
3. Falls back to rebuilding from live CRL data (slow, ~30 min)

To manually download snapshots:
```bash
# Download latest snapshots
cd server
mkdir -p data/g2 data/g3
curl -L -o data/g2/tree-snapshot.json.gz \
  https://github.com/moven0831/moica-revocation-smt/releases/download/snapshot-latest/g2-tree-snapshot.json.gz
curl -L -o data/g3/tree-snapshot.json.gz \
  https://github.com/moven0831/moica-revocation-smt/releases/download/snapshot-latest/g3-tree-snapshot.json.gz
```

Snapshots are available in two formats:
- **JSON** (`.json.gz`) — compatible with `@zk-kit/smt` v1.0.2, used by the server
- **Binary** (`.bin.gz` / `.bin`) — compact format for client-side WASM loading (see [Client-Side WASM](#client-side-wasm))

The latest Merkle roots for each issuer are displayed in the [snapshot-latest release notes](https://github.com/moven0831/moica-revocation-smt/releases/tag/snapshot-latest). Since the SMT is deterministic, anyone can independently verify a root by rebuilding from the same CRL data.

## API

### Usage Examples

```bash
# Check server status
curl localhost:3000/status

# Query a revoked certificate (membership proof)
curl localhost:3000/proof/g2/100048210dd2df2e128096a9282b5ec5

# Query a non-revoked certificate (non-membership proof)
curl localhost:3000/proof/g2/00000000000000000000000000000001
```

### `GET /proof/{issuerId}/{sn}`

Returns a membership or non-membership proof. Serial number accepts hex with or without `0x` prefix (max 32 hex chars).

```json
{
  "issuerId": "g2",
  "serialNumber": "0x100048210dd2df2e128096a9282b5ec5",
  "entry": ["0x...", "0x...", "0x1"],
  "matchingEntry": ["0x...", "0x...", "0x1"],
  "siblings": ["0x...", "0x...", "..."],
  "root": "0x3c2151...",
  "membership": false,
  "depth": 128
}
```

- `entry` — `[key, value, 1]` for the queried serial
- `matchingEntry` — present only for non-membership proofs
- `siblings` — up to 128 sibling hashes (varies by proof)
- All BigInt values are 0x-prefixed hex strings

### `GET /status`

```json
{
  "generations": {
    "g2": {
      "loaded": true,
      "count": 412404,
      "root": "0x3c2151...",
      "crlNumber": 2026031610,
      "loadedAt": "2026-03-16T08:00:00Z"
    }
  },
  "uptimeSeconds": 3600.5
}
```

### gRPC

Service `RevocationProofService` on port 50051 with `GetProof` and `GetStatus` RPCs. See `server/pkg/proto/revocation/revocation.proto`.

## Contract

**SMTRootStorage.sol** — on-chain registry for SMT roots.

**Deployed contract:**

| Network | Address |
|---------|---------|
| Ethereum Mainnet (production) | [`0xf3aAAe2D017dcC9cA901aDC9Da419f1C70362ab1`](https://etherscan.io/address/0xf3aAAe2D017dcC9cA901aDC9Da419f1C70362ab1) |
| Arbitrum Sepolia (legacy testnet) | [`0xc461326eb6e46F10A276B0F14BFFf8b256A43FFA`](https://sepolia.arbiscan.io/address/0xc461326eb6e46F10A276B0F14BFFf8b256A43FFA) |

The contract stores SMT Merkle roots on Ethereum Mainnet so anyone can verify certificate revocation status against a trusted root. A CI relayer updates roots after rebuilding from MOICA CRL data (cron runs twice daily; a transaction is sent only when the CRL actually changes). Each root is tied to a monotonically increasing CRL number to prevent stale updates. Anyone can call `getRoot(issuerId)` to read the latest root and verify proofs off-chain.

| Function | Description |
|----------|-------------|
| `setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber)` | Update root (relayer only, monotonic CRL number) |
| `getRoot(bytes32 issuerId) → uint256` | Read current root |

Issuer IDs: `keccak256("MOICA-G2")`, `keccak256("MOICA-G3")`

Deploy to Ethereum Mainnet (rehearse on the `sepolia` L1 testnet first):

1. Generate a fresh, dedicated relayer keypair: `cast wallet new`
2. Fund the relayer address with mainnet ETH (covers gas for ~2 `setRoot` tx per CRL change; keep a buffer for gas spikes)
3. Deploy with the optimizer-enabled `production` build profile:
```bash
cd onchain-contract
nvm use 22
pnpm install
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts \
  --network mainnet --build-profile production \
  --parameters '{"SMTRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```
4. Verify the source on Etherscan for transparency
5. Set GitHub Actions secrets (`RPC_URL` → mainnet RPC, `RELAYER_PRIVATE_KEY` hex no `0x`, `CONTRACT_ADDRESS` → deployed address) and optionally the variables `RELAYER_MAX_FEE_GWEI` / `RELAYER_TX_TIMEOUT_SEC` to tune the gas ceiling and confirmation timeout

Post roots on-chain manually (reads `root.json` files, skips gracefully if env vars are unset):
```bash
cd server
./bin/smtbuild --post-root
```

See [onchain-contract/README.md](onchain-contract/README.md) for detailed setup instructions.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3000 | REST API port |
| `GRPC_PORT` | 50051 | gRPC port |
| `DATA_DIR` | `./data` | Storage for CRL data and snapshots |
| `CRL_G2_URL` | MOICA G2 endpoint | CRL download URL for G2 issuer |
| `CRL_G3_URL` | MOICA G3 endpoint | CRL download URL for G3 issuer |
| `CRL_POLL_INTERVAL` | 21600 (6h) | CRL polling interval in seconds |
| `RPC_URL` | — | Ethereum JSON-RPC URL |
| `RELAYER_PRIVATE_KEY` | — | Hex private key for chain relayer |
| `CONTRACT_ADDRESS` | — | SMTRootStorage contract address |
| `RELAYER_MAX_FEE_GWEI` | 100 | Max gas fee per gas the relayer will pay; aborts posting above this ceiling |
| `RELAYER_TX_TIMEOUT_SEC` | 180 | Per-tx confirmation timeout before a same-nonce fee-bumped resend |
| `GITHUB_REPO` | `moven0831/moica-revocation-smt` | GitHub repo for snapshot releases |

## Client-Side WASM

A Go WASM module (`smt.wasm`, ~3.3MB) enables loading the full SMT and generating proofs entirely in the browser — no server round-trip needed.

### WASM API

Build: `cd server && make build-wasm` (requires Go 1.24+)

The WASM module exposes these functions on `globalThis`:

| Function | Signature | Description |
|----------|-----------|-------------|
| `smtInitTree` | `(nodeCount, depth)` | Pre-allocate tree structure |
| `smtAddNodeChunk` | `(Uint8Array)` → `number` | Stream binary nodes in chunks, returns count parsed |
| `smtFinalize` | `(rootHex, leafCount)` | Set root and finalize tree for proof generation |
| `smtCreateProof` | `(keyHex)` → `string` | Generate proof, returns JSON with `root`, `entry`, `matchingEntry`, `siblings` |
| `smtVerifyProof` | `(proofJSON)` → `boolean` | Verify a proof |
| `smtGetMemStats` | `()` → `string` | Go runtime memory stats as JSON |

Proof output format: all values are bare hex (no `0x` prefix), matching the server's REST API structure.

### Binary Snapshot Format

The binary format is a compact representation of the SMT node tree, designed for efficient chunked streaming to WASM.

**Header** (52 bytes, big-endian):
```
[0:2]   magic       uint16  0x534D ("SM")
[2:4]   version     uint16  1
[4:8]   nodeCount   uint32
[8:40]  rootHash    [32]byte
[40:44] depth       uint32
[44:52] crlNumber   uint64
```

**Per node** (variable size):
```
[0:1]   type    uint8   (0=branch, 1=leaf)
[1:33]  hash    [32]byte
Branch: [33:65] left [32]byte, [65:97] right [32]byte    (97 bytes total)
Leaf:   [33:65] key [32]byte, [65:97] value [32]byte, [97:129] entryMark [32]byte    (129 bytes total)
```

Build binary snapshots: `cd server && make build-binary` or `./bin/smtbuild --binary`

Convert existing JSON snapshot: `./bin/smtbuild --convert-binary data/g2/tree-snapshot.json.gz`

### Client Integration Guide

Web or mobile clients can use the published release artifacts to load an SMT and generate proofs locally:

1. **Load runtime** — include `wasm_exec.js` (Go's JS glue), instantiate `smt.wasm` via `WebAssembly.instantiateStreaming`
2. **Fetch snapshot** — download `g2-tree-snapshot.bin.gz` and decompress (browser: `DecompressionStream`, native: `gzip` library)
3. **Build tree** — parse 52-byte binary header → `smtInitTree(nodeCount, depth)` → stream nodes via `smtAddNodeChunk(chunk)` in batches of ~10,000 → `smtFinalize(rootHex, leafCount)`
4. **Generate proof** — `smtCreateProof(serialNumberHex)` → parse JSON → convert hex to decimal strings → pad siblings to depth 128
5. **Feed to circuit** — proof fields (`smtRoot`, `smtSiblings[128]`, `smtOldKey`, `smtOldValue`, `smtIsOld0`) become inputs for `SMTNonMembershipVerifier(128)` in the [zkID](https://github.com/privacy-ethereum/zkID/tree/RSA-X.509-Cert) circom circuit

### Release Assets

The `snapshot-latest` release (updated twice daily) includes:

| Asset | Size | Description |
|-------|------|-------------|
| `smt.wasm` | ~3.3MB | Go WASM module |
| `wasm_exec.js` | ~17KB | Go WASM runtime support |
| `g2-tree-snapshot.bin.gz` | ~71MB | Binary snapshot (gzip-compressed) |
| `g2-tree-snapshot.json.gz` | ~76MB | JSON snapshot (server) |

### Performance (Benchmark)

| Metric | Desktop (M3, Chrome) | iPhone (Safari) | Android (Chrome) |
|--------|---------------------|-----------------|------------------|
| Download | 6ms (localhost) | 103ms | 157ms |
| Decompress (.bin.gz) | 296ms | 14.4s | 14.9s |
| Tree load | 2.85s | 1.74s | 5.86s |
| Proof generation | <1ms | 1ms | 3ms |
| WASM heap | 442MB | 424MB | 424MB |
| **Total** | **3.2s** | **16.3s** | **21.1s** |

Run the benchmark locally: `cd server && make benchmark` → opens `http://localhost:8080/benchmark.html`

### Mobile Integration Path

The same binary snapshot and proof format work across platforms:

- **Web** — Go WASM (`smt.wasm`) loaded via `wasm_exec.js` + `WebAssembly.instantiateStreaming`
- **Mobile (iOS/Android)** — Go mobile bindings via `gomobile bind` (`.xcframework` for Swift, `.aar` for Kotlin) using the same binary format
- **Proof conversion** — hex-to-decimal + sibling padding is ~50 lines of platform-agnostic logic, easily ported to any language
- **Circuit** — same `SMTNonMembershipVerifier(128)` circom circuit, same witness format; [mopro](https://github.com/zkmopro/mopro) handles mobile proving

## CI/CD

**`ci.yml`** — runs on push/PR to main:
- Go server: `go test ./...` + build binary
- WASM: verify `smt.wasm` compiles (`GOOS=js GOARCH=wasm`)
- E2E integration: downloads real G2 snapshot (~412k entries), verifies proofs via REST + gRPC
- Contracts: `npx hardhat test` (Node 22)

**`update-smt.yml`** — runs twice daily at 12:00/00:00 UTC+8 (04:00/16:00 UTC):
1. Build server binary + WASM module
2. Fetch CRL, build SMT, export JSON + binary snapshots (skipped if merkle root is unchanged)
3. Upload snapshots, WASM module, and `wasm_exec.js` to GitHub Release (`snapshot-latest`)
4. Post root on-chain via `smtbuild --post-root` (Ethereum Mainnet), bounded by the `RELAYER_MAX_FEE_GWEI` gas ceiling

Required secrets: `RPC_URL`, `RELAYER_PRIVATE_KEY`, `CONTRACT_ADDRESS` (on-chain posting skips gracefully if unset)

## SMT Compatibility

Wire-compatible with `@zk-kit/smt` v1.0.2 (`bigNumbers` mode):

- **Hash:** Poseidon over P-256 base field via `go-poseidon-p256`
- **Tree depth:** 128 (sufficient for MOICA 64–128 bit serial numbers; halves proof size vs 256)
- **Path encoding:** LSB-first (`big.Int.Bit(i)` for i in 0..127)
- **Leaf node:** `Hash3(key, value, 1)` — the `1` is the entry mark
- **Branch node:** `Hash2(left, right)`
- **Proof:** entry `[key, value, 1]` + up to 128 siblings; non-membership includes optional matching entry

## Data Scale

| CRL | Revoked Certs | File Size |
|-----|--------------|-----------|
| G2  | ~412,000     | ~20MB DER |
| G3  | ~103,000     | ~5MB DER  |

## References

- [MOICA](https://moica.nat.gov.tw/) — Taiwan citizen digital certificate
- [Poseidon Hash](https://www.poseidon-hash.info/) — ZK-friendly hash function
- [Hadeshash spec](https://eprint.iacr.org/2019/458.pdf) — Round number parameters
- [zkID](https://github.com/privacy-ethereum/zkID/tree/RSA-X.509-Cert) — ZK identity verification project
