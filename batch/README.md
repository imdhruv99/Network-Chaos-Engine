# Lakehouse & Batch Processing (The Medallion Architecture)

## The Problem
When I first started scaling the Network Chaos Engine, I quickly ran into a classic Data Engineering wall: **The Small File Problem**.

My Python sink was diligently dumping thousands of `.json.gz` network telemetry logs into MinIO every minute. By the time I hit 32 million records, running a standard `spark.read.json()` on the bucket was causing massive JVM Garbage Collection pauses and eventual Out-Of-Memory (OOM) errors. Furthermore, relying on Spark's schema inference meant a full, incredibly expensive table scan before any processing even began.

Storing raw JSON is great for quick app appends, but it's terrible for analytical queries. To fix this, I built this batch processing layer using Apache Spark to implement a strict **Medallion Architecture**.

## The Architecture
This pipeline moves data through four stages, optimizing it for storage costs and query performance at each step:

1. **Raw (JSON):** The landing zone. Highly chaotic, highly fragmented.
2. **Bronze (Delta Lake):** To prevent OOMs, I used Spark Structured Streaming with `Trigger.AvailableNow`. This incrementally reads the massive JSON backlog in chunks of 500 files, securely writes them to a Delta table, tracks the offset, and shuts down.
3. **Silver (Optimized Parquet):** I take the Bronze Delta table, cleanse the nulls, and write it out as highly compressed Parquet files. I strictly `partitionBy("year", "month", "day")`. This enables **Partition Pruning**, allowing downstream analytical queries to scan petabytes of data in seconds by completely ignoring irrelevant folders.
4. **Gold (Aggregated Delta):** The final business-level aggregates (e.g., total bytes transferred per IP per day). I write this back to Delta to enable ACID transactions and `MERGE` capabilities for any downstream BI tools.

## What's in this folder?

I've included both my exploratory sandbox and the production-ready automated scripts:

* `lakehouse_pipeline.ipynb`: The original Jupyter Notebook I used to develop and test the pipeline interactively. It includes the performance benchmarks proving the 20x speedup of Parquet vs. JSON.
* `medallion_pipeline.py`: The headless, heavily-tuned PySpark script. It includes strict schema definitions and explicit JVM memory tuning (off-heap memory enabled) to chew through the 32M+ backlog without crashing.
* `run_spark_job.sh`: A shell wrapper that uses `docker exec` to submit the Python script directly into the running Spark container.

## How to Run It

### 1. Interactive Mode (JupyterLab)
If you want to step through the code and see the Spark UI:
1. Ensure your infrastructure is up (`docker-compose up -d`).
2. Grab your Jupyter token: `docker logs spark-jupyter | grep "token="`
3. Open the UI, navigate to `work/batch/lakehouse_pipeline.ipynb`, and run the cells.

### 2. Automated / Production Mode (Cron)
For headless execution (simulating a nightly batch job):

1. Make the shell script executable:
   ```bash
   chmod +x batch/run_spark_job.sh

2. You can run it manually to watch the pipeline execute in your terminal:

    ```
    ./batch/run_spark_job.sh
    ```

3. Schedule it: To run this pipeline every night at 2:00 AM, add it to your crontab `(crontab -e)`:

    ```
    0 2 * * * /absolute/path/to/Network-Chaos-Engine/batch/run_spark_job.sh >> /absolute/path/to/Network-Chaos-Engine/logs/spark_cron.log 2>&1
    ```

- (Note: I chose a simple cron schedule to orchestrate this for now to keep the infrastructure lightweight, but in a larger enterprise environment, this script is ready to be dropped directly into an Apache Airflow DAG).

### The Performance Win
By structuring the data this way, I entirely eliminated OOM errors during ingestion. More importantly, when running an anomaly detection SQL query ("Top 10 talkative IPs"), the execution time dropped from ~45 seconds (full table scan on raw JSON) down to ~2 seconds (partition pruning on Parquet).
