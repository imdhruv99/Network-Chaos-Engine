package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/config"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/generator"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/kafka"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/logger"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/models"
)

var (
	eventsProduced uint64 // Atomic counter to track EPS across all goroutines
)

func main() {

	logFile, err := logger.Init()
	if err != nil {
		// Fallback to standard error if logger fails to boot
		os.Stderr.WriteString("Critical failure initializing logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logFile.Close()

	slog.Info("Starting Telemetry Engine initialization")

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		slog.Error("Failed to initialize Kafka producer", "error", err)
	}
	defer producer.Close() // Ensure producer is closed on exit

	// Channel to handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup       // WaitGroup to wait for all producer goroutines to finish
	quit := make(chan struct{}) // Channel to signal goroutines to stop

	slog.Info("Chaos Engine configured",
		"workers", cfg.WorkerCount,
		"target_eps", cfg.EPSTarget,
		"chaos_mode", cfg.ChaosMode)

	// Calculate how mucj delay each worker needs to hit the exact EPS target
	epsPerWorker := float64(cfg.EPSTarget) / float64(cfg.WorkerCount)
	delayPerEvent := time.Duration(float64(time.Second) / epsPerWorker)

	// Standard Worker
	for i := 0; i < cfg.WorkerCount; i++ {
		wg.Add(1)
		go worker(i, cfg, producer, delayPerEvent, quit, &wg)
	}

	// Port Scan Attack worker
	if cfg.ChaosMode {
		wg.Add(1)
		go portScanAttacker(cfg, producer, quit, &wg)
	}

	go monitorEPS(quit)

	<-sigChan // Wait for shutdown signal
	slog.Info("Shutdown signal received. Flushing to Kafka and exiting...")
	close(quit) // Signal all goroutines to stop
	wg.Wait()   // Wait for all goroutines to finish
	slog.Info("Graceful shutdown complete.")
	slog.Info("Total events produced", "count", eventsProduced)
}

// worker simulates normal traffic by generating packets at a consistent rate defined by delay
func worker(id int, cfg *config.Config, producer *kafka.TelemetryProducer, delay time.Duration, quit <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	// Local random source for lock-free concurrency
	localRand := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id))) // Unique seed per worker
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			packet := generator.GeneratePacket(cfg, localRand)
			err := producer.Produce(packet)
			if err != nil {
				slog.Warn("Failed to produce packet", "worker_id", id, "error", err)
			} else {
				atomic.AddUint64(&eventsProduced, 1)
			}
		}
	}
}

// portScanAttacker simulates a rapid sequence of connection attempts to sequential ports
func portScanAttacker(cfg *config.Config, producer *kafka.TelemetryProducer, quit <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	localRand := rand.New(rand.NewSource(time.Now().UnixNano() + 999))

	// Attack every 15-30 seconds
	attackTimer := time.NewTimer(time.Duration(15+localRand.Intn(15)) * time.Second)

	for {
		select {
		case <-quit:
			return
		case <-attackTimer.C:
			slog.Warn("CHAOS INJECTED: Initiating Port Scan Anomaly")
			attackerIP := "192.168.1.99"
			targetIP := "10.0.5.55"

			// Scan 100 ports rapidly
			for port := 20; port < 120; port++ {
				p := models.Packet{
					Timestamp:     time.Now().UTC().Unix(),
					EventID:       fmt.Sprintf("scan-%d", localRand.Int63()),
					SrcIP:         attackerIP,
					DstIP:         targetIP,
					SrcPort:       localRand.Intn(10000) + 40000,
					DstPort:       port,
					Protocol:      "TCP",
					BytesSent:     64, // Just a SYN packet
					BytesReceived: 0,
					LatencyMS:     2,
					PacketsSent:   1,
					Flags:         []string{"SYN"},
					Tags:          models.Tags{Env: "prod", Region: "us-east-1"},
				}
				producer.Produce(p)
				atomic.AddUint64(&eventsProduced, 1)
				time.Sleep(2 * time.Millisecond) // Slight delay between scan packets
			}
			// Reset timer for next attack
			attackTimer.Reset(time.Duration(15+localRand.Intn(15)) * time.Second)
		}
	}
}

// monitorEPS prints the actual events produced per second
func monitorEPS(quit <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			current := atomic.SwapUint64(&eventsProduced, 0)
			slog.Info("Performance Metrics", "eps_throughput", current)
		}
	}
}
