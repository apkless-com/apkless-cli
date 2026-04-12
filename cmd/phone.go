package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func runCommand(name string, args ...string) {
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Run()
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

		// Step 1: Create
		var phoneID string
		_, err := runWithSpinner("Creating phone in "+cyan.Render(region), func() (string, error) {
			result, err := apiRequest("POST", "/v1/phones", map[string]string{"region": region})
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
				dim.Render("Server  "),
				serverURL,
			))
		fmt.Fprintln(os.Stderr, box)

		// Save as default context
		r, _ := apiRequest("GET", "/v1/phones/"+phoneID, nil)
		token, _ := r["server_token"].(string)
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
				timeAgo(created),
				expiresIn(expires),
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
		serverURL, _ := result["server_url"].(string)

		fmt.Println()
		printKV(
			"Phone ID", id,
			"Status", statusStyle(status),
			"Region", region,
			"Created", timeAgo(created),
			"Expires", expiresIn(expires),
		)
		if serverURL != "" {
			printKV("Server", serverURL)
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

var phoneConnectCmd = &cobra.Command{
	Use:   "connect <phone-id>",
	Short: "Connect to phone via ADB + scrcpy",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := apiRequest("GET", "/v1/phones/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		serverURL, _ := result["server_url"].(string)
		if serverURL == "" {
			fmt.Fprintf(os.Stderr, "  %s Phone not ready\n", fail)
			os.Exit(1)
		}

		ip := serverURL[7:]
		for i := len(ip) - 1; i >= 0; i-- {
			if ip[i] == ':' {
				ip = ip[:i]
				break
			}
		}

		addr := ip + ":5555"
		fmt.Printf("\n  %s  %s\n", dim.Render("ADB"), cyan.Render(addr))
		fmt.Printf("\n  Run:\n")
		fmt.Printf("  %s\n", bold.Render("adb connect "+addr))
		fmt.Printf("  %s\n\n", bold.Render("scrcpy -s "+addr+" -b 2M -m 720 --max-fps 10 --no-audio"))

		doExec, _ := cmd.Flags().GetBool("exec")
		if doExec {
			runCommand("adb", "connect", addr)
			runCommand("scrcpy", "-s", addr, "-b", "2M", "-m", "720", "--max-fps", "10", "--no-audio")
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

func init() {
	phoneCreateCmd.Flags().String("region", "beijing", "Region")
	phoneCreateCmd.Flags().Bool("wait", true, "Wait for phone to be ready")
	phoneConnectCmd.Flags().Bool("exec", false, "Auto-run adb + scrcpy")

	phoneCmd.AddCommand(phoneCreateCmd)
	phoneCmd.AddCommand(phoneListCmd)
	phoneCmd.AddCommand(phoneShowCmd)
	phoneCmd.AddCommand(phoneDestroyCmd)
	phoneCmd.AddCommand(phoneConnectCmd)
	phoneCmd.AddCommand(phoneUseCmd)
}
