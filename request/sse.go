package request

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"EasyWired/wg"
)

func (c *Client) StartSSE(ctx context.Context, nodeID string, privKey wg.PrivateKey, interfaceName string, ready chan<- struct{}) {
	var once sync.Once

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := c.connectSSE(context.Background(), nodeID, privKey, interfaceName, func() {
			once.Do(func() {
				close(ready)
			})
		})

		if err != nil {
			fmt.Printf("SSE disconnected: %v, retrying in 5s\n", err)
		}

		time.Sleep(5 * time.Second)
	}
}

func (c *Client) connectSSE(ctx context.Context, nodeID string, privKey wg.PrivateKey, interfaceName string, onReady func()) error {
	resp, err := c.SubscribeNodeEvents(ctx, nodeID)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	cfg, err := c.GetNodeConfig(ctx, nodeID)
	if err != nil {
		return err
	}
	if name := strings.TrimSpace(interfaceName); name != "" {
		cfg.InterfaceName = name
	}

	if err := wg.ApplyConfig(privKey, cfg, true); err != nil {
		return err
	}

	if onReady != nil {
		onReady()
	}

	scanner := bufio.NewScanner(resp.Body)
	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if eventType == "peer-added" {
				handlePeerAdded(data)
			}

			eventType = ""
		}
	}

	return scanner.Err()
}

func handlePeerAdded(data string) {
	var peer wg.PeerEntry
	if err := json.Unmarshal([]byte(data), &peer); err != nil {
		return
	}

	if err := wg.ApplyPeerHotUpdate(peer); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying peer hot update: %v\n", err)
	}
}
