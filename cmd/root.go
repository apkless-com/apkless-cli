package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiBase = "https://api.apkless.com"
var apiKey string

var rootCmd = &cobra.Command{
	Use:   "apkless",
	Short: "APKless — Cloud Android packet capture",
	Long:  "Android HTTPS traffic capture API — no device, no root, no setup.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiKey, "key", "", "API key (or set APKLESS_KEY env)")

	rootCmd.AddCommand(phoneCmd)
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(captureCmd)
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
