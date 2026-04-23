package wg

import (
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const defaultAdapterName = "EasyWired"

type PrivateKey = wgtypes.Key

type NodeConfig struct {
	NodeID             string      `json:"nodeId,omitempty"`
	InterfaceName      string      `json:"interfaceName,omitempty"`
	ListenPort         int         `json:"listenPort,omitempty"`
	Address            string      `json:"address,omitempty"`
	Addresses          []string    `json:"addresses,omitempty"`
	AllowedCIDRs       []string    `json:"allowedCIDRs,omitempty"`
	DNS                []string    `json:"dns,omitempty"`
	SuggestedDNS       []string    `json:"suggestedDNS,omitempty"`
	MTU                int         `json:"mtu,omitempty"`
	SuggestedMTU       int         `json:"suggestedMTU,omitempty"`
	SuggestedKeepalive int         `json:"suggestedKeepalive,omitempty"`
	Peers              []PeerEntry `json:"peers,omitempty"`
}

type PeerEntry struct {
	PublicKey           string   `json:"peerPublicKey"`
	PresharedKey        string   `json:"presharedKey,omitempty"`
	Endpoint            string   `json:"peerEndpoint,omitempty"`
	AllowedIPs          []string `json:"allowedIPs,omitempty"`
	AllowedCIDRs        []string `json:"allowedCIDRs,omitempty"`
	PersistentKeepalive int      `json:"persistentKeepalive,omitempty"`
	Remove              bool     `json:"remove,omitempty"`
	UpdateOnly          bool     `json:"updateOnly,omitempty"`
}

func GeneratePrivateKey() (PrivateKey, error) {
	return wgtypes.GeneratePrivateKey()
}

func ParsePrivateKey(value string) (PrivateKey, error) {
	return wgtypes.ParseKey(strings.TrimSpace(value))
}

func (c *NodeConfig) adapterName() string {
	if c == nil {
		return defaultAdapterName
	}

	if name := strings.TrimSpace(c.InterfaceName); name != "" {
		return name
	}

	if name := strings.TrimSpace(c.NodeID); name != "" {
		return name
	}

	return defaultAdapterName
}

func (p PeerEntry) normalizedAllowedIPs() []string {
	if len(p.AllowedIPs) > 0 {
		return p.AllowedIPs
	}

	return p.AllowedCIDRs
}
