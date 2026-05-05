package util

import (
	"fmt"
	"net"
	"strings"
)

func HostIP(cidr string) (net.IP, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	if v4 := ip.To4(); v4 != nil {
		return v4, nil
	}
	return nil, fmt.Errorf("only ipv4 is supported: %s", cidr)
}

func HostCIDR32(cidr string) (string, error) {
	ip, err := HostIP(cidr)
	if err != nil {
		return "", err
	}
	return ip.String() + "/32", nil
}

func AddressIP(address string) string {
	if i := strings.IndexByte(address, '/'); i >= 0 {
		return address[:i]
	}
	return address
}

func WithPrefix32(address string) string {
	ip := AddressIP(address)
	if ip == "" {
		return ""
	}
	return ip + "/32"
}

func ContainsString(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}
