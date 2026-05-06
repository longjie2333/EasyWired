package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func endpointURL(rawURL, endpoint string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return rawURL
	}
	endpoint = "/" + strings.Trim(endpoint, "/")
	path := strings.TrimRight(u.Path, "/")
	if path == endpoint || strings.HasSuffix(path, endpoint) {
		return u.String()
	}
	if path == "" {
		u.Path = endpoint
	} else {
		u.Path = path + endpoint
	}
	return u.String()
}

func (c *Client) Connect(rawURL string, req model.ConnectRequest) (*model.ConnectResponse, error) {
	var resp model.ConnectResponse
	if err := c.postJSON(endpointURL(rawURL, "connect"), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Disconnect(rawURL string, req model.DisconnectRequest) (*model.DisconnectResponse, error) {
	var resp model.DisconnectResponse
	if err := c.postJSON(endpointURL(rawURL, "disconnect"), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Peers(rawURL string) ([]byte, error) {
	url := endpointURL(rawURL, "peers")
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
