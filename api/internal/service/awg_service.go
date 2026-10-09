// Package service holds the business logic of the dashboard: administrators,
// clients, their keys and addresses, and the AmneziaWG interface they use.
package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/repository"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/pkg/crypto"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/pkg/qrcode"
)

const (
	// onlineWindow is how recent the last handshake of an online peer is.
	// WireGuard renews the handshake every two minutes while traffic flows.
	onlineWindow       = 3 * time.Minute
	statsCacheTTL      = 3 * time.Second
	metricsInterval    = 5 * time.Minute
	defaultAWGPort     = "51820"
	defaultIfaceName   = "awg0"
	hiddenClientKey    = "<client-private-key-hidden>"
	allTrafficIPs      = "0.0.0.0/0, ::/0"
	trafficHistorySize = 24
	// peerHistorySize is 24 hours of points recorded every metricsInterval.
	peerHistorySize = 288
	topPeers        = 10
)

// Commander runs the host commands that manage the AmneziaWG interface: awg,
// ip and iptables. It is the boundary between the service and the system, so
// tests replace it instead of changing the host network.
type Commander interface {
	// Run runs the command and returns its standard output. A failed command
	// is reported as an error.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	// LookPath returns the path of the named executable, or an error if it is
	// not installed.
	LookPath(name string) (string, error)
}

// Locator finds where the server is reachable from the internet, for client
// configurations that have no endpoint set and for the dashboard.
type Locator interface {
	PublicIP(ctx context.Context) (string, error)
	Location(ctx context.Context) (*domain.ServerLocation, error)
}

// AWGService manages the clients of the AmneziaWG server and keeps the live
// interface in line with the stored configuration.
type AWGService interface {
	GetServerConfig(ctx context.Context) (*domain.ServerConfig, error)
	UpdateServerConfig(ctx context.Context, cfg *domain.ServerConfig) error
	GetClients(ctx context.Context) ([]domain.Peer, error)
	CreateClient(ctx context.Context, req domain.CreateClientRequest) (*domain.ClientResponse, error)
	DeleteClient(ctx context.Context, name string) error
	GetClientConfigText(ctx context.Context, name string) (string, error)
	GetClientQRCode(ctx context.Context, name string) (string, error)
	GetSystemStatus(ctx context.Context) (*domain.SystemStatus, error)
	// StartMetricsCollector records the traffic every five minutes until ctx
	// is done.
	StartMetricsCollector(ctx context.Context)
}

// AWGConfig is what an AWGService works with.
type AWGConfig struct {
	Repo      repository.ConfigRepository
	IPService IPService
	Commander Commander
	Locator   Locator
	// ClientsDir keeps the client configurations, the only place that has the
	// client private keys.
	ClientsDir string
	// InterfaceName is the AmneziaWG interface, awg0 if empty.
	InterfaceName string
}

type peerRuntimeStats struct {
	LatestHandshake int64
	RxBytes         int64
	TxBytes         int64
}

type awgService struct {
	repo          repository.ConfigRepository
	ipService     IPService
	cmd           Commander
	locator       Locator
	clientsDir    string
	interfaceName string
	now           func() time.Time

	statsCache     map[string]peerRuntimeStats
	statsCacheTime time.Time
	statsMu        sync.RWMutex
}

// NewAWGService returns an AWGService and brings the interface in line with
// the stored configuration, so a restarted container serves its clients
// without waiting for the first change.
func NewAWGService(ctx context.Context, cfg AWGConfig) AWGService {
	if err := os.MkdirAll(cfg.ClientsDir, 0755); err != nil {
		slog.Error("Failed to create clients directory", "path", cfg.ClientsDir, "error", err)
	}
	if cfg.InterfaceName == "" {
		cfg.InterfaceName = defaultIfaceName
	}
	svc := &awgService{
		repo:          cfg.Repo,
		ipService:     cfg.IPService,
		cmd:           cfg.Commander,
		locator:       cfg.Locator,
		clientsDir:    cfg.ClientsDir,
		interfaceName: cfg.InterfaceName,
		now:           time.Now,
		statsCache:    make(map[string]peerRuntimeStats),
	}
	svc.syncRuntime(ctx)
	return svc
}

