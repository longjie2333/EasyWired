package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"easywired/internal/api"
	"easywired/internal/backend"
	"easywired/internal/client"
	"easywired/internal/config"
	"easywired/internal/model"
	"easywired/internal/store"

	"github.com/spf13/cobra"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := newRootCommand()
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "easywired",
		Short:         "WireGuard control plane agent",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("expected command: serve, connect, peers, disconnect, export")
		},
	}
	cmd.AddCommand(
		newServeCommand(),
		newConnectCommand(),
		newPeersCommand(),
		newDisconnectCommand(),
		newExportCommand(),
	)
	return cmd
}

func newServeCommand() *cobra.Command {
	var opts serveOptions
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the EasyWired HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(opts)
		},
	}
	cmd.Flags().StringVar(&opts.configPath, "config", "./config.json", "config path")
	cmd.Flags().StringVar(&opts.device, "device", "wg0", "wireguard device")
	cmd.Flags().StringVar(&opts.listen, "listen", ":8080", "http listen address")
	cmd.Flags().StringVar(&opts.output, "output", "", "wireguard config output")
	return cmd
}

type serveOptions struct {
	configPath string
	device     string
	listen     string
	output     string
}

func runServe(opts serveOptions) error {
	st, err := store.Open(opts.configPath)
	if err != nil {
		return err
	}
	be, err := backend.Select("")
	if err != nil {
		return err
	}
	ctx := context.Background()
	if be.SupportsNativeApply() {
		if err := be.EnsureDevice(ctx, opts.device, st.Config()); err != nil {
			return err
		}
		if err := be.ApplyConfig(ctx, opts.device, st.Config()); err != nil {
			return err
		}
	} else if opts.output != "" {
		if err := exportConfig(ctx, be, st.Config(), opts.output); err != nil {
			return err
		}
	}
	return api.NewServer(api.Options{
		Store:      st,
		Backend:    be,
		DeviceName: opts.device,
		ListenAddr: opts.listen,
		OutputPath: opts.output,
	}).ListenAndServe()
}

func newConnectCommand() *cobra.Command {
	var opts connectOptions
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect this node to a remote EasyWired node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(opts)
		},
	}
	cmd.Flags().StringVar(&opts.configPath, "config", "./config.json", "config path")
	cmd.Flags().StringVar(&opts.device, "device", "wg0", "wireguard device")
	cmd.Flags().StringVar(&opts.url, "url", "", "connect url")
	cmd.Flags().StringVar(&opts.output, "output", "", "wireguard config output")
	return cmd
}

type connectOptions struct {
	configPath string
	device     string
	url        string
	output     string
}

func runConnect(opts connectOptions) error {
	if opts.url == "" {
		return fmt.Errorf("--url is required")
	}
	st, err := store.OpenOrCreate(opts.configPath)
	if err != nil {
		return err
	}
	cfg := st.Config()
	resp, err := client.New(10*time.Second).Connect(opts.url, config.SanitizeConnectRequest(model.ConnectRequest{
		NodeID: cfg.NodeID,
		Interface: model.WGInterface{
			PublicKey:  cfg.Interface.PublicKey,
			Address:    cfg.Interface.Address,
			ListenPort: cfg.Interface.ListenPort,
		},
		ExtField: cfg.ExtField,
	}))
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
	be, err := backend.Select("")
	if err != nil {
		return err
	}
	ctx := context.Background()
	if be.SupportsNativeApply() {
		if err := be.EnsureDevice(ctx, opts.device, cfg); err != nil {
			return err
		}
		if err := be.ApplyInterface(ctx, opts.device, cfg.Interface); err != nil {
			return err
		}
		return be.AddOrUpdatePeer(ctx, opts.device, resp.Peer)
	}
	return exportConfig(ctx, be, cfg, opts.output)
}

func newPeersCommand() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:   "peers",
		Short: "Fetch peers from a remote EasyWired node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPeers(url)
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "peers url")
	return cmd
}

func runPeers(url string) error {
	if url == "" {
		return fmt.Errorf("--url is required")
	}
	body, err := client.New(10 * time.Second).Peers(url)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(body)
	if err == nil && len(body) > 0 && body[len(body)-1] != '\n' {
		_, err = os.Stdout.Write([]byte("\n"))
	}
	return err
}

func newDisconnectCommand() *cobra.Command {
	var opts disconnectOptions
	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnect a peer from a remote EasyWired node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisconnect(opts)
		},
	}
	cmd.Flags().StringVar(&opts.configPath, "config", "./config.json", "config path")
	cmd.Flags().StringVar(&opts.device, "device", "wg0", "wireguard device")
	cmd.Flags().StringVar(&opts.url, "url", "", "disconnect url")
	cmd.Flags().StringVar(&opts.publicKey, "public-key", "", "peer public key")
	cmd.Flags().StringVar(&opts.output, "output", "", "wireguard config output")
	return cmd
}

type disconnectOptions struct {
	configPath string
	device     string
	url        string
	publicKey  string
	output     string
}

func runDisconnect(opts disconnectOptions) error {
	if opts.url == "" || opts.publicKey == "" {
		return fmt.Errorf("--url and --public-key are required")
	}
	st, err := store.OpenOrCreate(opts.configPath)
	if err != nil {
		return err
	}
	_, err = client.New(10*time.Second).Disconnect(opts.url, model.DisconnectRequest{
		NodeID:    st.Config().NodeID,
		PublicKey: opts.publicKey,
	})
	if err != nil {
		return err
	}
	_ = st.RemovePeer(opts.publicKey)
	if err := st.Save(); err != nil {
		return err
	}
	be, err := backend.Select("")
	if err != nil {
		return err
	}
	ctx := context.Background()
	if be.SupportsNativeApply() {
		return be.RemovePeer(ctx, opts.device, opts.publicKey)
	}
	return exportConfig(ctx, be, st.Config(), opts.output)
}

func newExportCommand() *cobra.Command {
	var opts exportOptions
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a WireGuard configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(opts)
		},
	}
	cmd.Flags().StringVar(&opts.configPath, "config", "./config.json", "config path")
	cmd.Flags().StringVar(&opts.output, "output", "", "wireguard config output")
	return cmd
}

type exportOptions struct {
	configPath string
	output     string
}

func runExport(opts exportOptions) error {
	if opts.output == "" {
		return fmt.Errorf("--output is required")
	}
	cfg, err := store.LoadOrCreateConfig(opts.configPath)
	if err != nil {
		return err
	}
	return exportConfig(context.Background(), backend.NewWGConfigFile(), cfg, opts.output)
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
