package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

type postgresConfigRepo struct {
	pool       *pgxpool.Pool
	configPath string
	mu         sync.Mutex // to serialize DB writes and file flushes
}

func NewPostgresConfigRepo(pool *pgxpool.Pool, configPath string) ConfigRepository {
	repo := &postgresConfigRepo{
		pool:       pool,
		configPath: configPath,
	}

	// Initial sync: if DB is empty but awg0.conf exists, we could migrate it.
	// For now, if DB has no server_config, we generate default and save it.
	cfg, err := repo.LoadServerConfig()
	if err == pgx.ErrNoRows {
		// Try to read from awg0.conf, if not, generate default.
		fileRepo := NewFileConfigRepo(configPath)
		fileCfg, fileErr := fileRepo.LoadServerConfig()
		if fileErr == nil && fileCfg != nil && fileCfg.Address != "" {
			_ = repo.SaveServerConfig(fileCfg)
			// Add all peers from file
			for _, peer := range fileCfg.Peers {
				_ = repo.AddPeer(peer)
			}
		}
	} else if err == nil && cfg != nil {
		// Ensure file matches DB on boot
		_ = repo.flushToDisk(cfg)
	}

	return repo
}

func (r *postgresConfigRepo) LoadServerConfig() (*domain.ServerConfig, error) {
	ctx := context.Background()

	query := `SELECT 
		private_key, public_key, address, listen_port, endpoint, dns, lan_allowed,
		persistent_keepalive, post_up, post_down,
		jc, jmin, jmax, s1, s2, s3, s4, h1, h2, h3, h4, i1, i2, i3, i4, i5
	FROM server_config WHERE id = 1`

	var cfg domain.ServerConfig
	err := r.pool.QueryRow(ctx, query).Scan(
		&cfg.PrivateKey, &cfg.PublicKey, &cfg.Address, &cfg.ListenPort, &cfg.Endpoint, &cfg.DNS, &cfg.LANAllowed,
		&cfg.PersistentKeepalive, &cfg.PostUp, &cfg.PostDown,
		&cfg.Obfuscation.Jc, &cfg.Obfuscation.Jmin, &cfg.Obfuscation.Jmax,
		&cfg.Obfuscation.S1, &cfg.Obfuscation.S2, &cfg.Obfuscation.S3, &cfg.Obfuscation.S4,
		&cfg.Obfuscation.H1, &cfg.Obfuscation.H2, &cfg.Obfuscation.H3, &cfg.Obfuscation.H4,
		&cfg.Obfuscation.I1, &cfg.Obfuscation.I2, &cfg.Obfuscation.I3, &cfg.Obfuscation.I4, &cfg.Obfuscation.I5,
	)

	if err != nil {
		return nil, err
	}

	if port := os.Getenv("AWG_PORT"); port != "" {
		cfg.ListenPort = port
	}

	// Fetch peers
	peersQuery := `SELECT name, public_key, preshared_key, ip, allowed_ips FROM peers ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, peersQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p domain.Peer
		if err := rows.Scan(&p.Name, &p.PublicKey, &p.PresharedKey, &p.IP, &p.AllowedIPs); err != nil {
			return nil, err
		}
		cfg.Peers = append(cfg.Peers, p)
	}

	return &cfg, nil
}

func (r *postgresConfigRepo) SaveServerConfig(cfg *domain.ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx := context.Background()
	query := `
		INSERT INTO server_config (
			id, private_key, public_key, address, listen_port, endpoint, dns, lan_allowed,
			persistent_keepalive, post_up, post_down,
			jc, jmin, jmax, s1, s2, s3, s4, h1, h2, h3, h4, i1, i2, i3, i4, i5
		) VALUES (
			1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)
		ON CONFLICT (id) DO UPDATE SET
			private_key=EXCLUDED.private_key, public_key=EXCLUDED.public_key,
			address=EXCLUDED.address, listen_port=EXCLUDED.listen_port, endpoint=EXCLUDED.endpoint,
			dns=EXCLUDED.dns, lan_allowed=EXCLUDED.lan_allowed,
			persistent_keepalive=EXCLUDED.persistent_keepalive,
			post_up=EXCLUDED.post_up, post_down=EXCLUDED.post_down,
			jc=EXCLUDED.jc, jmin=EXCLUDED.jmin, jmax=EXCLUDED.jmax,
			s1=EXCLUDED.s1, s2=EXCLUDED.s2, s3=EXCLUDED.s3, s4=EXCLUDED.s4,
			h1=EXCLUDED.h1, h2=EXCLUDED.h2, h3=EXCLUDED.h3, h4=EXCLUDED.h4,
			i1=EXCLUDED.i1, i2=EXCLUDED.i2, i3=EXCLUDED.i3, i4=EXCLUDED.i4, i5=EXCLUDED.i5,
			updated_at=CURRENT_TIMESTAMP
	`
	_, err := r.pool.Exec(ctx, query,
		cfg.PrivateKey, cfg.PublicKey, cfg.Address, cfg.ListenPort, cfg.Endpoint, cfg.DNS, cfg.LANAllowed,
		cfg.PersistentKeepalive, cfg.PostUp, cfg.PostDown,
		cfg.Obfuscation.Jc, cfg.Obfuscation.Jmin, cfg.Obfuscation.Jmax,
		cfg.Obfuscation.S1, cfg.Obfuscation.S2, cfg.Obfuscation.S3, cfg.Obfuscation.S4,
		cfg.Obfuscation.H1, cfg.Obfuscation.H2, cfg.Obfuscation.H3, cfg.Obfuscation.H4,
		cfg.Obfuscation.I1, cfg.Obfuscation.I2, cfg.Obfuscation.I3, cfg.Obfuscation.I4, cfg.Obfuscation.I5,
	)
	if err != nil {
		return err
	}

	return r.flushToDisk(cfg)
}

func (r *postgresConfigRepo) AddPeer(peer domain.Peer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	query := `
		INSERT INTO peers (name, public_key, preshared_key, ip, allowed_ips)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, peer.Name, peer.PublicKey, peer.PresharedKey, peer.IP, peer.AllowedIPs)
	if err != nil {
		return fmt.Errorf("failed to insert peer: %w", err)
	}

	return r.syncDiskNoLock()
}

