package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestBackendFlagIsNotUserFacing(t *testing.T) {
	commands := map[string]*cobra.Command{
		"serve":      newServeCommand(),
		"connect":    newConnectCommand(),
		"disconnect": newDisconnectCommand(),
	}

	for name, cmd := range commands {
		if cmd.Flags().Lookup("backend") != nil {
			t.Fatalf("%s should not expose --backend", name)
		}
	}
}
