package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

// fakeCommander answers commands from a table keyed by the command line and
// records every command it runs. Commands missing from the table fail.
type fakeCommander struct {
	outputs map[string]string
	missing []string // executables that LookPath does not find
	ran     []string
}

func (c *fakeCommander) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	line := strings.Join(append([]string{name}, args...), " ")
	c.ran = append(c.ran, line)
	out, ok := c.outputs[line]
	if !ok {
		return nil, errors.New("exit status 1")
	}
	return []byte(out), nil
}

func (c *fakeCommander) LookPath(name string) (string, error) {
	if slices.Contains(c.missing, name) {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

type fakeLocator struct {
	ip string // empty means detection fails
}

func (l fakeLocator) PublicIP(ctx context.Context) (string, error) {
	if l.ip == "" {
		return "", errors.New("no answer")
	}
	return l.ip, nil
}

func (l fakeLocator) Location(ctx context.Context) (*domain.ServerLocation, error) {
	return nil, errors.New("no answer")
}

// fakeConfigRepo serves a fixed server configuration and keeps no history.
type fakeConfigRepo struct {
	cfg domain.ServerConfig
}

func (r *fakeConfigRepo) LoadServerConfig(ctx context.Context) (*domain.ServerConfig, error) {
	cfg := r.cfg
	cfg.Peers = slices.Clone(r.cfg.Peers)
	return &cfg, nil
}

func (r *fakeConfigRepo) SaveServerConfig(ctx context.Context, cfg *domain.ServerConfig) error {
	r.cfg = *cfg
	return nil
}

func (r *fakeConfigRepo) AddPeer(ctx context.Context, peer domain.Peer) error {
	r.cfg.Peers = append(r.cfg.Peers, peer)
	return nil
}

func (r *fakeConfigRepo) DeletePeer(ctx context.Context, name string) error {
	r.cfg.Peers = slices.DeleteFunc(r.cfg.Peers, func(p domain.Peer) bool { return p.Name == name })
	return nil
}

func (r *fakeConfigRepo) GetPeerByName(ctx context.Context, name string) (*domain.Peer, error) {
	for _, p := range r.cfg.Peers {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeConfigRepo) RecordTraffic(ctx context.Context, rxBytes, txBytes int64) error {
	return nil
}

func (r *fakeConfigRepo) GetTrafficHistory(ctx context.Context, limit int) ([]domain.TrafficPoint, error) {
	return nil, nil
}

func (r *fakeConfigRepo) RecordPeerTraffic(ctx context.Context, timestamp time.Time, peers []domain.Peer) error {
	return nil
}

func (r *fakeConfigRepo) GetTopPeersTrafficHistory(ctx context.Context, limit int, topN int) (map[string][]domain.PeerTrafficPoint, error) {
	return nil, nil
}

func newTestAWGService(repo *fakeConfigRepo, cmd *fakeCommander, locator fakeLocator) *awgService {
	return &awgService{
		repo:          repo,
		ipService:     NewIPService(),
		cmd:           cmd,
		locator:       locator,
		interfaceName: "awg0",
		now:           func() time.Time { return testNow },
		statsCache:    make(map[string]peerRuntimeStats),
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "zero", bytes: 0, want: "0 B"},
		{name: "negative", bytes: -1, want: "0 B"},
		{name: "bytes", bytes: 500, want: "500 B"},
		{name: "one kilobyte", bytes: 1024, want: "1.00 KB"},
		{name: "fractional megabytes", bytes: 1572864, want: "1.50 MB"},
		{name: "one gigabyte", bytes: 1073741824, want: "1.00 GB"},
		{name: "fractional gigabytes", bytes: 2684354560, want: "2.50 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBytes(tt.bytes); got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatHandshake(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{name: "seconds", ago: 30 * time.Second, want: "Just now"},
		{name: "clock skew", ago: -10 * time.Second, want: "Just now"},
		{name: "minutes", ago: 5 * time.Minute, want: "5m ago"},
		{name: "hours", ago: 2 * time.Hour, want: "2h ago"},
		{name: "days", ago: 72 * time.Hour, want: "3d ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handshake := testNow.Add(-tt.ago).Unix()
			if got := FormatHandshake(handshake, testNow); got != tt.want {
				t.Errorf("FormatHandshake(now-%v) = %q, want %q", tt.ago, got, tt.want)
			}
		})
	}
}

func TestFormatHandshakeReportsNeverConnectedPeerOffline(t *testing.T) {
	if got := FormatHandshake(0, testNow); got != "Offline" {
		t.Errorf("FormatHandshake(0) = %q, want %q", got, "Offline")
	}
}

func TestGetClientsMarksPeersOnlineByRecentHandshake(t *testing.T) {
	repo := &fakeConfigRepo{cfg: domain.ServerConfig{Peers: []domain.Peer{
		{Name: "recent", PublicKey: "recentKey=", IP: "10.0.0.2"},
		{Name: "stale", PublicKey: "staleKey=", IP: "10.0.0.3"},
		{Name: "never", PublicKey: "neverKey=", IP: "10.0.0.4"},
	}}}
	dump := "privKey=\tpubKey=\t51820\toff\n" +
		"recentKey=\t(none)\t1.2.3.4:5000\t10.0.0.2/32\t" + unix(testNow.Add(-time.Minute)) + "\t100\t200\t0\n" +
		"staleKey=\t(none)\t1.2.3.5:5000\t10.0.0.3/32\t" + unix(testNow.Add(-time.Hour)) + "\t300\t400\t0\n"
	cmd := &fakeCommander{outputs: map[string]string{"awg show awg0 dump": dump}}

	clients, err := newTestAWGService(repo, cmd, fakeLocator{}).GetClients(t.Context())
	if err != nil {
		t.Fatalf("GetClients() error = %v", err)
	}

	online := map[string]bool{}
	for _, c := range clients {
		online[c.Name] = c.IsOnline
	}
	want := map[string]bool{"recent": true, "stale": false, "never": false}
	for name, w := range want {
		if online[name] != w {
			t.Errorf("client %s IsOnline = %v, want %v", name, online[name], w)
		}
	}
	if clients[0].RxBytes != 100 || clients[0].TxBytes != 200 {
		t.Errorf("client recent traffic = %d/%d, want 100/200", clients[0].RxBytes, clients[0].TxBytes)
	}
}

func TestEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		publicIP   string
		want       string
		wantAuto   bool
	}{
		{name: "host and port", configured: "vpn.example.com:1234", want: "vpn.example.com:1234"},
		{name: "host without port gets listen port", configured: " vpn.example.com ", want: "vpn.example.com:8443"},
		{name: "detected public IP", publicIP: "203.0.113.7", want: "203.0.113.7:8443", wantAuto: true},
		{name: "detection fails", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAWGService(&fakeConfigRepo{}, &fakeCommander{}, fakeLocator{ip: tt.publicIP})

			got, auto := svc.endpoint(t.Context(), tt.configured, "8443")
			if got != tt.want || auto != tt.wantAuto {
				t.Errorf("endpoint(%q, 8443) = %q, %v, want %q, %v", tt.configured, got, auto, tt.want, tt.wantAuto)
			}
		})
	}
}

