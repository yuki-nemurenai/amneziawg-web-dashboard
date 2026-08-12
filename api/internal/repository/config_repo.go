package repository

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/pkg/crypto"
)

type ConfigRepository interface {
	LoadServerConfig() (*domain.ServerConfig, error)
	SaveServerConfig(cfg *domain.ServerConfig) error
	AddPeer(peer domain.Peer) error
	DeletePeer(name string) error
	GetPeerByName(name string) (*domain.Peer, error)
	RecordTraffic(rxBytes, txBytes int64) error
	GetTrafficHistory(limit int) ([]domain.TrafficPoint, error)
	RecordPeerTraffic(timestamp time.Time, peers []domain.Peer) error
	GetTopPeersTrafficHistory(limit int, topN int) (map[string][]domain.PeerTrafficPoint, error)
}

type fileConfigRepo struct {
	configPath string
	mu         sync.RWMutex
}

func NewFileConfigRepo(configPath string) ConfigRepository {
	return &fileConfigRepo{
		configPath: configPath,
	}
}

func (r *fileConfigRepo) LoadServerConfig() (*domain.ServerConfig, error) {
	r.mu.RLock()
	file, err := os.Open(r.configPath)
	if err != nil {
		r.mu.RUnlock()
		if os.IsNotExist(err) {
			defaultCfg := r.createDefaultServerConfig()
			_ = r.SaveServerConfig(defaultCfg)
			return defaultCfg, nil
		}
		return nil, fmt.Errorf("failed to open config file %s: %w", r.configPath, err)
	}
	defer file.Close()
	defer r.mu.RUnlock()

	cfg := &domain.ServerConfig{
		DNS:                 "1.1.1.1, 1.0.0.1",
		PersistentKeepalive: "25",
	}

	scanner := bufio.NewScanner(file)
	currentSection := ""
	var currentPeer *domain.Peer
	lastComment := ""

	peerIPRegex := regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			commentContent := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if parts := strings.SplitN(commentContent, "=", 2); len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if k == "Endpoint" {
					cfg.Endpoint = v
				} else if k == "LANAllowed" {
					cfg.LANAllowed = v
				}
			}
			lastComment = commentContent
			continue
		}

		if line == "[Interface]" {
			currentSection = "interface"
			continue
		} else if line == "[Peer]" {
			currentSection = "peer"
			if currentPeer != nil {
				cfg.Peers = append(cfg.Peers, *currentPeer)
			}
			currentPeer = &domain.Peer{
				Name: lastComment,
			}
			lastComment = ""
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if currentSection == "interface" {
			switch key {
			case "PrivateKey":
				cfg.PrivateKey = val
				pubKey, _ := crypto.PublicFromPrivate(val)
				cfg.PublicKey = pubKey
			case "Address":
				cfg.Address = val
			case "ListenPort":
				cfg.ListenPort = val
			case "PostUp":
				cfg.PostUp = val
			case "PostDown":
				cfg.PostDown = val
			case "Jc":
				cfg.Obfuscation.Jc = val
			case "Jmin":
				cfg.Obfuscation.Jmin = val
			case "Jmax":
				cfg.Obfuscation.Jmax = val
			case "S1":
				cfg.Obfuscation.S1 = val
			case "S2":
				cfg.Obfuscation.S2 = val
			case "S3":
				cfg.Obfuscation.S3 = val
			case "S4":
				cfg.Obfuscation.S4 = val
			case "H1":
				cfg.Obfuscation.H1 = val
			case "H2":
				cfg.Obfuscation.H2 = val
			case "H3":
				cfg.Obfuscation.H3 = val
			case "H4":
				cfg.Obfuscation.H4 = val
			case "I1":
				cfg.Obfuscation.I1 = val
			case "I2":
				cfg.Obfuscation.I2 = val
			case "I3":
				cfg.Obfuscation.I3 = val
			case "I4":
				cfg.Obfuscation.I4 = val
			case "Endpoint":
				cfg.Endpoint = val
			case "LANAllowed":
				cfg.LANAllowed = val
			case "I5":
				cfg.Obfuscation.I5 = val
			}
		} else if currentSection == "peer" && currentPeer != nil {
			switch key {
			case "PublicKey":
				currentPeer.PublicKey = val
			case "PresharedKey":
				currentPeer.PresharedKey = val
			case "AllowedIPs":
				currentPeer.AllowedIPs = val
				match := peerIPRegex.FindString(val)
				if match != "" {
					currentPeer.IP = match
				}
			}
		}
	}

	if currentPeer != nil {
		cfg.Peers = append(cfg.Peers, *currentPeer)
	}

	// Fallback peer names if missing
	for i := range cfg.Peers {
		if cfg.Peers[i].Name == "" {
			if cfg.Peers[i].IP != "" {
				cfg.Peers[i].Name = fmt.Sprintf("user-%s", strings.ReplaceAll(cfg.Peers[i].IP, ".", "-"))
			} else {
				cfg.Peers[i].Name = fmt.Sprintf("user-%d", i+1)
			}
		}
	}

	if cfg.LANAllowed == "" {
		cfg.LANAllowed = "0.0.0.0/0, ::/0"
	}

	return cfg, scanner.Err()
}

