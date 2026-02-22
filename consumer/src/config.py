from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    # Kafka Configuration
    kafka_brokers: str = "localhost:19092, localhost:29092, localhost:39092"
    kafka_topic: str = "network-telemetry"
    kafka_group_id: str = "telemetry-sink-group"

    # MinIO Configuration
    minio_endpoint: str = "localhost:9000"
    minio_access_key: str = "admin"
    minio_secret_key: str = "password@123"
    minio_bucket_name: str = "telemetry-data"
    minio_secure: bool = False

    # Batching Configuration
    batch_size: int = 2000
    flush_interval: int = 5  # in seconds

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8")

settings = Settings()
