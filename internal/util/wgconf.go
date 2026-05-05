package util

import (
	"bytes"
	"fmt"
	"strings"

	"easywired/internal/model"
)

func ExportWGQuick(cfg *model.NodeConfig) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	var b bytes.Buffer
	b.WriteString("[Interface]\n")
	writeKV(&b, "PrivateKey", cfg.Interface.PrivateKey)
	writeKV(&b, "Address", cfg.Interface.Address)
	if cfg.Interface.ListenPort != 0 {
		writeKV(&b, "ListenPort", fmt.Sprint(cfg.Interface.ListenPort))
	}
	if cfg.Interface.MTU != 0 {
		writeKV(&b, "MTU", fmt.Sprint(cfg.Interface.MTU))
	}
	if len(cfg.Interface.DNS) > 0 {
		writeKV(&b, "DNS", strings.Join(cfg.Interface.DNS, ", "))
	}
	for _, peer := range cfg.Peers {
		b.WriteString("\n[Peer]\n")
		writeKV(&b, "PublicKey", peer.PublicKey)
		writeKV(&b, "Endpoint", peer.Endpoint)
		if len(peer.AllowedIPs) > 0 {
			writeKV(&b, "AllowedIPs", strings.Join(peer.AllowedIPs, ", "))
		}
		if peer.PersistentKeepalive != 0 {
			writeKV(&b, "PersistentKeepalive", fmt.Sprint(peer.PersistentKeepalive))
		}
	}
	return b.Bytes(), nil
}

func writeKV(b *bytes.Buffer, key, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "%s = %s\n", key, value)
}
