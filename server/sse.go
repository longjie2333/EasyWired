package server

import (
	"EasyWired/wg"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const EventPeerAdded = "peer-added"

type sseEvent struct {
	Name string
	Data string
}

type sseClient struct {
	Events chan sseEvent
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[string]*sseClient
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string]*sseClient),
	}
}

func (h *SSEHub) Subscribe(nodeID string, c *gin.Context) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	client := &sseClient{
		Events: make(chan sseEvent, 16),
	}

	h.mu.Lock()
	h.clients[nodeID] = client
	h.mu.Unlock()

	defer h.unsubscribe(nodeID, client)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher.Flush()

	keepAlive := time.NewTicker(30 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event := <-client.Events:
			if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Name, event.Data); err != nil {
				return
			}
			flusher.Flush()
		case <-keepAlive.C:
			if _, err := fmt.Fprint(c.Writer, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *SSEHub) Notify(nodeID, eventName string, payload wg.PeerEntry) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	client := h.clients[nodeID]
	h.mu.RUnlock()
	if client == nil {
		return
	}

	select {
	case client.Events <- sseEvent{Name: eventName, Data: string(data)}:
	default:
	}
}

func (h *SSEHub) unsubscribe(nodeID string, client *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	current, ok := h.clients[nodeID]
	if !ok {
		return
	}
	if current != client {
		return
	}

	delete(h.clients, nodeID)
}
