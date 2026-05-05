package backend

import (
	"context"

	"easywired/internal/model"
)

const (
	NameAuto           = "auto"
	NameLinuxNative    = "linux-native"
	NameWGConfigFile   = "wgconfig-file"
	NameWindowsService = "windows-service"
	NameWindowsNT      = "windows-nt"
	NameDarwinKit      = "darwin-kit"
	NameAndroidTunnel  = "android-tunnel"
)

type Backend interface {
	Name() string
	Platform() string
	EnsureDevice(ctx context.Context, deviceName string, cfg *model.NodeConfig) error
	ApplyInterface(ctx context.Context, deviceName string, iface model.WGInterface) error
	AddOrUpdatePeer(ctx context.Context, deviceName string, peer model.WGPeer) error
	RemovePeer(ctx context.Context, deviceName string, publicKey string) error
	ApplyConfig(ctx context.Context, deviceName string, cfg *model.NodeConfig) error
	ExportConfig(ctx context.Context, cfg *model.NodeConfig) ([]byte, error)
	SupportsNativeApply() bool
}
