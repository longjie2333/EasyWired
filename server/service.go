package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"EasyWired/wg"
	"github.com/gin-gonic/gin"
)

type Service struct {
	store      *Store
	sseHub     *SSEHub
	router     *gin.Engine
	httpServer *http.Server
}

func New() *Service {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	service := &Service{
		store:  NewStore(),
		sseHub: NewSSEHub(),
		router: router,
	}

	router.GET("/", service.handleWelcome)

	nodesRouter := router.Group("/nodes")
	nodesRouter.GET("/", service.handleListNodes)
	nodesRouter.POST("/register", service.handleRegisterNode)
	nodesRouter.GET("/:id/config", service.handleNodeConfig)
	nodesRouter.GET("/:id/events", service.handleNodeEvents)

	router.POST("/connect", service.handleConnect)

	return service
}

func (s *Service) Start(listenAddr string) error {
	addr := strings.TrimSpace(listenAddr)
	if addr == "" {
		return errors.New("server listen address cannot be empty")
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.httpServer = &http.Server{
		Addr:    listener.Addr().String(),
		Handler: s.router,
	}

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("embedded server stopped unexpectedly: %v\n", err)
		}
	}()

	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

func (s *Service) handleWelcome(c *gin.Context) {
	c.JSON(http.StatusOK, WelcomeResponse{
		Message: "EasyWired Control Plane API is running.",
	})
}

func (s *Service) handleListNodes(c *gin.Context) {
	c.JSON(http.StatusOK, s.store.All())
}

