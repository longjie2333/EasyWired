package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"easywired/internal/model"
)

type Client struct {
	http *http.Client
}

func New(timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{http: &http.Client{Timeout: timeout}}
}

func (c *Client) Connect(url string, req model.ConnectRequest) (*model.ConnectResponse, error) {
	var resp model.ConnectResponse
	if err := c.postJSON(url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Disconnect(url string, req model.DisconnectRequest) (*model.DisconnectResponse, error) {
	var resp model.DisconnectResponse
	if err := c.postJSON(url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Peers(url string) ([]byte, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s failed: status %d: %s", url, resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) postJSON(url string, req any, out any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := c.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s failed: status %d: %s", url, resp.StatusCode, string(respBody))
	}
	if len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}
