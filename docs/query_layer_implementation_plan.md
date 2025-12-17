# Query Layer 实现计划

## 概述

本文档描述 Query Layer 的详细实现计划，包括 API 设计、架构、实施步骤等。

**设计方案**: 基于方案 1（简化审计接口 - 只用 log_hash）  
**创建时间**: 2024-12-17  
**状态**: 待实施

---

## 📋 API 设计（3个核心接口）

### API 1: 任务状态查询

**接口**: `GET /v1/status/{request_id}`

**认证方式**: API Key（从 Nginx Headers）
- `X-API-Client-ID`: 客户端 ID
- `X-Client-Org-ID`: 组织 ID
- `X-Auth-Method`: api-key

**权限控制**: 
- 只能查询自己组织提交的日志
- 通过 `source_org_id` 匹配验证

**数据源**: State DB (`tbl_log_status`)

**返回示例**:
```json
{
  "request_id": "uuid-123",
  "log_hash": "sha256...",
  "source_org_id": "org-abc",
  "status": "COMPLETED",
  "received_timestamp": "2024-01-01T12:00:00Z",
  "processing_started_at": "2024-01-01T12:00:01Z",
  "processing_finished_at": "2024-01-01T12:00:05Z",
  "tx_hash": "0xabc...",
  "block_height": 12345
}
```

**状态说明**:
- `RECEIVED`: 已接收，等待处理
- `PROCESSING`: 正在处理中
- `COMPLETED`: 已成功上链
- `FAILED`: 处理失败

---

### API 2: 内容反向查询

**接口**: `POST /v1/query_by_content`

**认证方式**: API Key（同 API 1）

**权限控制**: 只能查询自己组织的日志

**请求体**:
```json
{
  "log_content": "your raw log string"
}
```

**处理逻辑**:
1. 计算 `log_content` 的 SHA256 hash
2. 通过 `log_hash` 在 State DB 查询
3. 验证 `source_org_id` 匹配

**数据源**: State DB

**返回内容**: 与 API 1 相同的格式

**使用场景**: 
- 适用于 Syslog、Kafka 等被动采集方式的用户
- 通过原始日志内容反向查询上链状态

---

### API 3: 链上公开审计

**接口**: `GET /v1/audit/log/{log_hash}`

**认证方式**: mTLS + IP 白名单（从 Nginx 传递）
- `X-Cert-Subject`: 证书 DN
- `X-Member-ID`: 联盟成员 ID
- `X-Auth-Method`: mtls

**权限控制**: 
- 联盟成员可审计所有日志
- 无组织限制

**数据源**: Blockchain（直接调用智能合约 `find_log_by_hash`）

**返回示例**:
```json
{
  "source": "blockchain",
  "log_hash": "sha256...",
  "log_content": "original log content",
  "sender_org_id": "org-abc",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

**注意事项**:
- 返回的是链上不可篡改的原始数据
- 审计者可以验证：给定日志内容 → 计算 hash → 对比链上记录

---

## 🏗️ 目录结构

```
config/
├── query.go                     # Query 服务配置结构
├── query.defaults.yml           # Query 服务默认配置
├── blockchain.go
├── database.go
└── ...

query/
├── README.md                    # 已存在，需更新
├── cmd/
│   └── server/
│       └── main.go              # Query 服务主入口
├── service/
│   ├── core/
│   │   └── service.go           # 核心查询业务逻辑
│   └── http/
│       └── handler.go           # HTTP API 处理器
└── auth/
    └── middleware.go            # 认证中间件
```

**复用现有组件**:
- `storage/store/` - 扩展查询方法
- `blockchain/client/` - 复用现有的 `FindLogByHash` 方法
- `internal/models/` - 复用数据模型

---

## 🔧 核心组件设计

### 1. 认证中间件 (`query/auth/middleware.go`)

**功能**: 从 Nginx 传递的 Headers 提取身份信息

**认证上下文结构**:
```go
type AuthContext struct {
    ClientID    string   // from X-API-Client-ID
    OrgID       string   // from X-Client-Org-ID
    AuthMethod  string   // "api-key" or "mtls"
    CertSubject string   // from X-Cert-Subject (mTLS only)
    MemberID    string   // from X-Member-ID (mTLS only)
}
```

**核心函数**:
```go
// 从 HTTP Headers 提取认证信息
func ExtractAuthContext(r *http.Request) (*AuthContext, error)

