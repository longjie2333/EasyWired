package config

import (
	"strings"

	"easywired/internal/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func EnsureInterfaceKeys(cfg *model.NodeConfig) (bool, error) {
	if cfg == nil {
		return false, nil
	}
	privateKey := strings.TrimSpace(cfg.Interface.PrivateKey)
	publicKey := strings.TrimSpace(cfg.Interface.PublicKey)
	if privateKey == "" {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return false, err
		}
		cfg.Interface.PrivateKey = key.String()
		cfg.Interface.PublicKey = key.PublicKey().String()
		return true, nil
	}
	key, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return false, err
	}
	if publicKey == "" {
		cfg.Interface.PrivateKey = privateKey
		cfg.Interface.PublicKey = key.PublicKey().String()
		return true, nil
	}
	cfg.Interface.PrivateKey = privateKey
	cfg.Interface.PublicKey = publicKey
	return false, nil
}
