# ChainMaker Deployment & Configuration Guide

This guide outlines the steps to deploy the ChainMaker blockchain and smart contracts required for the LogChain system.

## 1. ChainMaker Deployment

To deploy ChainMaker locally via the command line, please refer to the official Quick Start guide:
[ChainMaker Command Line Experience](https://docs.chainmaker.org.cn/quickstart/%E9%80%9A%E8%BF%87%E5%91%BD%E4%BB%A4%E8%A1%8C%E4%BD%93%E9%AA%8C%E9%93%BE.html)

## 2. Smart Contract Deployment

We use Rust for smart contract development. For detailed instructions on developing contracts with Rust, refer to:
[ChainMaker Rust Contract Development](https://docs.chainmaker.org.cn/v2.3.7/html/instructions/%E4%BD%BF%E7%94%A8Rust%E8%BF%9B%E8%A1%8C%E6%99%BA%E8%83%BD%E5%90%88%E7%BA%A6%E5%BC%80%E5%8F%91.html)

### Contract Implementation
The Rust contract source code can be found in `blockchain/contracts.md`. Please copy the code from there to create your contract project.

### ⚠️ Rust Version Requirements

**Crucial for Compilation:**
Although official documentation might mention older versions (like 1.63.0), our testing confirms:

-   ❌ **v1.63.0**: Too old, causes errors.
-   ❌ **v1.75+**: Encounter compilation issues.
-   ✅ **v1.71.1**: **Recommended and verified version.**

## 3. Client Configuration

After deploying ChainMaker, you must update the client configuration to point to your local installation.

**File:** `config/clients/chainmaker.yml`

Update the paths (e.g., `user_key_path`, `user_cert_path`, `ca_paths`) to match your actual ChainMaker installation directory.
