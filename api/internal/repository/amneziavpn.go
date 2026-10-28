package repository

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

const (
	// amneziaContainer is the AmneziaVPN container that runs AmneziaWG 2
	// and later; the protocol version tells the app which of them it is.
	amneziaContainer = "amnezia-awg2"
	// amneziaFormatVersion is the version of the profile document.
	amneziaFormatVersion = 1
	// amneziaMTU is the MTU the AmneziaVPN app sets on desktop systems.
	amneziaMTU = "1376"
	// The DNS servers of the profile are kept in dns1 and dns2 and
	// substituted into the configuration by the app.
	primaryDNSVar   = "$PRIMARY_DNS"
	secondaryDNSVar = "$SECONDARY_DNS"
)

// AmneziaVPNConfig is a client profile in the form the AmneziaVPN app
// imports. It adds to the client configuration file what the app keeps
// about the server: the client public key and the VPN subnet.
type AmneziaVPNConfig struct {
	Client          ClientConfig
	ClientPublicKey string
	// SubnetAddress is the network address of the VPN subnet, such as
	// 10.8.0.0.
	SubnetAddress string
}

// amneziaProfile is the JSON document behind a vpn:// key.
type amneziaProfile struct {
	Containers       []amneziaContainerConfig `json:"containers"`
	DefaultContainer string                   `json:"defaultContainer"`
	Description      string                   `json:"description"`
	DNS1             string                   `json:"dns1,omitempty"`
	DNS2             string                   `json:"dns2,omitempty"`
	FormatVersion    int                      `json:"format_version"`
	HostName         string                   `json:"hostName"`
}

type amneziaContainerConfig struct {
	AWG       amneziaServerConfig `json:"awg"`
	Container string              `json:"container"`
}

// amneziaParams are the AmneziaWG settings under the names the app uses,
// which are those of the configuration file.
type amneziaParams struct {
	Jc   string `json:"Jc,omitempty"`
	Jmin string `json:"Jmin,omitempty"`
	Jmax string `json:"Jmax,omitempty"`
	S1   string `json:"S1,omitempty"`
	S2   string `json:"S2,omitempty"`
	S3   string `json:"S3,omitempty"`
	S4   string `json:"S4,omitempty"`
	H1   string `json:"H1,omitempty"`
	H2   string `json:"H2,omitempty"`
	H3   string `json:"H3,omitempty"`
	H4   string `json:"H4,omitempty"`
	I1   string `json:"I1,omitempty"`
	I2   string `json:"I2,omitempty"`
	I3   string `json:"I3,omitempty"`
	I4   string `json:"I4,omitempty"`
	I5   string `json:"I5,omitempty"`

	HeaderProtectionKey    string `json:"HeaderProtectionKey,omitempty"`
	ContentPaddingAddition string `json:"ContentPaddingAddition,omitempty"`
	RekeyAfterTime         string `json:"RekeyAfterTime,omitempty"`
	RekeyTimeout           string `json:"RekeyTimeout,omitempty"`
	RejectAfterTime        string `json:"RejectAfterTime,omitempty"`
	KeepaliveTimeout       string `json:"KeepaliveTimeout,omitempty"`
	MaxHandshakeAttempts   string `json:"MaxHandshakeAttempts,omitempty"`
	RandomTrailers         string `json:"RandomTrailers,omitempty"`
	DisableCookies         string `json:"DisableCookies,omitempty"`
}

// amneziaServerConfig is the container settings. LastConfig is the client
// part, stored by the app as a JSON string.
type amneziaServerConfig struct {
	amneziaParams
	LastConfig      string `json:"last_config"`
	Port            string `json:"port"`
	ProtocolVersion string `json:"protocol_version"`
	SubnetAddress   string `json:"subnet_address,omitempty"`
	TransportProto  string `json:"transport_proto"`
}

type amneziaClientConfig struct {
	amneziaParams
	AllowedIPs          []string `json:"allowed_ips"`
	ClientID            string   `json:"clientId"`
	ClientIP            string   `json:"client_ip"`
	ClientPrivateKey    string   `json:"client_priv_key"`
	ClientPublicKey     string   `json:"client_pub_key"`
	Config              string   `json:"config"`
	HostName            string   `json:"hostName"`
	MTU                 string   `json:"mtu"`
	PersistentKeepalive string   `json:"persistent_keep_alive,omitempty"`
	Port                int      `json:"port"`
	PresharedKey        string   `json:"psk_key,omitempty"`
	ServerPublicKey     string   `json:"server_pub_key"`
}

