package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"easywired/internal/api"
	"easywired/internal/backend"
	"easywired/internal/client"
	"easywired/internal/config"
	"easywired/internal/model"
	"easywired/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected command: serve, connect, peers, disconnect, export")
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "connect":
		return runConnect(args[1:])
	case "peers":
		return runPeers(args[1:])
	case "disconnect":
		return runDisconnect(args[1:])
	case "export":
		return runExport(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	configPath := fs.String("config", "./config.json", "config path")
	device := fs.String("device", "wg0", "wireguard device")
	listen := fs.String("listen", ":8080", "http listen address")
	backendName := fs.String("backend", backend.NameAuto, "backend name")
	output := fs.String("output", "", "wireguard config output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := store.Open(*configPath)
	if err != nil {
		return err
	}
	be, err := backend.Select(*backendName)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if be.SupportsNativeApply() {
		if err := be.EnsureDevice(ctx, *device, st.Config()); err != nil {
			return err
		}
		if err := be.ApplyConfig(ctx, *device, st.Config()); err != nil {
			return err
		}
	} else if *output != "" {
		if err := exportConfig(ctx, be, st.Config(), *output); err != nil {
			return err
		}
	}
	srv := api.NewServer(api.Options{
		Store:      st,
		Backend:    be,
		DeviceName: *device,
		ListenAddr: *listen,
		OutputPath: *output,
	})
	return srv.ListenAndServe()
}

func runConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	configPath := fs.String("config", "./config.json", "config path")
	device := fs.String("device", "wg0", "wireguard device")
	url := fs.String("url", "", "connect url")
	backendName := fs.String("backend", backend.NameAuto, "backend name")
	output := fs.String("output", "", "wireguard config output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *url == "" {
		return fmt.Errorf("--url is required")
	}
	st, err := store.Open(*configPath)
	if err != nil {
		return err
	}
	cfg := st.Config()
	req := model.ConnectRequest{
		NodeID: cfg.NodeID,
		Interface: model.WGInterface{
			PublicKey:  cfg.Interface.PublicKey,
			Address:    cfg.Interface.Address,
			ListenPort: cfg.Interface.ListenPort,
		},
		ExtField: cfg.ExtField,
	}
	resp, err := client.New(10*time.Second).Connect(*url, config.SanitizeConnectRequest(req))
	if err != nil {
		return err
	}
	if resp.Assigned != nil && resp.Assigned.Address != "" {
		cfg.Interface.Address = resp.Assigned.Address
	}
	if resp.InterfaceRecommendation.MTU != 0 {
		cfg.Interface.MTU = resp.InterfaceRecommendation.MTU
	}
	if len(resp.InterfaceRecommendation.DNS) > 0 {
		cfg.Interface.DNS = resp.InterfaceRecommendation.DNS
	}
	_ = st.UpsertPeer(resp.Peer)
	if err := st.Save(); err != nil {
		return err
	}
	be, err := backend.Select(*backendName)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if be.SupportsNativeApply() {
		if err := be.ApplyInterface(ctx, *device, cfg.Interface); err != nil {
			return err
		}
		return be.AddOrUpdatePeer(ctx, *device, resp.Peer)
	}
	return exportConfig(ctx, be, cfg, *output)
}

func runPeers(args []string) error {
	fs := flag.NewFlagSet("peers", flag.ContinueOnError)
	url := fs.String("url", "", "peers url")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *url == "" {
		return fmt.Errorf("--url is required")
	}
	body, err := client.New(10 * time.Second).Peers(*url)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(body)
	if err == nil && len(body) > 0 && body[len(body)-1] != '\n' {
		_, err = os.Stdout.Write([]byte("\n"))
	}
	return err
}

func runDisconnect(args []string) error {
	fs := flag.NewFlagSet("disconnect", flag.ContinueOnError)
	configPath := fs.String("config", "./config.json", "config path")
	device := fs.String("device", "wg0", "wireguard device")
	url := fs.String("url", "", "disconnect url")
	publicKey := fs.String("public-key", "", "peer public key")
	backendName := fs.String("backend", backend.NameAuto, "backend name")
	output := fs.String("output", "", "wireguard config output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *url == "" || *publicKey == "" {
		return fmt.Errorf("--url and --public-key are required")
	}
	st, err := store.Open(*configPath)
	if err != nil {
		return err
	}
	_, err = client.New(10*time.Second).Disconnect(*url, model.DisconnectRequest{
		NodeID:    st.Config().NodeID,
		PublicKey: *publicKey,
	})
	if err != nil {
		return err
	}
	_ = st.RemovePeer(*publicKey)
	if err := st.Save(); err != nil {
		return err
	}
	be, err := backend.Select(*backendName)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if be.SupportsNativeApply() {
		return be.RemovePeer(ctx, *device, *publicKey)
	}
	return exportConfig(ctx, be, st.Config(), *output)
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	configPath := fs.String("config", "./config.json", "config path")
	output := fs.String("output", "", "wireguard config output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	cfg, err := store.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	return exportConfig(context.Background(), backend.NewWGConfigFile(), cfg, *output)
}

func exportConfig(ctx context.Context, be backend.Backend, cfg *model.NodeConfig, output string) error {
	if output == "" {
		return nil
	}
	b, err := be.ExportConfig(ctx, cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(output, b, 0o600)
}
