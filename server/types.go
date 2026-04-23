package server

import "EasyWired/wg"

const (
	DefaultSuggestedMTU       = 1420
	DefaultSuggestedKeepalive = 25
	DefaultSuggestedDNS       = "8.8.8.8"
)

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error   string        `json:"error"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type WelcomeResponse struct {
	Message string `json:"message"`
}

type RegisterNodeRequest struct {
	NodeID             string   `json:"nodeId"`
	PublicKey          string   `json:"publicKey"`
	Endpoint           string   `json:"endpoint,omitempty"`
	Address            string   `json:"address,omitempty"`
	AllowedCIDRs       []string `json:"allowedCIDRs,omitempty"`
	ListenPort         int      `json:"listenPort,omitempty"`
	SuggestedMTU       *int     `json:"suggestedMTU,omitempty"`
	SuggestedDNS       []string `json:"suggestedDNS,omitempty"`
	SuggestedKeepalive *int     `json:"suggestedKeepalive,omitempty"`
}

type RegisterNodeResponse struct {
	Ok bool `json:"ok"`
}

type ConnectRequest struct {
	FromNodeID string `json:"fromNodeId"`
	ToNodeID   string `json:"toNodeId"`
}

type ConnectResponse struct {
	Ok               bool    `json:"ok"`
	AllocatedIP      *string `json:"allocatedIP,omitempty"`
	AlreadyConnected *bool   `json:"alreadyConnected,omitempty"`
}

type NodeRecord struct {
	NodeID             string         `json:"nodeId"`
	PublicKey          string         `json:"publicKey"`
	Endpoint           string         `json:"endpoint,omitempty"`
	Address            string         `json:"address,omitempty"`
	AllowedCIDRs       []string       `json:"allowedCIDRs,omitempty"`
	ListenPort         int            `json:"listenPort,omitempty"`
	SuggestedMTU       int            `json:"suggestedMTU,omitempty"`
	SuggestedDNS       []string       `json:"suggestedDNS,omitempty"`
	SuggestedKeepalive int            `json:"suggestedKeepalive,omitempty"`
	RegisteredAt       string         `json:"registeredAt"`
	Peers              []wg.PeerEntry `json:"peers,omitempty"`
}

func BuildRegisterNodeRequest(nodeID string, privateKey wg.PrivateKey) RegisterNodeRequest {
	return RegisterNodeRequest{
		NodeID:    nodeID,
		PublicKey: privateKey.PublicKey().String(),
	}
}

func BuildConnectRequest(fromNodeID, toNodeID string) ConnectRequest {
	return ConnectRequest{
		FromNodeID: fromNodeID,
		ToNodeID:   toNodeID,
	}
}
