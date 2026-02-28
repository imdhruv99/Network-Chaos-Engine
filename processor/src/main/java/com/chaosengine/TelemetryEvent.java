package com.chaosengine;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;

@JsonIgnoreProperties(ignoreUnknown = true)
public class TelemetryEvent {
    // Only mapping the fields which are required for the anomaly detection model. The rest of the fields will be ignored.
    public Instant timestamp;
    @JsonProperty("event_id") public String eventId;
    @JsonProperty("src_ip") public String srcIp;
    @JsonProperty("dst_ip") public String dstIp;
    @JsonProperty("bytes_sent") public long bytesSent;
    @JsonProperty("latency_ms") public int latencyMs;

    @Override
    public String toString() {
        return "IP: " + srcIp + " | Bytes: " + bytesSent + " | Latency: " + latencyMs + "ms | Time: " + timestamp;
    }
}