func (s *Service) handleRegisterNode(c *gin.Context) {
	payload := RegisterNodeRequest{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	payload.NodeID = strings.TrimSpace(payload.NodeID)
	payload.PublicKey = strings.TrimSpace(payload.PublicKey)
	payload.Endpoint = strings.TrimSpace(payload.Endpoint)
	payload.Address = strings.TrimSpace(payload.Address)
	payload.AllowedCIDRs = cleanStrings(payload.AllowedCIDRs)
	payload.SuggestedDNS = cleanStrings(payload.SuggestedDNS)

	if details := validateRegisterPayload(payload); len(details) > 0 {
		respondError(c, http.StatusBadRequest, "Invalid request body", details)
		return
	}

	existing, ok := s.store.Get(payload.NodeID)
	peers := []wg.PeerEntry(nil)
	if ok {
		peers = append(peers, existing.Peers...)
	}

	record := NodeRecord{
		NodeID:             payload.NodeID,
		PublicKey:          payload.PublicKey,
		Endpoint:           payload.Endpoint,
		Address:            payload.Address,
		AllowedCIDRs:       payload.AllowedCIDRs,
		ListenPort:         payload.ListenPort,
		SuggestedMTU:       defaultInt(payload.SuggestedMTU, DefaultSuggestedMTU),
		SuggestedDNS:       defaultStrings(payload.SuggestedDNS, []string{DefaultSuggestedDNS}),
		SuggestedKeepalive: defaultInt(payload.SuggestedKeepalive, DefaultSuggestedKeepalive),
		RegisteredAt:       time.Now().UTC().Format(time.RFC3339),
		Peers:              peers,
	}

	s.store.Set(record.NodeID, record)
	c.JSON(http.StatusOK, RegisterNodeResponse{Ok: true})
}

func (s *Service) handleNodeConfig(c *gin.Context) {
	nodeID := strings.TrimSpace(c.Param("id"))
	node, ok := s.store.Get(nodeID)
	if !ok {
		respondError(c, http.StatusNotFound, "Node not found", nil)
		return
	}

	c.JSON(http.StatusOK, wg.NodeConfig{
		Address:    node.Address,
		ListenPort: node.ListenPort,
		MTU:        node.SuggestedMTU,
		DNS:        node.SuggestedDNS,
		Peers:      node.Peers,
	})
}

func (s *Service) handleNodeEvents(c *gin.Context) {
	nodeID := strings.TrimSpace(c.Param("id"))
	if _, ok := s.store.Get(nodeID); !ok {
		c.Status(http.StatusNotFound)
		return
	}

	s.sseHub.Subscribe(nodeID, c)
}

func (s *Service) handleConnect(c *gin.Context) {
	payload := ConnectRequest{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	payload.FromNodeID = strings.TrimSpace(payload.FromNodeID)
	payload.ToNodeID = strings.TrimSpace(payload.ToNodeID)
	if payload.FromNodeID == "" || payload.ToNodeID == "" {
		details := []ErrorDetail{}
		if payload.FromNodeID == "" {
			details = append(details, ErrorDetail{Field: "fromNodeId", Message: "fromNodeId is required"})
		}
		if payload.ToNodeID == "" {
			details = append(details, ErrorDetail{Field: "toNodeId", Message: "toNodeId is required"})
		}
		respondError(c, http.StatusBadRequest, "Invalid request body", details)
		return
	}

	if payload.FromNodeID == payload.ToNodeID {
		respondError(c, http.StatusBadRequest, "cannot connect to self", nil)
		return
	}

	fromNode, ok := s.store.Get(payload.FromNodeID)
	if !ok {
		respondError(c, http.StatusNotFound, "self node unregistered", nil)
		return
	}

	toNode, ok := s.store.Get(payload.ToNodeID)
	if !ok {
		respondError(c, http.StatusNotFound, "target node not found", nil)
		return
	}

	if strings.TrimSpace(toNode.Endpoint) == "" {
		respondError(c, http.StatusBadRequest, "target node endpoint is missing", nil)
		return
	}

	if len(toNode.AllowedCIDRs) == 0 {
		respondError(c, http.StatusBadRequest, "target node allowedCIDRs is missing", nil)
		return
	}

	if strings.TrimSpace(toNode.Address) == "" {
		respondError(c, http.StatusBadRequest, "target node interface address is missing", nil)
		return
	}

	if hasPeer(fromNode.Peers, toNode.PublicKey) {
		alreadyConnected := true
		c.JSON(http.StatusOK, ConnectResponse{
			Ok:               true,
			AlreadyConnected: &alreadyConnected,
		})
		return
	}

	usedIPs := make([]string, 0, len(toNode.Peers)+1)
	usedIPs = append(usedIPs, toNode.Address)
	for _, peer := range toNode.Peers {
		if len(peer.AllowedIPs) > 0 {
			usedIPs = append(usedIPs, peer.AllowedIPs[0])
			continue
		}
	}

	allocatedIP := allocateIPv4(toNode.AllowedCIDRs[0], usedIPs)
	if allocatedIP == "" {
		respondError(c, http.StatusInternalServerError, "no available IP", nil)
		return
	}

	prefix := cidrPrefix(toNode.AllowedCIDRs[0])
	if prefix <= 0 {
		respondError(c, http.StatusInternalServerError, "failed to create peer entries", nil)
		return
	}

	peerForFrom := wg.PeerEntry{
		PublicKey:           toNode.PublicKey,
		Endpoint:            toNode.Endpoint,
		AllowedIPs:          []string{"0.0.0.0/0"},
		PersistentKeepalive: normalizeKeepalive(toNode.SuggestedKeepalive),
	}

	peerForTo := wg.PeerEntry{
		PublicKey:           fromNode.PublicKey,
		Endpoint:            fromNode.Endpoint,
		AllowedIPs:          []string{fmt.Sprintf("%s/%d", allocatedIP, prefix)},
		PersistentKeepalive: normalizeKeepalive(fromNode.SuggestedKeepalive),
	}

	fromNode.Peers = append(fromNode.Peers, peerForFrom)
	toNode.Peers = append(toNode.Peers, peerForTo)
	s.store.Set(fromNode.NodeID, fromNode)
	s.store.Set(toNode.NodeID, toNode)

	s.sseHub.Notify(fromNode.NodeID, EventPeerAdded, peerForFrom)
	s.sseHub.Notify(toNode.NodeID, EventPeerAdded, peerForTo)

	c.JSON(http.StatusOK, ConnectResponse{
		Ok:          true,
		AllocatedIP: &allocatedIP,
	})
}

func respondError(c *gin.Context, statusCode int, message string, details []ErrorDetail) {
	payload := ErrorResponse{
		Error: message,
	}
	if len(details) > 0 {
		payload.Details = details
	}

	c.JSON(statusCode, payload)
}

func cleanStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func defaultStrings(input []string, fallback []string) []string {
	cleaned := cleanStrings(input)
	if len(cleaned) > 0 {
		return cleaned
	}

	return append([]string(nil), fallback...)
}

func defaultInt(input *int, fallback int) int {
	if input == nil {
		return fallback
	}
	if *input <= 0 {
		return fallback
	}

	return *input
}

func normalizeKeepalive(keepalive int) int {
	if keepalive <= 0 {
		return DefaultSuggestedKeepalive
	}

	return keepalive
}

func hasPeer(peers []wg.PeerEntry, peerPublicKey string) bool {
	targetKey := strings.TrimSpace(peerPublicKey)
	if targetKey == "" {
		return false
	}

	for _, peer := range peers {
		if strings.TrimSpace(peer.PublicKey) == targetKey {
			return true
		}
	}

	return false
}

func validateRegisterPayload(payload RegisterNodeRequest) []ErrorDetail {
	details := make([]ErrorDetail, 0)

	if payload.NodeID == "" {
		details = append(details, ErrorDetail{Field: "nodeId", Message: "nodeId is required"})
	}

	if payload.PublicKey == "" {
		details = append(details, ErrorDetail{Field: "publicKey", Message: "publicKey is required"})
	}

	if payload.ListenPort < 0 || payload.ListenPort > 65535 {
		details = append(details, ErrorDetail{Field: "listenPort", Message: "listenPort must be between 1 and 65535"})
	}

	if payload.SuggestedMTU != nil && *payload.SuggestedMTU <= 0 {
		details = append(details, ErrorDetail{Field: "suggestedMTU", Message: "suggestedMTU must be greater than 0"})
	}

	if payload.SuggestedKeepalive != nil && *payload.SuggestedKeepalive <= 0 {
		details = append(details, ErrorDetail{Field: "suggestedKeepalive", Message: "suggestedKeepalive must be greater than 0"})
	}

	if payload.Address != "" {
		if _, _, err := net.ParseCIDR(payload.Address); err != nil {
			details = append(details, ErrorDetail{Field: "address", Message: "address must be a valid CIDR"})
		}
	}

	for idx, cidr := range payload.AllowedCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			details = append(details, ErrorDetail{
				Field:   fmt.Sprintf("allowedCIDRs[%d]", idx),
				Message: "must be a valid CIDR",
			})
		}
	}

	return details
}
