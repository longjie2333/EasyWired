package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"EasyWired/request"
	ctrl "EasyWired/server"
	"EasyWired/wg"
	"github.com/denisbrodbeck/machineid"
	"github.com/spf13/cobra"
)

const (
	flagServer             = "server"
	flagPrivKey            = "privkey"
	flagInterface          = "interface"
	flagEndpoint           = "endpoint"
	flagAddress            = "address"
	flagAllowedCIDRs       = "allowed-cidrs"
	flagListenPort         = "listen-port"
	flagSuggestedMTU       = "mtu"
	flagSuggestedDNS       = "dns"
	flagSuggestedKeepalive = "keepalive"
	flagConnect            = "connect"
)

var (
	runCmdLongText = fmt.Sprintf(`All flag values listed below can also be provided via environment variables.
Just prefix the flag key name (in uppercase) with %s.
For example %s=10.0.0.1, %s=53.`,
		envPrefix, envKey(flagAddress), envKey(flagListenPort))
	runCmd = &cobra.Command{
		Use:     "run",
		Short:   "Run with default parameters or specified parameters",
		Long:    runCmdLongText,
		Args:    cobra.NoArgs,
		PreRunE: PreRunCmdRun,
		RunE:    RunCmdMain,
	}
)

var (
	server             string
	privKey            string
	interfaceName      string
	endpoint           string
	address            string
	allowedCIDRs       []string
	listenPort         int
	suggestedMTU       int
	suggestedDNS       []string
	suggestedKeepalive int
	toNodeID           string

	regPayload     ctrl.RegisterNodeRequest
	serverSettings serverOption
)

func InitRunCmd() {
	rootCmd.AddCommand(runCmd)

	flags := runCmd.Flags()
	flags.StringVarP(&server, flagServer, "s", envOrDefault(flagServer, defaultServer), "controller base URL or run server")
	flags.StringVarP(&privKey, flagPrivKey, "P", envOrDefault(flagPrivKey, ""), "WireGuard private key")
	flags.StringVarP(&interfaceName, flagInterface, "i", envOrDefault(flagInterface, "EasyWired"), "WireGuard adapter/interface name")
	flags.StringVarP(&endpoint, flagEndpoint, "e", envOrDefault(flagEndpoint, ""), "node endpoint")
	flags.StringVarP(&address, flagAddress, "a", envOrDefault(flagAddress, ""), "node address")
	flags.StringSliceVarP(&allowedCIDRs, flagAllowedCIDRs, "c", envCSV(flagAllowedCIDRs), "node allowed CIDRs")
	flags.IntVarP(&listenPort, flagListenPort, "p", envIntOrDefault(flagListenPort, 0), "node listen port")
	flags.IntVarP(&suggestedMTU, flagSuggestedMTU, "m", envIntOrDefault(flagSuggestedMTU, 0), "recommended MTU values for end nodes")
	flags.StringSliceVarP(&suggestedDNS, flagSuggestedDNS, "d", envCSV(flagSuggestedDNS), "recommended DNS values for end nodes")
	flags.IntVarP(&suggestedKeepalive, flagSuggestedKeepalive, "k", envIntOrDefault(flagSuggestedKeepalive, 0), "recommended Keepalive values for end nodes")
	flags.StringVarP(&toNodeID, flagConnect, "C", envOrDefault(flagConnect, ""), "pre connect node to specified node")
}

func PreRunCmdRun(cmd *cobra.Command, args []string) error {
	regPayload = ctrl.RegisterNodeRequest{}

	resolved, err := parseServerOption(server)
	if err != nil {
		return err
	}
	serverSettings = resolved

	if endpoint != "" {
		regPayload.Endpoint = endpoint
	}

	if address != "" {
		regPayload.Address = address
	}

	if values := cleanValues(allowedCIDRs); len(values) > 0 {
		regPayload.AllowedCIDRs = values
	}

	if listenPort < 0 || listenPort > 65535 {
		return fmt.Errorf("invalid port %d, must be 1~65535", listenPort)
	} else if listenPort != 0 {
		regPayload.ListenPort = listenPort
	}

	if suggestedMTU != 0 {
		v := suggestedMTU
		regPayload.SuggestedMTU = &v
	}

	if values := cleanValues(suggestedDNS); len(values) > 0 {
		regPayload.SuggestedDNS = values
	}

	if suggestedKeepalive != 0 {
		regPayload.SuggestedKeepalive = &suggestedKeepalive
	}

	return nil
}

