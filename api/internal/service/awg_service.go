package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/repository"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/pkg/crypto"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/pkg/qrcode"
)

type AWGService interface {
	GetServerConfig() (*domain.ServerConfig, error)
	UpdateServerConfig(cfg *domain.ServerConfig) error
	GetClients() ([]domain.Peer, error)
	CreateClient(req domain.CreateClientRequest) (*domain.ClientResponse, error)
	DeleteClient(name string) error
	GetClientConfigText(name string) (string, error)
	GetClientQRCode(name string) (string, error)
	GetSystemStatus() (*domain.SystemStatus, error)
	StartMetricsCollector()
}

type peerRuntimeStats struct {
	LatestHandshake int64
	RxBytes         int64
	TxBytes         int64
}

type awgService struct {
	repo          repository.ConfigRepository
	ipService     IPService
	clientsDir    string
	interfaceName string

	statsCache     map[string]peerRuntimeStats
	statsCacheTime time.Time
	statsMu        sync.RWMutex
}

func NewAWGService(repo repository.ConfigRepository, ipService IPService, clientsDir string, interfaceName string) AWGService {
	_ = os.MkdirAll(clientsDir, 0755)
	if interfaceName == "" {
		interfaceName = "awg0"
	}
	svc := &awgService{
		repo:          repo,
		ipService:     ipService,
		clientsDir:    clientsDir,
		interfaceName: interfaceName,
		statsCache:    make(map[string]peerRuntimeStats),
	}
	svc.syncRuntime()
	return svc
}

func (s *awgService) GetServerConfig() (*domain.ServerConfig, error) {
	return s.repo.LoadServerConfig()
}

func (s *awgService) UpdateServerConfig(newCfg *domain.ServerConfig) error {
	current, err := s.repo.LoadServerConfig()
	if err != nil {
		return err
	}
	newCfg.Peers = current.Peers
	if err := s.repo.SaveServerConfig(newCfg); err != nil {
		return err
	}
	s.syncRuntime()
	s.regenerateClientConfigs(newCfg)
	return nil
}

func (s *awgService) regenerateClientConfigs(cfg *domain.ServerConfig) {
	files, err := os.ReadDir(s.clientsDir)
	if err != nil {
		return
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".conf") {
			name := strings.TrimSuffix(f.Name(), ".conf")

			content, err := os.ReadFile(filepath.Join(s.clientsDir, f.Name()))
			if err != nil {
				continue
			}

			privKey := ""
			for _, line := range strings.Split(string(content), "\n") {
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

			var clientIP string
			var psk string
			for _, p := range cfg.Peers {
				if p.Name == name {
					clientIP = p.IP
					psk = p.PresharedKey
					break
				}
			}

			if clientIP != "" {
				newConf := s.buildClientConfig(cfg, privKey, psk, clientIP)
				_ = os.WriteFile(filepath.Join(s.clientsDir, f.Name()), []byte(newConf), 0600)
			}
		}
	}
}

func (s *awgService) GetClients() ([]domain.Peer, error) {
	cfg, err := s.repo.LoadServerConfig()
	if err != nil {
		return nil, err
	}

	runtimeStats := s.fetchPeerRuntimeStats()
	now := time.Now().Unix()

	for i := range cfg.Peers {
		peer := &cfg.Peers[i]
		if stat, ok := runtimeStats[peer.PublicKey]; ok {
			peer.LatestHandshake = stat.LatestHandshake
			peer.RxBytes = stat.RxBytes
			peer.TxBytes = stat.TxBytes
		}

		peer.RxFormatted = FormatBytes(peer.RxBytes)
		peer.TxFormatted = FormatBytes(peer.TxBytes)
		peer.LatestHandshakeText = FormatHandshake(peer.LatestHandshake)
		// Peer is considered online if last handshake occurred within the last 3 minutes (180 seconds)
		peer.IsOnline = peer.LatestHandshake > 0 && (now-peer.LatestHandshake) <= 180
	}

	SortPeersByIP(cfg.Peers)
	return cfg.Peers, nil
}

