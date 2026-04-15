package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func newExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// Legacy group (hidden)
var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage apps on a cloud phone",
}

// UUID pattern for detecting phone IDs
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ── Top-level commands ──

var appsCmd = &cobra.Command{
	Use:     "apps",
	Short:   "List installed third-party apps",
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := adbCmd("shell", "pm", "list", "packages", "-3")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, out)
			os.Exit(1)
		}
		if out == "" {
			fmt.Println(dim.Render("  No third-party apps installed."))
			return
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "package:") {
				fmt.Println("  " + strings.TrimPrefix(line, "package:"))
			}
		}
	},
}

var installCmd = &cobra.Command{
	Use:     "install <apk-path>",
	Short:   "Install an APK file",
	Args:    cobra.ExactArgs(1),
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		if _, err := os.Stat(source); err != nil {
			fmt.Fprintf(os.Stderr, "  %s File not found: %s\n", fail, source)
			os.Exit(1)
		}
		_, err := runWithSpinner("Installing "+cyan.Render(source), func() (string, error) {
			out, err := adbCmd("install", "-r", source)
			if err != nil {
				return "", fmt.Errorf("%s", out)
			}
			return out, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <package>",
	Short:   "Uninstall an app",
	Args:    cobra.ExactArgs(1),
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		pkg := args[0]
		_, err := runWithSpinner("Uninstalling "+cyan.Render(pkg), func() (string, error) {
			out, err := adbCmd("shell", "pm", "uninstall", pkg)
			if err != nil {
				return "", fmt.Errorf("%s", out)
			}
			return out, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
	},
}

var launchCmd = &cobra.Command{
	Use:     "launch <package>",
	Short:   "Launch an app",
	Args:    cobra.ExactArgs(1),
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		pkg := args[0]
		out, err := adbCmd("shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, out)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  %s Launched %s\n", success, cyan.Render(pkg))
	},
}

var screenCmd = &cobra.Command{
	Use:     "screen [output-path]",
	Short:   "Take a screenshot",
	Args:    cobra.MaximumNArgs(1),
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		outPath := "screenshot.png"
		if len(args) > 0 {
			outPath = args[0]
		}
		_, err := runWithSpinner("Taking screenshot", func() (string, error) {
			adbCmd("shell", "screencap", "-p", "/sdcard/screenshot.png")
			out, err := adbCmd("pull", "/sdcard/screenshot.png", outPath)
			if err != nil {
				return "", fmt.Errorf("%s", out)
			}
			return out, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  %s Saved to %s\n", dim.Render("ℹ"), cyan.Render(outPath))
	},
}

var shellCmd = &cobra.Command{
	Use:     "shell [phone-id | command...]",
	Short:   "Open ADB shell or run a command",
	GroupID: "adb",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		// If first arg looks like a UUID, auto-connect then open shell
		if len(args) > 0 && uuidPattern.MatchString(args[0]) {
			runConnect(cmd, args[:1])
			addr := requireADB()
			c := newExecCommand("adb", "-s", addr, "shell")
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Run()
			return
		}

		addr := requireADB()
		if len(args) == 0 {
			c := newExecCommand("adb", "-s", addr, "shell")
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Run()
			return
		}
		out, err := adbCmd(append([]string{"shell"}, args...)...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", out)
			os.Exit(1)
		}
		fmt.Println(out)
	},
}

var pushCmd = &cobra.Command{
	Use:     "push <local-path> <remote-path>",
	Short:   "Push a file to the phone",
	Args:    cobra.ExactArgs(2),
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := runWithSpinner("Pushing "+cyan.Render(args[0]), func() (string, error) {
			out, err := adbCmd("push", args[0], args[1])
			if err != nil {
				return "", fmt.Errorf("%s", out)
			}
			return out, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
	},
}

var pullCmd = &cobra.Command{
	Use:     "pull <remote-path> [local-path]",
	Short:   "Pull a file from the phone",
	Args:    cobra.RangeArgs(1, 2),
	GroupID: "adb",
	Run: func(cmd *cobra.Command, args []string) {
		local := "."
		if len(args) > 1 {
			local = args[1]
		}
		_, err := runWithSpinner("Pulling "+cyan.Render(args[0]), func() (string, error) {
			out, err := adbCmd("pull", args[0], local)
			if err != nil {
				return "", fmt.Errorf("%s", out)
			}
			return out, nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %v\n", fail, err)
			os.Exit(1)
		}
	},
}

func init() {
	// Legacy aliases under "app" subcommand
	appCmd.AddCommand(
		&cobra.Command{Use: "list", Short: appsCmd.Short, Aliases: []string{"ls"}, Run: appsCmd.Run},
		&cobra.Command{Use: "install", Short: installCmd.Short, Args: cobra.ExactArgs(1), Run: installCmd.Run},
		&cobra.Command{Use: "uninstall", Short: uninstallCmd.Short, Args: cobra.ExactArgs(1), Run: uninstallCmd.Run},
		&cobra.Command{Use: "launch", Short: launchCmd.Short, Args: cobra.ExactArgs(1), Run: launchCmd.Run},
		&cobra.Command{Use: "screenshot", Short: screenCmd.Short, Args: cobra.MaximumNArgs(1), Run: screenCmd.Run},
		&cobra.Command{Use: "shell", Short: shellCmd.Short, Run: shellCmd.Run},
		&cobra.Command{Use: "push", Short: pushCmd.Short, Args: cobra.ExactArgs(2), Run: pushCmd.Run},
		&cobra.Command{Use: "pull", Short: pullCmd.Short, Args: cobra.RangeArgs(1, 2), Run: pullCmd.Run},
	)
}
