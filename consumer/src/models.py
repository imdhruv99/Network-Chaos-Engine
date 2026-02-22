from typing import List
from datetime import datetime
from pydantic import BaseModel


class AppLayer(BaseModel):
    protocol: str
    method: str
    host: str
    status_code: int


class Tags(BaseModel):
    env: str
    region: str
    security_group: str


class Packet(BaseModel):
    timestamp: datetime
    event_id: str
    src_ip: str
    dst_ip: str
    src_port: int
    dst_port: int
    protocol: str
    bytes_sent: int
    bytes_received: int
    latency_ms: int
    packets_sent: int
    flags: List[str]
    app_layer: AppLayer
    tags: Tags
