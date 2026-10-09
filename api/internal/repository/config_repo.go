package repository

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/pkg/crypto"
)

// ConfigRepository stores the server configuration, its peers and the traffic
// history. Lookups of a missing peer or configuration return an error wrapping
// domain.ErrNotFound.
type ConfigRepository interface {
	LoadServerConfig(ctx context.Context) (*domain.ServerConfig, error)
	SaveServerConfig(ctx context.Context, cfg *domain.ServerConfig) error
	AddPeer(ctx context.Context, peer domain.Peer) error
	DeletePeer(ctx context.Context, name string) error
	GetPeerByName(ctx context.Context, name string) (*domain.Peer, error)
	RecordTraffic(ctx context.Context, rxBytes, txBytes int64) error
	GetTrafficHistory(ctx context.Context, limit int) ([]domain.TrafficPoint, error)
	RecordPeerTraffic(ctx context.Context, timestamp time.Time, peers []domain.Peer) error
	GetTopPeersTrafficHistory(ctx context.Context, limit int, topN int) (map[string][]domain.PeerTrafficPoint, error)
}

type fileConfigRepo struct {
	configPath string
	mu         sync.RWMutex
}

// NewFileConfigRepo returns a ConfigRepository that keeps everything in the
// AmneziaWG configuration file at configPath. It keeps no traffic history and
// is used to import an existing file into PostgreSQL.
func NewFileConfigRepo(configPath string) ConfigRepository {
	return &fileConfigRepo{
		configPath: configPath,
	}
}

func (r *fileConfigRepo) LoadServerConfig(ctx context.Context) (*domain.ServerConfig, error) {
	r.mu.RLock()
	file, err := os.Open(r.configPath)
	if err != nil {
		r.mu.RUnlock()
		if errors.Is(err, fs.ErrNotExist) {
			defaultCfg := r.createDefaultServerConfig()
			// The generated config is still usable: the caller stores it in
			// PostgreSQL, which rewrites the file later.
			if err := r.SaveServerConfig(ctx, defaultCfg); err != nil {
				slog.Warn("Failed to write default server config", "path", r.configPath, "error", err)
			}
			return defaultCfg, nil
		}
		return nil, fmt.Errorf("read server config: %w", err)
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

		if after, ok := strings.CutPrefix(line, "#"); ok {
			commentContent := strings.TrimSpace(after)
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

// RecordTraffic does nothing: the configuration file has no place for history.
func (r *fileConfigRepo) RecordTraffic(ctx context.Context, rxBytes, txBytes int64) error {
	return nil
}

// GetTrafficHistory returns no history: the configuration file has none.
func (r *fileConfigRepo) GetTrafficHistory(ctx context.Context, limit int) ([]domain.TrafficPoint, error) {
	return nil, nil
}

// RecordPeerTraffic does nothing: the configuration file has no place for
// history.
func (r *fileConfigRepo) RecordPeerTraffic(ctx context.Context, timestamp time.Time, peers []domain.Peer) error {
	return nil
}

// GetTopPeersTrafficHistory returns no history: the configuration file has
// none.
func (r *fileConfigRepo) GetTopPeersTrafficHistory(ctx context.Context, limit int, topN int) (map[string][]domain.PeerTrafficPoint, error) {
	return nil, nil
}

func (r *fileConfigRepo) SaveServerConfig(ctx context.Context, cfg *domain.ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return os.WriteFile(r.configPath, []byte(formatServerConfig(cfg)), 0600)
}

func (r *fileConfigRepo) AddPeer(ctx context.Context, peer domain.Peer) error {
	cfg, err := r.LoadServerConfig(ctx)
	if err != nil {
		return err
	}

	for _, existing := range cfg.Peers {
		if strings.EqualFold(existing.Name, peer.Name) {
			return fmt.Errorf("%w: peer with name %q already exists", domain.ErrConflict, peer.Name)
		}
		if existing.IP == peer.IP && peer.IP != "" {
			return fmt.Errorf("%w: peer with IP %q already exists", domain.ErrConflict, peer.IP)
		}
	}

	cfg.Peers = append(cfg.Peers, peer)
	return r.SaveServerConfig(ctx, cfg)
}

func (r *fileConfigRepo) DeletePeer(ctx context.Context, name string) error {
	cfg, err := r.LoadServerConfig(ctx)
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
		return fmt.Errorf("peer %q: %w", name, domain.ErrNotFound)
	}

	cfg.Peers = newPeers
	return r.SaveServerConfig(ctx, cfg)
}

func (r *fileConfigRepo) GetPeerByName(ctx context.Context, name string) (*domain.Peer, error) {
	cfg, err := r.LoadServerConfig(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range cfg.Peers {
		if strings.EqualFold(p.Name, name) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("peer %q: %w", name, domain.ErrNotFound)
}

func (r *fileConfigRepo) createDefaultServerConfig() *domain.ServerConfig {
	kp := crypto.GenerateKeyPair()
	obf := crypto.GenerateObfuscationParams()
	randomPort := fmt.Sprintf("%d", crypto.RandomIntInRange(10000, 60000))

	return &domain.ServerConfig{
		PrivateKey:          kp.PrivateKey,
		PublicKey:           kp.PublicKey,
		Address:             "172.20.0.1/16",
		ListenPort:          randomPort,
		Endpoint:            "",
		DNS:                 "1.1.1.1, 1.0.0.1",
		LANAllowed:          "0.0.0.0/0, ::/0",
		PersistentKeepalive: "25",
		Obfuscation: domain.ObfuscationParams{
			Jc:   obf.Jc,
			Jmin: obf.Jmin,
			Jmax: obf.Jmax,
			S1:   obf.S1,
			S2:   obf.S2,
			S3:   obf.S3,
			S4:   obf.S4,
			H1:   obf.H1,
			H2:   obf.H2,
			H3:   obf.H3,
			H4:   obf.H4,
			I1:   "<r 2><b 0x858000010001000000000669636c6f756403636f6d0000010001c00c000100010000105a00044d583737>",
			I2:   "",
			I3:   "",
			I4:   "",
			I5:   "",
		},
		Peers: []domain.Peer{},
	}
}
