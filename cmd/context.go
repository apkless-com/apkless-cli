package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Context stores the current default phone ID
type Context struct {
	PhoneID   string `json:"phone_id"`
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
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

// printCurrentPhone prints a dim line showing which phone is being used
func printCurrentPhone(phoneID string) {
	short := phoneID
	if len(short) > 8 {
		short = short[:8]
	}
	fmt.Fprintf(os.Stderr, "  %s %s\n", dim.Render("phone"), cyan.Render(short+"..."))
}
