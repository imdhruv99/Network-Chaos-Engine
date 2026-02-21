package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/config"
	"github.com/imdhruv99/Network-Chaos-Engine/producer/internal/models"
)

var (
	srcIPs = []string{
		"10.0.0.5", "10.0.1.12", "192.168.1.100", "172.16.0.50",
		"10.0.2.23", "10.0.3.44", "10.1.0.15", "10.2.5.200",
		"192.168.0.10", "192.168.10.25", "192.168.50.75",
		"172.16.1.10", "172.16.2.20", "172.20.10.5",
		"100.64.1.1", "100.64.2.2",
		"169.254.10.10",
		"192.0.2.10", "198.51.100.25", "203.0.113.45",
	}

	dstIPs = []string{
		"172.16.254.1", "10.0.5.55", "8.8.8.8", "1.1.1.1",
		"10.10.10.10", "10.20.30.40", "172.16.100.1",
		"192.168.100.100", "192.168.200.200",
		"8.8.4.4", "9.9.9.9", "208.67.222.222",
		"104.16.132.229", "151.101.1.69",
		"34.117.59.81", "52.95.110.1",
		"13.32.86.91", "54.239.28.85",
		"203.0.113.99", "198.51.100.88",
	}

	methods = []string{
		"GET", "POST", "PUT", "DELETE",
		"PATCH", "HEAD", "OPTIONS",
		"CONNECT", "TRACE",
		"PROPFIND", "PROPPATCH",
		"MKCOL", "COPY", "MOVE",
		"LOCK", "UNLOCK",
	}

	hosts = []string{
		"api.internal.prod", "auth.internal.prod", "db.internal.prod",
		"cache.internal.prod", "metrics.internal.prod",
		"billing.internal.prod", "search.internal.prod",
		"cdn.internal.prod", "gateway.internal.prod",
		"admin.internal.prod", "files.internal.prod",
		"notifications.internal.prod", "stream.internal.prod",
		"analytics.internal.prod", "reporting.internal.prod",
		"ml.internal.prod", "queue.internal.prod",
		"backup.internal.prod", "proxy.internal.prod",
		"edge.internal.prod",
	}

	flagsOk = [][]string{{"ACK", "PSH"}, {"SYN", "ACK"}, {"ACK"}}
)

func GeneratePacket(cfg *config.Config, localRand *rand.Rand) models.Packet {
	packet := models.Packet{
		Timestamp:     time.Now().UTC().Unix(),
		EventID:       uuid.New().String(),
		SrcIP:         srcIPs[localRand.Intn(len(srcIPs))],
		DstIP:         dstIPs[localRand.Intn(len(dstIPs))],
		SrcPort:       localRand.Intn(50000) + 1024,
		DstPort:       443,
		Protocol:      "TCP",
		BytesSent:     localRand.Intn(5000) + 200,
		BytesReceived: localRand.Intn(10000) + 500,
		LatencyMS:     localRand.Intn(40) + 10,
		PacketsSent:   localRand.Intn(50) + 5,
		Flags:         flagsOk[localRand.Intn(len(flagsOk))],
		AppLayer: models.AppLayer{
			Protocol:   "HTTP/1.1",
			Method:     methods[localRand.Intn(len(methods))],
			Host:       hosts[localRand.Intn(len(hosts))],
			StatusCode: 200,
		},
		Tags: models.Tags{
			Env:           "Production",
			Region:        "us-east-1",
			SecurityGroup: fmt.Sprintf("sg-%06d", localRand.Intn(999999)),
		},
	}

	if cfg.ChaosMode {
		injectChaos(&packet, cfg, localRand)
	}

	return packet
}

func injectChaos(p *models.Packet, cfg *config.Config, localRand *rand.Rand) {
	chaosRoll := localRand.Float64()

	// Anomaly 1: Latency Spike
	if chaosRoll < cfg.LatencySpikeProb {
		p.LatencyMS = 2000 + localRand.Intn(5000) // 2s to 7s latency
	}

	// Anomaly 2: Data Exfiltration
	if chaosRoll < cfg.DataExfilProb {
		p.BytesSent = 10000000 + localRand.Intn(50000000) // 10MB to 60MB
		p.DstPort = 53                                    // Simulating DNS Tunneling
		p.Protocol = "UDP"

	}

	// Anomaly 3: Connection Reset / Service Crash
	if chaosRoll < cfg.ErrorRateProb {
		p.Flags = []string{"RST", "ACK"}
		p.AppLayer.StatusCode = 503
	}
}
