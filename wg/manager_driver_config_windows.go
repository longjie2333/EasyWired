//go:build windows

package wg

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"golang.zx2c4.com/wireguard/windows/driver"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

func buildDriverFullConfiguration(cfg wgtypes.Config) (*driver.Interface, uint32, error) {
	if cfg.PrivateKey == nil {
		return nil, 0, errors.New("wg: missing private key in config")
	}

	iface := driver.Interface{
		Flags:      driver.InterfaceHasPrivateKey,
		PrivateKey: *cfg.PrivateKey,
		PeerCount:  uint32(len(cfg.Peers)),
	}
	if cfg.ReplacePeers {
		iface.Flags |= driver.InterfaceReplacePeers
	}
	if cfg.ListenPort != nil {
		if *cfg.ListenPort < 0 || *cfg.ListenPort > 65535 {
			return nil, 0, fmt.Errorf("wg: invalid listen port: %d", *cfg.ListenPort)
		}
		iface.Flags |= driver.InterfaceHasListenPort
		iface.ListenPort = uint16(*cfg.ListenPort)
	}

	return buildDriverConfiguration(iface, cfg.Peers)
}

func buildDriverPeerHotUpdateConfiguration(peer wgtypes.PeerConfig) (*driver.Interface, uint32, error) {
	iface := driver.Interface{PeerCount: 1}
	return buildDriverConfiguration(iface, []wgtypes.PeerConfig{peer})
}

func buildDriverConfiguration(iface driver.Interface, peers []wgtypes.PeerConfig) (*driver.Interface, uint32, error) {
	var builder driver.ConfigBuilder
	builder.Preallocate(estimateDriverConfigSize(peers))
	builder.AppendInterface(&iface)

	for i := range peers {
		if err := appendDriverPeerConfig(&builder, peers[i]); err != nil {
			return nil, 0, err
		}
	}

	interfaze, size := builder.Interface()
	return interfaze, size, nil
}

func estimateDriverConfigSize(peers []wgtypes.PeerConfig) uint32 {
	size := int(unsafe.Sizeof(driver.Interface{})) + len(peers)*int(unsafe.Sizeof(driver.Peer{}))
	for _, peer := range peers {
		size += len(peer.AllowedIPs) * int(unsafe.Sizeof(driver.AllowedIP{}))
	}
	return uint32(size)
}

func appendDriverPeerConfig(builder *driver.ConfigBuilder, peer wgtypes.PeerConfig) error {
	driverPeer := driver.Peer{
		Flags:           driver.PeerHasPublicKey,
		PublicKey:       peer.PublicKey,
		AllowedIPsCount: uint32(len(peer.AllowedIPs)),
	}

	if peer.PresharedKey != nil {
		driverPeer.Flags |= driver.PeerHasPresharedKey
		driverPeer.PresharedKey = *peer.PresharedKey
	}
	if peer.PersistentKeepaliveInterval != nil {
		seconds, err := keepaliveSeconds(*peer.PersistentKeepaliveInterval)
		if err != nil {
			return err
		}
		driverPeer.Flags |= driver.PeerHasPersistentKeepalive
		driverPeer.PersistentKeepalive = seconds
	}
	if peer.Endpoint != nil {
		endpoint, err := endpointToRawSockaddr(peer.Endpoint)
		if err != nil {
			return err
		}
		driverPeer.Flags |= driver.PeerHasEndpoint
		driverPeer.Endpoint = endpoint
	}
	if peer.ReplaceAllowedIPs {
		driverPeer.Flags |= driver.PeerReplaceAllowedIPs
	}
	if peer.Remove {
		driverPeer.Flags |= driver.PeerRemove
	}
	if peer.UpdateOnly {
		driverPeer.Flags |= driver.PeerUpdateOnly
	}

	builder.AppendPeer(&driverPeer)

	for _, allowedIP := range peer.AllowedIPs {
		nativeAllowedIP, err := allowedIPNetToDriver(allowedIP)
		if err != nil {
			return err
		}
		builder.AppendAllowedIP(&nativeAllowedIP)
	}

	return nil
}

