package model

type NodeConfig struct {
	NodeID    string      `json:"nodeId"`
	Interface WGInterface `json:"interface"`
	Peers     []WGPeer    `json:"peers"`
	ExtField  ExtField    `json:"extField"`
	Leases    []Lease     `json:"leases,omitempty"`
}

type WGInterface struct {
	PrivateKey string   `json:"privateKey,omitempty"`
	PublicKey  string   `json:"publicKey"`
	Address    string   `json:"address,omitempty"`
	ListenPort int      `json:"listenPort,omitempty"`
	MTU        int      `json:"mtu,omitempty"`
	DNS        []string `json:"dns,omitempty"`
}

type WGPeer struct {
	NodeID              string   `json:"nodeId,omitempty"`
	PublicKey           string   `json:"publicKey"`
	AllowedIPs          []string `json:"allowedIPs,omitempty"`
	Endpoint            string   `json:"endpoint,omitempty"`
	PersistentKeepalive int      `json:"persistentKeepalive,omitempty"`
	Address             string   `json:"address,omitempty"`
	APIEndpoint         string   `json:"apiEndpoint,omitempty"`
	WGEndpoint          string   `json:"wgEndpoint,omitempty"`
}

type ExtField struct {
	MTU                 int      `json:"mtu,omitempty"`
	DNS                 []string `json:"dns,omitempty"`
	Endpoint            string   `json:"endpoint,omitempty"`
	AllowedIPs          []string `json:"allowedIPs,omitempty"`
	PersistentKeepalive int      `json:"persistentKeepalive,omitempty"`
	APIEndpoint         string   `json:"apiEndpoint,omitempty"`
}

type Lease struct {
	NodeID    string `json:"nodeId"`
	PublicKey string `json:"publicKey"`
	Address   string `json:"address"`
	CreatedAt string `json:"createdAt"`
}
