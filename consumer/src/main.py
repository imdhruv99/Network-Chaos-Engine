import sys
import time
import uuid
import signal
from typing import List
from confluent_kafka import Consumer, KafkaError, KafkaException
from pydantic import ValidationError

from src.config import settings
from src.logger import logger
from src.models import Packet
from src.storage import StorageClient

class TelemetrySink:
    def __init__(self):
        self.storage = StorageClient()
        self.consumer = Consumer({
            'bootstrap.servers': settings.kafka_brokers,
            'group.id': settings.kafka_group_id,
            'auto.offset.reset': 'earliest',
            'enable.auto.commit': False  # We commit manually AFTER successful S3 upload
        })
        self.consumer.subscribe([settings.kafka_topic])
        self.running = True

    def shutdown(self, signum, frame):
        logger.info("Shutdown signal received. Committing final offsets and closing...")
        self.running = False

    def run(self) -> None:
        signal.signal(signal.SIGINT, self.shutdown)
        signal.signal(signal.SIGTERM, self.shutdown)

        batch: List[Packet] = []
        last_flush_time = time.time()

        logger.info(f"Starting consumer loop. Target batch size: {settings.batch_size}")

        try:
            while self.running:
                # Poll frequently
                msg = self.consumer.poll(timeout=1.0)

                if msg is None:
                    # Check if we need to flush due to timeout, even if no new messages arrived
                    self._check_flush(batch, last_flush_time)
                    continue

                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        continue
                    else:
                        logger.error(f"Kafka error: {msg.error()}")
                        raise KafkaException(msg.error())

                # Validate and parse the message using Pydantic
                try:
                    raw_data = msg.value().decode('utf-8')
                    packet = Packet.model_validate_json(raw_data)
                    batch.append(packet)
                except ValidationError as e:
                    logger.warning(f"Data validation failed for message: {e}. Skipping malformed record.")
                except Exception as e:
                    logger.error(f"Unexpected error processing message: {e}")

                # Check if we need to flush due to batch size or timeout
                last_flush_time = self._check_flush(batch, last_flush_time)

        finally:
            # Final flush on shutdown
            if batch:
                self._flush_batch(batch)
            self.consumer.close()
            logger.info("Consumer safely shut down.")

    def _check_flush(self, batch: List[Packet], last_flush_time: float) -> float:
        current_time = time.time()
        time_elapsed = current_time - last_flush_time

        if len(batch) >= settings.batch_size or (time_elapsed >= settings.flush_interval and len(batch) > 0):
            self._flush_batch(batch)
            return current_time # Reset timer

        return last_flush_time

    def _flush_batch(self, batch: List[Packet]) -> None:
        batch_id = uuid.uuid4().hex
        try:
            self.storage.upload_batch(batch_id, batch)
            # Only commit to Kafka AFTER successful storage to prevent data loss
            self.consumer.commit(asynchronous=False)
            batch.clear()
        except Exception as e:
            logger.error(f"Failed to process batch {batch_id}. Halting to prevent data loss. Error: {e}")
            sys.exit(1)

if __name__ == "__main__":
    sink = TelemetrySink()
    sink.run()
