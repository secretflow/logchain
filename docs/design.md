# Trusted Log Attestation System

## I. Component Responsibilities Overview

| Layer | Core Component | Primary Responsibilities | Tech Stack |
|------|----------|----------|--------|
| **Ingress Layer** | `API Gateway` | Traffic entry and routing | `Traefik`/`Nginx` |
| **Ingestion Layer** | `Benthos` Adapters | Heterogeneous protocol conversion and data normalization | `Benthos` |
| **Ingestion Layer** | Log Ingestion Service | Log reception, hash calculation, queue insertion | `Go` + `gRPC`/`HTTP` |
| **Processing Layer** | Blockchain Processing Service | Message queue consumption and blockchain submission | `Go` + `Kafka` |
| **Storage Layer** | State Database | Task lifecycle tracking | `PostgreSQL` |
| **Blockchain Layer** | ChainMaker | Log attestation and immutable storage | `ChainMaker` |
| **Query Layer** | Query Service | Multi-dimensional queries and status retrieval | `Go` + `REST` |

---

## II. System Architecture

![Simplified Architecture](images/Architecture(Simplified).jpg)

![Detailed Architecture](images/Architecture(Detailed).jpg)

### 1. Ingress Layer - Traffic Entry and Routing

**Component**: `API Gateway` (`Traefik` / `Nginx Ingress`)  
**Responsibilities**: `TLS` termination, unified authentication, standard protocol routing, load balancing

**Key Workflows**:
* **`TLS` Termination**: Handles all `HTTPS` requests, decrypts traffic, allowing internal services to operate without dealing with `SSL`/`TLS` certificates.
* **Unified Authentication**: Authenticates and authorizes all external requests:
    * **`API Key`**: Validates API client identity
    * **`mTLS` + `IP` Whitelist**: Validates consortium member identity
    * Failed authentication results in immediate rejection; successful authentication passes user identity to backend services via HTTP Headers
* **Standard Protocol Routing**:
    * **`HTTP`/`gRPC` Paths**: `POST /v1/logs` or `gRPC SubmitLog` → Forwarded directly to Log Ingestion Service
    * **Query Paths**: `GET /status/...`, `POST /query_by_content`, `GET /log/by_tx/...` etc. → Forwarded to Query Layer

---

### 2. Ingestion Layer - Log Processing and Queuing

#### A. Adapters - Heterogeneous Protocol Conversion

**Component**: `Benthos` (implemented via `Redpanda Connect` configuration in `ingestion/adapters/`)  
**Responsibilities**: Heterogeneous protocol adapter that directly receives external heterogeneous protocol data and converts it to a unified format.

**Key Workflows**:
* Directly receives external heterogeneous protocol data such as `Syslog`, `Kafka`, `S3`
* Performs protocol parsing and data normalization (timestamp formatting, field mapping, etc.)
* Batch processing and buffering optimization (aggregates small requests to improve processing efficiency)
* Forwards processed logs uniformly to Log Ingestion Service

**Implementation Status & Adapter Types**:
* ✅ Implemented via `redpanda-data/connect` (Benthos-compatible) with configuration files under `ingestion/adapters/`
* Supported adapters in the first phase:
  * **Syslog Adapter (`syslog.yml`)**: Listens on UDP `5514` / TCP `6514`, parses syslog messages, maps raw message body into `log_content`, and forwards to `POST /v1/logs`
  * **Kafka Adapter (`kafka-consumer.yml`)**: Consumes from configured Kafka topics (supporting TLS/mTLS), normalizes messages, and forwards to Log Ingestion Service
  * **S3 Adapter (`s3-processor.yml`)**: Connects to `AWS S3` or S3-compatible storage (e.g. MinIO), reads objects line-by-line, wraps each line as `log_content`, and sends batched HTTP requests to Log Ingestion Service
* All adapters support optional rate limiting (`RATE_LIMIT_COUNT` / `RATE_LIMIT_PERIOD` / `RATE_LIMIT_ENABLED`)

