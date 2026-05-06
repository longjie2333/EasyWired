package config

import (
	"fmt"
	"strings"

	"easywired/internal/model"

	"github.com/denisbrodbeck/machineid"
)

const nodeIDAppID = "easywired"

func GenerateNodeID() (string, error) {
	id, err := machineid.ProtectedID(nodeIDAppID)
	if err != nil {
		return "", err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("machine id is empty")
	}
	return id, nil
}

func EnsureNodeID(cfg *model.NodeConfig, generate func() (string, error)) (bool, error) {
	if cfg == nil || strings.TrimSpace(cfg.NodeID) != "" {
		return false, nil
	}
	id, err := generate()
	if err != nil {
		return false, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("generated node id is empty")
	}
	cfg.NodeID = id
	return true, nil
}