func (r *postgresConfigRepo) DeletePeer(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	query := `DELETE FROM peers WHERE name = $1`
	_, err := r.pool.Exec(ctx, query, name)
	if err != nil {
		return err
	}

	return r.syncDiskNoLock()
}

func (r *postgresConfigRepo) GetPeerByName(name string) (*domain.Peer, error) {
	ctx := context.Background()
	query := `SELECT name, public_key, preshared_key, ip, allowed_ips FROM peers WHERE name = $1`
	var p domain.Peer
	err := r.pool.QueryRow(ctx, query, name).Scan(&p.Name, &p.PublicKey, &p.PresharedKey, &p.IP, &p.AllowedIPs)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postgresConfigRepo) syncDiskNoLock() error {
	cfg, err := r.LoadServerConfig()
	if err != nil {
		return err
	}
	return r.flushToDisk(cfg)
}

func (r *postgresConfigRepo) flushToDisk(cfg *domain.ServerConfig) error {
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
		builder.WriteString("\n[Peer]\n")
		builder.WriteString(fmt.Sprintf("# Name = %s\n", peer.Name))
		if peer.PresharedKey != "" {
			builder.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}
		builder.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		builder.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
	}

	return os.WriteFile(r.configPath, []byte(builder.String()), 0600)
}

func (r *postgresConfigRepo) RecordTraffic(rxBytes, txBytes int64) error {
	ctx := context.Background()
	query := `INSERT INTO traffic_history (total_rx_bytes, total_tx_bytes) VALUES ($1, $2)`
	_, err := r.pool.Exec(ctx, query, rxBytes, txBytes)
	return err
}

func (r *postgresConfigRepo) GetTrafficHistory(limit int) ([]domain.TrafficPoint, error) {
	ctx := context.Background()
	query := `
		SELECT EXTRACT(EPOCH FROM timestamp)::BIGINT, total_rx_bytes, total_tx_bytes
		FROM traffic_history
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.TrafficPoint
	for rows.Next() {
		var p domain.TrafficPoint
		if err := rows.Scan(&p.Timestamp, &p.RxBytes, &p.TxBytes); err != nil {
			return nil, err
		}
		history = append(history, p)
	}

	// Reverse to chronological order
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	return history, nil
}

func (r *postgresConfigRepo) RecordPeerTraffic(timestamp time.Time, peers []domain.Peer) error {
	ctx := context.Background()
	batch := &pgx.Batch{}

	query := `INSERT INTO peer_traffic_history (timestamp, public_key, rx_bytes, tx_bytes) VALUES ($1, $2, $3, $4)`
	for _, p := range peers {
		batch.Queue(query, timestamp, p.PublicKey, p.RxBytes, p.TxBytes)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	_, err := br.Exec()
	return err
}

func (r *postgresConfigRepo) GetTopPeersTrafficHistory(limit int, topN int) (map[string][]domain.PeerTrafficPoint, error) {
	ctx := context.Background()

	// 1. Find the top N peers by total rx+tx over the last 'limit' records (approx 24h)
	// We can approximate by looking at their most recent record in the time window minus their oldest record in the window
	// Since we just want top N active peers, a simple way is to find peers with the highest (rx_bytes + tx_bytes) in their latest record
	topPeersQuery := `
		SELECT public_key
		FROM peer_traffic_history
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		GROUP BY public_key
		ORDER BY MAX(rx_bytes + tx_bytes) DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, topPeersQuery, topN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topPubKeys []string
	for rows.Next() {
		var pk string
		if err := rows.Scan(&pk); err == nil {
			topPubKeys = append(topPubKeys, pk)
		}
	}
	rows.Close()

	if len(topPubKeys) == 0 {
		return make(map[string][]domain.PeerTrafficPoint), nil
	}

	// 2. Fetch the actual history for these top N peers
	historyQuery := `
		SELECT EXTRACT(EPOCH FROM timestamp)::BIGINT, public_key, rx_bytes, tx_bytes
		FROM (
			SELECT timestamp, public_key, rx_bytes, tx_bytes,
				   ROW_NUMBER() OVER(PARTITION BY public_key ORDER BY timestamp DESC) as rn
			FROM peer_traffic_history
			WHERE public_key = ANY($1)
		) sub
		WHERE rn <= $2
		ORDER BY timestamp ASC
	`
	historyRows, err := r.pool.Query(ctx, historyQuery, topPubKeys, limit)
	if err != nil {
		return nil, err
	}
	defer historyRows.Close()

	result := make(map[string][]domain.PeerTrafficPoint)
	for historyRows.Next() {
		var p domain.PeerTrafficPoint
		var pubKey string
		if err := historyRows.Scan(&p.Timestamp, &pubKey, &p.RxBytes, &p.TxBytes); err == nil {
			result[pubKey] = append(result[pubKey], p)
		}
	}

	return result, nil
}
