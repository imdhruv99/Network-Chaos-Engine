package com.chaosengine;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.windowing.ProcessWindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.time.Time;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;

public class AnomalyDetectorJob {

    private static final Logger LOG = LoggerFactory.getLogger(AnomalyDetectorJob.class);

    private static final String WEBHOOK_URL = "WEBHOOK_URL_PLACEHOLDER"; // Replace with actual Webhook URL

    public static void main(String[] args) throws Exception {
        final StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // kafka cluster bootstrap servers, since we are running flink job inside the same docker network,
        // we can use the service names defined in docker-compose.yml
        KafkaSource<String> source = KafkaSource.<String>builder()
                .setBootstrapServers("kafka-1:9092,kafka-2:9092,kafka-3:9092")
                .setTopics("network-telemetry")
                .setGroupId("flink-anomaly-group")
                .setStartingOffsets(OffsetsInitializer.latest())
                .setValueOnlyDeserializer(new SimpleStringSchema())
                .build();

        // Handling out-of-order events with a 5-second tolerance.
        WatermarkStrategy<TelemetryEvent> watermarkStrategy = WatermarkStrategy
                .<TelemetryEvent>forBoundedOutOfOrderness(Duration.ofSeconds(5))
                .withTimestampAssigner((event, timestamp) -> event.timestamp.toEpochMilli());

        // Ingesting telemetry data from Kafka, parsing JSON, and assigning timestamps/watermarks.
        DataStream<TelemetryEvent> telemetryStream = env
                .fromSource(source, WatermarkStrategy.noWatermarks(), "Kafka Source")
                .map(jsonString -> {
                    ObjectMapper mapper = new ObjectMapper();
                    mapper.registerModule(new JavaTimeModule());
                    return mapper.readValue(jsonString, TelemetryEvent.class);
                })
                .name("JSON Parser")
                .assignTimestampsAndWatermarks(watermarkStrategy)
                .name("Watermarks");

        // Keying by Source IP and applying a sliding window to aggregate bytes sent and packet count, then applying the anomaly detection logic.
        telemetryStream
                .keyBy(event -> event.srcIp) // Group all traffic by Source IP
                .window(SlidingEventTimeWindows.of(Time.minutes(1), Time.seconds(10))) // 60s window, updating every 10s
                .process(new ProcessWindowFunction<TelemetryEvent, AnomalyAlert, String, TimeWindow>() {

                    @Override
                    public void process(String srcIp, Context context, Iterable<TelemetryEvent> elements, Collector<AnomalyAlert> out) {
                        long totalBytes = 0;
                        int packetCount = 0;

                        // Aggregate the state within this window
                        for (TelemetryEvent event : elements) {
                            totalBytes += event.bytesSent;
                            packetCount++;
                        }

                        // The Detection Rule (e.g., Data Exfiltration Anomaly > 50MB per minute)
                        if (totalBytes > 50_000_000) {
                            String alertMsg = String.format("DATA EXFILTRATION DETECTED! IP: %s transferred %d bytes in 60s.", srcIp, totalBytes);
                            out.collect(new AnomalyAlert(srcIp, totalBytes, alertMsg));
                        }
                    }
                })
                .name("Bandwidth Anomaly Detector")
                .addSink(new AlertSink(WEBHOOK_URL))
                .name("Redis Deduplication & Webhook Sink");

        LOG.info("Submitting Network Chaos Anomaly Detector Job...");
        env.execute("Network Chaos Anomaly Detector");
    }
}
