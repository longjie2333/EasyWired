package backend

import (
	"context"
	"runtime"

	"easywired/internal/model"
	"easywired/internal/util"
)

type WGConfigFileBackend struct{}

func NewWGConfigFile() *WGConfigFileBackend { return &WGConfigFileBackend{} }

func (b *WGConfigFileBackend) Name() string              { return NameWGConfigFile }
func (b *WGConfigFileBackend) Platform() string          { return runtime.GOOS }
func (b *WGConfigFileBackend) SupportsNativeApply() bool { return false }

func (b *WGConfigFileBackend) EnsureDevice(context.Context, string, *model.NodeConfig) error {
	return nil
}

func (b *WGConfigFileBackend) ApplyInterface(context.Context, string, model.WGInterface) error {
	return nil
}

func (b *WGConfigFileBackend) AddOrUpdatePeer(context.Context, string, model.WGPeer) error {
	return nil
}

func (b *WGConfigFileBackend) RemovePeer(context.Context, string, string) error {
	return nil
}

func (b *WGConfigFileBackend) ApplyConfig(context.Context, string, *model.NodeConfig) error {
	return nil
}

func (b *WGConfigFileBackend) ExportConfig(ctx context.Context, cfg *model.NodeConfig) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return util.ExportWGQuick(cfg)
	}
}