func (s *awgService) CreateClient(req domain.CreateClientRequest) (*domain.ClientResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("client name cannot be empty")
	}

	cfg, err := s.repo.LoadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	// 1. IP Allocation
	clientIP := req.IP
	if clientIP == "" {
		subnetPrefix := ExtractSubnetPrefix(cfg.Address)
		allocated, err := s.ipService.AllocateNextIP(subnetPrefix, cfg.Peers)
		if err != nil {
			return nil, err
		}
		clientIP = allocated
	}

	// 2. Keys Generation
	clientKeyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client keypair: %w", err)
	}

	psk, err := crypto.GeneratePresharedKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate preshared key: %w", err)
	}

	// 3. Add Peer to Server Repo
	peer := domain.Peer{
		Name:         name,
		PublicKey:    clientKeyPair.PublicKey,
		PresharedKey: psk,
		AllowedIPs:   fmt.Sprintf("%s/32", clientIP),
		IP:           clientIP,
	}

	if err := s.repo.AddPeer(peer); err != nil {
		return nil, err
	}

	// 4. Generate Client Config Text
	clientConfigText := s.buildClientConfig(cfg, clientKeyPair.PrivateKey, psk, clientIP)

	// Save client .conf file locally
	clientFilePath := filepath.Join(s.clientsDir, fmt.Sprintf("%s.conf", name))
	_ = os.WriteFile(clientFilePath, []byte(clientConfigText), 0600)

	// 5. Generate QR Code Data URL
	qrDataURL, err := qrcode.GeneratePNGDataURL(clientConfigText, 320)
	if err != nil {
		slog.Warn("Failed to generate QR code data URL", "error", err)
	}

	// 6. Sync AWG Interface live in memory
	s.syncRuntime()

	return &domain.ClientResponse{
		Name:       name,
		IP:         clientIP,
		ConfigText: clientConfigText,
		QRCodeSVG:  qrDataURL,
	}, nil
}

func (s *awgService) DeleteClient(name string) error {
	peer, err := s.repo.GetPeerByName(name)
	if err != nil {
		return err
	}

	if err := s.repo.DeletePeer(name); err != nil {
		return err
	}

	clientFilePath := filepath.Join(s.clientsDir, fmt.Sprintf("%s.conf", name))
	_ = os.Remove(clientFilePath)

	s.syncRuntime()
	_ = peer
	return nil
}

func (s *awgService) GetClientConfigText(name string) (string, error) {
	clientFilePath := filepath.Join(s.clientsDir, fmt.Sprintf("%s.conf", name))
	if content, err := os.ReadFile(clientFilePath); err == nil {
		return string(content), nil
	}

	peer, err := s.repo.GetPeerByName(name)
	if err != nil {
		return "", err
	}

	cfg, err := s.repo.LoadServerConfig()
	if err != nil {
		return "", err
	}

	return s.buildClientConfig(cfg, "<client-private-key-hidden>", peer.PresharedKey, peer.IP), nil
}

func (s *awgService) GetClientQRCode(name string) (string, error) {
	configText, err := s.GetClientConfigText(name)
	if err != nil {
		return "", err
	}

	return qrcode.GeneratePNGDataURL(configText, 360)
}