// API Key 认证中间件
func RequireAPIKey(next http.Handler) http.Handler

// mTLS 认证中间件
func RequireMTLS(next http.Handler) http.Handler
```

**Nginx Headers 规范**:
```
# API Key 认证路径
X-API-Client-ID: client-001
X-Client-Org-ID: org-abc
X-Auth-Method: api-key

# mTLS 认证路径
X-Cert-Subject: CN=member01,O=Consortium
X-Member-ID: member-001
X-Auth-Method: mtls
```

---

### 2. Store 接口扩展 (`storage/store/store.go`)

**新增查询方法**:
```go
type Store interface {
    // 已有方法...
    
    // 按 request_id 查询日志状态
    GetLogStatusByRequestID(ctx context.Context, requestID string) (*LogStatus, error)
    
    // 按 log_hash 查询日志状态
    GetLogStatusByHash(ctx context.Context, logHash string) (*LogStatus, error)
}
```

**PostgreSQL 实现** (`storage/store/postgres.go`):

```go
func (s *PostgresStore) GetLogStatusByRequestID(ctx context.Context, requestID string) (*LogStatus, error) {
    query := `
        SELECT request_id, log_hash, source_org_id, received_timestamp, 
               status, received_at_db, processing_started_at, processing_finished_at,
               tx_hash, block_height, log_hash_on_chain, error_message, retry_count
        FROM tbl_log_status
        WHERE request_id = $1
    `
    var status LogStatus
    err := s.db.QueryRow(ctx, query, requestID).Scan(...)
    if err == pgx.ErrNoRows {
        return nil, ErrLogNotFound
    }
    return &status, err
}

func (s *PostgresStore) GetLogStatusByHash(ctx context.Context, logHash string) (*LogStatus, error) {
    query := `
        SELECT request_id, log_hash, source_org_id, received_timestamp,
               status, received_at_db, processing_started_at, processing_finished_at,
               tx_hash, block_height, log_hash_on_chain, error_message, retry_count
        FROM tbl_log_status
        WHERE log_hash = $1
    `
    var status LogStatus
    err := s.db.QueryRow(ctx, query, logHash).Scan(...)
    if err == pgx.ErrNoRows {
        return nil, ErrLogNotFound
    }
    return &status, err
}
```

**数据库索引优化**:
```sql
-- 添加必要的索引以提升查询性能
CREATE INDEX IF NOT EXISTS idx_log_status_request_id ON tbl_log_status(request_id);
CREATE INDEX IF NOT EXISTS idx_log_status_log_hash ON tbl_log_status(log_hash);
```

---

### 3. Blockchain 客户端集成

**复用现有接口**: `blockchain/client/interface.go` 中已有的 `FindLogByHash` 方法

```go
// 现有接口（无需修改）
type BlockchainClient interface {
    FindLogByHash(ctx context.Context, logHash string) (string, error)
    // 返回格式: "org_id=xxx&ts=xxx&content=xxx"
}
```

**ChainMaker 实现**: `blockchain/client/chainmaker/client.go` 已实现

**在 Query Service 中解析链上数据**:
```go
type OnChainLogData struct {
    OrgID     string
    Timestamp string
    Content   string
}

func parseOnChainData(raw string) (*OnChainLogData, error) {
    // 解析 "org_id=xxx&ts=xxx&content=xxx" 格式
    params := parseQueryString(raw)
    return &OnChainLogData{
        OrgID:     params["org_id"],
        Timestamp: params["ts"],
        Content:   params["content"],
    }, nil
}
```

---

### 4. 核心服务 (`query/service/core/service.go`)

**服务结构**:
```go
type Service struct {
    store      store.Store
    blockchain blockchain.BlockchainClient
    logger     *log.Logger
}

