package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage apps on a cloud phone",
}

var appListCmd = &cobra.Command{
	Use:   "list [phone-id]",
	Short: "List installed apps",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		data, err := serverRequest(serverURL, token, "GET", "/apps", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var apps []map[string]string
		json.Unmarshal(data, &apps)

		if len(apps) == 0 {
			fmt.Println("No third-party apps installed.")
			return
		}
		for _, a := range apps {
			fmt.Println(a["package"])
		}
	},
}

var appInstallCmd = &cobra.Command{
	Use:   "install [phone-id] <apk-path-or-url>",
	Short: "Install an APK",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var phoneID, source string
		if len(args) == 2 {
			phoneID = args[0]
			source = args[1]
		} else {
			phoneID = resolvePhoneID(nil, 0)
			source = args[0]
		}
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// URL install
		if len(source) > 4 && (source[:4] == "http") {
			fmt.Fprintf(os.Stderr, "Installing from URL: %s\n", source)
			data, err := serverRequest(serverURL, token, "POST", "/apps", map[string]string{
				"url": source,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			printJSON(data)
			return
		}

		// File install — upload via multipart
		fmt.Fprintf(os.Stderr, "Uploading %s...\n", source)

		file, err := os.Open(source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		// Use multipart upload
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		part, _ := writer.CreateFormFile("apk", filepath.Base(source))
		io.Copy(part, file)
		writer.Close()

		req, _ := http.NewRequest("POST", serverURL+"/apps", &b)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		printJSON(data)
	},
}

var appUninstallCmd = &cobra.Command{
	Use:   "uninstall [phone-id] <package>",
	Short: "Uninstall an app",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var phoneID, pkg string
		if len(args) == 2 {
			phoneID = args[0]
			pkg = args[1]
		} else {
			phoneID = resolvePhoneID(nil, 0)
			pkg = args[0]
		}
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		data, err := serverRequest(serverURL, token, "DELETE", "/apps/"+pkg, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		printJSON(data)
	},
}

func init() {
	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appInstallCmd)
	appCmd.AddCommand(appUninstallCmd)
}