**Security Access Control**:
* **S3 Access Control**: Platform provides dedicated S3 buckets, controls client write permissions through IAM policies and pre-signed URLs
* **Syslog Access Control**: Benthos listens on specified ports, combined with network firewalls and IP whitelists to restrict access sources
* **Kafka Access Control**: Prioritizes platform-managed Kafka clusters, controls producer permissions through ACLs; supports cross-cluster connections to client Kafka environments
* **Protocol Layer Validation**: Performs basic format and integrity validation for each protocol type
* **Rate Limiting**: Traffic control based on client identity to prevent data flooding attacks

#### B. Log Ingestion Service - Unified Processing Entry

**Component**: `Go` service with `gRPC` and `HTTP`/`REST` interfaces.  
**Responsibilities**: Unified processing entry for all log flows

**Key Workflows**:
* Receives standardized log content from direct clients (`HTTP`/`gRPC`)
* Obtains caller identity information from request context (from API Gateway or network layer identification)
* Calculates `SHA256` hash and generates `UUID` as `request_id`
* Immediately returns `request_id` and hash to the caller
* Asynchronous batch processing: writes to `State DB` and pushes to `Kafka`

---

### 3. Processing Layer - Blockchain Processing Service

#### A. Internal Message Queue (Internal MQ)

**Component**: `Kafka`  
**Responsibilities**: Serves as a buffer pool for load leveling between ingestion layer and processing services.

**Key Workflows**:
* Receives log tasks from the ingestion layer
* Distributes to different processing workers by partition
* Provides persistent storage and retry mechanisms

#### B. State Database (State DB - PostgreSQL)

**Component**: `PostgreSQL`  
**Primary Responsibilities**: Records and tracks the complete lifecycle status of log attestation tasks.  
**Secondary Purpose**: Provides fast status indexing and data association for the query layer.

**Key Workflows**:
* Receives initial status records from the ingestion layer
* Updates status changes during processing
* Provides query interfaces to the query layer

#### C. Blockchain Processing Service

**Component**: Multi-worker processing service developed in `Go`  
**Responsibilities**: Message queue consumer supporting horizontally scalable distributed processing.

**Key Workflows**:
* Consumes from `MQ` and updates `State DB` status to `PROCESSING`
* Calls blockchain interaction layer (`internal/pkg/blockchain`) for log attestation on chain
* Updates final status in `State DB` based on results (`COMPLETED`/`FAILED`)

---

### 4. Blockchain Layer - Trusted Storage Foundation

#### A. Blockchain Interaction Layer

**Component**: `Go` wrapper package (`internal/pkg/blockchain`)  
**Responsibilities**: Encapsulates ChainMaker `Go SDK` complexity (connection pooling, certificates, signing, calling `SubmitLog`, error handling).

**Key Workflows**:
* Manages connection pool to ChainMaker nodes
* Handles certificate and signing logic
* Encapsulates smart contract invocation interfaces

#### B. ChainMaker Consortium Blockchain

**Component**: `ChainMaker` consortium blockchain nodes  
**Responsibilities**: Provides immutable distributed ledger storage.

**Key Workflows**:
* Receives transaction requests from blockchain interaction layer
* Executes smart contract logic for log attestation
* Achieves multi-node consensus through consensus algorithms
* Generates block and transaction hash credentials

---

### 5. Query Layer - Multi-Dimensional Query Service

#### A. Query Service

**Component**: `Go` service with HTTP/REST interfaces  
**Responsibilities**: Provides unified multi-dimensional query interfaces.

**Architecture**:
```
Nginx Gateway (Authentication)
    ↓
HTTP Handlers (query/service/http/)
    ↓
Core Service Layer (query/service/core/)
    ↓
├─→ Store Interface (storage/store/) - for API 1 & 2
└─→ Blockchain Client (blockchain/client/) - for API 3
```

