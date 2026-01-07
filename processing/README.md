# Processing Worker

Core batch processing logic for blockchain attestation.

## Overview

The Worker consumes messages from Kafka, accumulates them into batches, and submits to blockchain for efficient processing.

## Batch Processing Flow

```
Kafka Messages → Batch Accumulator → Blockchain Submit → Database Update → Kafka ACK
                     ↓
            (BatchSize or Timeout)
```

### Batching Strategy

- **Size-based**: Accumulate up to `batch_size` messages (default: 100)
- **Time-based**: Submit batch after `batch_timeout` (default: 1s)
- **Trigger**: Whichever comes first

### Status Transitions

```
RECEIVED → PROCESSING → COMPLETED (with tx_hash, block_height)
                     ↓
                   FAILED (with error_message, retry_count)
```

## Configuration

From `config/engine.defaults.yml`:

```yaml
worker:
  concurrency: 4              # Number of parallel workers
  batch_size: 100            # Max messages per batch
  batch_timeout: "1s"        # Max wait time for batch
  consumer_retry_delay: "5s" # Delay on Kafka errors
  blockchain_timeout: "15s"  # Blockchain call timeout
```

## Key Features

### 1. Concurrent Workers
Multiple goroutines process batches in parallel (`concurrency` setting).

### 2. Atomic Batch Processing
- All-or-nothing blockchain submission
- Database updates in transaction
- Kafka ACKs only after success

### 3. Error Handling
- Failed messages marked `FAILED` with error details
- Retry count tracked for monitoring
- Max retries configurable (`max_task_retries`)

### 4. Deduplication
Uses `log_hash` as idempotent key - duplicate submissions are rejected by smart contract.

## Code Structure

**`worker.go`**:
- `New()` - Initialize worker with config
- `Run()` - Start worker pool
- `processMessagesInBatch()` - Main batch accumulation loop
- `submitBatchToBlockchain()` - Blockchain submission
- `updateBatchStatusInDB()` - Database updates

## Usage

See [`cmd/engine/README.md`](../cmd/engine/README.md) for running the engine service.
