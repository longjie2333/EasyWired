package tests

import (
	"context"
	"strings"
	"testing"

	"easywired/internal/backend"
	"easywired/internal/config"
	"easywired/internal/model"
)

func TestReady(t *testing.T) {
	cfg := &model.NodeConfig{NodeID: "nodeA", Interface: model.WGInterface{PrivateKey: "priv", PublicKey: "pub", Address: "10.0.0.1/24"}}
	if !config.Ready(cfg) {
		t.Fatal("expected ready config")
	}
	cfg.Interface.Address = ""
	if config.Ready(cfg) {
		t.Fatal("expected config without address to be not ready")
	}
}

func TestEnsureNodeID(t *testing.T) {
	cfg := &model.NodeConfig{}
	changed, err := config.EnsureNodeID(cfg, func() (string, error) { return "machine-node-id", nil })
	if err != nil {
		t.Fatal(err)
	}
	if !changed || cfg.NodeID != "machine-node-id" {
		t.Fatalf("expected generated node id, changed=%v nodeID=%q", changed, cfg.NodeID)
	}
	changed, err = config.EnsureNodeID(cfg, func() (string, error) { return "other", nil })
	if err != nil {
		t.Fatal(err)
	}
	if changed || cfg.NodeID != "machine-node-id" {
		t.Fatalf("expected existing node id to be preserved, changed=%v nodeID=%q", changed, cfg.NodeID)
	}
}

func TestWGConfigFileExport(t *testing.T) {
	cfg := &model.NodeConfig{
		Interface: model.WGInterface{PrivateKey: "priv", Address: "10.0.0.2/32", ListenPort: 51820, MTU: 1420, DNS: []string{"8.8.8.8", "1.1.1.1"}},
		Peers:     []model.WGPeer{{PublicKey: "peer", Endpoint: "nodeA:51820", AllowedIPs: []string{"0.0.0.0/0", "::/0"}, PersistentKeepalive: 25}},
	}
	b, err := backend.NewWGConfigFile().ExportConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	want := []string{"[Interface]", "PrivateKey = priv", "Address = 10.0.0.2/32", "ListenPort = 51820", "MTU = 1420", "DNS = 8.8.8.8, 1.1.1.1", "[Peer]", "PublicKey = peer", "Endpoint = nodeA:51820", "AllowedIPs = 0.0.0.0/0, ::/0", "PersistentKeepalive = 25"}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Fatalf("export missing %q in:\n%s", s, out)
		}
	}
}

func TestBackendAutoSelection(t *testing.T) {
	be, err := backend.SelectForOS(backend.NameAuto, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if be.Name() != backend.NameWGConfigFile {
		t.Fatalf("expected wgconfig-file on windows, got %s", be.Name())
	}
	be, err = backend.SelectForOS(backend.NameAuto, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	if be.Name() != backend.NameWGConfigFile {
		t.Fatalf("expected wgconfig-file on darwin, got %s", be.Name())
	}
	if _, err = backend.SelectForOS(backend.NameLinuxNative, "windows"); err == nil {
		t.Fatal("expected linux-native to be unsupported on windows")
	}
	be, err = backend.SelectForOS(backend.NameWindowsService, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if be.Name() != backend.NameWindowsService || !be.SupportsNativeApply() {
		t.Fatalf("expected windows-service native backend, got %s native=%v", be.Name(), be.SupportsNativeApply())
	}
	be, err = backend.SelectForOS(backend.NameWindowsNT, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if be.Name() != backend.NameWindowsNT || !be.SupportsNativeApply() {
		t.Fatalf("expected windows-nt native backend, got %s native=%v", be.Name(), be.SupportsNativeApply())
	}
}
