package backend

import (
	"context"

	"easywired/internal/model"
)

type UnsupportedBackend struct {
	name     string
	platform string
}

func (b *UnsupportedBackend) Name() string              { return b.name }
func (b *UnsupportedBackend) Platform() string          { return b.platform }
func (b *UnsupportedBackend) SupportsNativeApply() bool { return false }

func (b *UnsupportedBackend) EnsureDevice(context.Context, string, *model.NodeConfig) error {
	return ErrBackendNotSupported
}

func (b *UnsupportedBackend) ApplyInterface(context.Context, string, model.WGInterface) error {
	return ErrBackendNotSupported
}

func (b *UnsupportedBackend) AddOrUpdatePeer(context.Context, string, model.WGPeer) error {
	return ErrBackendNotSupported
}

func (b *UnsupportedBackend) RemovePeer(context.Context, string, string) error {
	return ErrBackendNotSupported
}

func (b *UnsupportedBackend) ApplyConfig(context.Context, string, *model.NodeConfig) error {
	return ErrBackendNotSupported
}

func (b *UnsupportedBackend) ExportConfig(context.Context, *model.NodeConfig) ([]byte, error) {
	return nil, ErrBackendNotSupported
}
