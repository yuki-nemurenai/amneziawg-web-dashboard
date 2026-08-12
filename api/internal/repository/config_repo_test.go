package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

func TestConfigRepo(t *testing.T) {
	tempDir := t.TempDir()
	confPath := filepath.Join(tempDir, "awg0.conf")

	sampleConf := `[Interface]
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

	if err := os.WriteFile(confPath, []byte(sampleConf), 0600); err != nil {
		t.Fatalf("Failed to write sample config: %v", err)
	}

	repo := NewFileConfigRepo(confPath)

	cfg, err := repo.LoadServerConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Address != "172.24.170.1/24" {
		t.Errorf("Expected address 172.24.170.1/24, got %s", cfg.Address)
	}
	if cfg.Obfuscation.Jc != "6" {
		t.Errorf("Expected Jc=6, got %s", cfg.Obfuscation.Jc)
	}
	if len(cfg.Peers) != 2 {
		t.Fatalf("Expected 2 peers, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].Name != "alice" || cfg.Peers[0].IP != "172.24.170.2" {
		t.Errorf("Unexpected peer 0: %+v", cfg.Peers[0])
	}
	if cfg.Peers[1].Name != "bob" || cfg.Peers[1].IP != "172.24.170.3" {
		t.Errorf("Unexpected peer 1: %+v", cfg.Peers[1])
	}

	// Add Peer
	newPeer := domain.Peer{
		Name:         "charlie",
		PublicKey:    "pubkeycharlie=",
		PresharedKey: "pskcharlie=",
		AllowedIPs:   "172.24.170.4/32",
		IP:           "172.24.170.4",
	}
	if err := repo.AddPeer(newPeer); err != nil {
		t.Fatalf("Failed to add peer: %v", err)
	}

	cfgUpdated, err := repo.LoadServerConfig()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if len(cfgUpdated.Peers) != 3 {
		t.Fatalf("Expected 3 peers after addition, got %d", len(cfgUpdated.Peers))
	}

	// Delete Peer
	if err := repo.DeletePeer("bob"); err != nil {
		t.Fatalf("Failed to delete peer: %v", err)
	}

	cfgFinal, err := repo.LoadServerConfig()
	if err != nil {
		t.Fatalf("Failed to reload config after deletion: %v", err)
	}
	if len(cfgFinal.Peers) != 2 {
		t.Fatalf("Expected 2 peers after deletion, got %d", len(cfgFinal.Peers))
	}
}