**Key Workflows**:
* Receives query requests forwarded from API Gateway (already authenticated)
* Extracts authentication context from HTTP headers (set by Nginx)
* Performs permission checks based on user identity and query type
* Routes to appropriate data source (State DB or Blockchain)
* Returns formatted query results with source indication

**Note**: Query layer trusts requests from API Gateway, no need for re-authentication, only performs permission control based on passed identity information.

#### B. Query API Interface Details

**API 1: Task Status Query (for API callers)**
* **Interface**: `GET /v1/query/status/{request_id}`
* **Authentication**: API Gateway authenticates via `API Key`
* **Authentication Headers**: `X-Auth-Method: api-key`, `X-API-Client-ID`, `X-Client-Org-ID`
* **Permission Scope**: Can only query status of self-submitted logs
* **Data Source**: State DB (PostgreSQL)
* **Purpose**: Allows "active push" clients to query attestation status using returned `request_id`

**API 2: Content-Based Reverse Query (for non-API users)**
* **Interface**: `POST /v1/query_by_content`
* **Authentication**: API Gateway authenticates via `API Key`
* **Authentication Headers**: `X-Auth-Method: api-key`, `X-API-Client-ID`, `X-Client-Org-ID`
* **Permission Scope**: Can only query logs produced by own system
* **Request Body**: `{"log_content": "your raw log string"}`
* **Data Source**: State DB (PostgreSQL) - queries by computed SHA-256 hash
* **Purpose**: Allows "passive ingestion" (`Syslog`, `Kafka`) users to reverse lookup on-chain credentials using log content

**API 3: On-Chain Public Audit (for consortium members)**
* **Interface**: `GET /v1/audit/log/{log_hash}`
* **Authentication**: API Gateway authenticates via `mTLS` + `IP` whitelist
* **Authentication Headers**: `X-Auth-Method: mtls`, `X-Cert-Subject`, `X-Member-ID`
* **Permission Scope**: Can audit all on-chain log data (no org restriction)
* **Data Source**: Blockchain (ChainMaker smart contract query)
* **Purpose**: Satisfies "transparent attestation" business requirements, allowing consortium members to verify on-chain content
* **Note**: Uses `log_hash` instead of `tx_hash` for simplified audit interface

---

## III. Complete Data Flow Paths

### Path 1: Standard HTTP/gRPC Clients
```
External Client → [Ingress Layer: TLS termination + authentication + routing] → [Log Ingestion Service: processing + queuing] → [Processing Service: on-chain attestation]
```

### Path 2: Heterogeneous Protocol Sources (Syslog, Kafka, S3)
```
External System → [Benthos Adapters: direct protocol reception + conversion] → [Log Ingestion Service: processing + queuing] → [Processing Service: on-chain attestation]
```

### Path 3: Query Requests
```
Query Client → [Ingress Layer: TLS termination + authentication + routing] → [Query Layer: query processing] → [State DB/Blockchain: data retrieval]
```

---

## IV. Authentication Architecture and Permission Control

### 1. Authentication Strategy Overview

| User Type | Authentication Method | Access Permissions | Typical Scenarios |
|----------|----------|----------|----------|
| **API Clients** | `API Key` | Log submission, status query | Business system integration |
| **Non-API Users** | `API Key` | Content-based reverse query | `Syslog`/`Kafka` users |
| **Consortium Members** | `mTLS`/`IP` whitelist | On-chain data audit | Regulators, auditors |
| **Internal Services** | `Service Token` | Inter-service communication | `Kafka`-`DB` etc. |
| **Heterogeneous Protocol Sources** | `Network/IP control` | Benthos direct access | `Syslog`/`Kafka`/`S3` clients |

---

### 2. Detailed Authentication Mechanisms

#### 2.1 API Key Management (Client Authentication)

**Applicable Scope**: Log Ingestion Service, first two `APIs` of Query Layer  

