package server

import (
	"encoding/binary"
	"net"
	"strings"
)

func allocateIPv4(subnetCIDR string, usedIPs []string) string {
	_, network, err := net.ParseCIDR(strings.TrimSpace(subnetCIDR))
	if err != nil || network == nil {
		return ""
	}

	baseIPv4 := network.IP.To4()
	if baseIPv4 == nil || len(network.Mask) != net.IPv4len {
		return ""
	}

	mask := binary.BigEndian.Uint32(network.Mask)
	networkAddr := binary.BigEndian.Uint32(baseIPv4) & mask
	broadcast := networkAddr | ^mask
	if broadcast-networkAddr <= 1 {
		return ""
	}

	usedSet := make(map[uint32]struct{}, len(usedIPs))
	for _, raw := range usedIPs {
		host := strings.TrimSpace(raw)
		if host == "" {
			continue
		}

		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}

		ipv4 := net.ParseIP(host).To4()
		if ipv4 == nil {
			continue
		}

		usedSet[binary.BigEndian.Uint32(ipv4)] = struct{}{}
	}

	for current := networkAddr + 1; current < broadcast; current++ {
		if _, exists := usedSet[current]; exists {
			continue
		}

		out := make(net.IP, net.IPv4len)
		binary.BigEndian.PutUint32(out, current)
		return out.String()
	}

	return ""
}

func cidrPrefix(subnetCIDR string) int {
	_, network, err := net.ParseCIDR(strings.TrimSpace(subnetCIDR))
	if err != nil || network == nil {
		return 0
	}

	ones, _ := network.Mask.Size()
	return ones
}