func (s *awgService) GetSystemStatus() (*domain.SystemStatus, error) {
	clients, err := s.GetClients()
	if err != nil {
		return nil, err
	}

	cfg, err := s.repo.LoadServerConfig()
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

	history, _ := s.repo.GetTrafficHistory(24)
	if history == nil {
		history = []domain.TrafficPoint{}
	}

	peerHistory, _ := s.repo.GetTopPeersTrafficHistory(288, 10) // 288 points = 24 hours of 5-min intervals
	if peerHistory == nil {
		peerHistory = make(map[string][]domain.PeerTrafficPoint)
	}

	mode := "Unknown"
	out, err2 := exec.Command("ip", "-d", "link", "show", s.interfaceName).Output()
	if err2 == nil {
		outStr := string(out)
		if strings.Contains(outStr, "amneziawg") || strings.Contains(outStr, "wireguard") {
			mode = "Kernel"
		} else if strings.Contains(outStr, "tun") {
			mode = "Userspace"
		}
	}

	status := &domain.SystemStatus{
		Interface:          s.interfaceName,
		IsRunning:          true,
		Mode:               mode,
		ActivePeers:        len(cfg.Peers),
		OnlinePeers:        onlineCount,
		TotalPeers:         len(cfg.Peers),
		Endpoint:           cfg.Endpoint,
		VPNSubnet:          cfg.Address,
		PublicKey:          cfg.PublicKey,
		ListenPort:         cfg.ListenPort,
		TotalRxBytes:       totalRx,
		TotalTxBytes:       totalTx,
		TotalRxFormatted:   FormatBytes(totalRx),
		TotalTxFormatted:   FormatBytes(totalTx),
		TrafficHistory:     history,
		PeerTrafficHistory: peerHistory,
		Location:           autoDetectLocation(),
	}

	// If endpoint is not set by user, auto-detect public IP dynamically
	endpointVal := strings.TrimSpace(cfg.Endpoint)
	port := os.Getenv("AWG_PORT")
	if port == "" {
		port = "51820"
	}
	if endpointVal == "" {
		if publicIP := autoDetectPublicIP(); publicIP != "" {
			status.Endpoint = fmt.Sprintf("%s:%s", publicIP, port)
			status.AutoEndpoint = true
		} else {
			status.Endpoint = ""
		}
	} else {
		if !strings.Contains(endpointVal, ":") {
			status.Endpoint = fmt.Sprintf("%s:%s", endpointVal, port)
		} else {
			status.Endpoint = endpointVal
		}
	}

	if _, err := exec.LookPath("awg"); err != nil {
		status.IsRunning = false
	}

	return status, nil
}

// createIPv4HTTPClient creates an HTTP client that explicitly uses IPv4 (tcp4)
// to prevent IPv6 timeout delays in Docker environments.
func createIPv4HTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: timeout,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		DisableKeepAlives: true,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

var (
	cachedPublicIP     string
	cachedPublicIPTime time.Time
	publicIPMutex      sync.RWMutex

	cachedLocation     *domain.ServerLocation
	cachedLocationTime time.Time
	locationMutex      sync.RWMutex
)

