# SMTRootStorage

On-chain registry for Sparse Merkle Tree roots from MOICA Certificate Revocation Lists.

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

```bash
npx hardhat ignition deploy ignition/modules/SMTRootStorage.ts --parameters '{"relayer":"0x..."}'
```
