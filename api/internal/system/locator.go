package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

const (
	lookupTimeout = 3 * time.Second
	publicIPTTL   = 10 * time.Minute
	locationTTL   = 24 * time.Hour
	locationURL   = "http://ip-api.com/json/"
)

// publicIPServices answer a GET request with the caller's IP address. Several
// are tried because any of them can be down or blocked.
var publicIPServices = []string{
	"http://api.ipify.org",
	"http://checkip.amazonaws.com",
	"http://icanhazip.com",
	"http://ifconfig.me/ip",
	"http://v4.ident.me",
	"http://ip-api.com/line?fields=query",
	"https://api.ipify.org",
	"https://checkip.amazonaws.com",
}

// Locator finds the public IPv4 address and the country of the server by
// asking public HTTP services. Answers are cached because they rarely change
// and a lookup can take seconds.
type Locator struct {
	staticIP string
	client   *http.Client

	mu         sync.Mutex
	ip         string
	ipAt       time.Time
	location   *domain.ServerLocation
	locationAt time.Time
}

// NewLocator returns a Locator. A valid IPv4 staticIP, the PUBLIC_IP setting,
// is returned as the public IP without asking anyone, for servers behind NAT
// or without outbound HTTP.
func NewLocator(staticIP string) *Locator {
	l := &Locator{client: newIPv4HTTPClient(lookupTimeout)}
	staticIP = strings.TrimSpace(staticIP)
	if ip := net.ParseIP(staticIP); ip != nil && ip.To4() != nil {
		l.staticIP = staticIP
	}
	return l
}

// newIPv4HTTPClient returns a client that dials over IPv4 only, because in
// Docker an IPv6 attempt can hang for the whole timeout.
func newIPv4HTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: timeout,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		DisableKeepAlives: true,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// PublicIP returns the public IPv4 address of the server.
func (l *Locator) PublicIP(ctx context.Context) (string, error) {
	if l.staticIP != "" {
		return l.staticIP, nil
	}

	l.mu.Lock()
	if l.ip != "" && time.Since(l.ipAt) < publicIPTTL {
		ip := l.ip
		l.mu.Unlock()
		return ip, nil
	}
	l.mu.Unlock()

	for _, url := range publicIPServices {
		ip, err := l.askPublicIP(ctx, url)
		if err != nil {
			slog.Debug("Public IP lookup failed", "url", url, "error", err)
			continue
		}

		l.mu.Lock()
		l.ip = ip
		l.ipAt = time.Now()
		l.mu.Unlock()
		slog.Info("Auto-detected server public IP", "ip", ip, "source", url)
		return ip, nil
	}

	return "", fmt.Errorf("detect public IP: none of %d services answered with an IPv4 address", len(publicIPServices))
}

func (l *Locator) askPublicIP(ctx context.Context, url string) (string, error) {
	body, err := l.get(ctx, url, 128)
	if err != nil {
		return "", err
	}

	raw := strings.TrimSpace(string(body))
	for _, r := range []string{"{", "}", "\"", "ip", ":", " ", "\n", "\r"} {
		raw = strings.ReplaceAll(raw, r, "")
	}

	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() == nil || ip.IsLoopback() {
		return "", fmt.Errorf("answer %q is not a public IPv4 address", raw)
	}
	return ip.String(), nil
}

// Location returns the country of the server public IP.
func (l *Locator) Location(ctx context.Context) (*domain.ServerLocation, error) {
	l.mu.Lock()
	if l.location != nil && time.Since(l.locationAt) < locationTTL {
		loc := l.location
		l.mu.Unlock()
		return loc, nil
	}
	l.mu.Unlock()

	body, err := l.get(ctx, locationURL, 4096)
	if err != nil {
		return nil, fmt.Errorf("detect location: %w", err)
	}

	var result struct {
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("detect location: %w", err)
	}
	if result.CountryCode == "" {
		return nil, errors.New("detect location: answer has no country code")
	}

	loc := &domain.ServerLocation{
		Country:     result.Country,
		CountryCode: result.CountryCode,
	}
	l.mu.Lock()
	l.location = loc
	l.locationAt = time.Now()
	l.mu.Unlock()
	return loc, nil
}

// get returns at most limit bytes of the body of a successful GET request.
func (l *Locator) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}
