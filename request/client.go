package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	ctrlserver "EasyWired/server"
	"EasyWired/wg"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) ListNodes(ctx context.Context) ([]ctrlserver.NodeRecord, error) {
	var out []ctrlserver.NodeRecord
	if err := c.doJSON(ctx, http.MethodGet, "/nodes", nil, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) RegisterNode(ctx context.Context, payload ctrlserver.RegisterNodeRequest) (*ctrlserver.RegisterNodeResponse, error) {
	var out ctrlserver.RegisterNodeResponse
	if err := c.doJSON(ctx, http.MethodPost, "/nodes/register", payload, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) GetNodeConfig(ctx context.Context, nodeID string) (*wg.NodeConfig, error) {
	var out wg.NodeConfig
	path := fmt.Sprintf("/nodes/%s/config", url.PathEscape(nodeID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) ConnectNodes(ctx context.Context, payload ctrlserver.ConnectRequest) (*ctrlserver.ConnectResponse, error) {
	var out ctrlserver.ConnectResponse
	if err := c.doJSON(ctx, http.MethodPost, "/connect", payload, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) SubscribeNodeEvents(ctx context.Context, nodeID string) (*http.Response, error) {
	path := fmt.Sprintf("/nodes/%s/events", url.PathEscape(nodeID))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, decodeAPIError(resp)
	}

	return resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, out any) error {
	req, err := c.newRequest(ctx, method, path, reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
		return fmt.Errorf("decode response (%s %s): %w", method, path, err)
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, reqBody any) (*http.Request, error) {
	var body io.Reader

	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request body (%s %s): %w", method, path, err)
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path), body)
	if err != nil {
		return nil, err
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) buildURL(path string) string {
	if strings.HasPrefix(path, "/") {
		return c.baseURL + path
	}

	return c.baseURL + "/" + path
}

func decodeAPIError(resp *http.Response) error {
	payload := ctrlserver.ErrorResponse{}
	body, _ := io.ReadAll(resp.Body)

	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Status:     resp.Status,
				Message:    payload.Error,
				Details:    payload.Details,
			}
		}
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Message:    msg,
	}
}
