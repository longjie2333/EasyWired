//go:build linux

package backend

import (
	"context"
	"errors"
	"net"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"easywired/internal/model"
	"easywired/internal/util"
)

type LinuxNativeBackend struct{}

func newLinuxNativeOrUnsupported(goos string) Backend   { return &LinuxNativeBackend{} }
func NewLinuxNative() *LinuxNativeBackend               { return &LinuxNativeBackend{} }
func (b *LinuxNativeBackend) Name() string              { return NameLinuxNative }
func (b *LinuxNativeBackend) Platform() string          { return runtime.GOOS }
func (b *LinuxNativeBackend) SupportsNativeApply() bool { return true }

func (b *LinuxNativeBackend) EnsureDevice(ctx context.Context, deviceName string, cfg *model.NodeConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := netlink.LinkByName(deviceName); err == nil {
		return nil
	}
	attrs := netlink.NewLinkAttrs()
	attrs.Name = deviceName
	link := &netlink.GenericLink{LinkAttrs: attrs, LinkType: "wireguard"}
	if err := netlink.LinkAdd(link); err != nil && !alreadyExists(err) {
		return err
	}
	return nil
}

func (b *LinuxNativeBackend) ApplyInterface(ctx context.Context, deviceName string, iface model.WGInterface) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	link, err := netlink.LinkByName(deviceName)
	if err != nil {
		return err
	}
	if iface.Address != "" {
		addr, err := netlink.ParseAddr(iface.Address)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(link, addr); err != nil && !alreadyExists(err) {
			return err
		}
	}
	if iface.MTU > 0 {
		if err := netlink.LinkSetMTU(link, iface.MTU); err != nil {
			return err
		}
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return err
	}
	return b.applyWGInterface(deviceName, iface)
}

func (b *LinuxNativeBackend) AddOrUpdatePeer(ctx context.Context, deviceName string, peer model.WGPeer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key, err := wgtypes.ParseKey(peer.PublicKey)
	if err != nil {
		return err
	}
	allowed, err := parseAllowedIPs(peer.AllowedIPs)
	if err != nil {
		return err
	}
	pc := wgtypes.PeerConfig{PublicKey: key, ReplaceAllowedIPs: true, AllowedIPs: allowed}
	if peer.Endpoint != "" {
		endpoint, err := net.ResolveUDPAddr("udp", peer.Endpoint)
		if err != nil {
			return err
		}
		pc.Endpoint = endpoint
	}
	if peer.PersistentKeepalive > 0 {
		d := time.Duration(peer.PersistentKeepalive) * time.Second
		pc.PersistentKeepaliveInterval = &d
	}
	return configureDevice(deviceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{pc}})
}

func (b *LinuxNativeBackend) RemovePeer(ctx context.Context, deviceName string, publicKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return err
	}
	return configureDevice(deviceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{{PublicKey: key, Remove: true}}})
}

func (b *LinuxNativeBackend) ApplyConfig(ctx context.Context, deviceName string, cfg *model.NodeConfig) error {
	if err := b.EnsureDevice(ctx, deviceName, cfg); err != nil {
		return err
	}
	if err := b.ApplyInterface(ctx, deviceName, cfg.Interface); err != nil {
		return err
	}
	for _, peer := range cfg.Peers {
		if err := b.AddOrUpdatePeer(ctx, deviceName, peer); err != nil {
			return err
		}
	}
	return nil
}

func (b *LinuxNativeBackend) ExportConfig(ctx context.Context, cfg *model.NodeConfig) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return util.ExportWGQuick(cfg)
	}
}

func (b *LinuxNativeBackend) applyWGInterface(deviceName string, iface model.WGInterface) error {
	privateKey, err := wgtypes.ParseKey(iface.PrivateKey)
	if err != nil {
		return err
	}
	cfg := wgtypes.Config{PrivateKey: &privateKey}
	if iface.ListenPort > 0 {
		cfg.ListenPort = &iface.ListenPort
	}
	return configureDevice(deviceName, cfg)
}

func configureDevice(deviceName string, cfg wgtypes.Config) error {
	client, err := wgctrl.New()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.ConfigureDevice(deviceName, cfg)
}

func parseAllowedIPs(values []string) ([]net.IPNet, error) {
	out := make([]net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, err
		}
		out = append(out, *network)
	}
	return out, nil
}

func alreadyExists(err error) bool {
	return err != nil && (errors.Is(err, syscall.EEXIST) || strings.Contains(strings.ToLower(err.Error()), "file exists"))
}
