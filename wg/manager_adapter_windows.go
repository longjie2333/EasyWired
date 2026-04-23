//go:build windows

package wg

import (
	"errors"
	"fmt"
	"strings"

	"golang.zx2c4.com/wireguard/windows/driver"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

func (m *adapterManager) ensureAdapterLocked(name string) (*driver.Adapter, winipcfg.LUID, error) {
	if err := validateAdapterName(name); err != nil {
		return nil, 0, err
	}

	if m.adapter != nil {
		if strings.EqualFold(m.adapterName, name) {
			return m.adapter, m.adapter.LUID(), nil
		}
		if err := m.closeActiveAdapterLocked(); err != nil {
			return nil, 0, err
		}
	}

	adapter, err := driver.CreateAdapter(name, "WireGuard", nil)
	if err != nil {
		return nil, 0, fmt.Errorf("wg: create adapter %q: %w", name, err)
	}

	m.adapter = adapter
	m.adapterName = name
	return adapter, adapter.LUID(), nil
}

func (m *adapterManager) closeActiveAdapterLocked() error {
	if m.adapter == nil {
		return nil
	}

	adapterName := m.adapterName
	stateErr := m.adapter.SetAdapterState(driver.AdapterStateDown)
	closeErr := m.adapter.Close()
	m.adapter = nil
	m.adapterName = ""

	if stateErr != nil && closeErr != nil {
		return fmt.Errorf("wg: set down failed for %q: %v; close failed: %w", adapterName, stateErr, closeErr)
	}
	if stateErr != nil {
		return fmt.Errorf("wg: set down failed for %q: %w", adapterName, stateErr)
	}
	if closeErr != nil {
		return fmt.Errorf("wg: close adapter %q failed: %w", adapterName, closeErr)
	}

	return nil
}

func validateAdapterName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("wg: adapter name is empty")
	}
	if len([]rune(trimmed)) > driver.AdapterNameMax {
		return fmt.Errorf("wg: adapter name too long: %d", len([]rune(trimmed)))
	}
	return nil
}
