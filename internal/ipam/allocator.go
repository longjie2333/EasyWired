package ipam

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	"easywired/internal/model"
	"easywired/internal/util"
)

var (
	ErrNoAddressPool = errors.New("no address pool")
	ErrNoAvailableIP = errors.New("no available ip")
)

type Allocator struct {
	mu sync.Mutex
}

func NewAllocator() *Allocator {
	return &Allocator{}
}

func (a *Allocator) Allocate(cfg *model.NodeConfig, nodeID, publicKey string) (model.Lease, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, lease := range cfg.Leases {
		if lease.PublicKey == publicKey || nodeID != "" && lease.NodeID == nodeID {
			return lease, nil
		}
	}
	ip, network, err := net.ParseCIDR(cfg.Interface.Address)
	if err != nil {
		return model.Lease{}, err
	}
	base := ip.To4()
	if base == nil {
		return model.Lease{}, ErrNoAddressPool
	}
	ones, bits := network.Mask.Size()
	if bits != 32 || ones >= 32 {
		return model.Lease{}, ErrNoAddressPool
	}
	used := usedIPs(cfg)
	start := ipToUint32(network.IP.To4()) + 1
	end := start + (1 << uint(32-ones)) - 3
	for n := start; n <= end; n++ {
		candidate := uint32ToIP(n).String()
		if used[candidate] {
			continue
		}
		lease := model.Lease{
			NodeID:    nodeID,
			PublicKey: publicKey,
			Address:   candidate + "/32",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		cfg.Leases = append(cfg.Leases, lease)
		return lease, nil
	}
	return model.Lease{}, ErrNoAvailableIP
}

func usedIPs(cfg *model.NodeConfig) map[string]bool {
	used := map[string]bool{}
	if ip := util.AddressIP(cfg.Interface.Address); ip != "" {
		used[ip] = true
	}
	for _, peer := range cfg.Peers {
		if ip := util.AddressIP(peer.Address); ip != "" {
			used[ip] = true
		}
		for _, allowed := range peer.AllowedIPs {
			if ip := util.AddressIP(allowed); ip != "" {
				used[ip] = true
			}
		}
	}
	for _, lease := range cfg.Leases {
		if ip := util.AddressIP(lease.Address); ip != "" {
			used[ip] = true
		}
	}
	return used
}

func ipToUint32(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
