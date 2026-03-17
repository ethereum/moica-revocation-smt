// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

contract SMTRootStorage {
    struct RootInfo {
        uint256 root;
        uint256 crlNumber;
        uint256 updatedAt;
    }

    address public relayer;
    mapping(bytes32 => RootInfo) public roots;

    event RootUpdated(
        bytes32 indexed issuerId,
        uint256 root,
        uint256 crlNumber
    );

    modifier onlyRelayer() {
        require(msg.sender == relayer, "unauthorized");
        _;
    }

    constructor(address _relayer) {
        relayer = _relayer;
    }

    function setRoot(
        bytes32 issuerId,
        uint256 newRoot,
        uint256 crlNumber
    ) external onlyRelayer {
        require(crlNumber > roots[issuerId].crlNumber, "stale CRL");
        roots[issuerId] = RootInfo(newRoot, crlNumber, block.timestamp);
        emit RootUpdated(issuerId, newRoot, crlNumber);
    }

    function getRoot(bytes32 issuerId) external view returns (uint256) {
        return roots[issuerId].root;
    }
}
