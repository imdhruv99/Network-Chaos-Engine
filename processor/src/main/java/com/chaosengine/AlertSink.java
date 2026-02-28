package com.chaosengine;

import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

public class AlertSink extends RichSinkFunction<AnomalyAlert> {
    private static final Logger LOG = LoggerFactory.getLogger(AlertSink.class);

    // transient ensures Flink doesn't try to serialize these network objects
    private transient JedisPool jedisPool;
    private transient HttpClient httpClient;

    private final String webhookUrl;

    public AlertSink(String webhookUrl) {
        this.webhookUrl = webhookUrl;
    }

    @Override
    public void open(Configuration parameters) {
        // Connect to the Redis container defined in docker-compose
        this.jedisPool = new JedisPool("redis", 6379);
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(5))
                .build();
    }

    @Override
    public void invoke(AnomalyAlert alert, Context context) throws Exception {
        try (Jedis jedis = jedisPool.getResource()) {
            String redisKey = "alert:throttle:" + alert.srcIp;

            // Check Redis for the lock
            if (jedis.exists(redisKey)) {
                LOG.info("Alert suppressed for IP: {} (Throttled)", alert.srcIp);
                return;
            }

            // Format the payload (Compatible with Slack and Discord)
            String jsonPayload = String.format("{\"content\": \"🚨 **%s**\"}", alert.message);

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(webhookUrl))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(jsonPayload))
                    .build();

            // Fire the Webhook
            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());

            // If successful, set the 15-minute (900 seconds) lock in Redis
            if (response.statusCode() >= 200 && response.statusCode() < 300) {
                jedis.setex(redisKey, 900, "active");
                LOG.warn("Webhook fired and Redis lock set for IP: {}", alert.srcIp);
            } else {
                LOG.error("Failed to send webhook. Status: {}", response.statusCode());
            }
        } catch (Exception e) {
            LOG.error("Error processing alert in sink", e);
        }
    }

    @Override
    public void close() {
        if (jedisPool != null) {
            jedisPool.close();
        }
    }
}