func RunCmdMain(_ *cobra.Command, _ []string) error {
	nodeID, err := machineid.ID()
	if err != nil {
		return fmt.Errorf("get machine id failed: %w", err)
	}
	if serverSettings.Mode != serverModeRemote {
		nodeID = "server-" + nodeID
	}

	privateKey, err := resolvePrivateKey(privKey)
	if err != nil {
		return err
	}

	regPayload.NodeID = nodeID
	regPayload.PublicKey = privateKey.PublicKey().String()

	var ctrlServer *ctrl.Service
	if serverSettings.Mode == serverModeOn || serverSettings.Mode == serverModeOnly {
		ctrlServer = ctrl.New()
		if err := ctrlServer.Start(serverSettings.ListenAddr); err != nil {
			return fmt.Errorf("start ctrl server failed: %w", err)
		}

		if serverSettings.Mode == serverModeOnly {
			fmt.Println("ctrl server only mode")
		}

		if err := validRegisterRequiredInServer(regPayload); err != nil {
			return fmt.Errorf("some param are missing when running the server: %w", err)
		}

		fmt.Printf("ctrl server started at http://%s\n", serverSettings.ListenAddr)

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = ctrlServer.Shutdown(shutdownCtx)
		}()
	}

	client := request.NewClient(serverSettings.BaseURL, nil)
	ctx := context.Background()

	if _, err := client.RegisterNode(ctx, regPayload); err != nil {
		return fmt.Errorf("register node failed: %w", err)
	}

	fmt.Printf("node %s registered.\n", nodeID)

	if serverSettings.Mode == serverModeOnly {
		waitForInterrupt()
		return nil
	}

	trimmedInterface := strings.TrimSpace(interfaceName)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})

	go client.StartSSE(ctx, nodeID, privateKey, trimmedInterface, ready)

	select {
	case <-ready:
		if toNodeID != "" {
			connRequest := ctrl.BuildConnectRequest(nodeID, toNodeID)
			resp, err := client.ConnectNodes(context.Background(), connRequest)
			if err != nil {
				return fmt.Errorf("connect nodes failed: %w", err)
			}

			if resp != nil && resp.AlreadyConnected != nil && *resp.AlreadyConnected {
				fmt.Printf("node %s already connected.\n", nodeID)
			}

			if resp != nil && resp.AllocatedIP != nil {
				fmt.Printf("node connected and allocated ip is %s\n", *resp.AllocatedIP)
				if err := applyAllocatedIPToInterface(client, nodeID, privateKey, trimmedInterface, *resp.AllocatedIP); err != nil {
					return err
				}
			}
		}
	}

	waitForInterrupt()

	if err := wg.Shutdown(); err != nil {
		return fmt.Errorf("wg shutdown error: %w", err)
	}

	fmt.Println("wg stopped")
	if serverSettings.Mode == serverModeOn {
		fmt.Println("server stopped")
	}

	return nil
}

func waitForInterrupt() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	<-sigCh
}

func applyAllocatedIPToInterface(client *request.Client, nodeID string, privateKey wg.PrivateKey, interfaceName, allocatedIP string) error {
	trimmedAllocatedIP := strings.TrimSpace(allocatedIP)
	if trimmedAllocatedIP == "" {
		return nil
	}

	cfg, err := client.GetNodeConfig(context.Background(), nodeID)
	if err != nil {
		return fmt.Errorf("get node config failed after connect: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("empty node config after connect")
	}

	cfg.Address = trimmedAllocatedIP
	if name := strings.TrimSpace(interfaceName); name != "" {
		cfg.InterfaceName = name
	}

	if err := wg.ApplyConfig(privateKey, cfg, true); err != nil {
		return fmt.Errorf("apply allocated ip to interface failed: %w", err)
	}

	fmt.Printf("allocated ip %s applied to interface %s\n", trimmedAllocatedIP, cfg.InterfaceName)
	return nil
}
