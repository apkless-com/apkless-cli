package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// flowToCurl converts a flow detail to a cURL command
func flowToCurl(flow map[string]interface{}) string {
	req, _ := flow["request"].(map[string]interface{})
	if req == nil {
		return ""
	}
	method, _ := req["method"].(string)
	u, _ := req["url"].(string)
	body, _ := req["body"].(string)

	parts := []string{"curl"}
	if method != "" && method != "GET" {
		parts = append(parts, "-X", method)
	}

	if headers, ok := req["headers"].([]interface{}); ok {
		for _, h := range headers {
			pair, _ := h.([]interface{})
			if len(pair) == 2 {
				k := fmt.Sprint(pair[0])
				v := fmt.Sprint(pair[1])
				if strings.EqualFold(k, "host") || strings.EqualFold(k, "content-length") {
					continue
				}
				parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, v))
			}
		}
	}

	if body != "" {
		parts = append(parts, "-d", fmt.Sprintf("'%s'", strings.ReplaceAll(body, "'", "\\'")))
	}

	parts = append(parts, fmt.Sprintf("'%s'", u))
	return strings.Join(parts, " ")
}

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

		format, _ := cmd.Flags().GetString("format")

		// Single flow detail
		if flowID != "" {
			data, err := serverRequest(serverURL, token, "GET", "/flows/"+flowID, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
				os.Exit(1)
			}

			var flow map[string]interface{}
			json.Unmarshal(data, &flow)

			if format == "curl" {
				fmt.Println(flowToCurl(flow))
				return
			}
			if format == "json" {
				b, _ := json.MarshalIndent(flow, "", "  ")
				fmt.Println(string(b))
				return
			}

			u, _ := flow["url"].(string)
			method, _ := flow["method"].(string)
			status, _ := flow["status"].(float64)
			reason, _ := flow["reason"].(string)

			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
			bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

			fmt.Printf("\n  %s %s → %s %s\n", methodStyle(method), cyan.Render(u), statusCodeStyle(int(status)), dim.Render(reason))

			if req, ok := flow["request"].(map[string]interface{}); ok {
				if headers, ok := req["headers"].([]interface{}); ok && len(headers) > 0 {
					fmt.Printf("\n  %s\n", headerStyle.Render("▶ Request Headers"))
					for _, h := range headers {
						if pair, ok := h.([]interface{}); ok && len(pair) == 2 {
							fmt.Printf("  %s: %s\n", dim.Render(fmt.Sprint(pair[0])), fmt.Sprint(pair[1]))
						}
					}
				}
				if body, ok := req["body"].(string); ok && body != "" {
					fmt.Printf("\n  %s\n", headerStyle.Render("▶ Request Body"))
					fmt.Printf("  %s\n", bodyStyle.Render(prettyJSON(body)))
				}
			}
			if resp, ok := flow["response"].(map[string]interface{}); ok {
				if headers, ok := resp["headers"].([]interface{}); ok && len(headers) > 0 {
					fmt.Printf("\n  %s\n", headerStyle.Render("◀ Response Headers"))
					for _, h := range headers {
						if pair, ok := h.([]interface{}); ok && len(pair) == 2 {
							fmt.Printf("  %s: %s\n", dim.Render(fmt.Sprint(pair[0])), fmt.Sprint(pair[1]))
						}
					}
				}
				if body, ok := resp["body"].(string); ok && body != "" {
					fmt.Printf("\n  %s\n", headerStyle.Render("◀ Response Body"))
					fmt.Printf("  %s\n", bodyStyle.Render(prettyJSON(body)))
				}
			}

			fmt.Printf("\n  %s\n\n", dim.Render("cURL: "+flowToCurl(flow)))
			return
		}

		// Flow list
		limit, _ := cmd.Flags().GetInt("limit")
		host, _ := cmd.Flags().GetString("host")
		method, _ := cmd.Flags().GetString("method")

		path := fmt.Sprintf("/flows?limit=%d", limit)
		if host != "" {
			path += "&host=" + url.QueryEscape(host)
		}
		if method != "" {
			path += "&method=" + url.QueryEscape(method)
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

		if format == "json" {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
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

var captureWatchCmd = &cobra.Command{
	Use:   "watch [phone-id]",
	Short: "Watch traffic in real-time (like tail -f)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		format, _ := cmd.Flags().GetString("format")
		seen := make(map[string]bool)
		fmt.Fprintf(os.Stderr, "  %s Watching traffic (Ctrl+C to stop)\n\n", dim.Render("ℹ"))

		for {
			data, err := serverRequest(serverURL, token, "GET", "/flows?limit=200", nil)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			var result map[string]interface{}
			json.Unmarshal(data, &result)
			flows, _ := result["flows"].([]interface{})

			for _, f := range flows {
				fl, _ := f.(map[string]interface{})
				id, _ := fl["id"].(string)
				if id == "" || seen[id] {
					continue
				}
				seen[id] = true

				m, _ := fl["method"].(string)
				h, _ := fl["host"].(string)
				p, _ := fl["path"].(string)
				s, _ := fl["status"].(float64)
				ms, _ := fl["duration_ms"].(float64)
				u, _ := fl["url"].(string)

				if format == "curl" {
					// Fetch full detail for cURL
					detail, err := serverRequest(serverURL, token, "GET", "/flows/"+id, nil)
					if err == nil {
						var flow map[string]interface{}
						json.Unmarshal(detail, &flow)
						fmt.Println(flowToCurl(flow))
					}
				} else {
					fmt.Printf("  %s %-4s %s%s → %s %s\n",
						dim.Render(id[:8]),
						methodStyle(m),
						h, p,
						statusCodeStyle(int(s)),
						dim.Render(fmt.Sprintf("%dms", int(ms))),
					)
					_ = u
				}
			}

			time.Sleep(2 * time.Second)
		}
	},
}

var captureExportCmd = &cobra.Command{
	Use:   "export [phone-id]",
	Short: "Export captured traffic as HAR",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		phoneID := resolvePhoneID(args, 0)
		printCurrentPhone(phoneID)
		serverURL, token, err := getPhoneConnection(phoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		output, _ := cmd.Flags().GetString("output")

		// Fetch all flows
		data, err := serverRequest(serverURL, token, "GET", "/flows?limit=1000", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}

		var result map[string]interface{}
		json.Unmarshal(data, &result)
		flows, _ := result["flows"].([]interface{})

		// Build HAR
		var entries []map[string]interface{}
		for _, f := range flows {
			fl, _ := f.(map[string]interface{})
			id, _ := fl["id"].(string)
			m, _ := fl["method"].(string)
			u, _ := fl["url"].(string)
			s, _ := fl["status"].(float64)
			ms, _ := fl["duration_ms"].(float64)

			// Fetch detail for headers
			detail, err := serverRequest(serverURL, token, "GET", "/flows/"+id, nil)
			if err != nil {
				continue
			}
			var flow map[string]interface{}
			json.Unmarshal(detail, &flow)

			req, _ := flow["request"].(map[string]interface{})
			resp, _ := flow["response"].(map[string]interface{})

			reqHeaders := convertHeaders(req)
			respHeaders := convertHeaders(resp)
			reqBody, _ := req["body"].(string)
			respBody, _ := resp["body"].(string)

			entry := map[string]interface{}{
				"startedDateTime": fl["timestamp"],
				"time":            ms,
				"request": map[string]interface{}{
					"method":      m,
					"url":         u,
					"httpVersion": "HTTP/1.1",
					"headers":     reqHeaders,
					"queryString": []interface{}{},
					"bodySize":    len(reqBody),
					"postData": map[string]interface{}{
						"mimeType": "",
						"text":     reqBody,
					},
				},
				"response": map[string]interface{}{
					"status":      int(s),
					"statusText":  "",
					"httpVersion": "HTTP/1.1",
					"headers":     respHeaders,
					"content": map[string]interface{}{
						"size":     len(respBody),
						"mimeType": "",
						"text":     respBody,
					},
					"bodySize": len(respBody),
				},
				"timings": map[string]interface{}{
					"send": 0, "wait": ms, "receive": 0,
				},
			}
			entries = append(entries, entry)
		}

		har := map[string]interface{}{
			"log": map[string]interface{}{
				"version": "1.2",
				"creator": map[string]string{"name": "APKless", "version": "0.3.0"},
				"entries": entries,
			},
		}

		harJSON, _ := json.MarshalIndent(har, "", "  ")

		if output == "" {
			fmt.Println(string(harJSON))
		} else {
			os.WriteFile(output, harJSON, 0644)
			fmt.Fprintf(os.Stderr, "  %s Exported %d flows to %s\n", success, len(entries), cyan.Render(output))
		}
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

// helpers

func prettyJSON(s string) string {
	var v interface{}
	if json.Unmarshal([]byte(s), &v) == nil {
		b, err := json.MarshalIndent(v, "  ", "  ")
		if err == nil {
			return string(b)
		}
	}
	return s
}

func convertHeaders(obj map[string]interface{}) []map[string]string {
	if obj == nil {
		return nil
	}
	headers, ok := obj["headers"].([]interface{})
	if !ok {
		return nil
	}
	var result []map[string]string
	for _, h := range headers {
		pair, _ := h.([]interface{})
		if len(pair) == 2 {
			result = append(result, map[string]string{
				"name":  fmt.Sprint(pair[0]),
				"value": fmt.Sprint(pair[1]),
			})
		}
	}
	return result
}

func init() {
	captureFlowsCmd.Flags().Int("limit", 50, "Max flows to show")
	captureFlowsCmd.Flags().String("host", "", "Filter by host")
	captureFlowsCmd.Flags().String("method", "", "Filter by method")
	captureFlowsCmd.Flags().String("format", "", "Output format: curl, json")

	captureWatchCmd.Flags().String("format", "", "Output format: curl")

	captureExportCmd.Flags().String("output", "", "Output file (default: stdout)")

	captureCmd.AddCommand(captureStartCmd)
	captureCmd.AddCommand(captureStopCmd)
	captureCmd.AddCommand(captureStatusCmd)
	captureCmd.AddCommand(captureFlowsCmd)
	captureCmd.AddCommand(captureWatchCmd)
	captureCmd.AddCommand(captureExportCmd)
	captureCmd.AddCommand(captureClearCmd)
}
