//go:build windows

package backend

import (
	"context"
	"runtime"

	"easywired/internal/model"
	"easywired/internal/util"
)

// WindowsNTBackend is reserved for WireGuardNT, a lower-level API not used initially.
type WindowsNTBackend struct{}

func newWindowsNTBackend(string) Backend     { return &WindowsNTBackend{} }
func NewWindowsNTBackend() *WindowsNTBackend { return &WindowsNTBackend{} }

func (b *WindowsNTBackend) Name() string              { return NameWindowsNT }
func (b *WindowsNTBackend) Platform() string          { return runtime.GOOS }
func (b *WindowsNTBackend) SupportsNativeApply() bool { return true }

func (b *WindowsNTBackend) EnsureDevice(context.Context, string, *model.NodeConfig) error {
	return notImplemented(NameWindowsNT)
}

func (b *WindowsNTBackend) ApplyInterface(context.Context, string, model.WGInterface) error {
	return notImplemented(NameWindowsNT)
}

func (b *WindowsNTBackend) AddOrUpdatePeer(context.Context, string, model.WGPeer) error {
	return notImplemented(NameWindowsNT)
}

func (b *WindowsNTBackend) RemovePeer(context.Context, string, string) error {
	return notImplemented(NameWindowsNT)
}

func (b *WindowsNTBackend) ApplyConfig(context.Context, string, *model.NodeConfig) error {
	return notImplemented(NameWindowsNT)
}

func (b *WindowsNTBackend) ExportConfig(ctx context.Context, cfg *model.NodeConfig) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return util.ExportWGQuick(cfg)
	}
}
