from datetime import datetime, timedelta
from feast import Entity, FeatureView, Field, FileSource, ValueType
from feast.types import Int64

# Defining the entity [ Primary Key ]
ip_entity = Entity(
    name="src_ip",
    join_keys=["src_ip"],
    value_type=ValueType.STRING,
    description="The source IP address of the network traffic."
)

# Defining the offline data source
silver_parquet_source = FileSource(
    name="silver_telemetry_source",
    path= "s3://telemetry-data/silver/telemetry_parquet",
    timestamp_field="parsed_timestamp",
)

# Defining the Feature View
daily_traffic_view = FeatureView(
    name = "daily_ip_traffic",
    entities = [ip_entity],
    ttl = timedelta(days=1), # If an IP hasn't had activity in 1 day, don't use stale features for inference.
    schema = [
        Field(name="bytes_sent", dtype=Int64),
        Field(name="latency_ms", dtype=Int64),
    ],
    online=True, # Allow these features to be materialized to Redis for real-time serving
    source=silver_parquet_source,
    tags={"team": "network_security", "materialized_to_redis": "true"}
)
