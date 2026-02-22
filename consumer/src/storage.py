import io
import gzip
import json
from typing import List
from minio import Minio
from src.config import settings
from src.models import Packet
from src.logger import logger

class StorageClient:
    def __init__(self):
        self.client = Minio(
            settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_secure
        )
        self._ensure_bucket()

    def _ensure_bucket(self) -> None:
        if not self.client.bucket_exists(settings.minio_bucket_name):
            self.client.make_bucket(settings.minio_bucket_name)
            logger.info(f"Created bucket: {settings.minio_bucket_name}")
        else:
            logger.info(f"Bucket already exists: {settings.minio_bucket_name}")

    def upload_batch(self, batch_id: str, packets: List[Packet]) -> None:

        if not packets:
            logger.warning(f"Empty batch {batch_id} - skipping upload")
            return

        json_data = "\n".join([p.model_dump_json()for p in packets]).encode('utf-8')
        compressed_data = io.BytesIO()
        with  gzip.GzipFile(fileobj=compressed_data, mode='w') as f:
            f.write(json_data)

        # compressed_data.seek(0) is necessary to reset the buffer's position to the beginning before uploading
        compressed_data.seek(0)

        object_name = f"ingest/batch_{batch_id}.json.gz"

        try:
            self.client.put_object(
                bucket_name=settings.minio_bucket_name,
                object_name=object_name,
                data=compressed_data,
                length=compressed_data.getbuffer().nbytes,
                content_type="application/gzip"
            )
            logger.info(f"Successfully uploaded batch {batch_id} with {len(packets)} packets to MinIO")
        except Exception as e:
            logger.error(f"Failed to upload batch {batch_id} to MinIO: {e}")
            raise