func NewService(store store.Store, bc blockchain.BlockchainClient, logger *log.Logger) *Service {
    return &Service{
        store:      store,
        blockchain: bc,
        logger:     logger,
    }
}
```

**API 1: 按 request_id 查询状态**:
```go
func (s *Service) GetStatusByRequestID(ctx context.Context, requestID, callerOrgID string) (*LogStatusResponse, error) {
    // 1. 从 State DB 查询
    status, err := s.store.GetLogStatusByRequestID(ctx, requestID)
    if err != nil {
        if errors.Is(err, store.ErrLogNotFound) {
            return nil, ErrLogNotFound
        }
        return nil, fmt.Errorf("failed to query database: %w", err)
    }
    
    // 2. 权限检查：只能查询自己组织的日志
    if status.SourceOrgID != callerOrgID {
        return nil, ErrPermissionDenied
    }
    
    // 3. 转换为响应格式
    return convertToResponse(status), nil
}
```

**API 2: 按内容查询**:
```go
func (s *Service) QueryByContent(ctx context.Context, logContent, callerOrgID string) (*LogStatusResponse, error) {
    // 1. 计算 log_hash
    logHash := calculateSHA256(logContent)
    
    // 2. 从 State DB 查询
    status, err := s.store.GetLogStatusByHash(ctx, logHash)
    if err != nil {
        if errors.Is(err, store.ErrLogNotFound) {
            return nil, ErrLogNotFound
        }
        return nil, fmt.Errorf("failed to query database: %w", err)
    }
    
    // 3. 权限检查
    if status.SourceOrgID != callerOrgID {
        return nil, ErrPermissionDenied
    }
    
    return convertToResponse(status), nil
}
```

**API 3: 链上审计查询（无权限限制）**:
```go
func (s *Service) AuditLogByHash(ctx context.Context, logHash string) (*OnChainLogResponse, error) {
    // 1. 调用 Blockchain Client 查询链上数据
    rawData, err := s.blockchain.FindLogByHash(ctx, logHash)
    if err != nil {
        return nil, fmt.Errorf("failed to query blockchain: %w", err)
    }
    
    if rawData == "" {
        return nil, ErrLogNotFound
    }
    
    // 2. 解析链上数据
    logData, err := parseOnChainData(rawData)
    if err != nil {
        return nil, fmt.Errorf("failed to parse on-chain data: %w", err)
    }
    
    // 3. 返回结构化响应
    return &OnChainLogResponse{
        Source:      "blockchain",
        LogHash:     logHash,
        LogContent:  logData.Content,
        SenderOrgID: logData.OrgID,
        Timestamp:   logData.Timestamp,
    }, nil
}
```

**辅助函数**:
```go
func calculateSHA256(content string) string {
    hash := sha256.Sum256([]byte(content))
    return hex.EncodeToString(hash[:])
}

func convertToResponse(status *store.LogStatus) *LogStatusResponse {
    resp := &LogStatusResponse{
        RequestID:         status.RequestID,
        LogHash:           status.LogHash,
        SourceOrgID:       status.SourceOrgID,
        Status:            string(status.Status),
        ReceivedTimestamp: status.ReceivedTimestamp,
    }
    
    if status.ProcessingStartedAt != nil {
        resp.ProcessingStartedAt = status.ProcessingStartedAt
    }
    if status.ProcessingFinishedAt != nil {
        resp.ProcessingFinishedAt = status.ProcessingFinishedAt
    }
    if status.TxHash != nil {
        resp.TxHash = *status.TxHash
    }
    if status.BlockHeight != nil {
        resp.BlockHeight = *status.BlockHeight
    }
    
    return resp
}
```

---

### 5. HTTP Handlers (`query/service/http/handler.go`)

**Handler 结构**:
```go
type Handler struct {
    svc    *core.Service
    logger *log.Logger
}