func keepaliveSeconds(interval time.Duration) (uint16, error) {
	if interval < 0 {
		return 0, fmt.Errorf("wg: invalid persistent keepalive: %s", interval)
	}
	if interval%time.Second != 0 {
		return 0, fmt.Errorf("wg: persistent keepalive must be whole seconds: %s", interval)
	}

	seconds := int(interval / time.Second)
	if seconds > 65535 {
		return 0, fmt.Errorf("wg: invalid persistent keepalive: %d", seconds)
	}

	return uint16(seconds), nil
}

func endpointToRawSockaddr(addr *net.UDPAddr) (winipcfg.RawSockaddrInet, error) {
	if addr == nil {
		return winipcfg.RawSockaddrInet{}, errors.New("wg: endpoint is nil")
	}
	if addr.Port < 0 || addr.Port > 65535 {
		return winipcfg.RawSockaddrInet{}, fmt.Errorf("wg: invalid endpoint port in %q", addr.String())
	}

	ip, ok := netip.AddrFromSlice(addr.IP)
	if !ok {
		return winipcfg.RawSockaddrInet{}, fmt.Errorf("wg: invalid endpoint ip in %q", addr.String())
	}
	if ip.Is4In6() {
		ip = ip.Unmap()
	}
	if ip.Is6() && addr.Zone != "" {
		ip = ip.WithZone(addr.Zone)
	}

	var out winipcfg.RawSockaddrInet
	if err := out.SetAddrPort(netip.AddrPortFrom(ip, uint16(addr.Port))); err != nil {
		return winipcfg.RawSockaddrInet{}, fmt.Errorf("wg: build endpoint sockaddr: %w", err)
	}
	return out, nil
}

func allowedIPNetToDriver(network net.IPNet) (driver.AllowedIP, error) {
	cidrBits, ipBits := network.Mask.Size()
	if cidrBits < 0 {
		return driver.AllowedIP{}, fmt.Errorf("wg: invalid allowed ip mask: %s", network.String())
	}

	out := driver.AllowedIP{Cidr: uint8(cidrBits)}
	if ipv4 := network.IP.To4(); ipv4 != nil {
		if ipBits != 32 {
			return driver.AllowedIP{}, fmt.Errorf("wg: invalid ipv4 allowed ip %s", network.String())
		}
		out.AddressFamily = windows.AF_INET
		copy(out.Address[:4], ipv4)
		return out, nil
	}

	ipv6 := network.IP.To16()
	if ipv6 == nil || ipBits != 128 {
		return driver.AllowedIP{}, fmt.Errorf("wg: invalid ipv6 allowed ip %s", network.String())
	}

	out.AddressFamily = windows.AF_INET6
	copy(out.Address[:], ipv6)
	return out, nil
}

func toWGConfig(privKey PrivateKey, cfg *NodeConfig) (wgtypes.Config, error) {
	peers := make([]wgtypes.PeerConfig, 0, len(cfg.Peers))
	for i := range cfg.Peers {
		peerConfig, err := toWGPeerConfig(cfg.Peers[i])
		if err != nil {
			return wgtypes.Config{}, err
		}
		peers = append(peers, peerConfig)
	}

	privateKey := privKey
	out := wgtypes.Config{
		PrivateKey:   &privateKey,
		ReplacePeers: true,
		Peers:        peers,
	}

	if cfg.ListenPort > 0 {
		listenPort := cfg.ListenPort
		out.ListenPort = &listenPort
	}

	return out, nil
}

func toWGPeerConfig(peer PeerEntry) (wgtypes.PeerConfig, error) {
	publicKey, err := parseWGKey(peer.PublicKey, "public")
	if err != nil {
		return wgtypes.PeerConfig{}, err
	}

	out := wgtypes.PeerConfig{
		PublicKey:  publicKey,
		Remove:     peer.Remove,
		UpdateOnly: peer.UpdateOnly,
	}

	if preshared := strings.TrimSpace(peer.PresharedKey); preshared != "" {
		presharedKey, err := parseWGKey(preshared, "preshared")
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
		out.PresharedKey = &presharedKey
	}
	if peer.PersistentKeepalive > 0 {
		keepalive := time.Duration(peer.PersistentKeepalive) * time.Second
		out.PersistentKeepaliveInterval = &keepalive
	}
	if endpoint := strings.TrimSpace(peer.Endpoint); endpoint != "" {
		udpAddr, err := net.ResolveUDPAddr("udp", endpoint)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("wg: parse endpoint %q: %w", endpoint, err)
		}
		out.Endpoint = udpAddr
	}

	allowedIPs, err := parseAllowedIPNets(peer.normalizedAllowedIPs())
	if err != nil {
		return wgtypes.PeerConfig{}, err
	}
	out.AllowedIPs = allowedIPs
	if len(allowedIPs) > 0 {
		out.ReplaceAllowedIPs = true
	}

	return out, nil
}

