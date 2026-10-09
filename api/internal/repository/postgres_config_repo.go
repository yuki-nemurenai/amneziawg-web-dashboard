package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
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

// NewPostgresConfigRepo returns a ConfigRepository backed by PostgreSQL that
// rewrites the AmneziaWG configuration file at configPath after every change,
// because awg reads the interface settings from that file. On the first start
// it imports an existing file, or a generated default configuration, into the
// database.
func NewPostgresConfigRepo(ctx context.Context, pool *pgxpool.Pool, configPath string) ConfigRepository {
	repo := &postgresConfigRepo{
		pool:       pool,
		configPath: configPath,
	}

	cfg, err := repo.LoadServerConfig(ctx)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		repo.importFile(ctx)
	case err != nil:
		slog.Error("Failed to load server config on startup", "error", err)
	default:
		// The file could have been edited while the API was down.
		if err := repo.flushToDisk(cfg); err != nil {
			slog.Error("Failed to write server config file on startup", "path", configPath, "error", err)
		}
	}

	return repo
}

// importFile stores the configuration file, or a generated default if there
// is none, in the database.
func (r *postgresConfigRepo) importFile(ctx context.Context) {
	fileCfg, err := NewFileConfigRepo(r.configPath).LoadServerConfig(ctx)
	if err != nil {
		slog.Error("Failed to read server config file for import", "path", r.configPath, "error", err)
		return
	}
	if fileCfg.Address == "" {
		return
	}

	if err := r.SaveServerConfig(ctx, fileCfg); err != nil {
		slog.Error("Failed to import server config", "error", err)
	}
	for _, peer := range fileCfg.Peers {
		if err := r.AddPeer(ctx, peer); err != nil {
			slog.Error("Failed to import peer", "peer", peer.Name, "error", err)
		}
	}
}

func (r *postgresConfigRepo) LoadServerConfig(ctx context.Context) (*domain.ServerConfig, error) {
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("server config: %w", domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("load server config: %w", err)
	}

	if port := os.Getenv("AWG_PORT"); port != "" {
		cfg.ListenPort = port
	}

	rows, err := r.pool.Query(ctx, `SELECT name, public_key, preshared_key, ip, allowed_ips FROM peers ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("load peers: %w", err)
	}
	cfg.Peers, err = pgx.CollectRows(rows, scanPeer)
	if err != nil {
		return nil, fmt.Errorf("load peers: %w", err)
	}

	return &cfg, nil
}

func scanPeer(row pgx.CollectableRow) (domain.Peer, error) {
	var p domain.Peer
	err := row.Scan(&p.Name, &p.PublicKey, &p.PresharedKey, &p.IP, &p.AllowedIPs)
	return p, err
}

func (r *postgresConfigRepo) SaveServerConfig(ctx context.Context, cfg *domain.ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

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
		return fmt.Errorf("save server config: %w", err)
	}

	return r.flushToDisk(cfg)
}

func (r *postgresConfigRepo) AddPeer(ctx context.Context, peer domain.Peer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	query := `
		INSERT INTO peers (name, public_key, preshared_key, ip, allowed_ips)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := r.pool.Exec(ctx, query, peer.Name, peer.PublicKey, peer.PresharedKey, peer.IP, peer.AllowedIPs); err != nil {
		return fmt.Errorf("insert peer %q: %w", peer.Name, err)
	}

	return r.syncDiskNoLock(ctx)
}

func (r *postgresConfigRepo) DeletePeer(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.pool.Exec(ctx, `DELETE FROM peers WHERE name = $1`, name); err != nil {
		return fmt.Errorf("delete peer %q: %w", name, err)
	}

	return r.syncDiskNoLock(ctx)
}

func (r *postgresConfigRepo) GetPeerByName(ctx context.Context, name string) (*domain.Peer, error) {
	query := `SELECT name, public_key, preshared_key, ip, allowed_ips FROM peers WHERE name = $1`
	var p domain.Peer
	err := r.pool.QueryRow(ctx, query, name).Scan(&p.Name, &p.PublicKey, &p.PresharedKey, &p.IP, &p.AllowedIPs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("peer %q: %w", name, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get peer %q: %w", name, err)
	}
	return &p, nil
}

func (r *postgresConfigRepo) syncDiskNoLock(ctx context.Context) error {
	cfg, err := r.LoadServerConfig(ctx)
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

func (r *postgresConfigRepo) RecordTraffic(ctx context.Context, rxBytes, txBytes int64) error {
	query := `INSERT INTO traffic_history (total_rx_bytes, total_tx_bytes) VALUES ($1, $2)`
	if _, err := r.pool.Exec(ctx, query, rxBytes, txBytes); err != nil {
		return fmt.Errorf("record traffic: %w", err)
	}
	return nil
}

func (r *postgresConfigRepo) GetTrafficHistory(ctx context.Context, limit int) ([]domain.TrafficPoint, error) {
	query := `
		SELECT EXTRACT(EPOCH FROM timestamp)::BIGINT, total_rx_bytes, total_tx_bytes
		FROM traffic_history
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get traffic history: %w", err)
	}
	history, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.TrafficPoint, error) {
		var p domain.TrafficPoint
		err := row.Scan(&p.Timestamp, &p.RxBytes, &p.TxBytes)
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("get traffic history: %w", err)
	}

	// Reverse to chronological order
	slices.Reverse(history)

	return history, nil
}

func (r *postgresConfigRepo) RecordPeerTraffic(ctx context.Context, timestamp time.Time, peers []domain.Peer) error {
	batch := &pgx.Batch{}
	query := `INSERT INTO peer_traffic_history (timestamp, public_key, rx_bytes, tx_bytes) VALUES ($1, $2, $3, $4)`
	for _, p := range peers {
		batch.Queue(query, timestamp, p.PublicKey, p.RxBytes, p.TxBytes)
	}

	// Close reads the results of every queued insert, not only the first one.
	if err := r.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("record peer traffic: %w", err)
	}
	return nil
}

func (r *postgresConfigRepo) GetTopPeersTrafficHistory(ctx context.Context, limit int, topN int) (map[string][]domain.PeerTrafficPoint, error) {
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
		return nil, fmt.Errorf("get top peers: %w", err)
	}
	topPubKeys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("get top peers: %w", err)
	}

	result := make(map[string][]domain.PeerTrafficPoint)
	if len(topPubKeys) == 0 {
		return result, nil
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
		return nil, fmt.Errorf("get peer traffic history: %w", err)
	}

	var p domain.PeerTrafficPoint
	var pubKey string
	_, err = pgx.ForEachRow(historyRows, []any{&p.Timestamp, &pubKey, &p.RxBytes, &p.TxBytes}, func() error {
		result[pubKey] = append(result[pubKey], p)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get peer traffic history: %w", err)
	}

	return result, nil
}