// autoDetectPublicIP attempts to determine the server's public IP address
// dynamically by querying external IPv4 HTTP/HTTPS IP services.
func autoDetectPublicIP() string {
	if envIP := strings.TrimSpace(os.Getenv("PUBLIC_IP")); envIP != "" {
		if ip := net.ParseIP(envIP); ip != nil && ip.To4() != nil {
			slog.Info("Using public IP from PUBLIC_IP environment variable", "ip", envIP)
			return envIP
		}
	}

	publicIPMutex.RLock()
	if time.Since(cachedPublicIPTime) < 10*time.Minute && cachedPublicIP != "" {
		ip := cachedPublicIP
		publicIPMutex.RUnlock()
		return ip
	}
	publicIPMutex.RUnlock()

	endpoints := []string{
		"http://api.ipify.org",
		"http://checkip.amazonaws.com",
		"http://icanhazip.com",
		"http://ifconfig.me/ip",
		"http://v4.ident.me",
		"http://ip-api.com/line?fields=query",
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
	}

	client := createIPv4HTTPClient(3 * time.Second)

	for _, url := range endpoints {
		resp, err := client.Get(url)
		if err != nil {
			slog.Debug("autoDetectPublicIP request failed", "url", url, "error", err)
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
		_ = resp.Body.Close()
		if err != nil {
			slog.Debug("autoDetectPublicIP read body failed", "url", url, "error", err)
			continue
		}

		raw := strings.TrimSpace(string(body))
		for _, r := range []string{"{", "}", "\"", "ip", ":", " ", "\n", "\r"} {
			raw = strings.ReplaceAll(raw, r, "")
		}
		raw = strings.TrimSpace(raw)

		if parsedIP := net.ParseIP(raw); parsedIP != nil && parsedIP.To4() != nil && !parsedIP.IsLoopback() {
			ipStr := parsedIP.String()
			publicIPMutex.Lock()
			cachedPublicIP = ipStr
			cachedPublicIPTime = time.Now()
			publicIPMutex.Unlock()
			slog.Info("Auto-detected server public IP", "ip", ipStr, "source", url)
			return ipStr
		}
	}

	slog.Warn("Failed to auto-detect public IPv4 address from all external services")
	return ""
}

func autoDetectLocation() *domain.ServerLocation {
	locationMutex.RLock()
	if time.Since(cachedLocationTime) < 24*time.Hour && cachedLocation != nil {
		loc := cachedLocation
		locationMutex.RUnlock()
		return loc
	}
	locationMutex.RUnlock()

	client := createIPv4HTTPClient(3 * time.Second)
	resp, err := client.Get("http://ip-api.com/json/")
	if err != nil {
		slog.Debug("autoDetectLocation request failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.CountryCode != "" {
		loc := &domain.ServerLocation{
			Country:     result.Country,
			CountryCode: result.CountryCode,
		}
		locationMutex.Lock()
		cachedLocation = loc
		cachedLocationTime = time.Now()
		locationMutex.Unlock()
		return loc
	}
	return nil
}

func (s *awgService) fetchPeerRuntimeStats() map[string]peerRuntimeStats {
	s.statsMu.RLock()
	if time.Since(s.statsCacheTime) < 3*time.Second && s.statsCache != nil {
		stats := s.statsCache
		s.statsMu.RUnlock()
		return stats
	}
	s.statsMu.RUnlock()

	stats := make(map[string]peerRuntimeStats)

	if _, err := exec.LookPath("awg"); err != nil {
		return stats
	}

	cmd := exec.Command("awg", "show", s.interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("Failed to execute awg show dump", "error", err)
		return stats
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	// First line is interface details, subsequent lines are peers
	isFirstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if isFirstLine {
			isFirstLine = false
			continue
		}

		fields := strings.Fields(line)
		// Dump format: public_key preshared_key endpoint allowed_ips latest_handshake rx_bytes tx_bytes persistent_keepalive
		if len(fields) >= 7 {
			pubKey := fields[0]
			handshake, _ := strconv.ParseInt(fields[4], 10, 64)
			rx, _ := strconv.ParseInt(fields[5], 10, 64)
			tx, _ := strconv.ParseInt(fields[6], 10, 64)

			stats[pubKey] = peerRuntimeStats{
				LatestHandshake: handshake,
				RxBytes:         rx,
				TxBytes:         tx,
			}
		}
	}

	s.statsMu.Lock()
	s.statsCache = stats
	s.statsCacheTime = time.Now()
	s.statsMu.Unlock()

	return stats
}

func (s *awgService) buildClientConfig(serverCfg *domain.ServerConfig, clientPrivKey string, psk string, clientIP string) string {
	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("Address = %s/32\n", clientIP))
	if serverCfg.DNS != "" {
		sb.WriteString(fmt.Sprintf("DNS = %s\n", serverCfg.DNS))
	}
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", clientPrivKey))

	if serverCfg.Obfuscation.Jc != "" {
		sb.WriteString(fmt.Sprintf("Jc = %s\n", serverCfg.Obfuscation.Jc))
		sb.WriteString(fmt.Sprintf("Jmin = %s\n", serverCfg.Obfuscation.Jmin))
		sb.WriteString(fmt.Sprintf("Jmax = %s\n", serverCfg.Obfuscation.Jmax))
		sb.WriteString(fmt.Sprintf("S1 = %s\n", serverCfg.Obfuscation.S1))
		sb.WriteString(fmt.Sprintf("S2 = %s\n", serverCfg.Obfuscation.S2))
		sb.WriteString(fmt.Sprintf("S3 = %s\n", serverCfg.Obfuscation.S3))
		sb.WriteString(fmt.Sprintf("S4 = %s\n", serverCfg.Obfuscation.S4))
		sb.WriteString(fmt.Sprintf("H1 = %s\n", serverCfg.Obfuscation.H1))
		sb.WriteString(fmt.Sprintf("H2 = %s\n", serverCfg.Obfuscation.H2))
		sb.WriteString(fmt.Sprintf("H3 = %s\n", serverCfg.Obfuscation.H3))
		sb.WriteString(fmt.Sprintf("H4 = %s\n", serverCfg.Obfuscation.H4))
		if serverCfg.Obfuscation.I1 != "" {
			sb.WriteString(fmt.Sprintf("I1 = %s\n", serverCfg.Obfuscation.I1))
		}
		if serverCfg.Obfuscation.I2 != "" {
			sb.WriteString(fmt.Sprintf("I2 = %s\n", serverCfg.Obfuscation.I2))
		}
		if serverCfg.Obfuscation.I3 != "" {
			sb.WriteString(fmt.Sprintf("I3 = %s\n", serverCfg.Obfuscation.I3))
		}
		if serverCfg.Obfuscation.I4 != "" {
			sb.WriteString(fmt.Sprintf("I4 = %s\n", serverCfg.Obfuscation.I4))
		}
		if serverCfg.Obfuscation.I5 != "" {
			sb.WriteString(fmt.Sprintf("I5 = %s\n", serverCfg.Obfuscation.I5))
		}
	}

	sb.WriteString("\n[Peer]\n")
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", serverCfg.PublicKey))
	if psk != "" {
		sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", psk))
	}

	allowed := strings.TrimSpace(serverCfg.LANAllowed)
	if allowed == "" || allowed == "0.0.0.0/0, ::/0" || allowed == "0.0.0.0/0" {
		allowed = "0.0.0.0/0, ::/0"
	} else {
		clientSubnet := fmt.Sprintf("%s/32", clientIP)
		if !strings.Contains(allowed, clientSubnet) {
			allowed = fmt.Sprintf("%s, %s", clientSubnet, allowed)
		}
	}
	sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", allowed))

	endpoint := strings.TrimSpace(serverCfg.Endpoint)
	port := os.Getenv("AWG_PORT")
	if port == "" {
		port = "51820"
	}
	if endpoint == "" {
		if publicIP := autoDetectPublicIP(); publicIP != "" {
			endpoint = fmt.Sprintf("%s:%s", publicIP, port)
		}
	} else {
		if !strings.Contains(endpoint, ":") {
			endpoint = fmt.Sprintf("%s:%s", endpoint, port)
		}
	}
	sb.WriteString(fmt.Sprintf("Endpoint = %s\n", endpoint))
	if serverCfg.PersistentKeepalive != "" {
		sb.WriteString(fmt.Sprintf("PersistentKeepalive = %s\n", serverCfg.PersistentKeepalive))
	}

	return sb.String()
}

func (s *awgService) syncPeerAdd(peer domain.Peer) {
	if _, err := exec.LookPath("awg"); err != nil {
		slog.Info("awg CLI tool not found in PATH — skipping live peer sync", "peer", peer.Name)
		return
	}

	args := []string{"set", s.interfaceName, "peer", peer.PublicKey, "allowed-ips", peer.AllowedIPs}
	if peer.PresharedKey != "" {
		tmpFile, err := os.CreateTemp("", "psk-*")
		if err == nil {
			_, _ = tmpFile.WriteString(peer.PresharedKey)
			_ = tmpFile.Close()
			defer os.Remove(tmpFile.Name())
			args = append(args, "preshared-key", tmpFile.Name())
		}
	}

	cmd := exec.Command("awg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to add peer in live runtime", "peer", peer.Name, "error", err, "output", string(output))
	} else {
		slog.Info("Successfully added/updated peer in live runtime", "peer", peer.Name, "ip", peer.IP)
	}
}

