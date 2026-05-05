package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"easywired/internal/api"
	"easywired/internal/model"
	"easywired/internal/store"
	"easywired/internal/util"
)

type mockBackend struct{}

func (mockBackend) Name() string                                                    { return "mock" }
func (mockBackend) Platform() string                                                { return "test" }
func (mockBackend) SupportsNativeApply() bool                                       { return false }
func (mockBackend) EnsureDevice(context.Context, string, *model.NodeConfig) error   { return nil }
func (mockBackend) ApplyInterface(context.Context, string, model.WGInterface) error { return nil }
func (mockBackend) AddOrUpdatePeer(context.Context, string, model.WGPeer) error     { return nil }
func (mockBackend) RemovePeer(context.Context, string, string) error                { return nil }
func (mockBackend) ApplyConfig(context.Context, string, *model.NodeConfig) error    { return nil }
func (mockBackend) ExportConfig(context.Context, *model.NodeConfig) ([]byte, error) {
	return []byte("ok"), nil
}

func TestJoinModeUsesExtFieldAllowedIPs(t *testing.T) {
	cfg := readyNodeA()
	cfg.ExtField.AllowedIPs = []string{"0.0.0.0/0", "::/0"}
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.New(path, cfg)
	srv := api.NewServer(api.Options{Store: st, Backend: mockBackend{}, DeviceName: "wg0", ListenAddr: ":8080", OutputPath: filepath.Join(t.TempDir(), "wg0.conf")})
	req := model.ConnectRequest{NodeID: "nodeB", Interface: model.WGInterface{PublicKey: "bbb"}}
	resp := postConnect(t, srv.Handler(), req)
	if resp.Assigned == nil || resp.Assigned.Address != "10.0.0.2/32" {
		t.Fatalf("expected assigned 10.0.0.2/32, got %#v", resp.Assigned)
	}
	if got := resp.Peer.AllowedIPs; len(got) != 2 || got[0] != "0.0.0.0/0" || got[1] != "::/0" {
		t.Fatalf("expected extField allowedIPs, got %#v", got)
	}
	if len(st.Config().Leases) != 1 || len(st.Config().Peers) != 1 {
		t.Fatalf("expected one lease and one peer, got leases=%d peers=%d", len(st.Config().Leases), len(st.Config().Peers))
	}
	if st.Config().Peers[0].AllowedIPs[0] != "10.0.0.2/32" {
		t.Fatalf("expected target to add requester /32, got %#v", st.Config().Peers[0].AllowedIPs)
	}
}

func TestPeerModeDoesNotAllocateAndUsesAddress32(t *testing.T) {
	cfg := readyNodeA()
	cfg.ExtField.AllowedIPs = []string{"0.0.0.0/0"}
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.New(path, cfg)
	srv := api.NewServer(api.Options{Store: st, Backend: mockBackend{}, DeviceName: "wg0", ListenAddr: ":8080"})
	req := model.ConnectRequest{NodeID: "nodeB", Interface: model.WGInterface{PublicKey: "bbb", Address: "10.0.0.2/32"}}
	resp := postConnect(t, srv.Handler(), req)
	if resp.Assigned != nil {
		t.Fatalf("expected no assigned config in peer mode, got %#v", resp.Assigned)
	}
	if got := resp.Peer.AllowedIPs; len(got) != 1 || got[0] != "10.0.0.1/32" {
		t.Fatalf("expected target address/32 only, got %#v", got)
	}
	if len(st.Config().Leases) != 0 {
		t.Fatalf("expected no leases, got %#v", st.Config().Leases)
	}
}

func TestNotReadyConnectReturnsNodeNotReady(t *testing.T) {
	cfg := readyNodeA()
	cfg.Interface.Address = ""
	st := store.New(filepath.Join(t.TempDir(), "config.json"), cfg)
	srv := api.NewServer(api.Options{Store: st, Backend: mockBackend{}})
	req := model.ConnectRequest{NodeID: "nodeB", Interface: model.WGInterface{PublicKey: "bbb"}}
	body, _ := json.Marshal(req)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/connect", bytes.NewReader(body)))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
	var errResp model.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	if errResp.Code != util.CodeNodeNotReady {
		t.Fatalf("expected NODE_NOT_READY, got %#v", errResp)
	}
}

func postConnect(t *testing.T, handler http.Handler, req model.ConnectRequest) model.ConnectResponse {
	t.Helper()
	body, _ := json.Marshal(req)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/connect", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp model.ConnectResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

func readyNodeA() *model.NodeConfig {
	return &model.NodeConfig{
		NodeID: "nodeA",
		Interface: model.WGInterface{
			PrivateKey: "priv",
			PublicKey:  "aaa",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
		},
		Peers: []model.WGPeer{},
		ExtField: model.ExtField{
			MTU:                 1420,
			DNS:                 []string{"8.8.8.8"},
			PersistentKeepalive: 25,
			Endpoint:            "nodeA-public-ip:51820",
		},
	}
}
