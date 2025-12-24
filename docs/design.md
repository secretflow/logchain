# Trusted Log Attestation System

## I. Component Responsibilities Overview

| Layer | Core Component | Primary Responsibilities | Tech Stack |
|------|----------|----------|--------|
| **Ingress Layer** | `API Gateway` | Traffic entry and routing | `Nginx + OpenResty` |
| **Ingestion Layer** | `Benthos` Adapters | Heterogeneous protocol conversion and data normalization | `Redpanda Connect` |
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

**Component**: `API Gateway` (Nginx with OpenResty for Lua scripting)  
**Responsibilities**: `TLS` termination, unified authentication, standard protocol routing, load balancing, rate limiting, audit logging

**Key Workflows**:
* **`TLS` Termination**: Handles all `HTTPS` requests (port 443) and gRPC (port 50052), decrypts traffic, allowing internal services to operate without dealing with `SSL`/`TLS` certificates.
* **Unified Authentication**: Lua-based authentication in Nginx:
    * **`API Key`**: File/Redis/external service-based validation (configured via `API_KEY_AUTH_METHOD`)
    * **`mTLS` + `IP` Whitelist**: Dual authentication for consortium members with certificate validation and IP whitelist checking
    * Failed authentication results in immediate rejection; successful authentication passes identity via HTTP headers (`X-API-Client-ID`, `X-Client-Org-ID`, `X-Auth-Method`, etc.)
* **Rate Limiting**: Configurable per-endpoint limits (100 req/s for submission, 50 req/s for query, 20 req/s for audit)
* **Standard Protocol Routing**:
    * **`HTTP` Paths**: `POST /v1/logs` → Ingestion Service HTTP endpoint (port 8091)
    * **`gRPC` Paths**: `SubmitLog` → Ingestion Service gRPC endpoint (port 50051)
    * **Query Paths**: `GET /v1/query/status/{request_id}`, `POST /v1/query_by_content` → Query Service (port 8083)
    * **Audit Paths**: `GET /v1/audit/log/{log_hash}` → Query Service

---

### 2. Ingestion Layer - Log Processing and Queuing

#### A. Adapters - Heterogeneous Protocol Conversion

**Component**: `Redpanda Connect` (Benthos-compatible stream processor)  
**Responsibilities**: Heterogeneous protocol adapter that directly receives external protocol data and converts it to a unified format.

**Key Workflows**:
* Directly receives external heterogeneous protocol data such as `Syslog`, `Kafka`, `S3`
* Performs protocol parsing and data normalization (timestamp formatting, field mapping, etc.)
* Optional rate limiting with configurable thresholds
* Forwards processed logs uniformly to Log Ingestion Service via `POST /v1/logs`

**Implementation Status & Adapter Types**:
* Implemented using `redpanda-data/connect` Docker image
* Three adapter configurations:
  * **Syslog Adapter**: Listens on UDP `5514` and TCP `6514`, parses RFC5424 format, forwards to Ingestion Service
  * **Kafka Adapter**: Consumes from configurable Kafka topics, supports TLS/mTLS authentication
  * **S3 Adapter**: Connects to AWS S3 or S3-compatible storage (MinIO), reads objects line-by-line
* All adapters support optional rate limiting and environment variable configuration

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
HTTP Handlers
    ↓
Core Service Layer
    ↓
