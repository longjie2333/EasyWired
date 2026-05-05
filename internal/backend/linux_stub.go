//go:build !linux

package backend

func newLinuxNativeOrUnsupported(goos string) Backend {
	return &UnsupportedBackend{name: NameLinuxNative, platform: goos}
}
