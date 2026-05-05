//go:build !windows

package backend

func newWindowsServiceBackend(goos string) Backend {
	return &UnsupportedBackend{name: NameWindowsService, platform: goos}
}

func newWindowsNTBackend(goos string) Backend {
	return &UnsupportedBackend{name: NameWindowsNT, platform: goos}
}
