package backend

import (
	"fmt"
	"runtime"
)

func Select(name string) (Backend, error) {
	return SelectForOS(name, runtime.GOOS)
}

func SelectForOS(name, goos string) (Backend, error) {
	if name == "" || name == NameAuto {
		if goos == "linux" {
			return newLinuxNativeOrUnsupported(goos), nil
		}
		return NewWGConfigFile(), nil
	}
	switch name {
	case NameWGConfigFile:
		return NewWGConfigFile(), nil
	case NameLinuxNative:
		if goos != "linux" {
			return nil, unsupported(name, goos)
		}
		return newLinuxNativeOrUnsupported(goos), nil
	case NameWindowsService:
		if goos != "windows" {
			return nil, unsupported(name, goos)
		}
		return newWindowsServiceBackend(goos), nil
	case NameWindowsNT:
		if goos != "windows" {
			return nil, unsupported(name, goos)
		}
		return newWindowsNTBackend(goos), nil
	case NameDarwinKit, NameAndroidTunnel:
		return nil, notImplemented(name)
	default:
		return nil, unsupported(name, goos)
	}
}

func unsupported(name, goos string) error {
	return &BackendError{
		Code:    "BACKEND_NOT_SUPPORTED",
		Message: fmt.Sprintf("%s backend is not supported on %s; use wgconfig-file", name, goos),
		Err:     ErrBackendNotSupported,
	}
}
