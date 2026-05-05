//go:build windows

package backend

import (
	"context"
	"runtime"

	"easywired/internal/model"
	"easywired/internal/util"
)

// WindowsServiceBackend is reserved for WireGuard embeddable-dll-service and
// wireguard-windows enterprise management integration.
type WindowsServiceBackend struct{}

func newWindowsServiceBackend(string) Backend          { return &WindowsServiceBackend{} }
func NewWindowsServiceBackend() *WindowsServiceBackend { return &WindowsServiceBackend{} }

func (b *WindowsServiceBackend) Name() string              { return NameWindowsService }
func (b *WindowsServiceBackend) Platform() string          { return runtime.GOOS }
func (b *WindowsServiceBackend) SupportsNativeApply() bool { return true }

func (b *WindowsServiceBackend) EnsureDevice(context.Context, string, *model.NodeConfig) error {
	return notImplemented(NameWindowsService)
}

func (b *WindowsServiceBackend) ApplyInterface(context.Context, string, model.WGInterface) error {
	return notImplemented(NameWindowsService)
}

func (b *WindowsServiceBackend) AddOrUpdatePeer(context.Context, string, model.WGPeer) error {
	return notImplemented(NameWindowsService)
}

func (b *WindowsServiceBackend) RemovePeer(context.Context, string, string) error {
	return notImplemented(NameWindowsService)
}

func (b *WindowsServiceBackend) ApplyConfig(context.Context, string, *model.NodeConfig) error {
	return notImplemented(NameWindowsService)
}

func (b *WindowsServiceBackend) ExportConfig(ctx context.Context, cfg *model.NodeConfig) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return util.ExportWGQuick(cfg)
	}
}

func (b *WindowsServiceBackend) InstallTunnelService(confPath string) error {
	return InstallTunnelService(confPath)
}

func (b *WindowsServiceBackend) UninstallTunnelService(name string) error {
	return UninstallTunnelService(name)
}

func (b *WindowsServiceBackend) UpdateTunnelConfig(confPath string) error {
	return UpdateTunnelConfig(confPath)
}

func InstallTunnelService(confPath string) error { return notImplemented(NameWindowsService) }
func UninstallTunnelService(name string) error   { return notImplemented(NameWindowsService) }
func UpdateTunnelConfig(confPath string) error   { return notImplemented(NameWindowsService) }
