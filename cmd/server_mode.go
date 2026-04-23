package cmd

import (
	ctrl "EasyWired/server"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	serverModeOnPrefix   = "on://"
	serverModeOnlyPrefix = "only://"
	defaultCtrlAddr      = "127.0.0.1:8080"
)

type serverMode int

const (
	serverModeRemote serverMode = iota
	serverModeOn
	serverModeOnly
)

type serverOption struct {
	Mode       serverMode
	BaseURL    string
	ListenAddr string
}

func parseServerOption(raw string) (serverOption, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "http://" + defaultCtrlAddr
	}

	switch {
	case strings.HasPrefix(trimmed, serverModeOnPrefix):
		return parseCtrlServerOption(strings.TrimPrefix(trimmed, serverModeOnPrefix), serverModeOn)
	case strings.HasPrefix(trimmed, serverModeOnlyPrefix):
		return parseCtrlServerOption(strings.TrimPrefix(trimmed, serverModeOnlyPrefix), serverModeOnly)
	default:
		return serverOption{
			Mode:    serverModeRemote,
			BaseURL: trimmed,
		}, nil
	}
}

func parseCtrlServerOption(value string, mode serverMode) (serverOption, error) {
	target := strings.TrimSpace(value)
	if target == "" {
		target = defaultCtrlAddr
	}

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return serverOption{}, fmt.Errorf("invalid server format %q, expected %s", mode.prefix()+value, mode.prefix()+"host:port")
	}

	if host == "" {
		host = "127.0.0.1"
	}

	parsedPort, err := strconv.Atoi(port)
	if err != nil || parsedPort < 1 || parsedPort > 65535 {
		return serverOption{}, fmt.Errorf("invalid server port %q", port)
	}

	listenAddr := net.JoinHostPort(host, port)
	clientHost := normalizeClientHost(host)
	baseURL := "http://" + net.JoinHostPort(clientHost, port)

	return serverOption{
		Mode:       mode,
		BaseURL:    baseURL,
		ListenAddr: listenAddr,
	}, nil
}

func normalizeClientHost(host string) string {
	trimmed := strings.Trim(strings.TrimSpace(host), "[]")
	switch trimmed {
	case "", "0.0.0.0", "::":
		return "127.0.0.1"
	default:
		return trimmed
	}
}

func validRegisterRequiredInServer(payload ctrl.RegisterNodeRequest) error {
	if payload.Endpoint == "" {
		return fmt.Errorf("need endpoint")
	}

	if len(payload.AllowedCIDRs) <= 0 {
		return fmt.Errorf("need allowed CIDRs")
	}

	if payload.Address == "" {
		return fmt.Errorf("need address")
	}

	return nil
}

func (m serverMode) prefix() string {
	switch m {
	case serverModeOn:
		return serverModeOnPrefix
	case serverModeOnly:
		return serverModeOnlyPrefix
	default:
		return ""
	}
}
