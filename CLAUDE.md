# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go+Solidity pipeline that fetches Taiwan MOICA Certificate Revocation Lists (CRLs), builds Sparse Merkle Trees (SMTs) from revoked serial numbers, serves ZK-friendly membership/non-membership proofs via REST and gRPC, and posts roots on-chain. Wire-compatible with `@zk-kit/smt` v1.0.2 (bigNumbers mode).

## Build & Test Commands

### Go Server (run from `server/`)

```bash
make build              # Compile bin/smtserver
make build-cli          # Compile bin/smtbuild
make test               # Unit tests (excludes integration)
make test-integration   # Live CRL fetch tests (~30 min, requires network)
make proto              # Regenerate gRPC stubs from .proto
make run                # Build and run smtserver

# Single test
go test ./internal/smt -run TestAdd -v
```

### Solidity Contracts (run from `onchain-contract/`)

```bash
source ~/.nvm/nvm.sh && nvm use 22  # Hardhat 3 requires Node >= 22.10.0
pnpm install
npx hardhat test
```

## Architecture

Two entry points in `server/cmd/`:
- **smtserver** — Long-running REST (port 3000) + gRPC (port 50051) server with CRL polling
- **smtbuild** — One-shot CLI: fetch CRL → build SMT → export snapshot (used in CI cron)

### Key packages (`server/internal/`)

| Package | Purpose |
|---------|---------|
| `smt/` | Core SMT: Poseidon hash, 256-depth tree, proof generation/verification |
| `crl/` | CRL HTTP fetcher, DER parser, periodic watcher goroutine |
| `manager/` | Thread-safe per-issuer (`g2`, `g3`) tree management via `TreeManager` |
| `api/rest/` | Chi router: `GET /proof/{issuerId}/{sn}`, `GET /status` |
| `api/grpcapi/` | gRPC mirror of REST API |
| `chain/` | Ethereum client wrapper + relayer for `setRoot` transactions |
| `snapshot/` | Gzip JSON export/import + GitHub Release download |
| `store/` | Store interface with MemoryStore and BadgerStore implementations |
| `config/` | Environment variable loader |

### Startup flow (smtserver)

1. Load config from env vars
2. Import snapshots: local file → GitHub Release fallback → live CRL rebuild
3. Start CRL watcher (polls every 6h, atomic tree replacement via `SetTree`)
4. Start REST + gRPC servers
5. Graceful shutdown on SIGINT/SIGTERM

### Smart Contract (`onchain-contract/contracts/SMTRootStorage.sol`)

Simple root registry: `setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber)` with relayer-only access and monotonic CRL number enforcement. Issuer IDs are `keccak256("MOICA-G2")` / `keccak256("MOICA-G3")`.

## SMT Implementation Details

- **Hash:** Poseidon over P-256 (secq256r1) scalar field via `go-poseidon-p256`
- **Depth:** 128 (sufficient for MOICA 64–128 bit serial numbers; keys exceeding depth are rejected)
- **Path encoding:** LSB-first — `key.Bit(i)` for i in 0..127
- **Leaf node:** `Hash3(key, value, 1)` stored as `[key, value, 1]`
- **Branch node:** `Hash2(left, right)` stored as `[left, right]`
- **Entry value:** Always `1` (membership marker)
- **Thread safety:** RWMutex on all public SMT and TreeManager methods

## CI/CD

- **ci.yml** — On push/PR: Go unit tests + build, Hardhat contract tests
- **update-smt.yml** — Cron (04:00 & 16:00 UTC): smtbuild → commit snapshots → upload to `snapshot-latest` release

## Data Scale

| CRL | Revoked Certs | DER Size |
|-----|---------------|----------|
| G2  | ~412,000      | ~20MB    |
| G3  | ~103,000      | ~5MB     |

Integration tests (`//go:build integration`) fetch live data and take ~30 minutes.