func (s *awgService) GetServerConfig(ctx context.Context) (*domain.ServerConfig, error) {
	return s.repo.LoadServerConfig(ctx)
}

func (s *awgService) UpdateServerConfig(ctx context.Context, newCfg *domain.ServerConfig) error {
	current, err := s.repo.LoadServerConfig(ctx)
	if err != nil {
		return err
	}
	newCfg.Peers = current.Peers
	if err := s.repo.SaveServerConfig(ctx, newCfg); err != nil {
		return err
	}
	s.syncRuntime(ctx)
	s.regenerateClientConfigs(ctx, newCfg)
	return nil
}

// regenerateClientConfigs rewrites the stored client configurations after a
// server change, keeping the private key of each client.
func (s *awgService) regenerateClientConfigs(ctx context.Context, cfg *domain.ServerConfig) {
	files, err := os.ReadDir(s.clientsDir)
	if err != nil {
		slog.Error("Failed to list client configs", "path", s.clientsDir, "error", err)
		return
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".conf") {
			continue
		}
		name := strings.TrimSuffix(f.Name(), ".conf")
		path := filepath.Join(s.clientsDir, f.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("Failed to read client config", "path", path, "error", err)
			continue
		}

		privKey := ""
		for line := range strings.SplitSeq(string(content), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "PrivateKey") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					privKey = strings.TrimSpace(parts[1])
				}
				break
			}
		}
		if privKey == "" {
			continue
		}

		i := slices.IndexFunc(cfg.Peers, func(p domain.Peer) bool { return p.Name == name })
		if i < 0 || cfg.Peers[i].IP == "" {
			continue
		}
		newConf := s.buildClientConfig(ctx, cfg, privKey, cfg.Peers[i].PresharedKey, cfg.Peers[i].IP)
		if err := os.WriteFile(path, []byte(newConf), 0600); err != nil {
			slog.Error("Failed to write client config", "path", path, "error", err)
		}
	}
}

func (s *awgService) GetClients(ctx context.Context) ([]domain.Peer, error) {
	cfg, err := s.repo.LoadServerConfig(ctx)
	if err != nil {
		return nil, err
	}

	runtimeStats := s.fetchPeerRuntimeStats(ctx)
	now := s.now()

	for i := range cfg.Peers {
		peer := &cfg.Peers[i]
		if stat, ok := runtimeStats[peer.PublicKey]; ok {
			peer.LatestHandshake = stat.LatestHandshake
			peer.RxBytes = stat.RxBytes
			peer.TxBytes = stat.TxBytes
		}

		peer.RxFormatted = FormatBytes(peer.RxBytes)
		peer.TxFormatted = FormatBytes(peer.TxBytes)
		peer.LatestHandshakeText = FormatHandshake(peer.LatestHandshake, now)
		peer.IsOnline = peer.LatestHandshake > 0 && now.Unix()-peer.LatestHandshake <= int64(onlineWindow/time.Second)
	}

	SortPeersByIP(cfg.Peers)
	return cfg.Peers, nil
}

