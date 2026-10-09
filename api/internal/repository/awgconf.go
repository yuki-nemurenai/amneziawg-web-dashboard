package repository

import (
	"fmt"
	"strings"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

// ClientConfig is the content of a client configuration file.
type ClientConfig struct {
	// Address is the client address with its prefix length, such as
	// 10.8.0.2/32.
	Address             string
	DNS                 string
	PrivateKey          string
	Obfuscation         domain.ObfuscationParams
	ServerPublicKey     string
	PresharedKey        string
	AllowedIPs          string
	Endpoint            string
	PersistentKeepalive string
}

// FormatClientConfig returns the client configuration file that the
// AmneziaVPN app imports.
func FormatClientConfig(c ClientConfig) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	writeSetting(&b, "Address", c.Address)
	writeOptional(&b, "DNS", c.DNS)
	writeSetting(&b, "PrivateKey", c.PrivateKey)
	writeObfuscation(&b, c.Obfuscation)

	b.WriteString("\n[Peer]\n")
	writeSetting(&b, "PublicKey", c.ServerPublicKey)
	writeOptional(&b, "PresharedKey", c.PresharedKey)
	writeSetting(&b, "AllowedIPs", c.AllowedIPs)
	writeSetting(&b, "Endpoint", c.Endpoint)
	writeOptional(&b, "PersistentKeepalive", c.PersistentKeepalive)
	return b.String()
}

// FormatSyncConfig returns the server configuration in the form awg syncconf
// accepts: without the wg-quick settings Address, PostUp and PostDown, which it
// rejects.
func FormatSyncConfig(cfg *domain.ServerConfig) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	writeSetting(&b, "PrivateKey", cfg.PrivateKey)
	writeOptional(&b, "ListenPort", cfg.ListenPort)
	writeObfuscation(&b, cfg.Obfuscation)

	for _, peer := range cfg.Peers {
		b.WriteString("\n[Peer]\n")
		writeSetting(&b, "PublicKey", peer.PublicKey)
		writeOptional(&b, "PresharedKey", peer.PresharedKey)
		writeSetting(&b, "AllowedIPs", peer.AllowedIPs)
	}
	return b.String()
}

// formatServerConfig returns the server configuration file. The endpoint, the
// LAN routes and the peer names are written as comments, which awg ignores, so
// that LoadServerConfig of the file repository can import them back.
func formatServerConfig(cfg *domain.ServerConfig) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	writeSetting(&b, "PrivateKey", cfg.PrivateKey)
	writeSetting(&b, "Address", cfg.Address)
	writeSetting(&b, "ListenPort", cfg.ListenPort)
	if cfg.Endpoint != "" {
		fmt.Fprintf(&b, "# Endpoint = %s\n", cfg.Endpoint)
	}
	if cfg.LANAllowed != "" {
		fmt.Fprintf(&b, "# LANAllowed = %s\n", cfg.LANAllowed)
	}
	writeObfuscation(&b, cfg.Obfuscation)

	if cfg.PostUp != "" {
		b.WriteString("\n")
		writeSetting(&b, "PostUp", cfg.PostUp)
	}
	writeOptional(&b, "PostDown", cfg.PostDown)

	for _, peer := range cfg.Peers {
		b.WriteString("\n")
		if peer.Name != "" {
			fmt.Fprintf(&b, "# %s\n", peer.Name)
		}
		b.WriteString("[Peer]\n")
		writeSetting(&b, "PublicKey", peer.PublicKey)
		writeOptional(&b, "PresharedKey", peer.PresharedKey)
		writeSetting(&b, "AllowedIPs", peer.AllowedIPs)
	}
	return b.String()
}

// writeObfuscation writes the obfuscation settings of the [Interface]
// section. They are written only when Jc is set, because a configuration
// without them is a plain WireGuard one.
func writeObfuscation(b *strings.Builder, o domain.ObfuscationParams) {
	if o.Jc == "" {
		return
	}
	writeSetting(b, "Jc", o.Jc)
	writeSetting(b, "Jmin", o.Jmin)
	writeSetting(b, "Jmax", o.Jmax)
	writeSetting(b, "S1", o.S1)
	writeSetting(b, "S2", o.S2)
	writeSetting(b, "S3", o.S3)
	writeSetting(b, "S4", o.S4)
	writeSetting(b, "H1", o.H1)
	writeSetting(b, "H2", o.H2)
	writeSetting(b, "H3", o.H3)
	writeSetting(b, "H4", o.H4)
	writeOptional(b, "I1", o.I1)
	writeOptional(b, "I2", o.I2)
	writeOptional(b, "I3", o.I3)
	writeOptional(b, "I4", o.I4)
	writeOptional(b, "I5", o.I5)
}

func writeSetting(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s = %s\n", key, value)
}

func writeOptional(b *strings.Builder, key, value string) {
	if value != "" {
		writeSetting(b, key, value)
	}
}
