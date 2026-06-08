# SMTRootStorage

On-chain registry for Sparse Merkle Tree roots from MOICA Certificate Revocation Lists.

## Deployed Contract

| Network | Address |
|---------|---------|
| Ethereum Mainnet (production) | [`0xf3aAAe2D017dcC9cA901aDC9Da419f1C70362ab1`](https://etherscan.io/address/0xf3aAAe2D017dcC9cA901aDC9Da419f1C70362ab1) |
| Arbitrum Sepolia (legacy testnet) | [`0xc461326eb6e46F10A276B0F14BFFf8b256A43FFA`](https://sepolia.arbiscan.io/address/0xc461326eb6e46F10A276B0F14BFFf8b256A43FFA) |

The CRL pipeline (fetch → parse → build SMT) produces a Merkle root per issuer, which the CI relayer posts on Ethereum Mainnet via `setRoot()`. The contract enforces monotonic CRL numbers to prevent stale updates. Anyone can read the latest root with `getRoot(issuerId)` and verify membership/non-membership proofs off-chain.

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

Send mainnet ETH to the relayer address — enough to cover gas for ~2 `setRoot` transactions per CRL change, with a buffer for gas spikes. Use a fresh, dedicated key (do not reuse a testnet key).

### 3. Deploy contract

Rehearse on the Ethereum Sepolia L1 testnet first (requires `SEPOLIA_RPC_URL` and `SEPOLIA_PRIVATE_KEY`) — it exercises real EIP-1559 and the relayer fee ceiling, unlike the Arbitrum testnet:
```bash
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts \
  --network sepolia --build-profile production \
  --parameters '{"SMTRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```

Ethereum Mainnet (requires `MAINNET_RPC_URL` and `MAINNET_PRIVATE_KEY`), deployed with the optimizer-enabled `production` profile:
```bash
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts \
  --network mainnet --build-profile production \
  --parameters '{"SMTRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```

Then verify the contract source on Etherscan for transparency.

### 4. Configure GitHub Actions secrets

Set these repository secrets for automated on-chain posting:

| Secret | Value |
|--------|-------|
| `RPC_URL` | Ethereum Mainnet RPC endpoint (e.g. from Alchemy/Infura) |
| `RELAYER_PRIVATE_KEY` | Hex private key without `0x` prefix |
| `CONTRACT_ADDRESS` | Deployed `SMTRootStorage` contract address |

Optionally set these repository **variables** to tune mainnet gas behavior (defaults apply if unset):

| Variable | Default | Value |
|----------|---------|-------|
| `RELAYER_MAX_FEE_GWEI` | 100 | Max gas fee per gas the relayer will pay; it skips posting above this ceiling |
| `RELAYER_TX_TIMEOUT_SEC` | 180 | Per-tx confirmation timeout before a same-nonce fee-bumped resend |

## CI/CD Integration

The `update-smt.yml` workflow posts roots on-chain automatically via `smtbuild --post-root` after building SMT snapshots. It reads `root.json` files and calls `SMTRootStorage.setRoot()` for each issuer. Skips gracefully when secrets are not configured (forks/PRs).
