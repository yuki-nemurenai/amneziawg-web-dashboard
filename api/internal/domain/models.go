package domain

// ObfuscationParams are the AmneziaWG parameters that disguise WireGuard
// traffic. They are kept as strings because they are written to the
// configuration file verbatim and H1–H4 are ranges.
type ObfuscationParams struct {
	Jc   string `json:"jc"`
	Jmin string `json:"jmin"`
	Jmax string `json:"jmax"`
	S1   string `json:"s1"`
	S2   string `json:"s2"`
	S3   string `json:"s3"`
	S4   string `json:"s4"`
	H1   string `json:"h1"`
	H2   string `json:"h2"`
	H3   string `json:"h3"`
	H4   string `json:"h4"`
	I1   string `json:"i1"`
	I2   string `json:"i2"`
	I3   string `json:"i3"`
	I4   string `json:"i4"`
	I5   string `json:"i5"`
}

// ServerConfig is the server side of the AmneziaWG configuration with all its
// peers. Endpoint, DNS, LANAllowed and PersistentKeepalive are not server
// settings: they only shape the client configurations.
type ServerConfig struct {
	Address             string            `json:"address"`
	ListenPort          string            `json:"listen_port"`
	PrivateKey          string            `json:"private_key"`
	PublicKey           string            `json:"public_key"`
	DNS                 string            `json:"dns"`
	Endpoint            string            `json:"endpoint"`
	LANAllowed          string            `json:"lan_allowed"`
	PersistentKeepalive string            `json:"persistent_keepalive"`
	PostUp              string            `json:"post_up"`
	PostDown            string            `json:"post_down"`
	Obfuscation         ObfuscationParams `json:"obfuscation"`
	Peers               []Peer            `json:"peers"`
}

// Peer is a client of the server. The fields from LatestHandshake on are the
// runtime state reported by awg show and are not stored.
type Peer struct {
	Name                string `json:"name"`
	PublicKey           string `json:"public_key"`
	PresharedKey        string `json:"preshared_key"`
	AllowedIPs          string `json:"allowed_ips"`
	IP                  string `json:"ip"`
	CreatedAt           string `json:"created_at,omitempty"`
	LatestHandshake     int64  `json:"latest_handshake,omitempty"`
	LatestHandshakeText string `json:"latest_handshake_text,omitempty"`
	RxBytes             int64  `json:"rx_bytes"`
	TxBytes             int64  `json:"tx_bytes"`
	RxFormatted         string `json:"rx_formatted"`
	TxFormatted         string `json:"tx_formatted"`
	IsOnline            bool   `json:"is_online"`
}

// CreateClientRequest is the body of POST /api/clients. An empty IP means the
// next free address of the subnet.
type CreateClientRequest struct {
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
}

// ClientResponse is a created client with its configuration and QR code.
type ClientResponse struct {
	Name       string `json:"name"`
	IP         string `json:"ip"`
	ConfigText string `json:"config_text"`
	QRCodeSVG  string `json:"qr_code_svg"`
}

// SystemStatus is the live state of the interface and its traffic for the
// dashboard.
type SystemStatus struct {
	Interface          string                        `json:"interface"`
	IsRunning          bool                          `json:"is_running"`
	Mode               string                        `json:"mode"`
	ActivePeers        int                           `json:"active_peers"`
	OnlinePeers        int                           `json:"online_peers"`
	TotalPeers         int                           `json:"total_peers"`
	Endpoint           string                        `json:"endpoint"`
	AutoEndpoint       bool                          `json:"auto_endpoint"`
	VPNSubnet          string                        `json:"vpn_subnet"`
	PublicKey          string                        `json:"public_key"`
	ListenPort         string                        `json:"listen_port"`
	TotalRxBytes       int64                         `json:"total_rx_bytes"`
	TotalTxBytes       int64                         `json:"total_tx_bytes"`
	TotalRxFormatted   string                        `json:"total_rx_formatted"`
	TotalTxFormatted   string                        `json:"total_tx_formatted"`
	TrafficHistory     []TrafficPoint                `json:"traffic_history"`
	PeerTrafficHistory map[string][]PeerTrafficPoint `json:"peer_traffic_history"`
	Location           *ServerLocation               `json:"location"`
}

// ServerLocation is the country the server public IP belongs to.
type ServerLocation struct {
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

// TrafficPoint is the total traffic of the server at one moment.
type TrafficPoint struct {
	Timestamp int64 `json:"timestamp"` // unix timestamp
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
}

// PeerTrafficPoint is the traffic of one peer at one moment.
type PeerTrafficPoint struct {
	Timestamp int64 `json:"timestamp"` // unix timestamp
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
}