func NewHandler(svc *core.Service, logger *log.Logger) *Handler {
    return &Handler{
        svc:    svc,
        logger: logger,
    }
}
```

**路由设置**:
```go
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
    // API 1 & 2: 需要 API Key 认证
    mux.Handle("/v1/status/", 
        auth.RequireAPIKey(http.HandlerFunc(h.GetStatus)))
    mux.Handle("/v1/query_by_content", 
        auth.RequireAPIKey(http.HandlerFunc(h.QueryByContent)))
    
    // API 3: 需要 mTLS 认证
    mux.Handle("/v1/audit/log/", 
        auth.RequireMTLS(http.HandlerFunc(h.AuditLog)))
    
    // 健康检查
    mux.HandleFunc("/health", h.Health)
}
```

**Handler 实现**:
```go
// API 1: GET /v1/status/{request_id}
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        h.respondError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // 提取认证上下文
    authCtx := auth.GetAuthContext(r.Context())
    if authCtx == nil {
        h.respondError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // 提取 request_id (从 /v1/status/{request_id} 路径)
    requestID := extractRequestID(r.URL.Path)
    if requestID == "" {
        h.respondError(w, "Invalid request_id", http.StatusBadRequest)
        return
    }
    
    // 调用 Service 层
    result, err := h.svc.GetStatusByRequestID(r.Context(), requestID, authCtx.OrgID)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }
    
    h.respondJSON(w, result, http.StatusOK)
}

