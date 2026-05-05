//go:build android

package backend

// AndroidTunnelBackend is reserved for com.wireguard.android:tunnel integration.
type AndroidTunnelBackend struct{}

func NewAndroidTunnelBackend() (*AndroidTunnelBackend, error) {
	return nil, ErrBackendNotImplemented
}
