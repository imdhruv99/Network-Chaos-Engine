package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	KafkaBrokers     string  `mapstructure:"KAFKA_BROKERS"`
	KafkaTopic       string  `mapstructure:"KAFKA_TOPIC"`
	EPSTarget        int     `mapstructure:"EPS_TARGET"`
	WorkerCount      int     `mapstructure:"WORKER_COUNT"`
	ChaosMode        bool    `mapstructure:"CHAOS_MODE"`
	LatencySpikeProb float64 `mapstructure:"LATENCY_SPIKE_PROB"`
	ErrorRateProb    float64 `mapstructure:"ERROR_RATE_PROB"`
	DataExfilProb    float64 `mapstructure:"DATA_EXFIL_PROB"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("KAFKA_BROKERS", "localhost:19092,localhost:29092,localhost:39092")
	viper.SetDefault("KAFKA_TOPIC", "network-telemetry")
	viper.SetDefault("EPS_TARGET", 5000)
	viper.SetDefault("WORKER_COUNT", 10)
	viper.SetDefault("CHAOS_MODE", true)
	viper.SetDefault("LATENCY_SPIKE_PROB", 0.1)
	viper.SetDefault("ERROR_RATE_PROB", 0.05)
	viper.SetDefault("DATA_EXFIL_PROB", 0.02)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