// API 2: POST /v1/query_by_content
func (h *Handler) QueryByContent(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        h.respondError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }
    
    authCtx := auth.GetAuthContext(r.Context())
    if authCtx == nil {
        h.respondError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // 解析请求体
    var req struct {
        LogContent string `json:"log_content"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    if req.LogContent == "" {
        h.respondError(w, "log_content is required", http.StatusBadRequest)
        return
    }
    
    // 调用 Service 层
    result, err := h.svc.QueryByContent(r.Context(), req.LogContent, authCtx.OrgID)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }
    
    h.respondJSON(w, result, http.StatusOK)
}

// API 3: GET /v1/audit/log/{log_hash}
func (h *Handler) AuditLog(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        h.respondError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }
    
    authCtx := auth.GetAuthContext(r.Context())
    if authCtx == nil || authCtx.AuthMethod != "mtls" {
        h.respondError(w, "Forbidden: mTLS authentication required", http.StatusForbidden)
        return
    }
    
    // 提取 log_hash
    logHash := extractLogHash(r.URL.Path)
    if logHash == "" {
        h.respondError(w, "Invalid log_hash", http.StatusBadRequest)
        return
    }
    
    // 调用 Service 层（无权限限制）
    result, err := h.svc.AuditLogByHash(r.Context(), logHash)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }
    
    h.respondJSON(w, result, http.StatusOK)
}

// 辅助函数
func extractRequestID(path string) string {
    // 从 /v1/status/{request_id} 提取
    parts := strings.Split(strings.TrimPrefix(path, "/v1/status/"), "/")
    if len(parts) > 0 {
        return parts[0]
    }
    return ""
}

func extractLogHash(path string) string {
    // 从 /v1/audit/log/{log_hash} 提取
    parts := strings.Split(strings.TrimPrefix(path, "/v1/audit/log/"), "/")
    if len(parts) > 0 {
        return parts[0]
    }
    return ""
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, core.ErrLogNotFound):
        h.respondError(w, "Log not found", http.StatusNotFound)
    case errors.Is(err, core.ErrPermissionDenied):
        h.respondError(w, "Permission denied", http.StatusForbidden)
    default:
        h.logger.Printf("Service error: %v", err)
        h.respondError(w, "Internal server error", http.StatusInternalServerError)
    }
}

func (h *Handler) respondJSON(w http.ResponseWriter, data interface{}, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, message string, status int) {
    h.respondJSON(w, map[string]string{"error": message}, status)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
    h.respondJSON(w, map[string]string{"status": "healthy"}, http.StatusOK)
}
```

---

### 6. 配置管理 (`config/query.go` & `config/query.defaults.yml`)

**配置结构** (`config/query.go`):
```go
package config

import (
    "fmt"
    "os"
    "time"
    "gopkg.in/yaml.v3"
)

type QueryConfig struct {
    Server     QueryServerConfig     `yaml:"server"`
    Database   DatabaseConfig        `yaml:"database"`
    Blockchain QueryBlockchainConfig `yaml:"blockchain"`
    Logging    LoggingConfig         `yaml:"logging"`
}

type QueryServerConfig struct {
    HTTPPort     int           `yaml:"http_port"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type QueryBlockchainConfig struct {
    Enabled           bool   `yaml:"enabled"`
    ChainMakerConfig  string `yaml:"chainmaker_config"`
}

type LoggingConfig struct {
    Level        string `yaml:"level"`
    Format       string `yaml:"format"`
    AuditEnabled bool   `yaml:"audit_enabled"`
    AuditFile    string `yaml:"audit_file"`
}

func LoadQueryConfig(configPath string) (*QueryConfig, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var cfg QueryConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    return &cfg, nil
}
```

**默认配置** (`config/query.defaults.yml`):
```yaml
server:
  http_port: 8082
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

database:
  host: postgres
  port: 5432
  user: logchain
  password: changeme
  dbname: logchain
  sslmode: disable
  max_conns: 30
  min_conns: 5
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m

blockchain:
  enabled: true
  chainmaker_config: /app/config/clients/chainmaker.yml

logging:
  level: info
  format: json
  audit_enabled: true
  audit_file: /var/log/query/audit.log
```

---

### 7. 主程序入口 (`query/cmd/server/main.go`)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "tlng/blockchain/client/factory"
    "tlng/config"
    "tlng/query/auth"
    "tlng/query/service/core"
    queryhttp "tlng/query/service/http"
    "tlng/storage/store"
)

func main() {
    logger := log.New(os.Stdout, "[QUERY] ", log.LstdFlags|log.Lshortfile)
    
    // 1. 加载配置
    configPath := os.Getenv("QUERY_CONFIG_PATH")
    if configPath == "" {
        configPath = "/app/config/query.defaults.yml"
    }
    
    cfg, err := config.LoadQueryConfig(configPath)
    if err != nil {
        logger.Fatalf("Failed to load config: %v", err)
    }
    
    logger.Printf("Query service starting with config: %s", configPath)
    
    // 2. 初始化数据库
    ctx := context.Background()
    dbDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
        cfg.Database.User,
        cfg.Database.Password,
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.DBName,
        cfg.Database.SSLMode,
    )
    
    storeDB, err := store.NewPostgresStore(
        ctx,
        dbDSN,
        cfg.Database.MaxConns,
        cfg.Database.MinConns,
        logger,
    )
    if err != nil {
        logger.Fatalf("Failed to initialize database: %v", err)
    }
    defer storeDB.Close()
    
    // 3. 初始化 Blockchain Client
    var blockchainClient blockchain.BlockchainClient
    if cfg.Blockchain.Enabled {
        bcCfg, err := config.LoadBlockchainConfig(cfg.Blockchain.ChainMakerConfig)
        if err != nil {
            logger.Fatalf("Failed to load blockchain config: %v", err)
        }
        
        blockchainClient, err = factory.NewClient(bcCfg, logger)
        if err != nil {
            logger.Fatalf("Failed to initialize blockchain client: %v", err)
        }
        defer blockchainClient.Close()
    }
    
    // 4. 初始化 Service 层
    svc := core.NewService(storeDB, blockchainClient, logger)
    
    // 5. 初始化 HTTP Handler
    handler := queryhttp.NewHandler(svc, logger)
    
    // 6. 设置路由
    mux := http.NewServeMux()
    handler.SetupRoutes(mux)
    
    // 7. 创建 HTTP Server
    addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
    server := &http.Server{
        Addr:         addr,
        Handler:      mux,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        IdleTimeout:  cfg.Server.IdleTimeout,
    }
    
    // 8. 启动服务器
    go func() {
        logger.Printf("Query service listening on %s", addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatalf("Server failed: %v", err)
        }
    }()
    
    // 9. 优雅关闭
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    logger.Println("Shutting down server...")
    
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    if err := server.Shutdown(shutdownCtx); err != nil {
        logger.Fatalf("Server forced to shutdown: %v", err)
    }
    
    logger.Println("Query service stopped")
}
```

---

## 📝 实施步骤（6个阶段）

### Phase 1: Nginx 配置补充 ⚠️

**目标**: 为 mTLS 路径添加 Header 传递

**文件**: `ingress/nginx/lua/mtls-ip-auth.lua`

**需要添加** (在认证成功后):
```lua
-- 在认证成功的部分添加 Headers
ngx.req.set_header("X-Cert-Subject", client_cert_dn or "-")
ngx.req.set_header("X-Member-ID", member_id or "-")
ngx.req.set_header("X-Auth-Method", "mtls")
```

**验证**: 
- 通过 mTLS 请求，检查后端是否收到正确的 Headers

---

### Phase 2: 数据层扩展

**任务清单**:
- [ ] 扩展 `storage/store/store.go` 接口
  - 添加 `GetLogStatusByRequestID` 方法
  - 添加 `GetLogStatusByHash` 方法
- [ ] 实现 `storage/store/postgres.go` 查询方法
- [ ] 添加数据库索引 Migration

**Migration 文件** (`storage/migrations/003_add_query_indexes.sql`):
```sql
-- Add indexes for query performance
CREATE INDEX IF NOT EXISTS idx_log_status_request_id ON tbl_log_status(request_id);
CREATE INDEX IF NOT EXISTS idx_log_status_log_hash ON tbl_log_status(log_hash);

-- Comments
COMMENT ON INDEX idx_log_status_request_id IS 'Index for query by request_id (API 1)';
COMMENT ON INDEX idx_log_status_log_hash IS 'Index for query by log_hash (API 2)';
```

**运行 Migration**:
```bash
psql -U logchain -d logchain -f storage/migrations/003_add_query_indexes.sql
```

**验证**:
```bash
# 检查索引是否创建成功
psql -U logchain -d logchain -c "\d tbl_log_status"
```

---

### Phase 3: Query 服务基础架构

**任务清单**:
- [ ] 创建 `config/query.go` 配置结构
- [ ] 创建 `config/query.defaults.yml` 默认配置
- [ ] 创建 `query/auth/middleware.go` 认证中间件
- [ ] 创建 `query/cmd/server/main.go` 服务启动框架

**依赖包**:
```bash
# 无需新增依赖，复用现有的包
```

---

### Phase 4: 核心业务逻辑

**任务清单**:
- [ ] 实现 `query/service/core/service.go`
  - `GetStatusByRequestID` 方法
  - `QueryByContent` 方法
  - `AuditLogByHash` 方法
  - 辅助函数（hash 计算、数据转换等）
- [ ] 定义错误类型和响应结构

**测试**:
```bash
go test ./query/service/core/... -v
```

---

### Phase 5: HTTP Handlers

**任务清单**:
- [ ] 实现 `query/service/http/handler.go`
  - `GetStatus` handler
  - `QueryByContent` handler
  - `AuditLog` handler
  - 错误处理和响应格式化
- [ ] 路由设置
- [ ] 健康检查接口

**测试**:
```bash
go test ./query/service/http/... -v
```

---

### Phase 6: 集成与部署

**任务清单**:
- [ ] 创建 `query/Dockerfile`
- [ ] 更新 `docker-compose.yml` 添加 query 服务
- [ ] 更新 Nginx 配置添加路由规则
- [ ] 编写集成测试
- [ ] 更新 `query/README.md` 文档

**Dockerfile** (`query/Dockerfile`):
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o query-server ./query/cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/query-server .
COPY --from=builder /build/config ./config

EXPOSE 8082

CMD ["./query-server"]
```

**docker-compose.yml 更新**:
```yaml
services:
  query:
    build:
      context: .
      dockerfile: query/Dockerfile
    container_name: logchain-query
    ports:
      - "8082:8082"
    environment:
      - QUERY_CONFIG_PATH=/app/config/query.defaults.yml
    volumes:
      - ./config:/app/config:ro
      - ./logs/query:/var/log/query
    depends_on:
      - postgres
      - ingestion
    networks:
      - logchain-network
```

**Nginx 路由更新** (`ingress/nginx/nginx.conf`):
```nginx
# Query Service - API Key routes
location ~ ^/v1/(status|query_by_content) {
    access_by_lua_file /etc/nginx/lua/api-key-auth.lua;
    
    proxy_pass http://query:8082;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}

# Query Service - mTLS audit route
location ~ ^/v1/audit/ {
    access_by_lua_file /etc/nginx/lua/mtls-ip-auth.lua;
    
    proxy_pass http://query:8082;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

---

## 🔑 关键实现要点

### 1. 认证信任链

```
External Request 
  → Nginx (验证 API Key/mTLS) 
  → 设置 Headers (X-API-Client-ID, X-Client-Org-ID, X-Auth-Method)
  → Query Service (信任 Headers，不重新验证)
  → 权限检查（基于 org_id）
```

**重要**: Query Service 完全信任来自 Nginx 的请求，只做权限控制，不做认证。

---

### 2. 错误处理

**定义标准错误**:
```go
var (
    ErrLogNotFound      = errors.New("log not found")
    ErrPermissionDenied = errors.New("permission denied")
    ErrInvalidRequest   = errors.New("invalid request")
    ErrBlockchainError  = errors.New("blockchain query failed")
)
```

**HTTP 状态码映射**:
- `200 OK`: 查询成功
- `400 Bad Request`: 请求参数错误 (ErrInvalidRequest)
- `403 Forbidden`: 权限不足 (ErrPermissionDenied)
- `404 Not Found`: 日志不存在 (ErrLogNotFound)
- `500 Internal Server Error`: 服务器错误
- `503 Service Unavailable`: Blockchain 不可用

---

### 3. 审计日志格式

**日志字段**:
```
timestamp|client_ip|client_id|org_id|method|path|status_code|response_time|error
```

**示例**:
```
2024-01-01T12:00:00Z|192.168.1.10|client-001|org-abc|GET|/v1/status/uuid-123|200|50ms|-
2024-01-01T12:01:00Z|192.168.1.20|member-001|-|GET|/v1/audit/log/hash123|200|120ms|-
2024-01-01T12:02:00Z|192.168.1.10|client-002|org-xyz|POST|/v1/query_by_content|404|30ms|log not found
```

**写入位置**: `/var/log/query/audit.log`

---

### 4. 性能优化

**数据库连接池**:
- 复用 `storage/store/postgres.go` 的连接池机制
- 最大连接数: 30
- 最小连接数: 5
- 连接最大生命周期: 1h
- 空闲连接超时: 30m

**查询超时**:
- HTTP 请求超时: 30s
- 数据库查询超时: 10s
- Blockchain 查询超时: 15s

**缓存策略** (Phase 1 不实现):
- 未来可考虑添加 Redis 缓存热点查询
- 缓存 TTL: 5-10 分钟

---

## 🧪 测试策略

### 单元测试

**测试文件**:
- `query/auth/middleware_test.go`: Header 提取逻辑测试
- `query/service/core/service_test.go`: 业务逻辑测试（Mock Store 和 Blockchain）

**Mock 示例**:
```go
type MockStore struct {
    mock.Mock
}

func (m *MockStore) GetLogStatusByRequestID(ctx context.Context, requestID string) (*store.LogStatus, error) {
    args := m.Called(ctx, requestID)
    return args.Get(0).(*store.LogStatus), args.Error(1)
}

// 测试用例
func TestGetStatusByRequestID_Success(t *testing.T) {
    mockStore := new(MockStore)
    mockBC := new(MockBlockchain)
    
    expectedStatus := &store.LogStatus{
        RequestID:   "req-123",
        SourceOrgID: "org-abc",
        Status:      store.StatusCompleted,
    }
    
    mockStore.On("GetLogStatusByRequestID", mock.Anything, "req-123").Return(expectedStatus, nil)
    
    svc := core.NewService(mockStore, mockBC, logger)
    result, err := svc.GetStatusByRequestID(context.Background(), "req-123", "org-abc")
    
    assert.NoError(t, err)
    assert.Equal(t, "req-123", result.RequestID)
    mockStore.AssertExpectations(t)
}
```

---

### 集成测试

**测试环境准备**:
```bash
# 启动测试数据库
docker-compose -f docker-compose.test.yml up -d postgres

# 运行 migrations
psql -U logchain_test -d logchain_test -f storage/migrations/*.sql

# 插入测试数据
psql -U logchain_test -d logchain_test -f scripts/db/test-data.sql
```

**测试场景**:
1. API Key 用户查询自己的日志 - 成功
2. API Key 用户查询其他组织的日志 - 403 Forbidden
3. 查询不存在的日志 - 404 Not Found
4. mTLS 用户审计任意日志 - 成功
5. 内容反向查询 - 成功

---

### 端到端测试

**通过 Nginx 测试**:
```bash
# API 1: 状态查询
curl -X GET http://localhost/v1/status/req-123 \
  -H "X-API-Key: example-api-key-12345"

# API 2: 内容查询
curl -X POST http://localhost/v1/query_by_content \
  -H "X-API-Key: example-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "test log content"}'

# API 3: 链上审计（需要 mTLS 证书）
curl -X GET https://localhost/v1/audit/log/hash123 \
  --cert client.crt \
  --key client.key \
  --cacert ca.crt
```

---

## 📦 依赖关系

| 组件 | 状态 | 备注 |
|------|------|------|
| State DB Store 接口 | ✅ 已存在 | 需扩展 2 个查询方法 |
| LogStatus 数据模型 | ✅ 已存在 | 无需修改 |
| Blockchain Client | ✅ 已存在 | 复用 `FindLogByHash` 方法 |
| Nginx API Key 认证 | ✅ 已实现 | Headers 已设置 |
| Nginx mTLS 认证 | ⚠️ 需补充 | 需添加 Header 设置 |
| DB 索引 | ❌ 需添加 | Migration: idx_request_id, idx_log_hash |

---

## 📊 预估工作量

| 阶段 | 预估时间 | 复杂度 |
|------|---------|--------|
| Phase 1: Nginx 补充 | 0.5h | 简单 |
| Phase 2: 数据层扩展 | 1h | 简单 |
| Phase 3: 服务框架 | 2h | 中等 |
| Phase 4: 业务逻辑 | 2h | 中等 |
| Phase 5: HTTP Handlers | 1.5h | 中等 |
| Phase 6: 集成部署 | 1.5h | 中等 |
| **总计** | **~8.5h** | - |

---

## 📋 待办事项 Checklist

### Phase 1: Nginx 配置
- [ ] 修改 `ingress/nginx/lua/mtls-ip-auth.lua` 添加 Headers
- [ ] 测试 mTLS 认证和 Header 传递

### Phase 2: 数据层
- [ ] 扩展 `storage/store/store.go` 接口
- [ ] 实现 `storage/store/postgres.go` 查询方法
- [ ] 创建并运行 Migration 添加索引

### Phase 3: 服务框架
- [ ] 创建 `config/query.go`
- [ ] 创建 `config/query.defaults.yml`
- [ ] 实现 `query/auth/middleware.go`
- [ ] 创建 `query/cmd/server/main.go`

### Phase 4: 业务逻辑
- [ ] 实现 `query/service/core/service.go`
- [ ] 编写单元测试

### Phase 5: HTTP 层
- [ ] 实现 `query/service/http/handler.go`
- [ ] 设置路由
- [ ] 编写 Handler 测试

### Phase 6: 部署集成
- [ ] 创建 `query/Dockerfile`
- [ ] 更新 `docker-compose.yml`
- [ ] 更新 Nginx 路由配置
- [ ] 端到端测试
- [ ] 更新文档

---

## 🔄 后续优化计划（Phase 2+）

### 性能优化
- [ ] 添加 Redis 缓存层
- [ ] 实现查询结果缓存（TTL 5-10分钟）
- [ ] 添加 Prometheus 监控指标

### 功能增强
- [ ] 支持批量查询接口
- [ ] 添加分页支持（查询历史记录）
- [ ] 支持高级过滤（按时间范围、状态等）

### 安全增强
- [ ] 实现 Rate Limiting（基于用户）
- [ ] 添加查询审计追踪
- [ ] 敏感数据脱敏（审计日志）

### 合约增强（可选）
- [ ] 修改智能合约支持 tx_hash 索引
- [ ] 实现 `find_logs_by_tx` 方法
- [ ] 添加 `GET /v1/audit/tx/{tx_hash}/logs` 接口

---

## 📚 参考文档

- [设计文档](design.md) - 系统整体架构设计
- [Blockchain Contracts](../blockchain/contracts.md) - 智能合约接口说明
- [Ingestion Service](../ingestion/README.md) - 日志接入服务文档
- [Nginx Configuration](../ingress/README.md) - 网关配置说明

---

**文档版本**: v1.0  
**最后更新**: 2024-12-17  
**状态**: 待实施
