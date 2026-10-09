// Package crypto generates the key material and obfuscation parameters of
// AmneziaWG configurations.
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"golang.org/x/crypto/curve25519"
)

// KeyPair holds a Curve25519 key pair in the base64 form that AmneziaWG
// configuration files use.
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKeyPair returns a new key pair. The private key is clamped as
// WireGuard requires, so it has the same form as the output of awg genkey.
func GenerateKeyPair() KeyPair {
	var priv [32]byte
	rand.Read(priv[:])

	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(priv[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(pub[:]),
	}
}

// GeneratePresharedKey returns a random 32-byte preshared key in base64.
func GeneratePresharedKey() string {
	var psk [32]byte
	rand.Read(psk[:])
	return base64.StdEncoding.EncodeToString(psk[:])
}

// PublicFromPrivate derives the base64 public key of a base64 private key.
// The public key is not stored in the server configuration file, so it is
// recomputed whenever the file is read.
func PublicFromPrivate(privateKey string) (string, error) {
	priv, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("decode private key: %w", err)
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("derive public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

// RandomIntInRange returns a cryptographically random integer in [lo, hi].
// It returns lo when the range is empty.
func RandomIntInRange(lo, hi int64) int64 {
	if lo >= hi {
		return lo
	}
	n, err := rand.Int(rand.Reader, big.NewInt(hi-lo+1))
	if err != nil {
		return lo
	}
	return lo + n.Int64()
}

// GenerateRandomObfuscationParams returns random AmneziaWG obfuscation
// parameters in the order Jc, Jmin, Jmax, S1–S4, H1–H4. The H ranges do not
// overlap, as AmneziaWG requires distinct message type headers.
func GenerateRandomObfuscationParams() (string, string, string, string, string, string, string, string, string, string, string) {
	jc := RandomIntInRange(3, 10)
	jmin := RandomIntInRange(10, 40)
	jmax := RandomIntInRange(jmin+10, 100)

	s1 := RandomIntInRange(15, 160)
	s2 := RandomIntInRange(15, 160)
	s3 := RandomIntInRange(15, 160)
	s4 := RandomIntInRange(15, 160)

	h1Min := RandomIntInRange(1000000000, 1500000000)
	h1Max := RandomIntInRange(h1Min+500000, h1Min+200000000)

	h2Min := RandomIntInRange(2000000000, 2050000000)
	h2Max := RandomIntInRange(h2Min+500000, h2Min+20000000)

	h3Min := RandomIntInRange(2060000000, 2100000000)
	h3Max := RandomIntInRange(h3Min+500000, h3Min+20000000)

	h4Min := RandomIntInRange(2110000000, 2140000000)
	h4Max := RandomIntInRange(h4Min+500000, h4Min+7000000)

	return fmt.Sprintf("%d", jc),
		fmt.Sprintf("%d", jmin),
		fmt.Sprintf("%d", jmax),
		fmt.Sprintf("%d", s1),
		fmt.Sprintf("%d", s2),
		fmt.Sprintf("%d", s3),
		fmt.Sprintf("%d", s4),
		fmt.Sprintf("%d-%d", h1Min, h1Max),
		fmt.Sprintf("%d-%d", h2Min, h2Max),
		fmt.Sprintf("%d-%d", h3Min, h3Max),
		fmt.Sprintf("%d-%d", h4Min, h4Max)
}
