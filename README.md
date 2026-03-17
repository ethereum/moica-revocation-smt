# Moica Revocation SMT

Pipeline that fetches Taiwan MOICA Certificate Revocation Lists (CRLs), builds a Sparse Merkle Tree (SMT) from revoked serial numbers, serves ZK-friendly membership/non-membership proofs via REST and gRPC, and posts roots on-chain. The Go counterpart to [moica-revocation-smt-ts](https://github.com/moven0831/moica-revocation-smt).

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

# integration tests (fetches live CRL data, builds full SMT, ~30min)
make test-integration
```

## API

### `GET /proof/{issuerId}/{sn}`

Returns a membership or non-membership proof. Serial number accepts hex with or without `0x` prefix (max 64 hex chars).

```json
{
  "issuerId": "g2",
  "serialNumber": "0x100048210dd2df2e128096a9282b5ec5",
  "entry": ["0x...", "0x...", "0x1"],
  "matchingEntry": ["0x...", "0x...", "0x1"],
  "siblings": ["0x...", "0x...", "..."],
  "root": "0x3c2151...",
  "membership": false
}
```

- `entry` — `[key, value, 1]` for the queried serial
- `matchingEntry` — present only for non-membership proofs
- `siblings` — 256 sibling hashes
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

| Function | Description |
|----------|-------------|
| `setRoot(bytes32 issuerId, uint256 root, uint256 crlNumber)` | Update root (relayer only, monotonic CRL number) |
| `getRoot(bytes32 issuerId) → uint256` | Read current root |

Issuer IDs: `keccak256("MOICA-G2")`, `keccak256("MOICA-G3")`

Deploy:
```bash
cd onchain-contract
nvm use 22
pnpm install
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts --parameters '{"relayer":"0x..."}'
```

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
| `GITHUB_REPO` | `moven0831/moica-revocation-smt` | GitHub repo for snapshot releases |

## CI/CD

**`ci.yml`** — runs on push/PR to main:
- Go server: `go test ./...` + build binary (integration tests excluded via `//go:build integration` tag)
- Contracts: `npx hardhat test` (Node 22)

**`update-smt.yml`** — runs every 6 hours (cron):
1. Build server binary
2. Fetch CRL, build SMT, export snapshot
3. Upload snapshot to GitHub Release
4. Post root on-chain

Required secrets: `RPC_URL`, `RELAYER_PRIVATE_KEY`, `CONTRACT_ADDRESS`

## SMT Compatibility

Wire-compatible with `@zk-kit/smt` v1.0.2 (`bigNumbers` mode):

- **Hash:** Poseidon over secq256r1 scalar field via `go-poseidon-p256`
- **Tree depth:** 256
- **Path encoding:** LSB-first (`big.Int.Bit(i)` for i in 0..255)
- **Leaf node:** `Hash3(key, value, 1)` — the `1` is the entry mark
- **Branch node:** `Hash2(left, right)`
- **Proof:** entry `[key, value, 1]` + 256 siblings; non-membership includes optional matching entry

## Data Scale

| CRL | Revoked Certs | File Size |
|-----|--------------|-----------|
| G2  | ~412,000     | ~20MB DER |
| G3  | ~103,000     | ~5MB DER  |

## References

- [MOICA](https://moica.nat.gov.tw/) — Taiwan citizen digital certificate
- [Poseidon Hash](https://www.poseidon-hash.info/) — ZK-friendly hash function
- [Hadeshash spec](https://eprint.iacr.org/2019/458.pdf) — Round number parameters
- [zkID](https://github.com/zkmopro/zkID) — ZK identity verification project
