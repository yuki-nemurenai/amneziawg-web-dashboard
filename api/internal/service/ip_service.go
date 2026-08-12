package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

type IPService interface {
	AllocateNextIP(subnetPrefix string, existingPeers []domain.Peer) (string, error)
}

type ipService struct{}

func NewIPService() IPService {
	return &ipService{}
}

// AllocateNextIP parses subnet prefix (e.g., "172.24.170") and finds the lowest available octet starting from .2 up to .254
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

	return "", fmt.Errorf("no available IP addresses left in subnet %s.0/24", subnetPrefix)
}

func ExtractSubnetPrefix(address string) string {
	// Address like "172.24.170.1/24" -> "172.24.170"
	ipOnly := strings.Split(address, "/")[0]
	parts := strings.Split(ipOnly, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[0:3], ".")
	}
	return "172.20.0"
}

func SortPeersByIP(peers []domain.Peer) {
	sort.Slice(peers, func(i, j int) bool {
		octetI := extractLastOctet(peers[i].IP)
		octetJ := extractLastOctet(peers[j].IP)
		return octetI < octetJ
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