├─→ Database Store - for API 1 & 2
└─→ Blockchain Client - for API 3
```

**Key Workflows**:
* Receives query requests forwarded from API Gateway (already authenticated)
* Extracts authentication context from HTTP headers (set by Nginx)
* Performs permission checks based on user identity and query type
* Routes to appropriate data source (State DB or Blockchain)
* Returns formatted query results with source indication

**Note**: Query layer trusts requests from API Gateway (Nginx), authentication is already completed. Query service only extracts auth context from HTTP headers and performs permission checks.

#### B. Query API Interface Details

**API 1: Task Status Query (for API callers)**
* **Interface**: `GET /v1/query/status/{request_id}`
* **Authentication**: Nginx validates via `API Key`
* **Authentication Headers**: `X-Auth-Method: api-key`, `X-API-Client-ID`, `X-Client-Org-ID`
* **Permission Scope**: Can only query logs submitted by own organization
* **Data Source**: State DB (PostgreSQL)
* **Purpose**: Allows "active push" clients to query attestation status using returned `request_id`

**API 2: Content-Based Reverse Query (for non-API users)**
* **Interface**: `POST /v1/query_by_content`
* **Authentication**: Nginx validates via `API Key`
* **Authentication Headers**: `X-Auth-Method: api-key`, `X-API-Client-ID`, `X-Client-Org-ID`
* **Permission Scope**: Can only query logs from own organization
* **Request Body**: `{"log_content": "your raw log string"}`
* **Process**: Service computes SHA-256 hash and queries by hash
* **Data Source**: State DB (PostgreSQL)
* **Purpose**: Allows "passive ingestion" (`Syslog`, `Kafka`) users to reverse lookup on-chain credentials

**API 3: On-Chain Public Audit (for consortium members)**
* **Interface**: `GET /v1/audit/log/{log_hash}`
* **Authentication**: Nginx validates via `mTLS` + `IP` whitelist
* **Authentication Headers**: `X-Auth-Method: mtls`, `X-Cert-Subject`, `X-Member-ID`
* **Permission Scope**: Can audit all on-chain log data (no organization restriction)
* **Data Source**: Blockchain (ChainMaker)
* **Purpose**: Allows consortium members to verify on-chain content and satisfy transparent attestation requirements

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
| **Consortium Members** | `mTLS`+`IP` whitelist | On-chain data audit | Regulators, auditors |
| **Internal Services** | Service mesh/network isolation | Inter-service communication | `Ingestion`-`Engine`-`Query` |
| **Heterogeneous Protocol Sources** | Network/IP control + Rate limiting | Benthos direct access | `Syslog`/`Kafka`/`S3` clients |

---

### 2. Detailed Authentication Mechanisms

#### 2.1 API Key Management (Client Authentication)

**Applicable Scope**: Log Ingestion Service, Query APIs

**Management Mechanism**:
* **Storage Methods**: File-based (development), Redis-based (production), or external auth service
* **Key Structure**: Contains `client_id`, `org_id`, `status`, `permissions`, and expiration
* **Validation**: Nginx Lua scripts validate keys and inject identity headers
* **Header Injection**: Sets `X-API-Client-ID`, `X-Client-Org-ID`, `X-Auth-Method: api-key`

**Security Measures:**
* Rate limiting per endpoint
* Audit logging of authentication attempts
* Support for key rotation and expiration

#### 2.2 Consortium Member Authentication (mTLS + IP Whitelist)

**Applicable Scope**: Query Layer audit API (`/v1/audit/log/{log_hash}`)

**Dual Authentication Mechanism:**

**`mTLS` Certificate Authentication:**
* Nginx validates client certificates against consortium CA
* Certificate subject passed via `X-Cert-Subject` header
* Supports certificate revocation and rotation

**`IP` Address Whitelist:**
* Configured per consortium member with allowed IP addresses
* Lua script validates client IP against member's whitelist
* Member ID passed via `X-Member-ID` header

#### 2.3 Internal Service Authentication

**Applicable Scope**: Inter-service communication

**Security Mechanism**:
* Services communicate within isolated Docker network
* Internal ports not exposed externally
* Database credentials via environment variables

#### 2.4 Heterogeneous Protocol Source Access Control

**Applicable Scope**: Benthos adapters (Syslog, Kafka, S3)

**Security Mechanisms**:
* **Network Layer Control**: Docker network isolation with selective port exposure
* **Rate Limiting**: Configurable per adapter to prevent data flooding
* **Protocol Validation**: Format validation and integrity checks for each protocol
* **Organization Tagging**: Default organization ID injected for tracking

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

All external HTTPS/gRPC requests pass through Nginx authentication:
* API Key routes validated by Lua scripts, inject identity headers
* mTLS routes validate certificates and IP whitelist
* Failed authentication returns 401/403 immediately with audit logging
* Successful authentication forwards request with identity headers

Benthos adapters bypass Nginx, using network isolation and rate limiting.

#### 4.2 Authentication Context Propagation

Nginx injects identity via HTTP headers after successful authentication:
* **API Key**: `X-Auth-Method: api-key`, `X-API-Client-ID`, `X-Client-Org-ID`
* **mTLS**: `X-Auth-Method: mtls`, `X-Cert-Subject`, `X-Member-ID`

Backend services extract headers and perform permission checks without re-authentication.

#### 4.3 Permission Control

Backend services perform permission control based on authentication headers:
* **Ingestion Service**: Associates logs with organization from `X-Client-Org-ID`
* **Query Service API Key routes**: Filters results by organization (users see own logs only)
* **Query Service mTLS routes**: No organization filtering (consortium members see all logs)

#### 4.4 Audit Logging

All authentication and critical operations are logged:
* **Authentication Events**: Success/failure with timestamp, client IP, auth method, client ID
* **Access Logs**: Standard format with custom fields for client identifiers
* **Error Logs**: Authentication failures, rate limit violations, TLS errors

#### 4.5 Security Configuration Requirements

* **Principle of Least Privilege**: Each identity receives only the minimum permissions needed to complete its tasks
* **Periodic Review**: Quarterly review of permission configurations and access logs
* **Emergency Response**: Supports immediate revocation of suspicious identity access privileges