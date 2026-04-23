//go:build !windows

package wg

import (
	"fmt"
	"runtime"
)

func ApplyConfig(privKey PrivateKey, cfg *NodeConfig, bringUp bool) error {
	return fmt.Errorf("wg: wireguard-nt integration only supports windows (current: %s)", runtime.GOOS)
}

func ApplyPeerHotUpdate(peer PeerEntry) error {
	return fmt.Errorf("wg: wireguard-nt integration only supports windows (current: %s)", runtime.GOOS)
}

func Shutdown() error {
	return nil
}

func ActiveAdapterName() string {
	return ""
}
