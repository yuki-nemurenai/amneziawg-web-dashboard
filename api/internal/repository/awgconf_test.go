package repository

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

// TestFlushedServerConfigKeepsPeerNames guards the import of a file written by
// the PostgreSQL repository. It used to write "# Name = alice" after [Peer],
// which the file repository read as the name of the next peer.
func TestFlushedServerConfigKeepsPeerNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "awg0.conf")
	cfg := &domain.ServerConfig{
		Address:    "10.0.0.1/24",
		ListenPort: "51820",
		LANAllowed: "0.0.0.0/0, ::/0",
		Peers: []domain.Peer{
			{Name: "alice", PublicKey: "a=", AllowedIPs: "10.0.0.2/32"},
			{Name: "bob", PublicKey: "b=", AllowedIPs: "10.0.0.3/32"},
		},
	}
	if err := (&postgresConfigRepo{configPath: path}).flushToDisk(cfg); err != nil {
		t.Fatalf("flushToDisk() error = %v", err)
	}

	got, err := NewFileConfigRepo(path).LoadServerConfig(t.Context())
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}
	var names []string
	for _, p := range got.Peers {
		names = append(names, p.Name)
	}
	if len(names) != 2 || names[0] != "alice" || names[1] != "bob" {
		t.Errorf("peer names = %q, want [alice bob]", names)
	}
}

func TestFormatClientConfig(t *testing.T) {
	got := FormatClientConfig(ClientConfig{
		Address:    "10.0.0.2/32",
		PrivateKey: "clientPriv=",
		Obfuscation: domain.ObfuscationParams{
			Jc: "4", Jmin: "10", Jmax: "50", S1: "20", S2: "30", S3: "40", S4: "50",
			H1: "1-2", H2: "3-4", H3: "5-6", H4: "7-8", I1: "<b 0x01>",
		},
		ServerPublicKey: "serverPub=",
		AllowedIPs:      "0.0.0.0/0, ::/0",
		Endpoint:        "203.0.113.7:51820",
	})

	want := `[Interface]
Address = 10.0.0.2/32
PrivateKey = clientPriv=
Jc = 4
Jmin = 10
Jmax = 50
S1 = 20
S2 = 30
S3 = 40
S4 = 50
H1 = 1-2
H2 = 3-4
H3 = 5-6
H4 = 7-8
I1 = <b 0x01>

[Peer]
PublicKey = serverPub=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 203.0.113.7:51820
`
	if got != want {
		t.Errorf("FormatClientConfig() =\n%s\nwant\n%s", got, want)
	}
}

func TestFormatSyncConfigOmitsWGQuickSettings(t *testing.T) {
	got := FormatSyncConfig(&domain.ServerConfig{
		PrivateKey: "serverPriv=",
		Address:    "10.0.0.1/24",
		ListenPort: "51820",
		PostUp:     "iptables -A FORWARD",
		PostDown:   "iptables -D FORWARD",
		Peers:      []domain.Peer{{PublicKey: "a=", AllowedIPs: "10.0.0.2/32"}},
	})

	for _, setting := range []string{"Address", "PostUp", "PostDown"} {
		if strings.Contains(got, setting) {
			t.Errorf("FormatSyncConfig() contains %s, which awg syncconf rejects:\n%s", setting, got)
		}
	}
	if !strings.Contains(got, "PublicKey = a=\n") {
		t.Errorf("FormatSyncConfig() has no peer:\n%s", got)
	}
}
