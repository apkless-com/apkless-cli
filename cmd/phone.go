package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	return fmt.Errorf("unsupported platform")
}

var phoneCmd = &cobra.Command{
	Use:   "phone",
	Short: "Manage cloud phones",
}

var phoneCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cloud phone",
	Run: func(cmd *cobra.Command, args []string) {
		region, _ := cmd.Flags().GetString("region")
		hours, _ := cmd.Flags().GetInt("hours")

		// Step 1: Create
		var phoneID string
		_, err := runWithSpinner(fmt.Sprintf("Creating phone in %s (%dh)", cyan.Render(region), hours), func() (string, error) {
			result, err := apiRequest("POST", "/v1/phones", map[string]interface{}{"region": region, "hours": hours})
			if err != nil {
				return "", err
			}
			phoneID, _ = result["id"].(string)
			return phoneID, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n  %s %v\n", fail, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "  %s Phone %s\n", dim.Render("ID"), bold.Render(phoneID[:8]+"..."))

		// Step 2: Wait for ready
		wait, _ := cmd.Flags().GetBool("wait")
		if !wait {
			return
		}

		var serverURL string
		start := time.Now()
		_, err = runWithSpinner("Provisioning cloud phone", func() (string, error) {
			for i := 0; i < 60; i++ {
				time.Sleep(5 * time.Second)
				r, err := apiRequest("GET", "/v1/phones/"+phoneID, nil)
				if err != nil {
					continue
				}
				status, _ := r["status"].(string)
				if status == "ready" {
					serverURL, _ = r["server_url"].(string)
					return "ready", nil
				}
				if status == "error" {
					return "", fmt.Errorf("provisioning failed")
				}
			}
			return "", fmt.Errorf("timeout after 5 minutes")
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n  %s %v\n", fail, err)
			os.Exit(1)
		}

		elapsed := time.Since(start).Round(time.Second)
		fmt.Fprintf(os.Stderr, "\n")

		// Get full details including web_url
		r, _ := apiRequest("GET", "/v1/phones/"+phoneID, nil)
		token, _ := r["server_token"].(string)
		webURL, _ := r["web_url"].(string)

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(0, 2).
			Render(fmt.Sprintf(
				"%s  Ready in %s\n\n"+
					"  %s  %s\n"+
					"  %s  %s",
				green.Render("●"),
				bold.Render(elapsed.String()),
				dim.Render("Phone ID"),
				phoneID,
				dim.Render("Web URL "),
				cyan.Render(webURL),
			))
		fmt.Fprintln(os.Stderr, box)

		saveContext(Context{PhoneID: phoneID, ServerURL: serverURL, Token: token})
		fmt.Fprintf(os.Stderr, "\n  %s Set as default phone\n", dim.Render("ℹ"))
	},
}

var phoneListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List your cloud phones",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		req, _ := http.NewRequest("GET", apiBase+"/v1/phones", nil)
		req.Header.Set("Authorization", "Bearer "+getAPIKey())
		r, err := httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		defer r.Body.Close()
		data, _ := io.ReadAll(r.Body)

		var phones []map[string]interface{}
		json.Unmarshal(data, &phones)

		if len(phones) == 0 {
			fmt.Println(dim.Render("  No phones. Create one with: apkless phone create"))
			return
		}

		var rows [][]string
		for _, p := range phones {
			id, _ := p["id"].(string)
			status, _ := p["status"].(string)
			region, _ := p["region"].(string)
			created, _ := p["created_at"].(string)
			expires, _ := p["expires_at"].(string)

			rows = append(rows, []string{
				id,
				statusStyle(status),
				region,
				formatTime(created),
				formatExpiry(expires),
			})
		}

		fmt.Println(renderTable(
			[]string{"ID", "STATUS", "REGION", "CREATED", "EXPIRES"},
			rows,
		))
	},
}