func (s *awgService) CreateClient(ctx context.Context, req domain.CreateClientRequest) (*domain.ClientResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: client name is empty", domain.ErrInvalidInput)
	}

	cfg, err := s.repo.LoadServerConfig(ctx)
	if err != nil {
		return nil, err
	}

	clientIP := req.IP
	if clientIP == "" {
		allocated, err := s.ipService.AllocateNextIP(ExtractSubnetPrefix(cfg.Address), cfg.Peers)
		if err != nil {
			return nil, err
		}
		clientIP = allocated
	}

	clientKeyPair := crypto.GenerateKeyPair()
	psk := crypto.GeneratePresharedKey()

	peer := domain.Peer{
		Name:         name,
		PublicKey:    clientKeyPair.PublicKey,
		PresharedKey: psk,
		AllowedIPs:   clientIP + "/32",
		IP:           clientIP,
	}
	if err := s.repo.AddPeer(ctx, peer); err != nil {
		return nil, err
	}

	clientConfigText := s.buildClientConfig(ctx, cfg, clientKeyPair.PrivateKey, psk, clientIP)

	// The response below is the only other copy of the private key: without
	// this file later downloads show a placeholder instead.
	clientFilePath := filepath.Join(s.clientsDir, name+".conf")
	if err := os.WriteFile(clientFilePath, []byte(clientConfigText), 0600); err != nil {
		slog.Error("Failed to save client config", "path", clientFilePath, "error", err)
	}

	qrDataURL, err := qrcode.GeneratePNGDataURL(clientConfigText, 320)
	if err != nil {
		slog.Warn("Failed to generate QR code data URL", "error", err)
	}

	s.syncRuntime(ctx)

	return &domain.ClientResponse{
		Name:       name,
		IP:         clientIP,
		ConfigText: clientConfigText,
		QRCodeSVG:  qrDataURL,
	}, nil
}

func (s *awgService) DeleteClient(ctx context.Context, name string) error {
	if _, err := s.repo.GetPeerByName(ctx, name); err != nil {
		return err
	}

	if err := s.repo.DeletePeer(ctx, name); err != nil {
		return err
	}

	clientFilePath := filepath.Join(s.clientsDir, name+".conf")
	if err := os.Remove(clientFilePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("Failed to remove client config", "path", clientFilePath, "error", err)
	}

	s.syncRuntime(ctx)
	return nil
}

func (s *awgService) GetClientConfigText(ctx context.Context, name string) (string, error) {
	clientFilePath := filepath.Join(s.clientsDir, name+".conf")
	if content, err := os.ReadFile(clientFilePath); err == nil {
		return string(content), nil
	}

	peer, err := s.repo.GetPeerByName(ctx, name)
	if err != nil {
		return "", err
	}

	cfg, err := s.repo.LoadServerConfig(ctx)
	if err != nil {
		return "", err
	}

	return s.buildClientConfig(ctx, cfg, hiddenClientKey, peer.PresharedKey, peer.IP), nil
}

func (s *awgService) GetClientQRCode(ctx context.Context, name string) (string, error) {
	configText, err := s.GetClientConfigText(ctx, name)
	if err != nil {
		return "", err
	}

	return qrcode.GeneratePNGDataURL(configText, 360)
}

