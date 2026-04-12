package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture HTTP/HTTPS traffic",
}

var captureStartCmd = &cobra.Command{
	Use:   "start [phone-id] <package>",
	Short: "Start capturing traffic from an app",
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
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		_, err = runWithSpinner("Starting capture for "+cyan.Render(pkg), func() (string, error) {
			_, err := serverRequest(serverURL, token, "POST", "/capture", map[string]string{
				"package": pkg,
			})
			return "", err
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "  %s App launched, capturing traffic\n", dim.Render("ℹ"))
	},
}

var captureStopCmd = &cobra.Command{
	Use:   "stop [phone-id]",
	Short: "Stop capturing",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		_, err = runWithSpinner("Stopping capture", func() (string, error) {
			_, err := serverRequest(serverURL, token, "DELETE", "/capture", nil)
			return "", err
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
	},
}

var captureStatusCmd = &cobra.Command{
	Use:   "status [phone-id]",
	Short: "Show capture status",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		data, err := serverRequest(serverURL, token, "GET", "/capture", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		var result map[string]interface{}
		json.Unmarshal(data, &result)

		status, _ := result["status"].(string)
		pkg, _ := result["package"].(string)
		flowCount, _ := result["flow_count"].(float64)

		fmt.Println()
		if status == "capturing" {
			printKV(
				"Status", green.Render("● capturing"),
				"Package", cyan.Render(pkg),
				"Flows", fmt.Sprintf("%d", int(flowCount)),
			)
		} else {
			printKV("Status", dim.Render("○ idle"))
		}
		fmt.Println()
	},
}

var captureFlowsCmd = &cobra.Command{
	Use:   "flows [phone-id] [flow-id]",
	Short: "List or show captured traffic",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Detect if first arg is a flow ID (short) or phone ID (UUID)
		var phoneID string
		var flowID string
		for _, a := range args {
			if len(a) == 36 && a[8] == '-' {
				phoneID = a
			} else if len(a) > 0 {
				flowID = a
			}
		}
		if phoneID == "" {
			phoneID = resolvePhoneID(nil, 0)
		}
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		// Single flow detail
		if flowID != "" {
			data, err := serverRequest(serverURL, token, "GET", "/flows/"+flowID, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
				os.Exit(1)
			}

			var flow map[string]interface{}
			json.Unmarshal(data, &flow)

			url, _ := flow["url"].(string)
			method, _ := flow["method"].(string)
			status, _ := flow["status"].(float64)
			reason, _ := flow["reason"].(string)

			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
			bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

			fmt.Printf("\n  %s %s → %s %s\n", methodStyle(method), cyan.Render(url), statusCodeStyle(int(status)), dim.Render(reason))

			if req, ok := flow["request"].(map[string]interface{}); ok {
				if body, ok := req["body"].(string); ok && body != "" {
					fmt.Printf("\n  %s\n", headerStyle.Render("▶ Request Body"))
					fmt.Printf("  %s\n", bodyStyle.Render(body))
				}
			}
			if resp, ok := flow["response"].(map[string]interface{}); ok {
				if body, ok := resp["body"].(string); ok && body != "" {
					fmt.Printf("\n  %s\n", headerStyle.Render("◀ Response Body"))
					fmt.Printf("  %s\n", bodyStyle.Render(body))
				}
			}
			fmt.Println()
			return
		}

		// Flow list
		limit, _ := cmd.Flags().GetInt("limit")
		host, _ := cmd.Flags().GetString("host")
		method, _ := cmd.Flags().GetString("method")

		path := fmt.Sprintf("/flows?limit=%d", limit)
		if host != "" {
			path += "&host=" + host
		}
		if method != "" {
			path += "&method=" + method
		}

		data, err := serverRequest(serverURL, token, "GET", path, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		var result map[string]interface{}
		json.Unmarshal(data, &result)

		flows, _ := result["flows"].([]interface{})
		total, _ := result["total"].(float64)

		if len(flows) == 0 {
			fmt.Println(dim.Render("  No flows captured."))
			return
		}

		var rows [][]string
		for i, f := range flows {
			fl, _ := f.(map[string]interface{})
			m, _ := fl["method"].(string)
			h, _ := fl["host"].(string)
			p, _ := fl["path"].(string)
			s, _ := fl["status"].(float64)
			ms, _ := fl["duration_ms"].(float64)
			id, _ := fl["id"].(string)

			if len(p) > 45 {
				p = p[:45] + "..."
			}
			if len(h) > 28 {
				h = h[:28] + "..."
			}

			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1),
				methodStyle(m),
				h,
				p,
				statusCodeStyle(int(s)),
				fmt.Sprintf("%dms", int(ms)),
				dim.Render(id[:8]),
			})
		}

		fmt.Println(renderTable(
			[]string{"#", "METHOD", "HOST", "PATH", "STATUS", "TIME", "ID"},
			rows,
		))
		fmt.Fprintf(os.Stderr, "  %s\n", dim.Render(fmt.Sprintf("%d/%d flows", len(flows), int(total))))
	},
}

var captureClearCmd = &cobra.Command{
	Use:   "clear [phone-id]",
	Short: "Clear all captured traffic",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		serverRequest(serverURL, token, "DELETE", "/flows", nil)
		fmt.Fprintf(os.Stderr, "  %s Flows cleared\n", success)
	},
}

func init() {
	captureFlowsCmd.Flags().Int("limit", 50, "Max flows to show")
	captureFlowsCmd.Flags().String("host", "", "Filter by host")
	captureFlowsCmd.Flags().String("method", "", "Filter by method")

	captureCmd.AddCommand(captureStartCmd)
	captureCmd.AddCommand(captureStopCmd)
	captureCmd.AddCommand(captureStatusCmd)
	captureCmd.AddCommand(captureFlowsCmd)
	captureCmd.AddCommand(captureClearCmd)
}
