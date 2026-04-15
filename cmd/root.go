package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiBase = "https://api.apkless.com"
var apiKey string

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "apkless",
	Short:   "APKless — Cloud Android packet capture",
	Long:    "Cloud Android HTTPS capture — no device, no root, no setup.",
	Version: Version,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiKey, "key", "", "API key (or set APKLESS_KEY env)")

	// Help groups
	rootCmd.AddGroup(
		&cobra.Group{ID: "device", Title: "Device Management:"},
		&cobra.Group{ID: "adb", Title: "ADB Operations:"},
		&cobra.Group{ID: "capture", Title: "Traffic Capture:"},
	)

	// Device management (top-level)
	rootCmd.AddCommand(createCmd, lsCmd, showCmd, rmCmd,
		restartCmd, connectCmd, openCmd, useCmd, statusCmd)
	// ADB operations (top-level)
	rootCmd.AddCommand(shellCmd, installCmd, uninstallCmd,
		launchCmd, appsCmd, screenCmd, pushCmd, pullCmd)
	// Capture (subcommand group)
	rootCmd.AddCommand(captureCmd)

	// Hidden legacy groups (backward compat)
	phoneCmd.Hidden = true
	appCmd.Hidden = true
	rootCmd.AddCommand(phoneCmd, appCmd)
}

func getAPIKey() string {
	if apiKey != "" {
		return apiKey
	}
	if k := os.Getenv("APKLESS_KEY"); k != "" {
		return k
	}
	fmt.Fprintln(os.Stderr, "Error: API key required. Set APKLESS_KEY or use --key")
	os.Exit(1)
	return ""
}
