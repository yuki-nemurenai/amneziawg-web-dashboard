package repository

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

var awg31Params = domain.ObfuscationParams{
	Jc: "4", Jmin: "10", Jmax: "50", S1: "12", S2: "12", S3: "12", S4: "12",
	H1: "1", H2: "2", H3: "3", H4: "4", I1: "<r 2><b 0x01>",
	HeaderProtectionKey:  "hpKey=",
	RekeyAfterTime:       "100-120",
	RekeyTimeout:         "3-7",
	RejectAfterTime:      "150-180",
	KeepaliveTimeout:     "5-15",
	MaxHandshakeAttempts: "15-20",
	RandomTrailers:       "on",
	DisableCookies:       "on",
}

// decodeVPNKey reverses FormatAmneziaVPNConfig the way the AmneziaVPN app
// does: unpadded URL-safe base64, then Qt qUncompress.
func decodeVPNKey(t *testing.T, key string) map[string]any {
	t.Helper()
	data, ok := strings.CutPrefix(key, "vpn://")
	if !ok {
		t.Fatalf("key %q has no vpn:// prefix", key)
	}
	packed, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	r, err := zlib.NewReader(bytes.NewReader(packed[4:]))
	if err != nil {
		t.Fatalf("open zlib stream: %v", err)
	}
	doc, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read zlib stream: %v", err)
	}
	if n := binary.BigEndian.Uint32(packed[:4]); int(n) != len(doc) {
		t.Errorf("qCompress length = %d, want %d", n, len(doc))
	}
	var profile map[string]any
	if err := json.Unmarshal(doc, &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return profile
}

func TestFormatAmneziaVPNConfigIsImportableProfile(t *testing.T) {
	key, err := FormatAmneziaVPNConfig(AmneziaVPNConfig{
		Client: ClientConfig{
			Address:             "10.8.1.3/32",
			DNS:                 "1.1.1.1, 1.0.0.1",
			PrivateKey:          "clientPriv=",
			Obfuscation:         awg31Params,
			ServerPublicKey:     "serverPub=",
			PresharedKey:        "psk=",
			AllowedIPs:          "0.0.0.0/0, ::/0",
			Endpoint:            "vpn.example.com:601",
			PersistentKeepalive: "25-35",
		},
		ClientPublicKey: "clientPub=",
		SubnetAddress:   "10.8.1.0",
	})
	if err != nil {
		t.Fatalf("FormatAmneziaVPNConfig() error = %v", err)
	}

	profile := decodeVPNKey(t, key)
	for k, want := range map[string]string{
		"defaultContainer": "amnezia-awg2",
		"hostName":         "vpn.example.com",
		"dns1":             "1.1.1.1",
		"dns2":             "1.0.0.1",
	} {
		if got := profile[k]; got != want {
			t.Errorf("profile[%q] = %v, want %q", k, got, want)
		}
	}

	if profile["format_version"] != float64(1) {
		t.Errorf("profile[format_version] = %v, want 1", profile["format_version"])
	}

	container := profile["containers"].([]any)[0].(map[string]any)
	if container["container"] != "amnezia-awg2" {
		t.Errorf("container = %v, want amnezia-awg2", container["container"])
	}
	awg := container["awg"].(map[string]any)
	for k, want := range map[string]string{
		"protocol_version":    "3.1",
		"port":                "601",
		"subnet_address":      "10.8.1.0",
		"transport_proto":     "udp",
		"HeaderProtectionKey": "hpKey=",
		"RandomTrailers":      "on",
		"S4":                  "12",
	} {
		if got := awg[k]; got != want {
			t.Errorf("awg[%q] = %v, want %q", k, got, want)
		}
	}

	var last map[string]any
	if err := json.Unmarshal([]byte(awg["last_config"].(string)), &last); err != nil {
		t.Fatalf("decode last_config: %v", err)
	}
	for k, want := range map[string]any{
		"client_ip":             "10.8.1.3",
		"client_priv_key":       "clientPriv=",
		"client_pub_key":        "clientPub=",
		"clientId":              "clientPub=",
		"server_pub_key":        "serverPub=",
		"psk_key":               "psk=",
		"hostName":              "vpn.example.com",
		"port":                  float64(601),
		"mtu":                   "1376",
		"persistent_keep_alive": "25-35",
		"DisableCookies":        "on",
	} {
		if got := last[k]; got != want {
			t.Errorf("last_config[%q] = %v, want %v", k, got, want)
		}
	}
	var allowed []string
	for _, ip := range last["allowed_ips"].([]any) {
		allowed = append(allowed, ip.(string))
	}
	if !slices.Equal(allowed, []string{"0.0.0.0/0", "::/0"}) {
		t.Errorf("last_config allowed_ips = %q, want [0.0.0.0/0 ::/0]", allowed)
	}
	config := last["config"].(string)
	for _, want := range []string{"DNS = $PRIMARY_DNS, $SECONDARY_DNS\n", "HeaderProtectionKey = hpKey=\n", "Endpoint = vpn.example.com:601\n"} {
		if !strings.Contains(config, want) {
			t.Errorf("last_config config has no %q:\n%s", want, config)
		}
	}
}

func TestFormatAmneziaVPNConfigDuplicatesSingleDNS(t *testing.T) {
	key, err := FormatAmneziaVPNConfig(AmneziaVPNConfig{Client: ClientConfig{
		Address:  "10.8.1.3/32",
		DNS:      "9.9.9.9",
		Endpoint: "203.0.113.7:51820",
	}})
	if err != nil {
		t.Fatalf("FormatAmneziaVPNConfig() error = %v", err)
	}
	profile := decodeVPNKey(t, key)
	if profile["dns1"] != "9.9.9.9" || profile["dns2"] != "9.9.9.9" {
		t.Errorf("dns1, dns2 = %v, %v, want 9.9.9.9 for both", profile["dns1"], profile["dns2"])
	}
}

func TestFormatAmneziaVPNConfigReportsAWG2WithoutAWG3Settings(t *testing.T) {
	key, err := FormatAmneziaVPNConfig(AmneziaVPNConfig{Client: ClientConfig{
		Address:     "10.8.1.3/32",
		Endpoint:    "203.0.113.7:51820",
		Obfuscation: domain.ObfuscationParams{Jc: "4", H1: "1-2", H2: "3-4", H3: "5-6", H4: "7-8"},
	}})
	if err != nil {
		t.Fatalf("FormatAmneziaVPNConfig() error = %v", err)
	}
	awg := decodeVPNKey(t, key)["containers"].([]any)[0].(map[string]any)["awg"].(map[string]any)
	if awg["protocol_version"] != "2" {
		t.Errorf("protocol_version = %v, want 2", awg["protocol_version"])
	}
}

func TestFormatAmneziaVPNConfigRejectsEndpointWithoutPort(t *testing.T) {
	if _, err := FormatAmneziaVPNConfig(AmneziaVPNConfig{Client: ClientConfig{Endpoint: ""}}); err == nil {
		t.Error("FormatAmneziaVPNConfig(no endpoint) error = nil, want error")
	}
}
