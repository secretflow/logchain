# Benthos Adapters

This directory contains Benthos configuration files and adapters for heterogeneous data source integration.

## Purpose

According to the design document, these adapters handle protocol conversion for non-HTTP/gRPC data sources:

- **Syslog** - UDP/TCP syslog protocol (UDP 5514 / TCP 6514)
- **Kafka Topics** - Direct Kafka topic consumption
- **S3** - AWS S3 bucket file processing
- **Other protocols** - Any future heterogeneous data sources

## Architecture

These adapters work with the API Gateway to:
1. Receive heterogeneous protocol traffic
2. Parse and standardize data formats
3. Forward processed logs to the Log Ingestion Service

## Implementation Status

✅ **Benthos 适配器配置已提供**

本目录现在包含可直接运行的 Benthos 配置，支持将异构数据源转换为统一的 `log_content` JSON，并转发到日志接入服务（默认 `http://ingestion:8091/v1/logs`）。

## Configuration Files

当前配置文件：
- `syslog.yml` - Syslog 适配器（UDP 5514 / TCP 6514 默认）
- `kafka-consumer.yml` - Kafka Topic 适配器
- `s3-processor.yml` - S3 文件适配器（按行切分）

## 使用方法

### 环境变量（通用）
- `INGESTION_ENDPOINT`: 日志接入服务地址，默认 `http://ingestion:8091/v1/logs`
- `INGESTION_API_KEY`: 可选，若接入服务启用 API Key 则填写
- `DEFAULT_ORG_ID`: 写入 `client_source_org_id` 的默认值
- `HTTP_BATCH_COUNT` / `HTTP_BATCH_PERIOD`: HTTP 批量大小与时间窗口
- `RATE_LIMIT_COUNT` / `RATE_LIMIT_PERIOD` / `RATE_LIMIT_ENABLED`: 流控门限、窗口大小及开关

### Syslog 适配器
```bash
export SYSLOG_UDP_ADDR=0.0.0.0:5514
export SYSLOG_TCP_ADDR=0.0.0.0:6514
export RATE_LIMIT_COUNT=
export RATE_LIMIT_PERIOD=
export RATE_LIMIT_ENABLED=true
export DEFAULT_ORG_ID=dkb
export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
export INGESTION_API_KEY=dkbmtb
export HTTP_BATCH_COUNT= 
export HTTP_BATCH_PERIOD= 

./redpanda-connect lint ingestion/adapters/syslog.yml
./redpanda-connect run ingestion/adapters/syslog.yml
```

### Kafka 适配器
```bash
export KAFKA_BROKERS=localhost:8066
export KAFKA_TOPIC=yykj-topic-ssl
export KAFKA_CONSUMER_GROUP=benthos-adapter-kafka
export KAFKA_CLIENT_ID=kfk
export KAFKA_TLS_ENABLED=true
export KAFKA_CA_FILE=/root/logchain/ingestion/adapters/kafka/ssl/cert/ca-cert
export SKIP_SERVER_CERT_VERIFY=false
export KAFKA_CLIENT_CERT_FILE=/root/logchain/ingestion/adapters/kafka/ssl/cert/client/client.crt
export KAFKA_CLIENT_KEY_FILE=/root/logchain/ingestion/adapters/kafka/ssl/cert/client/client.key
export RATE_LIMIT_COUNT=
export RATE_LIMIT_PERIOD=
export RATE_LIMIT_ENABLED=true
export DEFAULT_ORG_ID=dkb
export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
export INGESTION_API_KEY=dkbmtb
export HTTP_BATCH_COUNT=
export HTTP_BATCH_PERIOD=

./redpanda-connect lint ingestion/adapters/kafka-consumer.yml
./redpanda-connect run ingestion/adapters/kafka-consumer.yml
```

### S3 适配器
```bash
export S3_BUCKET_NAME=
export S3_PREFIX=
export AWS_REGION=
export AWS_ACCESS_KEY_ID=
export AWS_SECRET_ACCESS_KEY=
export AWS_SESSION_TOKEN=
export S3_DELETE_AFTER_READ=
export RATE_LIMIT_COUNT=
export RATE_LIMIT_PERIOD=
export RATE_LIMIT_ENABLED=
export DEFAULT_ORG_ID=
export INGESTION_ENDPOINT=
export INGESTION_API_KEY=
export HTTP_BATCH_COUNT=
export HTTP_BATCH_PERIOD=
  
./redpanda-connect lint ingestion/adapters/s3-processor.yml
./redpanda-connect run ingestion/adapters/s3-processor.yml
```

## 安全建议
### 整体原则
- 在网络基础设施层面（安全组、防火墙、网络 ACL）实施基于源 IP 和端口的访问控制策略，限制仅允许授权的网络流量访问 Benthos 适配器服务。
- 如需增强传输层安全性或实施流量整形，可在私有网络内通过 Nginx/Ingress 等代理组件进行四层端口转发或七层反向代理，实现 TLS 终止和限速功能。
- 充分利用 Benthos Adapter 内置的认证机制（如客户端证书认证、Access Key/Secret Key 认证等）进行身份验证和授权，确保数据传输安全。

### Syslog 适配器（TCP/UDP）

**基础设施层面（需用户配置）：**
- 由于 UDP 协议不支持 TLS，请在 Benthos 所在节点的安全组/防火墙层配置 **IP 白名单 + 端口控制**，仅允许可信源 IP 访问 Syslog 监听端口。

**Benthos Adapter 已实现：**
- ✅ **服务端限流**：开启 `RATE_LIMIT_ENABLED=true` 时，内置的 `throttle` 处理器会按计数周期对处理管道中的超限流量施加背压而非丢弃。

### Kafka 适配器

**基础设施层面（需用户配置）：**
- 安全控制重心在 **Kafka 自身及所在节点的 ACL 等网络管控**，请确保 Kafka broker 所在节点的安全组/防火墙仅允许必要的访问源。

**Benthos Adapter 已实现：**
- ✅ **客户端证书认证**：Benthos 作为客户端连接 Kafka broker 时，支持配置客户端证书，并分发自己的 CA 证书给 Kafka，以此来实现 Kafka 对于客户端（Benthos）的证书认证。
- ✅ **服务端证书验证（可选）**：Benthos 内可选择性地对 Kafka 服务端证书进行验证。
- ✅ **服务端限流**：开启 `RATE_LIMIT_ENABLED=true` 时，内置的 `throttle` 处理器会按计数周期对处理管道中的超限流量施加背压而非丢弃。

### AWS S3 适配器

**基础设施层面（需用户配置）：**
- 安全控制重心在 **S3 服务端的 ACL/IAM 等网络管控**，请确保 S3 bucket 的访问策略和 IAM 权限配置正确，仅允许必要的访问源。

**Benthos Adapter 已实现：**
- ✅ **客户端认证**：Benthos 作为客户端连接 S3 时，支持通过配置 S3 的 Access Key、Secret Key、Session Token 等方式来实现 S3 服务端对于 Benthos 客户端的认证。
- ✅ **服务端限流**：开启 `RATE_LIMIT_ENABLED=true` 时，内置的 `throttle` 处理器会按计数周期对处理管道中的超限流量施加背压而非丢弃。