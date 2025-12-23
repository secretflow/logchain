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

✅ **Benthos adapter configuration provided**

This directory now contains Benthos configurations that can be run directly, supporting the conversion of heterogeneous data sources into a unified 'log_comtent' JSON and forwarding it to the log access service (default)` http://ingestion:8091/v1/logs `).

## Configuration Files

Current configuration file:
- `syslog.yml` - Syslog adapter (UDP 5514/TCP 6514 default)
- `kafka-consumer.yml` - Kafka Topic Adapter
- `s3-processor.yml` - S3 file adapter (split by line)

## Instructions for use

### benthos download

- `The original benthosdev/benthos were acquired by Redpanda in 2024. The original benthos has been split into the basic version Redpanda-data/benthos and the full connector version Redpanda-data/connect. This module adopts the full connector version`
- `download: https://github.com/redpanda-data/connect/releases`

### Environment variables (generic)
- `INGESTION_ENDPOINT`: Log access service address, default `http://ingestion:8091/v1/logs`
- `DEFAULT_ORG_ID`: Write the default value of `content_Source_org_id`
- `HTTP_BATCH_COUNT` / `HTTP_BATCH_PERIOD`: HTTP Batch Size and Time Window
- `RATE_LIMIT_COUNT` / `RATE_LIMIT_PERIOD` / `RATE_LIMIT_ENABLED`: Flow control threshold, window size, and switch

### Syslog adapter
```bash
export SYSLOG_UDP_ADDR=0.0.0.0:5514
export SYSLOG_TCP_ADDR=0.0.0.0:6514
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values (otherwise use the default values in the configuration)
# export RATE_LIMIT_COUNT=1000
# export RATE_LIMIT_PERIOD=1s
# export RATE_LIMIT_ENABLED=true
export DEFAULT_ORG_ID=dkb
export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values (otherwise use the default values in the configuration)
# export HTTP_BATCH_COUNT=500
# export HTTP_BATCH_BYTES=500000
# export HTTP_BATCH_PERIOD=2

./redpanda-connect lint ingestion/adapters/syslog.yml
./redpanda-connect run ingestion/adapters/syslog.yml

# The following method can be used to simulate the client sending data, and Benthos can consume it
echo '<14>1 2025-12-16T08:31:00Z myhost app 1234 - - Test TCP syslog 5424' | nc 127.0.0.1 6514
echo '<14>1 2025-12-16T08:30:00Z myhost app 1234 - - Test UDP syslog 5424' | nc -u -w1 127.0.0.1 5514
```

### Kafka adapter
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
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values (otherwise use the default values in the configuration)
# export RATE_LIMIT_COUNT=1000
# export RATE_LIMIT_PERIOD=1s
# export RATE_LIMIT_ENABLED=true
export DEFAULT_ORG_ID=dkb
export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values (otherwise use the default values in the configuration)
# export HTTP_BATCH_COUNT=500
# export HTTP_BATCH_BYTES=500000
# export HTTP_BATCH_PERIOD=2

./redpanda-connect lint ingestion/adapters/kafka-consumer.yml
./redpanda-connect run ingestion/adapters/kafka-consumer.yml

# The following method can be used to simulate the client sending data
# After completing the installation of Kafka, start the Kafka producer and send message, and Benthos can consume it
bin/kafka-console-consumer.sh --topic yykj-topic --from-beginning --bootstrap-server node1:9092
```

### S3 adapter
```bash
export S3_BUCKET_NAME=my-container-bucket
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values
# export S3_PREFIX=sdb
export AWS_REGION=local
export COMPATIBLE_END_POINT=http://127.0.0.1:9000
export FORCE_PATH_STYLE_URLS=true
export AWS_ACCESS_KEY_ID=minio
export AWS_SECRET_ACCESS_KEY=redpandaTieredStorage7
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values
# export AWS_SESSION_TOKEN=
# export S3_DELETE_AFTER_READ=false
# export RATE_LIMIT_COUNT=1000
# export RATE_LIMIT_PERIOD=1
# export RATE_LIMIT_ENABLED=true
export DEFAULT_ORG_ID=dkbmtb
export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
# Optional: To enable custom flow control thresholds, please uncomment and set appropriate values (otherwise use the default values in the configuration)
# export HTTP_BATCH_COUNT=500
# export HTTP_BATCH_BYTES=500000
# export HTTP_BATCH_PERIOD=2
  
./redpanda-connect lint ingestion/adapters/s3-processor.yml
./redpanda-connect run ingestion/adapters/s3-processor.yml

# The following method can be used to simulate the client sending data
# For example, after completing the installation of Minio, use MC to put the message file into the bucket, and Benthos can consume it
docker exec -it minio sh
echo "just test message" > /tmp/test.txt
mc cp /tmp/test.txt myminio/my-container-bucket
```

## Safety advice
### Overall principle
- Implement access control policies based on source IP and ports at the network infrastructure level (security groups, firewalls, network ACLs), limiting access to Benthos adapter services to only authorized network traffic.
- To enhance transport layer security or implement traffic shaping, layer 4 port forwarding or layer 7 reverse proxy can be performed through proxy components such as Nginx/Ingress within a private network to achieve TLS termination and speed limiting functions.
- Fully utilize the built-in authentication mechanisms of Benthos Adapter (such as client certificate authentication, Access Key/Screen Key authentication, etc.) for identity verification and authorization, ensuring secure data transmission.

### Syslog adapter（TCP/UDP）

**At the infrastructure level (requiring user configuration):**
- Due to UDP protocol not supporting TLS, please configure **IP whitelist+port control** in the security group/firewall layer of Benthos' node, allowing only trusted IP addresses to access Syslog listening ports.

**Benthos Adapter has been implemented:**
- ✅ **Server side flow restriction**：When `RATE_LIMIT-INABLED=true` is enabled, the built-in `rate_imit` processor will apply back pressure to the excess flow in the processing pipeline according to the counting cycle instead of discarding it.

### Kafka adapter

**At the infrastructure level (requiring user configuration):**
- The focus of security control is on network management such as the `ACL of Kafka itself and its nodes`. Please ensure that the security group/firewall of the node where the Kafka broker is located only allows necessary access sources.

**Benthos Adapter has been implemented:**
- ✅ **Client certificate authentication**：Benthos 作为客户端连接 Kafka broker 时，支持配置客户端证书，并分发自己的 CA 证书给 Kafka，以此来实现 Kafka 对于客户端（Benthos）的证书认证。
- ✅ **Server certificate verification (optional)**：Benthos 内可选择性地对 Kafka 服务端证书进行验证。
- ✅ **Server side flow restriction**：When `RATE_LIMIT-INABLED=true` is enabled, the built-in `rate_imit` processor will apply back pressure to the excess flow in the processing pipeline according to the counting cycle instead of discarding it.

### AWS S3 适配器

**At the infrastructure level (requiring user configuration):**
- The focus of security control is on network management such as `ACL/IAM on the S3 server`. Please ensure that the access policy and IAM permission configuration of the S3 bucket are correct, and only allow necessary access sources.

**Benthos Adapter has been implemented:**
- ✅ **Client certificate authenticatio**：When Benthos connects to S3 as a client, it supports authentication of the Benthos client by the S3 server through configuring S3's Access Key, Secret Key, Session Token, and other methods.
- ✅ **Server side flow restriction**：When `RATE_LIMIT-INABLED=true` is enabled, the built-in `rate_imit` processor will apply back pressure to the excess flow in the processing pipeline according to the counting cycle instead of discarding it.