// FormatAmneziaVPNConfig returns the profile as the vpn:// key that the
// AmneziaVPN app imports from a .vpn file, a link or the clipboard: the JSON
// profile compressed as Qt qCompress does, a big-endian length followed by a
// zlib stream, in unpadded URL-safe base64.
func FormatAmneziaVPNConfig(c AmneziaVPNConfig) (string, error) {
	host, port, err := net.SplitHostPort(c.Client.Endpoint)
	if err != nil {
		return "", fmt.Errorf("endpoint %q: %w", c.Client.Endpoint, err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("endpoint port %q: %w", port, err)
	}

	clientIP, _, _ := strings.Cut(c.Client.Address, "/")
	dns := splitList(c.Client.DNS)
	native := c.Client
	if len(dns) > 0 {
		native.DNS = primaryDNSVar + ", " + secondaryDNSVar
	}
	params := toAmneziaParams(c.Client.Obfuscation)

	client, err := json.Marshal(amneziaClientConfig{
		amneziaParams:       params,
		AllowedIPs:          splitList(c.Client.AllowedIPs),
		ClientID:            c.ClientPublicKey,
		ClientIP:            clientIP,
		ClientPrivateKey:    c.Client.PrivateKey,
		ClientPublicKey:     c.ClientPublicKey,
		Config:              FormatClientConfig(native),
		HostName:            host,
		MTU:                 amneziaMTU,
		PersistentKeepalive: c.Client.PersistentKeepalive,
		Port:                portNum,
		PresharedKey:        c.Client.PresharedKey,
		ServerPublicKey:     c.Client.ServerPublicKey,
	})
	if err != nil {
		return "", fmt.Errorf("encode client config: %w", err)
	}

	profile := amneziaProfile{
		Containers: []amneziaContainerConfig{{
			AWG: amneziaServerConfig{
				amneziaParams:   params,
				LastConfig:      string(client),
				Port:            port,
				ProtocolVersion: amneziaProtocolVersion(c.Client.Obfuscation),
				SubnetAddress:   c.SubnetAddress,
				TransportProto:  "udp",
			},
			Container: amneziaContainer,
		}},
		DefaultContainer: amneziaContainer,
		Description:      host,
		FormatVersion:    amneziaFormatVersion,
		HostName:         host,
	}
	if len(dns) > 0 {
		// A single DNS server fills both, so that the substituted
		// configuration has no empty entry.
		profile.DNS1, profile.DNS2 = dns[0], dns[0]
		if len(dns) > 1 {
			profile.DNS2 = dns[1]
		}
	}

	doc, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("encode profile: %w", err)
	}
	packed, err := qCompress(doc)
	if err != nil {
		return "", fmt.Errorf("compress profile: %w", err)
	}
	return "vpn://" + base64.RawURLEncoding.EncodeToString(packed), nil
}

// qCompress compresses data in the format of Qt qCompress, which the app
// reverses with qUncompress.
func qCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, uint32(len(data))); err != nil {
		return nil, err
	}
	w := zlib.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// amneziaProtocolVersion is the AmneziaWG version the app shows and expects:
// 3.1 when any AmneziaWG 3 setting is present, otherwise 2.
func amneziaProtocolVersion(o domain.ObfuscationParams) string {
	for _, v := range []string{
		o.HeaderProtectionKey, o.ContentPaddingAddition,
		o.RekeyAfterTime, o.RekeyTimeout, o.RejectAfterTime, o.KeepaliveTimeout, o.MaxHandshakeAttempts,
		o.RandomTrailers, o.DisableCookies,
	} {
		if v != "" {
			return "3.1"
		}
	}
	return "2"
}

func toAmneziaParams(o domain.ObfuscationParams) amneziaParams {
	return amneziaParams{
		Jc: o.Jc, Jmin: o.Jmin, Jmax: o.Jmax,
		S1: o.S1, S2: o.S2, S3: o.S3, S4: o.S4,
		H1: o.H1, H2: o.H2, H3: o.H3, H4: o.H4,
		I1: o.I1, I2: o.I2, I3: o.I3, I4: o.I4, I5: o.I5,
		HeaderProtectionKey:    o.HeaderProtectionKey,
		ContentPaddingAddition: o.ContentPaddingAddition,
		RekeyAfterTime:         o.RekeyAfterTime,
		RekeyTimeout:           o.RekeyTimeout,
		RejectAfterTime:        o.RejectAfterTime,
		KeepaliveTimeout:       o.KeepaliveTimeout,
		MaxHandshakeAttempts:   o.MaxHandshakeAttempts,
		RandomTrailers:         o.RandomTrailers,
		DisableCookies:         o.DisableCookies,
	}
}

// splitList splits a comma separated setting, such as DNS or AllowedIPs,
// into its trimmed entries.
func splitList(s string) []string {
	var items []string
	for item := range strings.SplitSeq(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
