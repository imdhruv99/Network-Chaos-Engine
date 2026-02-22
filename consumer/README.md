# Telemetry Data Sink (Python Consumer)

## Overview

I built this Python-based microservice to act as the resilient data sink for the High-Throughput Network Telemetry Generator. It is designed to consume 10,000+ events per second (EPS) of raw, chaotic network logs from Apache Kafka, validate the payload structure in real-time, and efficiently batch the data into an S3-compatible object store (MinIO).

Writing a firehose of individual logs directly to object storage is a well-known anti-pattern that leads to throttled networks and crashed storage nodes. I engineered this sink specifically to handle extreme throughput via intelligent, dual-trigger batching and in-memory compression.

## Architecture & Design Decisions

To ensure the consumer could keep pace with the Go producer without buckling under the load, I focused on strict typing, C-level bindings, and efficient I/O operations:

- **High-Performance Ingestion:** I utilized `confluent-kafka`, which relies on the highly optimized `librdkafka` C library, bypassing Python's native performance limitations for network I/O.
- **Dual-Trigger Batching:** The system buffers incoming messages in memory and flushes them to MinIO based on two conditions:
  1. **Volume threshold:** Reaching a specific `BATCH_SIZE` (e.g., 2000 records).
  2. **Time threshold:** Reaching a `FLUSH_INTERVAL_SEC` (e.g., 5 seconds) to prevent stale data from sitting in memory during low-traffic periods.
- **Strict Schema Validation:** Because the producer deliberately injects "chaos" (malformed data, dropped packets), I implemented `Pydantic` models. Every single JSON payload is strictly validated at runtime before it enters the batch.
- **Storage Optimization:** Before writing to MinIO, the JSON batches are heavily compressed in-memory into `json.gz` files, drastically reducing network payload size and long-term storage costs.
- **Guaranteed Delivery:** Auto-commit is explicitly disabled. The consumer only commits its Kafka offset _after_ a successful HTTP 200 OK response from the MinIO upload, guaranteeing zero data loss if the Python process crashes mid-batch.

## Configuration (The "Knobs")

The microservice is configured entirely via environment variables (managed by `pydantic-settings`), making it instantly deployable to Docker or Kubernetes without code changes.

| Environment Variable | Description                                          | Default Value                                     |
| :------------------- | :--------------------------------------------------- | :------------------------------------------------ |
| `KAFKA_BROKERS`      | Comma-separated list of Kafka broker addresses.      | `localhost:19092,localhost:29092,localhost:39092` |
| `KAFKA_TOPIC`        | The Kafka topic to consume telemetry from.           | `network-telemetry`                               |
| `KAFKA_GROUP_ID`     | The consumer group ID for offset tracking.           | `telemetry-sink-group`                            |
| `MINIO_ENDPOINT`     | The address of the MinIO/S3 cluster.                 | `localhost:9000`                                  |
| `MINIO_BUCKET`       | The target bucket for the batched files.             | `telemetry-data`                                  |
| `BATCH_SIZE`         | Number of records to hold in memory before flushing. | `2000`                                            |
| `FLUSH_INTERVAL_SEC` | Maximum seconds to wait before forcing a flush.      | `5`                                               |

## How to Run

### Local Execution (CLI)

You will need Python 3.9+ installed. It is highly recommended to run this inside a virtual environment.

**1. Setup the environment:**

```bash
cd consumer
python3 -m venv venv
source venv/bin/activate  # On Windows use: venv\Scripts\activate
pip install -r requirements.txt
```

**2. Ensure Infrastructure is Running:**
Make sure your Kafka brokers and MinIO instances are active and reachable.

**3. Run the Data Sink:**

```bash
python -m src.main
```

**Running with Custom Knobs:**
You can tune the batching aggressiveness by overriding the environment variables inline:

```
BATCH_SIZE=5000 FLUSH_INTERVAL_SEC=10 python -m src.main
```
