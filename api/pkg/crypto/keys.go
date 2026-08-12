package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

// KeyPair holds base64 encoded WireGuard private & public keys
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKeyPair creates a new Curve25519 keypair for WireGuard / AmneziaWG
func GenerateKeyPair() (*KeyPair, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Clamp key according to WireGuard spec
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return &KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(priv[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(pub[:]),
	}, nil
}

// GeneratePresharedKey creates a random 32-byte base64 preshared key
func GeneratePresharedKey() (string, error) {
	var psk [32]byte
	if _, err := rand.Read(psk[:]); err != nil {
		return "", fmt.Errorf("failed to generate preshared key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(psk[:]), nil
}

// PublicFromPrivate computes public key from base64 private key
func PublicFromPrivate(privateKeyBase64 string) (string, error) {
	priv, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil || len(priv) != 32 {
		return "", fmt.Errorf("invalid private key length or encoding")
	}

	var privBytes [32]byte
	copy(privBytes[:], priv)

	var pubBytes [32]byte
	curve25519.ScalarBaseMult(&pubBytes, &privBytes)

	return base64.StdEncoding.EncodeToString(pubBytes[:]), nil
}

// RandomIntInRange returns a cryptographically random integer in [min, max]
func RandomIntInRange(min, max int64) int64 {
	if min >= max {
		return min
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return min
	}
	return min + nBig.Int64()
}

// GenerateRandomObfuscationParams creates randomized AmneziaWG obfuscation parameters
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

// GetPublicIP tries to fetch the server's public IP address via env var or HTTP providers
func GetPublicIP() string {
	if envIP := strings.TrimSpace(os.Getenv("PUBLIC_IP")); envIP != "" {
		return envIP
	}

	client := &http.Client{Timeout: 3 * time.Second}
	providers := []string{
		"http://api.ipify.org",
		"http://checkip.amazonaws.com",
		"http://icanhazip.com",
		"http://ifconfig.me/ip",
		"https://api.ipify.org",
	}

	for _, url := range providers {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err == nil {
				ip := strings.TrimSpace(string(body))
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}

	return ""
}