func effectiveDNS(cfg *NodeConfig) []string {
	if cfg == nil {
		return nil
	}
	if len(cfg.DNS) > 0 {
		return cfg.DNS
	}
	return cfg.SuggestedDNS
}

func effectiveMTU(cfg *NodeConfig) int {
	if cfg == nil {
		return 0
	}
	if cfg.MTU > 0 {
		return cfg.MTU
	}
	return cfg.SuggestedMTU
}

func collectInterfaceAddresses(cfg *NodeConfig) ([]net.IPNet, error) {
	if cfg == nil {
		return nil, nil
	}

	raw := make([]string, 0, 1+len(cfg.Addresses))
	if addr := strings.TrimSpace(cfg.Address); addr != "" {
		raw = append(raw, addr)
	}
	raw = append(raw, cfg.Addresses...)

	out := make([]net.IPNet, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		network, err := parseAllowedIPNet(item)
		if err != nil {
			return nil, fmt.Errorf("wg: invalid interface address %q: %w", item, err)
		}
		key := network.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, network)
	}

	return out, nil
}

func collectPeerRoutes(cfg wgtypes.Config) []net.IPNet {
	out := make([]net.IPNet, 0)
	seen := map[string]struct{}{}
	for _, peer := range cfg.Peers {
		for _, route := range peer.AllowedIPs {
			key := route.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, route)
		}
	}
	return out
}

func parseWGKey(value string, label string) (wgtypes.Key, error) {
	key := strings.TrimSpace(value)
	if key == "" {
		return wgtypes.Key{}, fmt.Errorf("wg: empty %s key", label)
	}

	parsed, err := wgtypes.ParseKey(key)
	if err != nil {
		return wgtypes.Key{}, fmt.Errorf("wg: parse %s key: %w", label, err)
	}

	return parsed, nil
}

func parseAllowedIPNets(values []string) ([]net.IPNet, error) {
	out := make([]net.IPNet, 0, len(values))
	for _, value := range values {
		network, err := parseAllowedIPNet(value)
		if err != nil {
			return nil, err
		}
		out = append(out, network)
	}
	return out, nil
}

func parseAllowedIPNet(value string) (net.IPNet, error) {
	cidr := strings.TrimSpace(value)
	if cidr == "" {
		return net.IPNet{}, errors.New("wg: empty allowed ip")
	}

	if strings.Contains(cidr, "/") {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return net.IPNet{}, fmt.Errorf("wg: parse allowed ip %q: %w", cidr, err)
		}

		if ipv4 := network.IP.To4(); ipv4 != nil {
			network.IP = ipv4
			return *network, nil
		}

		ipv6 := network.IP.To16()
		if ipv6 == nil {
			return net.IPNet{}, fmt.Errorf("wg: invalid allowed ip %q", cidr)
		}
		network.IP = ipv6
		return *network, nil
	}

	ip := net.ParseIP(cidr)
	if ip == nil {
		return net.IPNet{}, fmt.Errorf("wg: invalid allowed ip %q", cidr)
	}

	if ipv4 := ip.To4(); ipv4 != nil {
		return net.IPNet{IP: ipv4, Mask: net.CIDRMask(32, 32)}, nil
	}

	ipv6 := ip.To16()
	if ipv6 == nil {
		return net.IPNet{}, fmt.Errorf("wg: invalid allowed ip %q", cidr)
	}

	return net.IPNet{IP: ipv6, Mask: net.CIDRMask(128, 128)}, nil
}
