package crypto

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestPublicFromPrivateMatchesRFC7748(t *testing.T) {
	// Alice's key pair from RFC 7748, section 6.1.
	priv, _ := hex.DecodeString("77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a")
	want, _ := hex.DecodeString("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a")

	got, err := PublicFromPrivate(base64.StdEncoding.EncodeToString(priv))
	if err != nil {
		t.Fatalf("PublicFromPrivate() error = %v", err)
	}
	if got != base64.StdEncoding.EncodeToString(want) {
		t.Errorf("PublicFromPrivate() = %s, want %s", got, base64.StdEncoding.EncodeToString(want))
	}
}

func TestPublicFromPrivateRejectsMalformedKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "not base64", key: "not-base64!"},
		{name: "short key", key: base64.StdEncoding.EncodeToString(make([]byte, 16))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := PublicFromPrivate(tt.key); err == nil {
				t.Errorf("PublicFromPrivate(%q) error = nil, want error", tt.key)
			}
		})
	}
}

func TestGenerateKeyPairPublicKeyMatchesPrivateKey(t *testing.T) {
	kp := GenerateKeyPair()

	pub, err := PublicFromPrivate(kp.PrivateKey)
	if err != nil {
		t.Fatalf("PublicFromPrivate() error = %v", err)
	}
	if pub != kp.PublicKey {
		t.Errorf("PublicFromPrivate(PrivateKey) = %s, want PublicKey %s", pub, kp.PublicKey)
	}
}