func (s *awgService) GetSystemStatus(ctx context.Context) (*domain.SystemStatus, error) {
	clients, err := s.GetClients(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := s.repo.LoadServerConfig(ctx)
	if err != nil {
		return nil, err
	}

	var onlineCount int
	var totalRx int64
	var totalTx int64

	for _, peer := range clients {
		if peer.IsOnline {
			onlineCount++
		}
		totalRx += peer.RxBytes
		totalTx += peer.TxBytes
	}

	history, err := s.repo.GetTrafficHistory(ctx, trafficHistorySize)
	if err != nil {
		slog.Warn("Failed to load traffic history", "error", err)
	}
	if history == nil {
		history = []domain.TrafficPoint{}
	}

	peerHistory, err := s.repo.GetTopPeersTrafficHistory(ctx, peerHistorySize, topPeers)
	if err != nil {
		slog.Warn("Failed to load peer traffic history", "error", err)
	}
	if peerHistory == nil {
		peerHistory = make(map[string][]domain.PeerTrafficPoint)
	}

	location, err := s.locator.Location(ctx)
	if err != nil {
		slog.Debug("Server location is unknown", "error", err)
	}

	endpoint, autoEndpoint := s.endpoint(ctx, cfg.Endpoint, cfg.ListenPort)

	_, lookErr := s.cmd.LookPath("awg")

	return &domain.SystemStatus{
		Interface:          s.interfaceName,
		IsRunning:          lookErr == nil,
		Mode:               s.interfaceMode(ctx),
		ActivePeers:        len(cfg.Peers),
		OnlinePeers:        onlineCount,
		TotalPeers:         len(cfg.Peers),
		Endpoint:           endpoint,
		AutoEndpoint:       autoEndpoint,
		VPNSubnet:          cfg.Address,
		PublicKey:          cfg.PublicKey,
		ListenPort:         cfg.ListenPort,
		TotalRxBytes:       totalRx,
		TotalTxBytes:       totalTx,
		TotalRxFormatted:   FormatBytes(totalRx),
		TotalTxFormatted:   FormatBytes(totalTx),
		TrafficHistory:     history,
		PeerTrafficHistory: peerHistory,
		Location:           location,
	}, nil
}

// interfaceMode reports whether the interface is served by the kernel module
// or by amneziawg-go, which shows up as a tun device.
func (s *awgService) interfaceMode(ctx context.Context) string {
	out, err := s.cmd.Run(ctx, "ip", "-d", "link", "show", s.interfaceName)
	if err != nil {
		return "Unknown"
	}
	link := string(out)
	switch {
	case strings.Contains(link, "amneziawg") || strings.Contains(link, "wireguard"):
		return "Kernel"
	case strings.Contains(link, "tun"):
		return "Userspace"
	default:
		return "Unknown"
	}
}

// endpoint returns the address clients connect to. A configured endpoint
// without a port gets listenPort, the port the server listens on. An empty one
// falls back to the detected public IP, and auto reports that fallback; it
// stays empty if detection fails.
func (s *awgService) endpoint(ctx context.Context, configured, listenPort string) (endpoint string, auto bool) {
	port := listenPort
	if port == "" {
		port = defaultAWGPort
	}

	configured = strings.TrimSpace(configured)
	if configured != "" {
		if strings.Contains(configured, ":") {
			return configured, false
		}
		return configured + ":" + port, false
	}

	publicIP, err := s.locator.PublicIP(ctx)
	if err != nil {
		slog.Warn("Failed to auto-detect public IPv4 address", "error", err)
		return "", false
	}
	return publicIP + ":" + port, true
}

func (s *awgService) fetchPeerRuntimeStats(ctx context.Context) map[string]peerRuntimeStats {
	s.statsMu.RLock()
	if time.Since(s.statsCacheTime) < statsCacheTTL && s.statsCache != nil {
		stats := s.statsCache
		s.statsMu.RUnlock()
		return stats
	}
	s.statsMu.RUnlock()

	if _, err := s.cmd.LookPath("awg"); err != nil {
		return map[string]peerRuntimeStats{}
	}

	output, err := s.cmd.Run(ctx, "awg", "show", s.interfaceName, "dump")
	if err != nil {
		slog.Warn("Failed to execute awg show dump", "error", err)
		return map[string]peerRuntimeStats{}
	}
	stats := parseDump(output)

	s.statsMu.Lock()
	s.statsCache = stats
	s.statsCacheTime = time.Now()
	s.statsMu.Unlock()

	return stats
}

// parseDump reads the peers from the output of awg show <interface> dump. The
// first line describes the interface; each following line is a peer:
// public_key preshared_key endpoint allowed_ips latest_handshake rx_bytes
// tx_bytes persistent_keepalive.
func parseDump(output []byte) map[string]peerRuntimeStats {
	stats := make(map[string]peerRuntimeStats)
	first := true
	for line := range strings.Lines(string(output)) {
		if first {
			first = false
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		handshake, _ := strconv.ParseInt(fields[4], 10, 64)
		rx, _ := strconv.ParseInt(fields[5], 10, 64)
		tx, _ := strconv.ParseInt(fields[6], 10, 64)
		stats[fields[0]] = peerRuntimeStats{
			LatestHandshake: handshake,
			RxBytes:         rx,
			TxBytes:         tx,
		}
	}
	return stats
}

// buildClientConfig returns the configuration file of a client. Unless the
// server limits clients to LAN routes, they send all their traffic through the
// tunnel.
func (s *awgService) buildClientConfig(ctx context.Context, serverCfg *domain.ServerConfig, clientPrivKey string, psk string, clientIP string) string {
	allowed := strings.TrimSpace(serverCfg.LANAllowed)
	if allowed == "" || allowed == allTrafficIPs || allowed == "0.0.0.0/0" {
		allowed = allTrafficIPs
	} else if clientSubnet := clientIP + "/32"; !strings.Contains(allowed, clientSubnet) {
		allowed = clientSubnet + ", " + allowed
	}

	endpoint, _ := s.endpoint(ctx, serverCfg.Endpoint, serverCfg.ListenPort)

	return repository.FormatClientConfig(repository.ClientConfig{
		Address:             clientIP + "/32",
		DNS:                 serverCfg.DNS,
		PrivateKey:          clientPrivKey,
		Obfuscation:         serverCfg.Obfuscation,
		ServerPublicKey:     serverCfg.PublicKey,
		PresharedKey:        psk,
		AllowedIPs:          allowed,
		Endpoint:            endpoint,
		PersistentKeepalive: serverCfg.PersistentKeepalive,
	})
}

// syncRuntime brings the live interface, its address, peers and NAT rules in
// line with the stored configuration. Failures are logged, not returned: the
// configuration is already saved and is applied again on the next change or
// restart.
func (s *awgService) syncRuntime(ctx context.Context) {
	// The configuration is already saved, so a client that hangs up must not
	// leave the interface half configured.
	ctx = context.WithoutCancel(ctx)

	if _, err := s.cmd.LookPath("awg"); err != nil {
		slog.Info("awg CLI tool not found in PATH — skipping live runtime sync")
		return
	}

	cfg, err := s.repo.LoadServerConfig(ctx)
	if err != nil {
		slog.Error("Failed to load server config for syncRuntime", "error", err)
		return
	}

	// 1. Ensure the interface exists, preferring the kernel module.
	if _, err := s.cmd.Run(ctx, "ip", "link", "show", s.interfaceName); err != nil {
		slog.Info("Attempting to create kernel interface", "interface", s.interfaceName)
		if _, err := s.cmd.Run(ctx, "ip", "link", "add", "dev", s.interfaceName, "type", "amneziawg"); err != nil {
			slog.Info("Kernel module not found or failed, falling back to amneziawg-go", "interface", s.interfaceName)
			if _, err := s.cmd.Run(ctx, "amneziawg-go", s.interfaceName); err != nil {
				slog.Error("Failed to start amneziawg-go", "interface", s.interfaceName, "error", err)
			}
		}
	}

	// 2. Configure the IP address of the interface.
	if cfg.Address != "" {
		if _, err := s.cmd.Run(ctx, "ip", "addr", "flush", "dev", s.interfaceName); err != nil {
			slog.Warn("Failed to flush interface addresses", "interface", s.interfaceName, "error", err)
		}
		if _, err := s.cmd.Run(ctx, "ip", "addr", "add", cfg.Address, "dev", s.interfaceName); err != nil {
			slog.Error("Failed to assign IP address to interface", "interface", s.interfaceName, "address", cfg.Address, "error", err)
		}
		if _, err := s.cmd.Run(ctx, "ip", "link", "set", "dev", s.interfaceName, "up"); err != nil {
			slog.Error("Failed to bring interface up", "interface", s.interfaceName, "error", err)
		}
	}

	// 3. Apply the keys, obfuscation and peers with awg syncconf, which does
	// not drop the sessions of unchanged peers.
	if err := s.syncConf(ctx, cfg); err != nil {
		slog.Error("Failed awg syncconf", "interface", s.interfaceName, "error", err)
	} else {
		slog.Info("Successfully synced awg interface runtime via syncconf", "interface", s.interfaceName)
	}

	// 4. Ensure forwarding and NAT for the clients.
	s.ensureIptablesRule(ctx, "filter", "FORWARD", "-i", s.interfaceName, "-j", "ACCEPT")
	s.ensureIptablesRule(ctx, "filter", "FORWARD", "-o", s.interfaceName, "-j", "ACCEPT")

	routes, err := s.cmd.Run(ctx, "ip", "route", "show", "default")
	if err != nil {
		slog.Warn("Failed to read the default route", "error", err)
		return
	}
	if defaultIf := defaultRouteInterface(routes); defaultIf != "" {
		if s.ensureIptablesRule(ctx, "nat", "POSTROUTING", "-o", defaultIf, "-j", "MASQUERADE") {
			slog.Info("Applied NAT MASQUERADE rule for AWG interface", "out_interface", defaultIf)
		}
	}
}

// syncConf applies cfg to the interface with awg syncconf, which reads the
// configuration from a file.
func (s *awgService) syncConf(ctx context.Context, cfg *domain.ServerConfig) error {
	tmpFile, err := os.CreateTemp("", "awg-sync-*.conf")
	if err != nil {
		return fmt.Errorf("create syncconf file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(repository.FormatSyncConfig(cfg)); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write syncconf file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("write syncconf file: %w", err)
	}

	_, err = s.cmd.Run(ctx, "awg", "syncconf", s.interfaceName, tmpFile.Name())
	return err
}

// ensureIptablesRule appends the rule, a chain with its match and target, to
// the table unless iptables -C finds it there, and reports whether it was
// appended.
func (s *awgService) ensureIptablesRule(ctx context.Context, table string, rule ...string) bool {
	if _, err := s.cmd.Run(ctx, "iptables", slices.Concat([]string{"-t", table, "-C"}, rule)...); err == nil {
		return false
	}
	if _, err := s.cmd.Run(ctx, "iptables", slices.Concat([]string{"-t", table, "-A"}, rule)...); err != nil {
		slog.Error("Failed to add iptables rule", "table", table, "rule", strings.Join(rule, " "), "error", err)
		return false
	}
	return true
}

// defaultRouteInterface returns the device of the first route in the output
// of ip route show default.
func defaultRouteInterface(routes []byte) string {
	for line := range strings.Lines(string(routes)) {
		fields := strings.Fields(line)
		if i := slices.Index(fields, "dev"); i >= 0 && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

// FormatBytes formats a byte count for the dashboard with binary units, such
// as "1.50 MB".
func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}

// FormatHandshake describes how long before now the handshake at the Unix
// time timestamp happened, such as "5m ago". A zero timestamp means the peer
// has never connected.
func FormatHandshake(timestamp int64, now time.Time) string {
	if timestamp == 0 {
		return "Offline"
	}
	diff := now.Unix() - timestamp
	switch {
	case diff < 60:
		return "Just now"
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%dh ago", diff/3600)
	default:
		return fmt.Sprintf("%dd ago", diff/86400)
	}
}

func (s *awgService) StartMetricsCollector(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(metricsInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.recordTraffic(ctx)
			}
		}
	}()
}

func (s *awgService) recordTraffic(ctx context.Context) {
	status, err := s.GetSystemStatus(ctx)
	if err != nil {
		slog.Error("Metrics collector failed to get status", "error", err)
		return
	}

	if err := s.repo.RecordTraffic(ctx, status.TotalRxBytes, status.TotalTxBytes); err != nil {
		slog.Error("Metrics collector failed to record traffic", "error", err)
	}

	// The status has totals only; the peers carry their own rx and tx.
	clients, err := s.GetClients(ctx)
	if err != nil {
		slog.Error("Metrics collector failed to get clients for peer traffic", "error", err)
		return
	}

	if err := s.repo.RecordPeerTraffic(ctx, s.now(), clients); err != nil {
		slog.Error("Metrics collector failed to record peer traffic", "error", err)
	}
}
