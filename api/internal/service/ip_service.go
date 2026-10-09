package service

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

// IPService picks addresses for new clients.
type IPService interface {
	AllocateNextIP(subnetPrefix string, existingPeers []domain.Peer) (string, error)
}

type ipService struct{}

// NewIPService returns an IPService that hands out addresses of a /24 subnet.
func NewIPService() IPService {
	return &ipService{}
}

// AllocateNextIP returns the lowest free address from .2 to .254 of the subnet
// with the given first three octets, such as "172.24.170". The server holds .1.
func (s *ipService) AllocateNextIP(subnetPrefix string, existingPeers []domain.Peer) (string, error) {
	usedOctets := make(map[int]bool)

	for _, peer := range existingPeers {
		if peer.IP == "" {
			continue
		}
		parts := strings.Split(peer.IP, ".")
		if len(parts) == 4 {
			if octet, err := strconv.Atoi(parts[3]); err == nil {
				usedOctets[octet] = true
			}
		}
	}

	// Server interface address is usually .1
	usedOctets[1] = true

	for octet := 2; octet <= 254; octet++ {
		if !usedOctets[octet] {
			return fmt.Sprintf("%s.%d", subnetPrefix, octet), nil
		}
	}

	return "", fmt.Errorf("%w: no free IP address left in subnet %s.0/24", domain.ErrConflict, subnetPrefix)
}

// ExtractSubnetPrefix returns the first three octets of an interface address,
// "172.24.170" for "172.24.170.1/24", or "172.20.0" if address is not IPv4.
func ExtractSubnetPrefix(address string) string {
	ipOnly, _, _ := strings.Cut(address, "/")
	parts := strings.Split(ipOnly, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[0:3], ".")
	}
	return "172.20.0"
}

// SortPeersByIP sorts peers by the last octet of their address, the order the
// dashboard lists clients in.
func SortPeersByIP(peers []domain.Peer) {
	slices.SortFunc(peers, func(a, b domain.Peer) int {
		return cmp.Compare(extractLastOctet(a.IP), extractLastOctet(b.IP))
	})
}

func extractLastOctet(ip string) int {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		v, _ := strconv.Atoi(parts[3])
		return v
	}
	return 0
}
