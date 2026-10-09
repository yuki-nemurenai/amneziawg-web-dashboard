package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

const sampleConf = `[Interface]
PrivateKey = testPrivateKey1234567890123456789012345=
Address = 172.24.170.1/24
ListenPort = 689
Jc = 6
Jmin = 10
Jmax = 50
S1 = 136
S2 = 20
S3 = 36
S4 = 10
H1 = 1027326130-2124574311
H2 = 2128283030-2131527662
H3 = 2139330923-2144622857
H4 = 2145845262-2147466530
I1 = <b 0xc0>
I2 = <b 0x49>
I3 = <b 0x53>

# alice
[Peer]
PublicKey = VShcwL0mDHJbGefD4t1deJxfysw6YNKRK4Sg0s0XJ3M=
PresharedKey = baANwHDJpiU7LaEQkka667SghIhCHv294avSpsPo9xw=
AllowedIPs = 172.24.170.2/32

# bob
[Peer]
PublicKey = +yhqLeSf8Tt/SxC80OOdD4f6+Gf+32it+1QPgIWR+Tg=
AllowedIPs = 172.24.170.3/32
`

// newSampleRepo returns a file repository over a copy of sampleConf.
func newSampleRepo(t *testing.T) ConfigRepository {
	t.Helper()
	confPath := filepath.Join(t.TempDir(), "awg0.conf")
	if err := os.WriteFile(confPath, []byte(sampleConf), 0600); err != nil {
		t.Fatalf("write sample config: %v", err)
	}
	return NewFileConfigRepo(confPath)
}

func TestFileConfigRepoLoadsInterfaceAndNamedPeers(t *testing.T) {
	cfg, err := newSampleRepo(t).LoadServerConfig(t.Context())
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}

	if cfg.Address != "172.24.170.1/24" {
		t.Errorf("Address = %s, want 172.24.170.1/24", cfg.Address)
	}
	if cfg.Obfuscation.Jc != "6" {
		t.Errorf("Obfuscation.Jc = %s, want 6", cfg.Obfuscation.Jc)
	}
	if len(cfg.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(cfg.Peers))
	}
	if cfg.Peers[0].Name != "alice" || cfg.Peers[0].IP != "172.24.170.2" {
		t.Errorf("Peers[0] = %+v, want alice at 172.24.170.2", cfg.Peers[0])
	}
	if cfg.Peers[1].Name != "bob" || cfg.Peers[1].IP != "172.24.170.3" {
		t.Errorf("Peers[1] = %+v, want bob at 172.24.170.3", cfg.Peers[1])
	}
}

func TestFileConfigRepoAddsAndDeletesPeers(t *testing.T) {
	ctx := t.Context()
	repo := newSampleRepo(t)

	charlie := domain.Peer{
		Name:         "charlie",
		PublicKey:    "pubkeycharlie=",
		PresharedKey: "pskcharlie=",
		AllowedIPs:   "172.24.170.4/32",
		IP:           "172.24.170.4",
	}
	if err := repo.AddPeer(ctx, charlie); err != nil {
		t.Fatalf("AddPeer(charlie) error = %v", err)
	}
	if err := repo.DeletePeer(ctx, "bob"); err != nil {
		t.Fatalf("DeletePeer(bob) error = %v", err)
	}

	cfg, err := repo.LoadServerConfig(ctx)
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}
	var names []string
	for _, p := range cfg.Peers {
		names = append(names, p.Name)
	}
	if len(names) != 2 || names[0] != "alice" || names[1] != "charlie" {
		t.Errorf("peer names = %v, want [alice charlie]", names)
	}
}

func TestFileConfigRepoAddPeerRejectsDuplicates(t *testing.T) {
	tests := []struct {
		name string
		peer domain.Peer
	}{
		{name: "same name in other case", peer: domain.Peer{Name: "Alice", PublicKey: "k1=", IP: "172.24.170.9"}},
		{name: "same IP", peer: domain.Peer{Name: "dave", PublicKey: "k2=", IP: "172.24.170.2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := newSampleRepo(t).AddPeer(t.Context(), tt.peer); !errors.Is(err, domain.ErrConflict) {
				t.Errorf("AddPeer(%+v) error = %v, want %v", tt.peer, err, domain.ErrConflict)
			}
		})
	}
}

func TestFileConfigRepoReportsMissingPeerAsNotFound(t *testing.T) {
	ctx := t.Context()
	repo := newSampleRepo(t)

	if _, err := repo.GetPeerByName(ctx, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetPeerByName(nobody) error = %v, want %v", err, domain.ErrNotFound)
	}
	if err := repo.DeletePeer(ctx, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("DeletePeer(nobody) error = %v, want %v", err, domain.ErrNotFound)
	}
}
