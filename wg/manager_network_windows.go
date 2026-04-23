//go:build windows

package wg

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

func applyInterfaceNetworkConfig(luid winipcfg.LUID, cfg *NodeConfig, wgConfig wgtypes.Config) error {
	addresses, err := collectInterfacePrefixes(cfg)
	if err != nil {
		return err
	}

	routes, err := collectPeerRoutePrefixes(wgConfig)
	if err != nil {
		return err
	}

	dnsServers := collectDNSServers(cfg)
	mtu := effectiveMTU(cfg)

	families := []winipcfg.AddressFamily{windows.AF_INET, windows.AF_INET6}
	for _, family := range families {
		if err := configureInterfaceFamily(luid, family, addresses, routes, dnsServers, mtu); err != nil {
			return err
		}
	}

	return nil
}

func configureInterfaceFamily(
	luid winipcfg.LUID,
	family winipcfg.AddressFamily,
	addresses []netip.Prefix,
	routes []netip.Prefix,
	dnsServers []netip.Addr,
	mtu int,
) error {
	familyRoutes, hasDefaultRoute := collectFamilyRoutes(routes, family)
	if len(familyRoutes) > 0 {
		if err := luid.SetRoutesForFamily(family, familyRoutes); err != nil {
			return fmt.Errorf("wg: set routes for family %d: %w", family, err)
		}
	}

	familyAddresses := filterPrefixesByFamily(addresses, family)
	if len(familyAddresses) > 0 {
		if err := luid.SetIPAddressesForFamily(family, familyAddresses); err != nil {
			return fmt.Errorf("wg: set addresses for family %d: %w", family, err)
		}
	}

	if len(familyRoutes) > 0 || len(familyAddresses) > 0 {
		ipif, err := luid.IPInterface(family)
		if err != nil {
			return fmt.Errorf("wg: query interface for family %d: %w", family, err)
		}

		ipif.RouterDiscoveryBehavior = winipcfg.RouterDiscoveryDisabled
		ipif.DadTransmits = 0
		ipif.ManagedAddressConfigurationSupported = false
		ipif.OtherStatefulConfigurationSupported = false

		if mtu > 0 {
			ipif.NLMTU = uint32(mtu)
		}
		if hasDefaultRoute {
			ipif.UseAutomaticMetric = false
			ipif.Metric = 0
		}
		if err := ipif.Set(); err != nil {
			return fmt.Errorf("wg: set interface params for family %d: %w", family, err)
		}
	}

	familyDNS := filterDNSByFamily(dnsServers, family)
	if len(familyDNS) > 0 {
		if err := luid.SetDNS(family, familyDNS, nil); err != nil {
			return fmt.Errorf("wg: set dns for family %d: %w", family, err)
		}
	}

	return nil
}

func collectFamilyRoutes(routes []netip.Prefix, family winipcfg.AddressFamily) ([]*winipcfg.RouteData, bool) {
	out := make([]*winipcfg.RouteData, 0, len(routes))
	hasDefaultRoute := false

	for _, route := range routes {
		if !isAddrInFamily(route.Addr(), family) {
			continue
		}

		routeData := winipcfg.RouteData{
			Destination: route.Masked(),
			NextHop:     unspecifiedNextHopForFamily(family),
			Metric:      0,
		}
		if route.Bits() == 0 {
			hasDefaultRoute = true
		}
		out = append(out, &routeData)
	}

	return out, hasDefaultRoute
}

func filterPrefixesByFamily(prefixes []netip.Prefix, family winipcfg.AddressFamily) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		if isAddrInFamily(prefix.Addr(), family) {
			out = append(out, prefix)
		}
	}
	return out
}

func filterDNSByFamily(dnsServers []netip.Addr, family winipcfg.AddressFamily) []netip.Addr {
	out := make([]netip.Addr, 0, len(dnsServers))
	for _, server := range dnsServers {
		if isAddrInFamily(server, family) {
			out = append(out, server)
		}
	}
	return out
}

func collectInterfacePrefixes(cfg *NodeConfig) ([]netip.Prefix, error) {
	networks, err := collectInterfaceAddresses(cfg)
	if err != nil {
		return nil, err
	}
	return ipNetsToPrefixes(networks, true)
}

func collectPeerRoutePrefixes(cfg wgtypes.Config) ([]netip.Prefix, error) {
	return ipNetsToPrefixes(collectPeerRoutes(cfg), false)
}

func ipNetsToPrefixes(networks []net.IPNet, deduplicate bool) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(networks))
	seen := map[string]struct{}{}
	if !deduplicate {
		seen = nil
	}

	for _, network := range networks {
		prefix, err := ipNetToPrefix(network)
		if err != nil {
			return nil, err
		}
		if seen != nil {
			key := prefix.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
		}
		out = append(out, prefix)
	}

	return out, nil
}

func ipNetToPrefix(network net.IPNet) (netip.Prefix, error) {
	cidrBits, ipBits := network.Mask.Size()
	if cidrBits < 0 {
		return netip.Prefix{}, fmt.Errorf("wg: invalid ip mask: %s", network.String())
	}

	if ipv4 := network.IP.To4(); ipv4 != nil {
		if ipBits != 32 {
			return netip.Prefix{}, fmt.Errorf("wg: invalid ipv4 network: %s", network.String())
		}
		addr, ok := netip.AddrFromSlice(ipv4)
		if !ok {
			return netip.Prefix{}, fmt.Errorf("wg: invalid ipv4 address: %s", network.String())
		}
		return netip.PrefixFrom(addr.Unmap(), cidrBits), nil
	}

	ipv6 := network.IP.To16()
	if ipv6 == nil || ipBits != 128 {
		return netip.Prefix{}, fmt.Errorf("wg: invalid ipv6 network: %s", network.String())
	}
	addr, ok := netip.AddrFromSlice(ipv6)
	if !ok {
		return netip.Prefix{}, fmt.Errorf("wg: invalid ipv6 address: %s", network.String())
	}
	return netip.PrefixFrom(addr, cidrBits), nil
}

func collectDNSServers(cfg *NodeConfig) []netip.Addr {
	raw := effectiveDNS(cfg)
	out := make([]netip.Addr, 0, len(raw))
	seen := make(map[netip.Addr]struct{}, len(raw))

	for _, item := range raw {
		ip := net.ParseIP(strings.TrimSpace(item))
		if ip == nil {
			continue
		}
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}
		if addr.Is4In6() {
			addr = addr.Unmap()
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}

	return out
}

func isAddrInFamily(addr netip.Addr, family winipcfg.AddressFamily) bool {
	switch family {
	case windows.AF_INET:
		return addr.Is4()
	case windows.AF_INET6:
		return addr.Is6()
	default:
		return false
	}
}

func unspecifiedNextHopForFamily(family winipcfg.AddressFamily) netip.Addr {
	if family == windows.AF_INET6 {
		return netip.IPv6Unspecified()
	}
	return netip.IPv4Unspecified()
}