func TestDefaultRouteInterface(t *testing.T) {
	tests := []struct {
		name   string
		routes string
		want   string
	}{
		{name: "via gateway", routes: "default via 172.17.0.1 dev eth0 \n", want: "eth0"},
		{name: "first of several", routes: "default via 10.0.0.1 dev ens3 proto dhcp metric 100\ndefault via 10.0.1.1 dev ens4\n", want: "ens3"},
		{name: "device route", routes: "default dev wg0 scope link\n", want: "wg0"},
		{name: "no default route", routes: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultRouteInterface([]byte(tt.routes)); got != tt.want {
				t.Errorf("defaultRouteInterface(%q) = %q, want %q", tt.routes, got, tt.want)
			}
		})
	}
}

func TestEnsureIptablesRuleAddsOnlyMissingRule(t *testing.T) {
	tests := []struct {
		name      string
		existing  bool
		wantAdded bool
		wantRan   []string
	}{
		{
			name:      "missing rule",
			wantAdded: true,
			wantRan: []string{
				"iptables -t nat -C POSTROUTING -o eth0 -j MASQUERADE",
				"iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE",
			},
		},
		{
			name:     "existing rule",
			existing: true,
			wantRan:  []string{"iptables -t nat -C POSTROUTING -o eth0 -j MASQUERADE"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &fakeCommander{outputs: map[string]string{"iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE": ""}}
			if tt.existing {
				cmd.outputs["iptables -t nat -C POSTROUTING -o eth0 -j MASQUERADE"] = ""
			}
			svc := newTestAWGService(&fakeConfigRepo{}, cmd, fakeLocator{})

			added := svc.ensureIptablesRule(t.Context(), "nat", "POSTROUTING", "-o", "eth0", "-j", "MASQUERADE")
			if added != tt.wantAdded {
				t.Errorf("ensureIptablesRule() = %v, want %v", added, tt.wantAdded)
			}
			if !slices.Equal(cmd.ran, tt.wantRan) {
				t.Errorf("commands = %q, want %q", cmd.ran, tt.wantRan)
			}
		})
	}
}

func TestGetSystemStatusReportsStoppedWithoutAWG(t *testing.T) {
	cmd := &fakeCommander{missing: []string{"awg"}}
	svc := newTestAWGService(&fakeConfigRepo{}, cmd, fakeLocator{})

	status, err := svc.GetSystemStatus(t.Context())
	if err != nil {
		t.Fatalf("GetSystemStatus() error = %v", err)
	}
	if status.IsRunning {
		t.Errorf("IsRunning = true without awg, want false")
	}
	if status.Mode != "Unknown" {
		t.Errorf("Mode = %q without the interface, want %q", status.Mode, "Unknown")
	}
}

// unix formats t as the Unix seconds that awg show dump prints.
func unix(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func TestCreateClientRejectsEmptyName(t *testing.T) {
	svc := newTestAWGService(&fakeConfigRepo{}, &fakeCommander{}, fakeLocator{})

	_, err := svc.CreateClient(t.Context(), domain.CreateClientRequest{Name: "  "})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("CreateClient(blank name) error = %v, want %v", err, domain.ErrInvalidInput)
	}
}

func TestDeleteClientReportsMissingClientAsNotFound(t *testing.T) {
	svc := newTestAWGService(&fakeConfigRepo{}, &fakeCommander{}, fakeLocator{})

	if err := svc.DeleteClient(t.Context(), "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("DeleteClient(nobody) error = %v, want %v", err, domain.ErrNotFound)
	}
}
