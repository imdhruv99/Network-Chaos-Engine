package models

type AppLayer struct {
	Protocol   string `json:"protocol"`
	Method     string `json:"method"`
	Host       string `json:"host"`
	StatusCode int    `json:"status_code"`
}

type Tags struct {
	Env           string `json:"env"`
	Region        string `json:"region"`
	SecurityGroup string `json:"security_group"`
}

type Packet struct {
	Timestamp     int64    `json:"timestamp"`
	EventID       string   `json:"event_id"`
	SrcIP         string   `json:"src_ip"`
	DstIP         string   `json:"dst_ip"`
	SrcPort       int      `json:"src_port"`
	DstPort       int      `json:"dst_port"`
	Protocol      string   `json:"protocol"`
	BytesSent     int      `json:"bytes_sent"`
	BytesReceived int      `json:"bytes_received"`
	LatencyMS     int      `json:"latency_ms"`
	PacketsSent   int      `json:"packets_sent"`
	Flags         []string `json:"flags"`
	AppLayer      AppLayer `json:"app_layer"`
	Tags          Tags     `json:"tags"`
}
