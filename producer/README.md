# Telemetry Engine (Go Producer)

## Overview

I built this high-performance Data Generator in Go to simulate real-world network traffic patterns at a scale of 10,000+ events per second (EPS). Standard load generators often produce perfectly clean, predictable data, but real systems are messy. I designed this engine to generate "Real-World" chaos—injecting latency spikes, dropped packets, suspicious payloads, and malformed headers—creating a robust dataset for downstream anomaly detection.

The engine produces strictly structured JSON payloads (simulating VPC Flow Logs or Firewall Logs) and pushes them to an Apache Kafka ingestion layer.

## Architecture & Design Decisions

I chose Go for the producer because its concurrency model (goroutines) allows me to spawn thousands of concurrent "devices" with minimal memory overhead, easily saturating the network interface on a standard machine.

To achieve 10k+ EPS efficiently, I implemented a few critical optimizations:

- **Worker Pool Pattern:** Instead of spawning a goroutine for every single message, I implemented a fixed pool of workers that run continuously. This keeps CPU usage stable and predictable.
- **Lock-Free Randomization:** Go's global `math/rand` uses a mutex lock, which becomes a massive bottleneck at high concurrency. I bypassed this by giving every worker its own locally seeded random number generator.
- **C-Level Kafka Batching:** I utilized `confluent-kafka-go` (a wrapper around `librdkafka`) to handle asynchronous batching, compression (`snappy`), and network I/O outside of the Go runtime.

## Chaos Engineering Features

To make this more than just a random number generator, I built in specific defect modes that can be toggled via environment variables:

1. **The "Latency Spike" (Performance Degradation):** Simulates a database backup or network congestion. A percentage of packets suddenly jump from a normal 10-50ms latency to 2000ms+.
2. **The "Port Scan" (Security Anomaly):** Simulates an attacker. A dedicated goroutine loop rapidly attempts connections across 100+ sequential ports from a single source IP to a target IP.
3. **The "Data Exfiltration" (Byte Anomaly):** Simulates data theft (e.g., DNS tunneling). Injects occasional connections with massive outbound payloads (10MB+) compared to tiny inbound ACKs, routed over UDP port 53.
4. **The "Connection Reset" (Reliability Failure):** Simulates a service crash by injecting a high volume of TCP RST flags and HTTP 503 status codes.

## Configuration (The "Knobs")

I designed the system to be fully configurable at runtime using environment variables via Viper, requiring no recompilation to tune the throughput or chaos levels.

| Environment Variable | Description                                                     | Default Value                                     |
| :------------------- | :-------------------------------------------------------------- | :------------------------------------------------ |
| `KAFKA_BROKERS`      | Comma-separated list of Kafka broker addresses.                 | `localhost:19092,localhost:29092,localhost:39092` |
| `KAFKA_TOPIC`        | The Kafka topic to produce telemetry to.                        | `network-telemetry`                               |
| `EPS_TARGET`         | Total Events Per Second to generate across all workers.         | `5000`                                            |
| `WORKER_COUNT`       | Number of concurrent Goroutines producing data.                 | `10`                                              |
| `CHAOS_MODE`         | Enable or disable the injection of network anomalies.           | `true`                                            |
| `LATENCY_SPIKE_PROB` | Probability (0.0 to 1.0) of a packet experiencing high latency. | `0.05`                                            |
| `ERROR_RATE_PROB`    | Probability of generating HTTP 5xx/4xx codes or RST flags.      | `0.02`                                            |
| `DATA_EXFIL_PROB`    | Probability of a massive outbound payload event.                | `0.01`                                            |

## How to Run

### Local Execution (CLI)

You will need Go installed and CGO enabled (a C compiler like `build-essential` or Xcode command line tools).

Ensure your Kafka cluster is running, then execute:

```bash
cd producer
go run ./cmd/telemetry/main.go
```

### Running with Custom Knobs:

You can inject the environment variables directly into the execution command to alter behavior on the fly. For example, to run at 15,000 EPS with chaos mode disabled:

```
EPS_TARGET=15000 WORKER_COUNT=20 CHAOS_MODE=false go run ./cmd/telemetry/main.go
```