var phoneShowCmd = &cobra.Command{
	Use:   "show <phone-id>",
	Short: "Show phone details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := apiRequest("GET", "/v1/phones/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		id, _ := result["id"].(string)
		status, _ := result["status"].(string)
		region, _ := result["region"].(string)
		created, _ := result["created_at"].(string)
		expires, _ := result["expires_at"].(string)
		fmt.Println()
		printKV(
			"Phone ID", id,
			"Status", statusStyle(status),
			"Region", region,
			"Created", formatTime(created),
			"Expires", formatExpiry(expires),
		)
		webURL, _ := result["web_url"].(string)
		if webURL != "" {
			printKV("Web", cyan.Render(webURL))
		}
		fmt.Println()
	},
}

var phoneDestroyCmd = &cobra.Command{
	Use:     "destroy <phone-id>",
	Short:   "Destroy a cloud phone",
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := runWithSpinner("Destroying "+args[0][:8]+"...", func() (string, error) {
			_, err := apiRequest("DELETE", "/v1/phones/"+args[0], nil)
			return "", err
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
	},
}


var phoneUseCmd = &cobra.Command{
	Use:   "use <phone-id>",
	Short: "Set default phone for subsequent commands",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Verify phone exists and get connection info
		result, err := apiRequest("GET", "/v1/phones/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		serverURL, _ := result["server_url"].(string)
		token, _ := result["server_token"].(string)
		saveContext(Context{PhoneID: args[0], ServerURL: serverURL, Token: token})
		fmt.Fprintf(os.Stderr, "  %s Default phone set to %s\n", success, cyan.Render(args[0][:8]+"..."))
	},
}

var phoneConnectCmd = &cobra.Command{
	Use:   "connect [phone-id]",
	Short: "Connect local ADB to a cloud phone",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Check adb is installed
		if _, err := exec.LookPath("adb"); err != nil {
			fmt.Fprintf(os.Stderr, "  %s adb not found. Install Android SDK Platform Tools first.\n", fail)
			fmt.Fprintf(os.Stderr, "    https://developer.android.com/tools/releases/platform-tools\n")
			os.Exit(1)
		}

		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		// Detect our public IP
		var myIP string
		_, err = runWithSpinner("Detecting your public IP", func() (string, error) {
			for _, svc := range []string{"https://ifconfig.me", "https://api.ipify.org", "https://ip.sb"} {
				resp, e := http.Get(svc)
				if e != nil {
					continue
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				ip := strings.TrimSpace(string(body))
				if ip != "" && !strings.Contains(ip, "<") {
					myIP = ip
					return ip, nil
				}
			}
			return "", fmt.Errorf("cannot detect public IP")
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		// Call /connect to whitelist our IP
		var result map[string]interface{}
		_, err = runWithSpinner("Opening ADB access for "+myIP, func() (string, error) {
			data, err := serverRequest(serverURL, token, "POST", "/connect", map[string]string{"ip": myIP})
			if err != nil {
				return "", err
			}
			json.Unmarshal(data, &result)
			return "", nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		adbHost, _ := result["adb_host"].(string)
		adbPort, _ := result["adb_port"].(float64)
		clientIP, _ := result["client_ip"].(string)
		addr := fmt.Sprintf("%s:%d", adbHost, int(adbPort))

		fmt.Fprintf(os.Stderr, "  %s Your IP %s allowed\n", dim.Render("ℹ"), clientIP)

		// Connect ADB
		_, err = runWithSpinner("Connecting ADB to "+cyan.Render(addr), func() (string, error) {
			out, err := exec.Command("adb", "connect", addr).CombinedOutput()
			output := strings.TrimSpace(string(out))
			if err != nil {
				return "", fmt.Errorf("%s", output)
			}
			if !strings.Contains(output, "connected") {
				return "", fmt.Errorf("%s", output)
			}
			return output, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "\n")
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(0, 2).
			Render(fmt.Sprintf(
				"%s  ADB connected\n\n"+
					"  %s  %s\n"+
					"  %s  %s\n"+
					"  %s  %s",
				green.Render("●"),
				dim.Render("ADB     "),
				cyan.Render(addr),
				dim.Render("Shell   "),
				fmt.Sprintf("adb -s %s shell", addr),
				dim.Render("Scrcpy  "),
				fmt.Sprintf("scrcpy -s %s", addr),
			))
		fmt.Fprintln(os.Stderr, box)

		// Save ADB addr to context
		ctx := loadContext()
		ctx.ADBAddr = addr
		saveContext(ctx)
	},
}

var phoneOpenCmd = &cobra.Command{
	Use:   "open [phone-id]",
	Short: "Open phone sandbox in browser",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		result, err := apiRequest("GET", "/v1/phones/"+phoneID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		webURL, _ := result["web_url"].(string)
		if webURL == "" {
			fmt.Fprintf(os.Stderr, "  %s Phone is not ready yet\n", fail)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  %s Opening %s\n", success, cyan.Render(webURL))
		openBrowser(webURL)
	},
}

var phoneRestartCmd = &cobra.Command{
	Use:   "restart <phone-id>",
	Short: "Restart a cloud phone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := runWithSpinner("Restarting "+args[0][:8]+"...", func() (string, error) {
			_, err := apiRequest("POST", "/v1/phones/"+args[0]+"/restart", nil)
			return "", err
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  %s Phone will be available in 1-2 minutes\n", dim.Render("ℹ"))
	},
}

var phoneStatusCmd = &cobra.Command{
	Use:   "status [phone-id]",
	Short: "Quick status overview of current phone",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)

		// Phone info from API
		result, err := apiRequest("GET", "/v1/phones/"+phoneID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		status, _ := result["status"].(string)
		region, _ := result["region"].(string)
		expires, _ := result["expires_at"].(string)
		webURL, _ := result["web_url"].(string)

		fmt.Println()
		printKV(
			"Phone", phoneID[:12]+"...",
			"Status", statusStyle(status),
			"Region", region,
			"Expires", formatExpiry(expires),
		)
		if webURL != "" {
			printKV("Web", cyan.Render(webURL))
		}

		// ADB connection
		ctx := loadContext()
		if ctx.ADBAddr != "" {
			out, err := adbCmd("shell", "echo", "ok")
			if err == nil && strings.Contains(out, "ok") {
				printKV("ADB", green.Render("● connected")+" "+dim.Render(ctx.ADBAddr))
			} else {
				printKV("ADB", red.Render("● disconnected")+" "+dim.Render(ctx.ADBAddr))
			}
		} else {
			printKV("ADB", dim.Render("not connected (run: apkless phone connect)"))
		}

		// Capture status
		if status == "ready" {
			serverURL, _ := result["server_url"].(string)
			token, _ := result["server_token"].(string)
			if serverURL != "" && token != "" {
				capData, err := serverRequest(serverURL, token, "GET", "/capture", nil)
				if err == nil {
					var capResult map[string]interface{}
					json.Unmarshal(capData, &capResult)
					capStatus, _ := capResult["status"].(string)
					capPkg, _ := capResult["package"].(string)
					flowCount, _ := capResult["flow_count"].(float64)
					if capStatus == "capturing" {
						printKV("Capture", green.Render("● "+capPkg)+" "+dim.Render(fmt.Sprintf("(%d flows)", int(flowCount))))
					} else {
						printKV("Capture", dim.Render("○ idle"))
					}
				}
			}
		}
		fmt.Println()
	},
}

func init() {
	phoneCreateCmd.Flags().String("region", "beijing", "Region (e.g. beijing, ap-southeast-1, us-east-1)")
	phoneCreateCmd.Flags().Int("hours", 1, "Hours to allocate (1-24)")
	phoneCreateCmd.Flags().Bool("wait", true, "Wait for phone to be ready")
	phoneCmd.AddCommand(phoneCreateCmd)
	phoneCmd.AddCommand(phoneListCmd)
	phoneCmd.AddCommand(phoneShowCmd)
	phoneCmd.AddCommand(phoneStatusCmd)
	phoneCmd.AddCommand(phoneDestroyCmd)
	phoneCmd.AddCommand(phoneRestartCmd)
	phoneCmd.AddCommand(phoneConnectCmd)
	phoneCmd.AddCommand(phoneOpenCmd)
	phoneCmd.AddCommand(phoneUseCmd)
}
