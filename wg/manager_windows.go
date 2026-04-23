//go:build windows

package wg

import (
	"errors"
	"fmt"
	"sync"

	"golang.zx2c4.com/wireguard/windows/driver"
)

var defaultManager = &adapterManager{}

type adapterManager struct {
	mu          sync.Mutex
	adapter     *driver.Adapter
	adapterName string
}

func ApplyConfig(privKey PrivateKey, cfg *NodeConfig, bringUp bool) error {
	return defaultManager.applyConfig(privKey, cfg, bringUp)
}

func ApplyPeerHotUpdate(peer PeerEntry) error {
	return defaultManager.applyPeerHotUpdate(peer)
}

func Shutdown() error {
	return defaultManager.shutdown()
}

func ActiveAdapterName() string {
	return defaultManager.activeAdapterName()
}

func (m *adapterManager) applyConfig(privKey PrivateKey, cfg *NodeConfig, bringUp bool) error {
	if cfg == nil {
		return errors.New("wg: nil node config")
	}
	if privKey == (PrivateKey{}) {
		return errors.New("wg: empty private key")
	}

	wgConfig, err := toWGConfig(privKey, cfg)
	if err != nil {
		return err
	}

	driverCfg, driverCfgSize, err := buildDriverFullConfiguration(wgConfig)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	adapter, luid, err := m.ensureAdapterLocked(cfg.adapterName())
	if err != nil {
		return err
	}

	if err := adapter.SetConfiguration(driverCfg, driverCfgSize); err != nil {
		return fmt.Errorf("wg: set adapter configuration: %w", err)
	}
	if err := adapter.SetLogging(driver.AdapterLogOnWithPrefix); err != nil {
		return fmt.Errorf("wg: enable adapter logging: %w", err)
	}

	state := driver.AdapterStateDown
	if bringUp {
		state = driver.AdapterStateUp
	}
	if err := adapter.SetAdapterState(state); err != nil {
		return fmt.Errorf("wg: set adapter state: %w", err)
	}

	if bringUp {
		if err := applyInterfaceNetworkConfig(luid, cfg, wgConfig); err != nil {
			return fmt.Errorf("wg: apply windows network config: %w", err)
		}
	}

	return nil
}

func (m *adapterManager) applyPeerHotUpdate(peer PeerEntry) error {
	peerConfig, err := toWGPeerConfig(peer)
	if err != nil {
		return err
	}

	driverCfg, driverCfgSize, err := buildDriverPeerHotUpdateConfiguration(peerConfig)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.adapter == nil {
		return errors.New("wg: adapter is not initialized")
	}
	if err := m.adapter.SetConfiguration(driverCfg, driverCfgSize); err != nil {
		return fmt.Errorf("wg: apply peer hot update: %w", err)
	}

	return nil
}

func (m *adapterManager) shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closeActiveAdapterLocked()
}

func (m *adapterManager) activeAdapterName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.adapterName
}
