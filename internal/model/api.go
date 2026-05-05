package model

type ConnectRequest struct {
	NodeID    string      `json:"nodeId"`
	Interface WGInterface `json:"interface"`
	ExtField  ExtField    `json:"extField"`
}

type ConnectResponse struct {
	Assigned                *AssignedConfig         `json:"assigned,omitempty"`
	Peer                    WGPeer                  `json:"peer"`
	InterfaceRecommendation InterfaceRecommendation `json:"interfaceRecommendation,omitempty"`
}

type AssignedConfig struct {
	Address string `json:"address"`
}

type InterfaceRecommendation struct {
	MTU int      `json:"mtu,omitempty"`
	DNS []string `json:"dns,omitempty"`
}

type DisconnectRequest struct {
	NodeID    string `json:"nodeId,omitempty"`
	PublicKey string `json:"publicKey"`
}

type DisconnectResponse struct {
	Success bool `json:"success"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PeersResponse struct {
	NodeID    string      `json:"nodeId"`
	Ready     bool        `json:"ready"`
	Message   string      `json:"message,omitempty"`
	Interface WGInterface `json:"interface"`
	Peers     []WGPeer    `json:"peers"`
}
