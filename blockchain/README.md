# Blockchain Module

Generic blockchain abstraction layer supporting multiple blockchain implementations.

## Structure

```
blockchain/
├── types/                # Common data structures
│   └── types.go         # LogEntry, Proof, BatchProof, AuditData, etc.
├── client/              # Client implementations
│   ├── interface.go     # BlockchainClient interface
│   ├── factory.go       # Factory methods
│   └── chainmaker/      # ChainMaker implementation
└── contracts.md         # Smart contract specifications
```

## Common Types

All blockchain implementations use unified types from `types/`:

- **LogEntry** - Log data structure for submission
- **Proof** - Single log attestation proof (tx_hash, block_height)
- **BatchProof** - Batch transaction proof
- **LogStatusInfo** - Processing status for batch results
- **AuditData** - On-chain audit data

## Client Interface

The `BlockchainClient` interface provides blockchain-agnostic operations:

- `SubmitLog()` - Submit single log entry
- `SubmitLogsBatch()` - Submit multiple logs in one transaction
- `FindLogByHash()` - Query log by hash
- `GetLogByTxHash()` - Get transaction details for audit
- `Close()` - Release resources

## Usage

```go
import blockchain "tlng/blockchain/client"

// Create client from config file
client, err := blockchain.NewBlockchainClientFromFile("./config/blockchain.defaults.yml", logger)

// Submit single log
proof, err := client.SubmitLog(ctx, logHash, logContent, orgID, timestamp)

// Submit batch
entries := []types.LogEntry{...}
batchProof, results, err := client.SubmitLogsBatch(ctx, entries)

// Query by hash
content, err := client.FindLogByHash(ctx, logHash)

// Get audit data
auditData, err := client.GetLogByTxHash(ctx, txHash)
```

## Configuration

```yaml
blockchain_type: "chainmaker"  # Currently supported: chainmaker

# Chain-specific config in config/clients/chainmaker.yml
```

## Supported Blockchains

- ✅ **ChainMaker** - Fully implemented
- ⏳ **Ethereum** - Planned
- ⏳ **Hyperledger Fabric** - Planned

## Adding New Blockchains

To add support for a new blockchain:

1. Create implementation package: `blockchain/client/<name>/`
2. Implement `BlockchainClient` interface
3. Use common types from `blockchain/types`
4. Add factory method in `factory.go`
5. Update smart contract specs in `contracts.md`

Example skeleton:
```go
package ethereum

import "tlng/blockchain/types"

type Client struct { /* ... */ }

func (c *Client) SubmitLog(ctx context.Context, logHash, logContent, senderOrgID, timestamp string) (*types.Proof, error) {
    // Implementation
}

func (c *Client) SubmitLogsBatch(ctx context.Context, entries []types.LogEntry) (*types.BatchProof, []types.LogStatusInfo, error) {
    // Implementation
}
```
