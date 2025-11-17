# SOC Check Documentation

## Overview

The SOC (Single Owner Chunk) check validates the upload and retrieval of SOC chunks in the Swarm network, with configurable redundancy levels and cache behavior.

## What it does

1. **SOC Chunk Creation**: Creates a SOC chunk with test payload "Hello Swarm :)" using cryptographic signing
2. **Upload**: Uploads the SOC chunk to a bee node with specified redundancy level
3. **Original Retrieval**: Downloads and validates the original chunk from the upload node
4. **Replica Testing**: Tests retrieval of replica chunks created by the redundancy system

## Key Features

### Configurable Redundancy

- **Upload Redundancy Level**: Controls how many replica chunks are created during upload
- **Download Redundancy Level**: Controls how many replica chunks are tested during download
- Supports levels: NONE, MEDIUM, STRONG, INSANE, PARANOID

### Cache Behavior Control

- **Cache = true**: Downloads replicas from local node storage (cache)
- **Cache = false**: Downloads replicas from network peers with retry logic

### Multi-Node Support

- **Cache = true**: Uses single node for upload and download
- **Cache = false**: Uses separate nodes (upload to node1, download replicas from node2)

### Retry Logic

When `Cache = false`, implements retry mechanism:

- Up to 5 retry attempts per replica download
- 1-second delay between retries
- Allows time for chunks to propagate through the network

## Validation Logic

The check validates that:

- Original chunk data matches after upload/download
- Expected number of replica retrievals succeed/fail based on redundancy levels
- Failed replica retrievals return expected error types (HTTP 500 "read chunk failed")
- Replica chunk data matches the original chunk data

## Use Cases

- **Network Dispersal Testing**: Verify chunks are properly distributed across peer nodes
- **Cache vs Network Performance**: Compare local cache vs network retrieval speeds
- **Redundancy Validation**: Ensure redundancy levels work as expected
- **Error Handling**: Test proper error responses when replicas are unavailable

## Configuration

```yaml
cache: true/false             # Local cache vs network retrieval
upload-r-level: 3             # Upload redundancy level  
download-r-level: 4           # Download redundancy level
request-timeout: 5m           # Timeout for operations
```
