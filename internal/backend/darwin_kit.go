//go:build darwin

package backend

// DarwinKitBackend is reserved for WireGuardKit via helper app, sidecar, IPC, or NetworkExtension.
type DarwinKitBackend struct{}

func NewDarwinKitBackend() (*DarwinKitBackend, error) {
	return nil, ErrBackendNotImplemented
}
