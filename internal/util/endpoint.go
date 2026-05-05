package util

import (
	"net"
	"strconv"
	"strings"

	"easywired/internal/model"
)

const DefaultWGPort = 51820

func PeerEndpoint(cfg *model.NodeConfig) string {
	if cfg.ExtField.Endpoint != "" {
		return cfg.ExtField.Endpoint
	}
	ip := AddressIP(cfg.Interface.Address)
	if ip == "" {
		return ""
	}
	port := cfg.Interface.ListenPort
	if port == 0 {
		port = DefaultWGPort
	}
	return net.JoinHostPort(ip, strconv.Itoa(port))
}

func WGEndpoint(cfg *model.NodeConfig) string {
	return PeerEndpoint(cfg)
}

func APIEndpoint(cfg *model.NodeConfig, listen string) string {
	if cfg.ExtField.APIEndpoint != "" {
		return cfg.ExtField.APIEndpoint
	}
	ip := AddressIP(cfg.Interface.Address)
	if ip == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		port = strings.TrimPrefix(listen, ":")
	}
	if port == "" {
		port = "8080"
	}
	return "http://" + net.JoinHostPort(ip, port)
}
