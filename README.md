# Net Chaos Engine: High-Throughput Telemetry Pipeline

## Executive Summary

I built this end-to-end data pipeline to solve a specific problem in machine learning and system observability: standard load generators produce perfectly clean, predictable data. But in the real world, systems are messy.

This project is a high-performance **Network Telemetry Generator and Ingestion Pipeline** designed to simulate real-world VPC Flow Logs and Firewall Logs at a scale of 10,000+ events per second (EPS). More importantly, it acts as a "Chaos Engine"—purposefully injecting realistic network anomalies (latency spikes, dropped packets, data exfiltration) into the data stream.

This serves as the foundational data generation layer for downstream Anomaly Detection and Machine Learning models.

---

## High-Level Architecture

The system is designed using a decoupled, event-driven microservice architecture to maximize throughput and scalability.

![Architecture](images/arch.png)

### 1. The Telemetry Engine (Go Producer)

A highly concurrent Go application that utilizes a worker pool pattern and lock-free randomization to generate synthetic JSON network logs. It saturates the network by batching messages asynchronously to Kafka using C-level bindings (`librdkafka`).

### 2. The Ingestion Layer (Apache Kafka KRaft)

A 3-node Apache Kafka cluster running in KRaft mode (Zookeeper-less). It acts as the shock absorber, decoupling the blazing-fast Go producer from the slower storage layer. It's configured with a replication factor of 3 for high availability.

### 3. The Data Sink (Python Consumer)

A resilient Python microservice that pulls messages from Kafka, strictly validates the schemas using `Pydantic`, and buffers the data. To prevent storage node crashes, it implements dual-trigger batching (by volume or time) and compresses the batches into `json.gz` files before writing to object storage.

### 4. Storage & Observability (MinIO, Prometheus, Grafana)

- **MinIO:** Acts as a local, S3-compatible object storage sink for the compressed batch files.
- **Prometheus & Grafana:** Monitors the entire pipeline in real-time, scraping metrics via `kafka-exporter` to visualize pipeline throughput (EPS), consumer lag, and cluster health without adding overhead to the brokers.

---

## Directory Structure

```text
.
├── consumer/                 # Python Data Sink Microservice
│   ├── README.md
│   ├── requirements.txt
│   └── src/                  # Pydantic models, batching logic, and MinIO client
├── docker-compose.yml        # Orchestrates Kafka, MinIO, Prometheus, and Grafana
├── grafana_dashboard/
│   └── kafka_dashboard.json  # Pre-built dashboard for EPS and Lag monitoring
├── images/
│   └── kafka_dashboard.png
├── monitoring/
│   └── prometheus.yml        # Scrape configs for kafka-exporter
└── producer/                 # Go Telemetry Engine Microservice
    ├── cmd/telemetry/        # Application entrypoint
    ├── go.mod
    ├── go.sum
    ├── internal/             # Config, Kafka producer wrapper, and Chaos generator
    └── README.md
```

## The Chaos Modes (Defect Injection)

I designed the generator to simulate specific infrastructure and security failures. These can be toggled dynamically via environment variables:

The "Latency Spike": Simulates performance degradation (e.g., a database backup) by randomly spiking packet latency from a standard 10-50ms up to 2000ms+.

The "Port Scan": Simulates a security anomaly by dedicating a worker thread to rapidly attempt connections across 100+ sequential ports from a single source IP.

The "Data Exfiltration": Simulates data theft by generating massive outbound byte payloads (10MB+) compared to tiny inbound ACKs, routed over UDP port 53 (DNS Tunneling).

The "Connection Reset": Simulates reliability failures by injecting high volumes of TCP RST flags and HTTP 5xx errors.

---

### Quick Start Guide

**1. Boot the Infrastructure:**

Start the Kafka KRaft cluster, MinIO storage, and the Prometheus/Grafana observability stack:

```bash
docker-compose up -d
```

- Grafana: `http://localhost:3000` -> Import the dashboard from grafana_dashboard/. After running producer and consumer it will look like below

  ![Grafana Dashboard](images/kafka_dashboard.png)

- MinIO UI: `http://localhost:9001`

**2. Start the Data Sink (Consumer)**

Open a new terminal, activate a Python environment, and start the sink so it's ready to catch data:

```
cd consumer
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python -m src.main
```

**3. Fire the Chaos Cannon (Producer)**

Open another terminal and start the Go generator. You can tune the throughput and chaos probabilities dynamically.

```
cd producer
# Run with defaults (~5000 EPS)
go run ./cmd/telemetry/main.go

# Or stress test it at 15,000 EPS with specific anomaly probabilities
EPS_TARGET=15000 WORKER_COUNT=20 DATA_EXFIL_PROB=0.05 go run ./cmd/telemetry/main.go
```
