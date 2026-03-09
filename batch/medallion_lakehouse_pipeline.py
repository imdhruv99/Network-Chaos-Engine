import sys
import time
from pyspark.sql import SparkSession
from pyspark.sql.types import StructType, StructField, StringType, LongType, IntegerType
from pyspark.sql.functions import col, year, month, dayofmonth, to_timestamp, sum, count

def run_pipeline():
    print("Initializing Spark Session...")
    spark = SparkSession.builder \
        .appName("NetChaos-Medallion-Pipeline") \
        .config("spark.sql.extensions", "io.delta.sql.DeltaSparkSessionExtension") \
        .config("spark.sql.catalog.spark_catalog", "org.apache.spark.sql.delta.catalog.DeltaCatalog") \
        .config("spark.hadoop.fs.s3a.endpoint", "http://minio:9000") \
        .config("spark.hadoop.fs.s3a.access.key", "admin") \
        .config("spark.hadoop.fs.s3a.secret.key", "password@123") \
        .config("spark.hadoop.fs.s3a.path.style.access", "true") \
        .config("spark.hadoop.fs.s3a.impl", "org.apache.hadoop.fs.s3a.S3AFileSystem") \
        .config("spark.hadoop.fs.s3a.connection.maximum", "100") \
        .config("spark.driver.memory", "4g") \
        .config("spark.memory.fraction", "0.6") \
        .config("spark.memory.offHeap.enabled", "true") \
        .config("spark.memory.offHeap.size", "1g") \
        .config("spark.sql.shuffle.partitions", "10") \
        .getOrCreate()

    spark.sparkContext.setLogLevel("WARN")

    # Strict Schema to prevent inference OOMs, matches exactly to go producer's json output
    telemetry_schema = StructType([
        StructField("timestamp", StringType(), True),
        StructField("event_id", StringType(), True),
        StructField("src_ip", StringType(), True),
        StructField("dst_ip", StringType(), True),
        StructField("bytes_sent", LongType(), True),
        StructField("latency_ms", IntegerType(), True)
    ])

    # Paths
    raw_path = "s3a://telemetry-data/ingest/" # python consumer data-sink layer path
    bronze_path = "s3a://telemetry-data/bronze/telemetry_delta"
    silver_path = "s3a://telemetry-data/silver/telemetry_parquet"
    gold_path = "s3a://telemetry-data/gold/telemetry_aggregates"
    checkpoint_path = "s3a://telemetry-data/checkpoints/bronze_ingest"

    # Raw -> Bronze
    print("Starting Raw -> Bronze Incremental Ingestion...")
    raw_stream = spark.readStream \
        .schema(telemetry_schema) \
        .option("maxFilesPerTrigger", 500) \
        .json(raw_path)

    bronze_query = raw_stream.writeStream \
        .format("delta") \
        .outputMode("append") \
        .option("checkpointLocation", checkpoint_path) \
        .trigger(availableNow=True) \
        .start(bronze_path)
    bronze_query.awaitTermination()

    # Bronze -> Silver
    print("Starting Bronze -> Silver Transformation...")
    df_bronze = spark.read.format("delta").load(bronze_path)
    df_silver = df_bronze \
        .filter(col("src_ip").isNotNull() & col("bytes_sent").isNotNull()) \
        .withColumn("parsed_timestamp", to_timestamp(col("timestamp"))) \
        .withColumn("year", year(col("parsed_timestamp"))) \
        .withColumn("month", month(col("parsed_timestamp"))) \
        .withColumn("day", dayofmonth(col("parsed_timestamp")))

    df_silver.write \
        .mode("overwrite") \
        .partitionBy("year", "month", "day") \
        .parquet(silver_path)

    # Silver -> Gold
    print("Starting Silver -> Gold Aggregation...")
    df_optimized = spark.read.parquet(silver_path)
    df_gold = df_optimized \
        .groupBy("year", "month", "day", "src_ip") \
        .agg(
            sum("bytes_sent").alias("total_bytes_transferred"),
            count("event_id").alias("total_connections")
        )

    df_gold.write.format("delta").mode("overwrite").save(gold_path)
    print("Pipeline Execution Completed Successfully.")
    spark.stop()

if __name__ == "__main__":
    run_pipeline()
