package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"EasyWired/request"
	"github.com/spf13/cobra"
)

var (
	listServer string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all nodes",
	Args:  cobra.NoArgs,
	RunE:  ListCmdMain,
}

func InitListCmd() {
	flags := listCmd.Flags()
	flags.StringVarP(&listServer, flagServer, "s", envOrDefault(flagServer, defaultServer), "controller base URL")
	rootCmd.AddCommand(listCmd)
}

func ListCmdMain(_ *cobra.Command, _ []string) error {
	client := request.NewClient(listServer, nil)
	nodes, err := client.ListNodes(context.Background())
	if err != nil {
		return fmt.Errorf("list nodes failed: %w", err)
	}
	if len(nodes) == 0 {
		fmt.Println("no nodes")
		return nil
	}

	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if id := strings.TrimSpace(node.NodeID); id != "" {
			ids = append(ids, id)
		}
	}

	sort.Strings(ids)
	for _, id := range ids {
		fmt.Println(" - " + id)
	}

	return nil
}