func (s *awgService) syncPeerRemove(pubKey string, name string) {
	if _, err := exec.LookPath("awg"); err != nil {
		return
	}

	cmd := exec.Command("awg", "set", s.interfaceName, "peer", pubKey, "remove")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to remove peer in live runtime", "peer", name, "error", err, "output", string(output))
	} else {
		slog.Info("Successfully removed peer from live runtime", "peer", name)
	}
}

func (s *awgService) syncRuntime() {
	if _, err := exec.LookPath("awg"); err != nil {
		slog.Info("awg CLI tool not found in PATH — skipping live runtime sync")
		return
	}

	cfg, err := s.repo.LoadServerConfig()
	if err != nil {
		slog.Error("Failed to load server config for syncRuntime", "error", err)
		return
	}

	// 1. Ensure interface awg0 exists
	if err := exec.Command("ip", "link", "show", s.interfaceName).Run(); err != nil {
		slog.Info("Attempting to create kernel interface", "interface", s.interfaceName)
		err := exec.Command("ip", "link", "add", "dev", s.interfaceName, "type", "amneziawg").Run()
		if err != nil {
			slog.Info("Kernel module not found or failed, falling back to amneziawg-go", "interface", s.interfaceName)
			out, err := exec.Command("amneziawg-go", s.interfaceName).CombinedOutput()
			if err != nil {
				slog.Error("Failed to start amneziawg-go", "interface", s.interfaceName, "error", err, "output", string(out))
			}
		}
	}

	// 2. Configure IP address on the interface
	if cfg.Address != "" {
		_ = exec.Command("ip", "addr", "flush", "dev", s.interfaceName).Run()
		if err := exec.Command("ip", "addr", "add", cfg.Address, "dev", s.interfaceName).Run(); err != nil {
			slog.Error("Failed to assign IP address to interface", "interface", s.interfaceName, "address", cfg.Address, "error", err)
		}
		_ = exec.Command("ip", "link", "set", "dev", s.interfaceName, "up").Run()
	}

	// 3. Build stripped configuration for awg syncconf
	tmpFile, err := os.CreateTemp("", "awg-sync-*.conf")
	if err != nil {
		slog.Error("Failed to create temporary file for awg syncconf", "error", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey))
	if cfg.ListenPort != "" {
		sb.WriteString(fmt.Sprintf("ListenPort = %s\n", cfg.ListenPort))
	}
	if cfg.Obfuscation.Jc != "" {
		sb.WriteString(fmt.Sprintf("Jc = %s\n", cfg.Obfuscation.Jc))
		sb.WriteString(fmt.Sprintf("Jmin = %s\n", cfg.Obfuscation.Jmin))
		sb.WriteString(fmt.Sprintf("Jmax = %s\n", cfg.Obfuscation.Jmax))
		sb.WriteString(fmt.Sprintf("S1 = %s\n", cfg.Obfuscation.S1))
		sb.WriteString(fmt.Sprintf("S2 = %s\n", cfg.Obfuscation.S2))
		sb.WriteString(fmt.Sprintf("S3 = %s\n", cfg.Obfuscation.S3))
		sb.WriteString(fmt.Sprintf("S4 = %s\n", cfg.Obfuscation.S4))
		sb.WriteString(fmt.Sprintf("H1 = %s\n", cfg.Obfuscation.H1))
		sb.WriteString(fmt.Sprintf("H2 = %s\n", cfg.Obfuscation.H2))
		sb.WriteString(fmt.Sprintf("H3 = %s\n", cfg.Obfuscation.H3))
		sb.WriteString(fmt.Sprintf("H4 = %s\n", cfg.Obfuscation.H4))
		if cfg.Obfuscation.I1 != "" {
			sb.WriteString(fmt.Sprintf("I1 = %s\n", cfg.Obfuscation.I1))
		}
		if cfg.Obfuscation.I2 != "" {
			sb.WriteString(fmt.Sprintf("I2 = %s\n", cfg.Obfuscation.I2))
		}
		if cfg.Obfuscation.I3 != "" {
			sb.WriteString(fmt.Sprintf("I3 = %s\n", cfg.Obfuscation.I3))
		}
		if cfg.Obfuscation.I4 != "" {
			sb.WriteString(fmt.Sprintf("I4 = %s\n", cfg.Obfuscation.I4))
		}
		if cfg.Obfuscation.I5 != "" {
			sb.WriteString(fmt.Sprintf("I5 = %s\n", cfg.Obfuscation.I5))
		}
	}

	for _, peer := range cfg.Peers {
		sb.WriteString("\n[Peer]\n")
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		if peer.PresharedKey != "" {
			sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
	}

	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		_ = tmpFile.Close()
		slog.Error("Failed to write to temp config file for awg syncconf", "error", err)
		return
	}
	_ = tmpFile.Close()

	// 4. Run awg syncconf
	cmd := exec.Command("awg", "syncconf", s.interfaceName, tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed awg syncconf", "interface", s.interfaceName, "error", err, "output", string(output))
	} else {
		slog.Info("Successfully synced awg interface runtime via syncconf", "interface", s.interfaceName)
	}

	// 5. Ensure iptables NAT & Forwarding rules are applied inside container
	_ = exec.Command("iptables", "-C", "FORWARD", "-i", s.interfaceName, "-j", "ACCEPT").Run()
	if err := exec.Command("iptables", "-C", "FORWARD", "-i", s.interfaceName, "-j", "ACCEPT").Run(); err != nil {
		_ = exec.Command("iptables", "-A", "FORWARD", "-i", s.interfaceName, "-j", "ACCEPT").Run()
	}
	if err := exec.Command("iptables", "-C", "FORWARD", "-o", s.interfaceName, "-j", "ACCEPT").Run(); err != nil {
		_ = exec.Command("iptables", "-A", "FORWARD", "-o", s.interfaceName, "-j", "ACCEPT").Run()
	}

	out, _ := exec.Command("sh", "-c", "ip route show default | awk '/default/ {print $5}' | head -n1").Output()
	defaultIf := strings.TrimSpace(string(out))
	if defaultIf != "" {
		if err := exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING", "-o", defaultIf, "-j", "MASQUERADE").Run(); err != nil {
			_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", defaultIf, "-j", "MASQUERADE").Run()
			slog.Info("Applied NAT MASQUERADE rule for AWG interface", "out_interface", defaultIf)
		}
	}
}

