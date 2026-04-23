package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"EasyWired/wg"
	"github.com/spf13/cobra"
)

const (
	defaultServer = "http://127.0.0.1:8080"
	envPrefix     = "EW_"
)

var rootCmd = &cobra.Command{
	Use:          "easywired",
	Short:        "WireGuard Automatic registration pull configuration tool.",
	SilenceUsage: true,
	Args:         cobra.NoArgs,
}

func Execute() error {
	InitRunCmd()
	InitListCmd()
	return rootCmd.Execute()
}

func resolvePrivateKey(raw string) (wg.PrivateKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		privateKey, err := wg.GeneratePrivateKey()
		if err != nil {
			return wg.PrivateKey{}, fmt.Errorf("generate private key failed: %w", err)
		}
		return privateKey, nil
	}

	privateKey, err := wg.ParsePrivateKey(trimmed)
	if err != nil {
		return wg.PrivateKey{}, fmt.Errorf("parse privkey failed: %w", err)
	}

	return privateKey, nil
}

func cleanValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func envKey(key string) string {
	key = strings.ReplaceAll(key, "-", "")
	return envPrefix + strings.ToUpper(key)
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(envKey(key)))
	if value != "" {
		return value
	}

	return fallback
}

func envCSV(key string) []string {
	value := strings.TrimSpace(os.Getenv(envKey(key)))
	if value == "" {
		return nil
	}

	return cleanValues(strings.Split(value, ","))
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(envKey(key)))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
