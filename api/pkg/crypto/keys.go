// Package crypto generates the key material and obfuscation parameters of
// AmneziaWG configurations.
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"strconv"

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

// ObfuscationParams are the AmneziaWG parameters that disguise WireGuard
// handshakes: junk packets (Jc, Jmin, Jmax), padding (S1–S4) and message type
// header ranges (H1–H4).
type ObfuscationParams struct {
	Jc, Jmin, Jmax string
	S1, S2, S3, S4 string
	H1, H2, H3, H4 string
}

// GenerateObfuscationParams returns random obfuscation parameters, so that
// servers set up from the dashboard do not share a fingerprint. The H ranges
// do not overlap, as AmneziaWG requires distinct headers.
func GenerateObfuscationParams() ObfuscationParams {
	jmin := RandomIntInRange(10, 40)
	return ObfuscationParams{
		Jc:   strconv.FormatInt(RandomIntInRange(3, 10), 10),
		Jmin: strconv.FormatInt(jmin, 10),
		Jmax: strconv.FormatInt(RandomIntInRange(jmin+10, 100), 10),
		S1:   strconv.FormatInt(RandomIntInRange(15, 160), 10),
		S2:   strconv.FormatInt(RandomIntInRange(15, 160), 10),
		S3:   strconv.FormatInt(RandomIntInRange(15, 160), 10),
		S4:   strconv.FormatInt(RandomIntInRange(15, 160), 10),
		H1:   randomRange(1000000000, 1500000000, 200000000, 2000000000-1),
		H2:   randomRange(2000000000, 2050000000, 20000000, 2060000000-1),
		H3:   randomRange(2060000000, 2100000000, 20000000, 2110000000-1),
		H4:   randomRange(2110000000, 2140000000, 7000000, math.MaxInt32),
	}
}

// minHeaderRangeWidth is the narrowest H range generated.
const minHeaderRangeWidth = 500000

// randomRange returns a header range "lo-hi" that starts in [startLo, startHi],
// is at most maxWidth wide and ends no later than ceiling, the start of the
// next header range minus one.
func randomRange(startLo, startHi, maxWidth, ceiling int64) string {
	lo := RandomIntInRange(startLo, startHi)
	hi := RandomIntInRange(lo+minHeaderRangeWidth, min(lo+maxWidth, ceiling))
	return fmt.Sprintf("%d-%d", lo, hi)
}
