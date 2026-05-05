//go:build windows

package wg

import (
	"errors"
	"fmt"
	"strings"

	"golang.zx2c4.com/wireguard/windows/driver"
)

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