func (r *fileConfigRepo) RecordTraffic(rxBytes, txBytes int64) error {
	// Stub for file repo, not implemented
	return nil
}

func (r *fileConfigRepo) GetTrafficHistory(limit int) ([]domain.TrafficPoint, error) {
	// Stub for file repo, not implemented
	return nil, nil
}

func (r *fileConfigRepo) RecordPeerTraffic(timestamp time.Time, peers []domain.Peer) error {
	return nil
}

func (r *fileConfigRepo) GetTopPeersTrafficHistory(limit int, topN int) (map[string][]domain.PeerTrafficPoint, error) {
	return nil, nil
}

func (r *fileConfigRepo) SaveServerConfig(cfg *domain.ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var builder strings.Builder
	builder.WriteString("[Interface]\n")
	builder.WriteString(fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey))
	builder.WriteString(fmt.Sprintf("Address = %s\n", cfg.Address))
	builder.WriteString(fmt.Sprintf("ListenPort = %s\n", cfg.ListenPort))
	if cfg.Endpoint != "" {
		builder.WriteString(fmt.Sprintf("# Endpoint = %s\n", cfg.Endpoint))
	}
	if cfg.LANAllowed != "" {
		builder.WriteString(fmt.Sprintf("# LANAllowed = %s\n", cfg.LANAllowed))
	}

	if cfg.Obfuscation.Jc != "" {
		builder.WriteString(fmt.Sprintf("Jc = %s\n", cfg.Obfuscation.Jc))
		builder.WriteString(fmt.Sprintf("Jmin = %s\n", cfg.Obfuscation.Jmin))
		builder.WriteString(fmt.Sprintf("Jmax = %s\n", cfg.Obfuscation.Jmax))
		builder.WriteString(fmt.Sprintf("S1 = %s\n", cfg.Obfuscation.S1))
		builder.WriteString(fmt.Sprintf("S2 = %s\n", cfg.Obfuscation.S2))
		builder.WriteString(fmt.Sprintf("S3 = %s\n", cfg.Obfuscation.S3))
		builder.WriteString(fmt.Sprintf("S4 = %s\n", cfg.Obfuscation.S4))
		builder.WriteString(fmt.Sprintf("H1 = %s\n", cfg.Obfuscation.H1))
		builder.WriteString(fmt.Sprintf("H2 = %s\n", cfg.Obfuscation.H2))
		builder.WriteString(fmt.Sprintf("H3 = %s\n", cfg.Obfuscation.H3))
		builder.WriteString(fmt.Sprintf("H4 = %s\n", cfg.Obfuscation.H4))
		if cfg.Obfuscation.I1 != "" {
			builder.WriteString(fmt.Sprintf("I1 = %s\n", cfg.Obfuscation.I1))
		}
		if cfg.Obfuscation.I2 != "" {
			builder.WriteString(fmt.Sprintf("I2 = %s\n", cfg.Obfuscation.I2))
		}
		if cfg.Obfuscation.I3 != "" {
			builder.WriteString(fmt.Sprintf("I3 = %s\n", cfg.Obfuscation.I3))
		}
		if cfg.Obfuscation.I4 != "" {
			builder.WriteString(fmt.Sprintf("I4 = %s\n", cfg.Obfuscation.I4))
		}
		if cfg.Obfuscation.I5 != "" {
			builder.WriteString(fmt.Sprintf("I5 = %s\n", cfg.Obfuscation.I5))
		}
	}

	if cfg.PostUp != "" {
		builder.WriteString(fmt.Sprintf("\nPostUp = %s\n", cfg.PostUp))
	}
	if cfg.PostDown != "" {
		builder.WriteString(fmt.Sprintf("PostDown = %s\n", cfg.PostDown))
	}

	for _, peer := range cfg.Peers {
		builder.WriteString("\n")
		if peer.Name != "" {
			builder.WriteString(fmt.Sprintf("# %s\n", peer.Name))
		}
		builder.WriteString("[Peer]\n")
		builder.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		if peer.PresharedKey != "" {
			builder.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}
		builder.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
	}

	return os.WriteFile(r.configPath, []byte(builder.String()), 0600)
}

