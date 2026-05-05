package config

import (
	"strings"

	"easywired/internal/model"
)

const NodeNotReadyMessage = "此节点未准备好连接服务"

func Ready(cfg *model.NodeConfig) bool {
	if cfg == nil {
		return false
	}
	return strings.TrimSpace(cfg.NodeID) != "" &&
		strings.TrimSpace(cfg.Interface.PrivateKey) != "" &&
		strings.TrimSpace(cfg.Interface.PublicKey) != "" &&
		strings.TrimSpace(cfg.Interface.Address) != ""
}

func PublicInterface(iface model.WGInterface) model.WGInterface {
	iface.PrivateKey = ""
	return iface
}

func SanitizeConnectRequest(req model.ConnectRequest) model.ConnectRequest {
	req.Interface.PrivateKey = ""
	return req
}
