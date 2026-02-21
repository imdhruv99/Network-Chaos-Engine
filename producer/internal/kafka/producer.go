package kafka

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/config"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/models"
)

type TelemetryProducer struct {
	producer *kafka.Producer
	topic    string
}

func NewProducer(cfg *config.Config) (*TelemetryProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":   cfg.KafkaBrokers,
		"client.id":           "Chaos-Producer-Go",
		"acks":                "1",      // Leader acknowledgment is sufficient for our use case
		"linger.ms":           5,        // Wait 5ms to fill batches
		"batch.size":          16384,    // 16KB batch size
		"compression.type":    "snappy", // Excellent CPU compression for JSON
		"go.delivery.reports": false,    // Disable deliver reports for higher throughput, fire and forget
	})

	if err != nil {
		return nil, err
	}

	return &TelemetryProducer{
		producer: p,
		topic:    cfg.KafkaTopic,
	}, nil
}

func (tp *TelemetryProducer) Produce(packet models.Packet) error {
	payload, err := json.Marshal(packet)

	if err != nil {
		return fmt.Errorf("Failed to marshal packet: %w", err)
	}

	err = tp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &tp.topic, Partition: kafka.PartitionAny},
		Value:          payload,
	}, nil) // nil delivery channel for fire-and-forget

	if err != nil {
		// In a production system, we need to handle backpressure here
		return fmt.Errorf("Failed to produce message: %w", err)
	}

	return nil
}

func (tp *TelemetryProducer) Close() {
	tp.producer.Flush(10000) // Wait up to 5 seconds for messages to be delivered
	tp.producer.Close()
}