// FormatBytes converts raw bytes to human readable string (KB, MB, GB, TB)
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

// FormatHandshake converts epoch timestamp to relative time string
func FormatHandshake(timestamp int64) string {
	if timestamp == 0 {
		return "Offline"
	}
	now := time.Now().Unix()
	diff := now - timestamp
	if diff < 0 {
		return "Just now"
	}
	if diff < 60 {
		return "Just now"
	}
	if diff < 3600 {
		return fmt.Sprintf("%dm ago", diff/60)
	}
	if diff < 86400 {
		return fmt.Sprintf("%dh ago", diff/3600)
	}
	return fmt.Sprintf("%dd ago", diff/86400)
}

func (s *awgService) StartMetricsCollector() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C
			status, err := s.GetSystemStatus()
			if err != nil {
				slog.Error("Metrics collector failed to get status", "error", err)
				continue
			}

			if err := s.repo.RecordTraffic(status.TotalRxBytes, status.TotalTxBytes); err != nil {
				slog.Error("Metrics collector failed to record traffic", "error", err)
			}

			// We need to fetch clients directly to have their latest rx/tx
			clients, err := s.GetClients()
			if err != nil {
				slog.Error("Metrics collector failed to get clients for peer traffic", "error", err)
				continue
			}

			if err := s.repo.RecordPeerTraffic(time.Now(), clients); err != nil {
				slog.Error("Metrics collector failed to record peer traffic", "error", err)
			}
		}
	}()
}