**Management Mechanism**:
* **Key Generation:** System administrator generates unique `API Key` for each client
* **Permission Binding:** `API Key` is bound to specific organization or system identifier
* **Access Control:** Supports separation of read/write permissions (submission vs query)
* **Key Rotation:** Supports periodic key rotation without affecting business operations

**Security Measures:**
* `API Keys` are stored as hashes, plaintext only used for verification
* Request signature verification: ensures requests are not tampered with
* Rate limiting: prevents `API` abuse and `DDoS` attacks

#### 2.2 Consortium Member Authentication (mTLS + IP Whitelist)

**Applicable Scope**: Third `API` of Query Layer (on-chain audit)  

**Dual Authentication Mechanism:**

**`mTLS` Certificate Authentication:**
* Each consortium member is issued a client certificate
* Certificates are signed by consortium `CA` authority
* Supports certificate revocation mechanism

**`IP` Address Whitelist:**
* Serves as supplementary protection to `mTLS`
* Only allows access from consortium members' specified `IPs`
* Supports `CIDR` subnet configuration

#### 2.3 Internal Service Authentication (Service Token)

**Applicable Scope**: Inter-service communication (Processing Layer → State Database, Processing Layer → Blockchain, etc.)

**Authentication Mechanism**:
* `JWT Token` contains service identifier and permission scope
* `Token` has short validity period, automatically refreshed periodically
* Role-Based Access Control (`RBAC`)

#### 2.4 Heterogeneous Protocol Source Access Control

**Applicable Scope**: Direct access to Benthos adapters (Syslog, Kafka, S3, etc.)

**Security Mechanisms**:
* **Network Layer Control**: Restricts access sources through firewalls and VPC network isolation
* **IP Whitelist**: Only allows pre-configured client IP addresses to access
* **Protocol Layer Validation**: Performs basic format and integrity validation for each protocol type
* **Rate Limiting**: Traffic control based on source IP to prevent data flooding attacks

---

### 3. Permission Control Matrix

| Operation Type | API Clients | Non-API Users | Consortium Members | Internal Services | Heterogeneous Protocol Sources |
|----------|-----------|-----------|----------|----------|------------|
| **Log Submission** | ✅ | ❌ | ❌ | ❌ | ✅ (restricted) |
| **Status Query** | ✅ (self) | ❌ | ❌ | ✅ | ❌ |
| **Content Reverse Query** | ✅ (self) | ✅ (self) | ❌ | ✅ | ❌ |
| **On-Chain Audit** | ❌ | ❌ | ✅ | ✅ | ❌ |
| **System Status** | ❌ | ❌ | ❌ | ✅ | ❌ |

---

### 4. Authentication Implementation Key Points

#### 4.1 Unified Authentication Gateway

All external requests via `HTTPS` must be authenticated through `Ingress Layer`. Failed authentication results in immediate rejection without passing to backend services. Heterogeneous protocol sources access directly through `Benthos`, using network layer security controls.

#### 4.2 Authentication Context Propagation

After successful authentication, API Gateway passes user identity and permission information to backend services via HTTP Headers. Backend services trust this information without re-authentication, only performing permission checks.

#### 4.3 Permission Control

Backend services perform fine-grained permission control based on passed identity information:
* **Query Layer**: Checks if user has permission to query specific log data
* **Log Ingestion Service**: Performs resource quota management based on user identity
* **Internal Inter-Service**: Implements trust through internal network isolation
* **Benthos Adapters**: Implements secure access based on network layer control and protocol validation

#### 4.4 Audit Logging

All authentication events (success/failure) and critical operations must be logged, including:
* Timestamp, client `IP`, user identifier
* Request type, resource, result
* Failure reason (if applicable)

#### 4.5 Security Configuration Requirements

* **Principle of Least Privilege**: Each identity receives only the minimum permissions needed to complete its tasks
* **Periodic Review**: Quarterly review of permission configurations and access logs
* **Emergency Response**: Supports immediate revocation of suspicious identity access privileges