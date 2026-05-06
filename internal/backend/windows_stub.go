//go:build !windows

package backend

import (
	"context"

	"easywired/internal/model"
	"easywired/internal/util"
)

type windowsReservedBackend struct {
	name     string
	platform string
}

func newWindowsServiceBackend(goos string) Backend {
	return &windowsReservedBackend{name: NameWindowsService, platform: goos}
}

func newWindowsNTBackend(goos string) Backend {
	return &windowsReservedBackend{name: NameWindowsNT, platform: goos}
}

func (b *windowsReservedBackend) Name() string              { return b.name }
func (b *windowsReservedBackend) Platform() string          { return b.platform }
func (b *windowsReservedBackend) SupportsNativeApply() bool { return true }

func (b *windowsReservedBackend) EnsureDevice(context.Context, string, *model.NodeConfig) error {
	return notImplemented(b.name)
}

func (b *windowsReservedBackend) ApplyInterface(context.Context, string, model.WGInterface) error {
	return notImplemented(b.name)
}

func (b *windowsReservedBackend) AddOrUpdatePeer(context.Context, string, model.WGPeer) error {
	return notImplemented(b.name)
}

func (b *windowsReservedBackend) RemovePeer(context.Context, string, string) error {
	return notImplemented(b.name)
}

func (b *windowsReservedBackend) ApplyConfig(context.Context, string, *model.NodeConfig) error {
	return notImplemented(b.name)
}

func (b *windowsReservedBackend) ExportConfig(ctx context.Context, cfg *model.NodeConfig) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return util.ExportWGQuick(cfg)
	}
}
