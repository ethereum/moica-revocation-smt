# SMTRootStorage

On-chain registry for Sparse Merkle Tree roots from MOICA Certificate Revocation Lists.

## Deployed Contract

| Network | Address |
|---------|---------|
| Arbitrum Sepolia | [`0xc461326eb6e46F10A276B0F14BFFf8b256A43FFA`](https://sepolia.arbiscan.io/address/0xc461326eb6e46F10A276B0F14BFFf8b256A43FFA) |

The CRL pipeline (fetch → parse → build SMT) produces a Merkle root per issuer, which the CI relayer posts on-chain via `setRoot()`. The contract enforces monotonic CRL numbers to prevent stale updates. Anyone can read the latest root with `getRoot(issuerId)` and verify membership/non-membership proofs off-chain.

## Contract

**SMTRootStorage.sol** (Solidity 0.8.28)

| Function | Description |
|----------|-------------|
| `constructor(address _relayer)` | Sets the authorized relayer |
| `setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber)` | Update root (relayer only) |
| `getRoot(bytes32 issuerId) → uint256` | Read current root |

**State:**
```solidity
address public relayer;
mapping(bytes32 => RootInfo) public roots;

struct RootInfo {
    uint256 root;
    uint256 crlNumber;
    uint256 updatedAt;
}
```

**Events:**
```solidity
event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber);
```

**Modifiers:**
- `onlyRelayer` — reverts with `"unauthorized"` for non-relayer callers
- `setRoot` requires `crlNumber > roots[issuerId].crlNumber` (monotonic)

## Issuer IDs

| Issuer | ID |
|--------|----|
| MOICA G2 | `keccak256("MOICA-G2")` |
| MOICA G3 | `keccak256("MOICA-G3")` |

## Setup

```bash
nvm use 22    # Hardhat 3 requires Node >= 22.10.0
pnpm install
```

## Test

```bash
npx hardhat test
```

## Deploy

### 1. Generate relayer keypair

```bash
cast wallet new
```

Save the private key (without `0x` prefix) and note the address.

### 2. Fund the relayer

Send Arbitrum Sepolia ETH to the relayer address. You can get testnet ETH from an [Arbitrum Sepolia faucet](https://www.alchemy.com/faucets/arbitrum-sepolia).

### 3. Deploy contract

Local/default network:
```bash
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts \
  --parameters '{"SMTRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```

Arbitrum Sepolia (requires `ARB_SEPOLIA_RPC_URL` and `ARB_SEPOLIA_PRIVATE_KEY` env vars):
```bash
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts \
  --network arbitrumSepolia \
  --parameters '{"SMTRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```

### 4. Configure GitHub Actions secrets

Set these repository secrets for automated on-chain posting:

| Secret | Value |
|--------|-------|
| `RPC_URL` | Arbitrum Sepolia RPC endpoint (e.g. from Alchemy/Infura) |
| `RELAYER_PRIVATE_KEY` | Hex private key without `0x` prefix |
| `CONTRACT_ADDRESS` | Deployed `SMTRootStorage` contract address |

## CI/CD Integration

The `update-smt.yml` workflow posts roots on-chain automatically via `smtbuild --post-root` after building SMT snapshots. It reads `root.json` files and calls `SMTRootStorage.setRoot()` for each issuer. Skips gracefully when secrets are not configured (forks/PRs).
