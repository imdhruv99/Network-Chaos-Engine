# Feature Store: Bridging Data Engineering and ML

## The Problem (Why I built this)
In enterprise ML, the biggest cause of model failure is **Training-Serving Skew**—when the data used to train the model looks subtly different from the data the model sees in production. Data Scientists usually write complex SQL to generate historical features, while Backend Engineers rewrite that logic in Java/Go to fetch real-time features. This duplication causes discrepancies.

To fix this in my Network Chaos pipeline, I introduced **Feast** (an open-source Feature Store). Feast acts as the single source of truth: it manages the features, ensures Point-in-Time correctness for training, and seamlessly syncs that data to a low-latency cache for real-time model inference.

## The Architecture
* **The Registry:** A local catalog (`data/registry.db`) holding the single-source-of-truth definitions for my features.
* **The Offline Store:** My MinIO Lakehouse (the "Silver Layer" Parquet files). Used for massive historical joins.
* **The Online Store:** A Redis container. Used for sub-10ms real-time inference lookups.
* **The Entity:** `src_ip` (Everything revolves around the IP address).

---

## Step-by-Step Execution & Hard-Learned Lessons

### 1. Environment Setup & The `s3` vs `s3a` Quirk
First, I set up a dedicated Python virtual environment and installed Feast.
Since my Spark Lakehouse uses the Hadoop `s3a://` prefix, but Feast uses PyArrow (which strictly expects `s3://`), I had to update my data source URIs. I also exported my MinIO credentials to the local environment so PyArrow could authenticate:

```bash
export AWS_ACCESS_KEY_ID=<DUMMY_ACCESS_KEY>
export AWS_SECRET_ACCESS_KEY=<DUMMY_SECRET_KEY>
export AWS_ENDPOINT_URL=http://localhost:9000
export AWS_DEFAULT_REGION=us-east-1
```

### 2. Defining the Data Model & Applying
I defined the Entity (`src_ip`) and the Feature View (`daily_ip_traffic`) inside `feature_repo/features.py`.

Engineering Note: Feast has a strict type system. I had to use `ValueType.STRING` for the Entity definition to avoid deprecation warnings from `typeguard`.

Once defined, I compiled the definitions into the registry:
```
cd feature_repo
feast apply
```

### 3. The Data Scientist Workflow (Preventing Data Leakage)
To train an ML model, you need historical data. But if an anomaly occurred at 10:05 AM, you can't let the model "cheat" by seeing features calculated at 10:06 AM.

I wrote `get_training_data.py` to execute a Time-Travel Join. Feast automatically handles the complex SQL to fetch features that were true exactly at the provided event timestamps.

```
python get_training_data.py
```

Result: Feast reached into MinIO, parsed the Parquet files, and returned a clean Pandas DataFrame ready for `scikit-learn` or `XGBoost`.

### 4. Materialization (Moving to Real-Time)
A production API can't wait for a Parquet file scan. It needs features in milliseconds. I used Feast's materialization engine to copy the latest feature states from MinIO into my Redis cache.

Gotcha: I initially got a `ModuleNotFoundError` for `s3fs`. Dask/Pandas need `s3fs` to resolve the object storage connections under the hood. After a quick `pip install s3fs`, I ran:

```
CURRENT_TIME=$(date -u +"%Y-%m-%dT%H:%M:%S")
feast materialize 2026-01-01T00:00:00 $CURRENT_TIME
```

Result: Success. Feast synced all records up to the current second directly into Redis.

### 5. The Backend Workflow (Online Serving)
Finally, I simulated a production backend making an inference request. I wrote `get_online_features.py` to ask Feast for the current state of a specific IP address.

```
python get_online_features.py
```

Output:

```
Connecting to Feast Feature Store...
Fetching sub-millisecond features for 2 IPs...

--- Real-Time Feature Vector for Inference ---
IP: 10.0.0.1
 - Bytes Sent: None
 - Latency MS: None

IP: 192.168.1.100
 - Bytes Sent: 3698
 - Latency MS: 16

```
Result: Instead of spinning up a heavy compute engine, Feast queried Redis and returned the latest `bytes_sent` and `latency_ms` for my target IPs in under 5 milliseconds.
