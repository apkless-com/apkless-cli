package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Context stores the current default phone ID and connection info
type Context struct {
	PhoneID   string `json:"phone_id"`
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	ADBAddr   string `json:"adb_addr,omitempty"` // e.g. "1.2.3.4:15555"
}

func contextPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".apkless", "context.json")
}

func saveContext(ctx Context) error {
	dir := filepath.Dir(contextPath())
	os.MkdirAll(dir, 0755)
	data, _ := json.Marshal(ctx)
	return os.WriteFile(contextPath(), data, 0600)
}

func loadContext() Context {
	data, err := os.ReadFile(contextPath())
	if err != nil {
		return Context{}
	}
	var ctx Context
	json.Unmarshal(data, &ctx)
	return ctx
}

// resolvePhoneID returns phone ID from arg or context
func resolvePhoneID(args []string, pos int) string {
	if pos < len(args) && args[pos] != "" {
		return args[pos]
	}
	ctx := loadContext()
	if ctx.PhoneID != "" {
		return ctx.PhoneID
	}
	fmt.Fprintln(os.Stderr, "  "+fail.Render("")+" No phone specified. Use <phone-id> or set default with: apkless phone use <id>")
	os.Exit(1)
	return ""
}

// requireADB returns the ADB address from context, exits if not connected
func requireADB() string {
	ctx := loadContext()
	if ctx.ADBAddr == "" {
		fmt.Fprintln(os.Stderr, "  "+fail.Render("")+" ADB not connected. Run first: apkless phone connect")
		os.Exit(1)
	}
	return ctx.ADBAddr
}

// adbCmd runs an adb command against the connected device
func adbCmd(args ...string) (string, error) {
	addr := requireADB()
	allArgs := append([]string{"-s", addr}, args...)
	out, err := exec.Command("adb", allArgs...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// printCurrentPhone prints a dim line showing which phone is being used
func printCurrentPhone(phoneID string) {
	short := phoneID
	if len(short) > 8 {
		short = short[:8]
	}
	fmt.Fprintf(os.Stderr, "  %s %s\n", dim.Render("phone"), cyan.Render(short+"..."))
}