func (r *fileConfigRepo) AddPeer(peer domain.Peer) error {
	cfg, err := r.LoadServerConfig()
	if err != nil {
		return err
	}

	for _, existing := range cfg.Peers {
		if strings.EqualFold(existing.Name, peer.Name) {
			return fmt.Errorf("peer with name '%s' already exists", peer.Name)
		}
		if existing.IP == peer.IP && peer.IP != "" {
			return fmt.Errorf("peer with IP '%s' already exists", peer.IP)
		}
	}

	cfg.Peers = append(cfg.Peers, peer)
	return r.SaveServerConfig(cfg)
}

func (r *fileConfigRepo) DeletePeer(name string) error {
	cfg, err := r.LoadServerConfig()
	if err != nil {
		return err
	}

	newPeers := make([]domain.Peer, 0, len(cfg.Peers))
	found := false
	for _, p := range cfg.Peers {
		if strings.EqualFold(p.Name, name) {
			found = true
			continue
		}
		newPeers = append(newPeers, p)
	}

	if !found {
		return fmt.Errorf("peer '%s' not found", name)
	}

	cfg.Peers = newPeers
	return r.SaveServerConfig(cfg)
}

func (r *fileConfigRepo) GetPeerByName(name string) (*domain.Peer, error) {
	cfg, err := r.LoadServerConfig()
	if err != nil {
		return nil, err
	}

	for _, p := range cfg.Peers {
		if strings.EqualFold(p.Name, name) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("peer '%s' not found", name)
}

func (r *fileConfigRepo) createDefaultServerConfig() *domain.ServerConfig {
	kp, err := crypto.GenerateKeyPair()
	privKey := ""
	pubKey := ""
	if err == nil {
		privKey = kp.PrivateKey
		pubKey = kp.PublicKey
	}

	jc, jmin, jmax, s1, s2, s3, s4, h1, h2, h3, h4 := crypto.GenerateRandomObfuscationParams()
	randomPort := fmt.Sprintf("%d", crypto.RandomIntInRange(10000, 60000))

	return &domain.ServerConfig{
		PrivateKey:          privKey,
		PublicKey:           pubKey,
		Address:             "172.20.0.1/16",
		ListenPort:          randomPort,
		Endpoint:            "",
		DNS:                 "1.1.1.1, 1.0.0.1",
		LANAllowed:          "0.0.0.0/0, ::/0",
		PersistentKeepalive: "25",
		Obfuscation: domain.ObfuscationParams{
			Jc:   jc,
			Jmin: jmin,
			Jmax: jmax,
			S1:   s1,
			S2:   s2,
			S3:   s3,
			S4:   s4,
			H1:   h1,
			H2:   h2,
			H3:   h3,
			H4:   h4,
			I1:   "<r 2><b 0x858000010001000000000669636c6f756403636f6d0000010001c00c000100010000105a00044d583737>",
			I2:   "",
			I3:   "",
			I4:   "",
			I5:   "",
		},
		Peers: []domain.Peer{},
	}
}
