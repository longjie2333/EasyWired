package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"easywired/internal/config"
	"easywired/internal/ipam"
	"easywired/internal/model"
	"easywired/internal/util"
)

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		util.WriteError(w, http.StatusMethodNotAllowed, util.CodeBadRequest, "method not allowed")
		return
	}
	var req model.ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, util.CodeBadRequest, err.Error())
		return
	}
	req = config.SanitizeConnectRequest(req)
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.store.Config()
	if !config.Ready(cfg) {
		util.WriteError(w, http.StatusServiceUnavailable, util.CodeNodeNotReady, config.NodeNotReadyMessage)
		return
	}
	if req.Interface.PublicKey == "" {
		util.WriteError(w, http.StatusBadRequest, util.CodeBadRequest, "publicKey is required")
		return
	}
	joinMode := req.Interface.Address == ""
	assigned := ""
	if joinMode {
		lease, err := s.allocator.Allocate(cfg, req.NodeID, req.Interface.PublicKey)
		if err != nil {
			s.writeIPAMError(w, err)
			return
		}
		assigned = lease.Address
	}
	peerAddress := req.Interface.Address
	if peerAddress == "" {
		peerAddress = assigned
	}
	peer := model.WGPeer{
		NodeID:              req.NodeID,
		PublicKey:           req.Interface.PublicKey,
		AllowedIPs:          []string{util.WithPrefix32(peerAddress)},
		Endpoint:            req.ExtField.Endpoint,
		PersistentKeepalive: req.ExtField.PersistentKeepalive,
		Address:             util.WithPrefix32(peerAddress),
		APIEndpoint:         req.ExtField.APIEndpoint,
		WGEndpoint:          req.ExtField.Endpoint,
	}
	_ = s.store.UpsertPeer(peer)
	if err := s.store.Save(); err != nil {
		util.WriteError(w, http.StatusInternalServerError, util.CodeStoreFailed, err.Error())
		return
	}
	if err := s.applyAfterChange(r.Context(), peer); err != nil {
		util.WriteError(w, http.StatusInternalServerError, util.CodeWGConfigFailed, err.Error())
		return
	}
	resp := model.ConnectResponse{
		Peer:                    s.selfPeer(cfg, joinMode),
		InterfaceRecommendation: model.InterfaceRecommendation{MTU: cfg.ExtField.MTU, DNS: cfg.ExtField.DNS},
	}
	if assigned != "" {
		resp.Assigned = &model.AssignedConfig{Address: assigned}
	}
	util.WriteJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		util.WriteError(w, http.StatusMethodNotAllowed, util.CodeBadRequest, "method not allowed")
		return
	}
	var req model.DisconnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, util.CodeBadRequest, err.Error())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.store.Config()
	if !config.Ready(cfg) {
		util.WriteError(w, http.StatusServiceUnavailable, util.CodeNodeNotReady, config.NodeNotReadyMessage)
		return
	}
	if req.PublicKey == "" {
		util.WriteError(w, http.StatusBadRequest, util.CodeBadRequest, "publicKey is required")
		return
	}
	_ = s.store.RemovePeer(req.PublicKey)
	_ = s.store.RemoveLease(req.PublicKey)
	if err := s.store.Save(); err != nil {
		util.WriteError(w, http.StatusInternalServerError, util.CodeStoreFailed, err.Error())
		return
	}
	if s.backend.SupportsNativeApply() {
		if err := s.backend.RemovePeer(r.Context(), s.deviceName, req.PublicKey); err != nil {
			util.WriteError(w, http.StatusInternalServerError, util.CodeWGConfigFailed, err.Error())
			return
		}
	} else if err := s.exportFile(r.Context()); err != nil {
		util.WriteError(w, http.StatusInternalServerError, util.CodeWGConfigFailed, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, model.DisconnectResponse{Success: true})
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		util.WriteError(w, http.StatusMethodNotAllowed, util.CodeBadRequest, "method not allowed")
		return
	}
	cfg := s.store.Config()
	resp := model.PeersResponse{
		NodeID:    cfg.NodeID,
		Ready:     config.Ready(cfg),
		Interface: config.PublicInterface(cfg.Interface),
		Peers:     cfg.Peers,
	}
	if !resp.Ready {
		resp.Message = config.NodeNotReadyMessage
	}
	util.WriteJSON(w, http.StatusOK, resp)
}

func (s *Server) selfPeer(cfg *model.NodeConfig, joinMode bool) model.WGPeer {
	allowed := []string{util.WithPrefix32(cfg.Interface.Address)}
	if joinMode && len(cfg.ExtField.AllowedIPs) > 0 {
		allowed = cfg.ExtField.AllowedIPs
	}
	return model.WGPeer{
		NodeID:              cfg.NodeID,
		PublicKey:           cfg.Interface.PublicKey,
		AllowedIPs:          allowed,
		Endpoint:            util.PeerEndpoint(cfg),
		PersistentKeepalive: cfg.ExtField.PersistentKeepalive,
		Address:             util.WithPrefix32(cfg.Interface.Address),
		APIEndpoint:         util.APIEndpoint(cfg, s.listenAddr),
		WGEndpoint:          util.WGEndpoint(cfg),
	}
}

func (s *Server) applyAfterChange(ctx context.Context, peer model.WGPeer) error {
	if s.backend.SupportsNativeApply() {
		if err := s.backend.ApplyInterface(ctx, s.deviceName, s.store.Config().Interface); err != nil {
			return err
		}
		return s.backend.AddOrUpdatePeer(ctx, s.deviceName, peer)
	}
	return s.exportFile(ctx)
}

func (s *Server) exportFile(ctx context.Context) error {
	if s.outputPath == "" {
		return nil
	}
	b, err := s.backend.ExportConfig(ctx, s.store.Config())
	if err != nil {
		return err
	}
	return os.WriteFile(s.outputPath, b, 0o600)
}

func (s *Server) writeIPAMError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ipam.ErrNoAddressPool):
		util.WriteError(w, http.StatusConflict, util.CodeNoAddressPool, err.Error())
	case errors.Is(err, ipam.ErrNoAvailableIP):
		util.WriteError(w, http.StatusConflict, util.CodeNoAvailableIP, err.Error())
	default:
		util.WriteError(w, http.StatusInternalServerError, util.CodeInternalError, err.Error())
	}
}
