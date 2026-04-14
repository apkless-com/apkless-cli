package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func newExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage apps on a cloud phone (requires: apkless phone connect first)",
}

var appListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List installed third-party apps",
	Aliases: []string{"ls"},
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

var appInstallCmd = &cobra.Command{
	Use:   "install <apk-path>",
	Short: "Install an APK file",
	Args:  cobra.ExactArgs(1),
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

var appUninstallCmd = &cobra.Command{
	Use:   "uninstall <package>",
	Short: "Uninstall an app",
	Args:  cobra.ExactArgs(1),
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

var appLaunchCmd = &cobra.Command{
	Use:   "launch <package>",
	Short: "Launch an app",
	Args:  cobra.ExactArgs(1),
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

var appScreenshotCmd = &cobra.Command{
	Use:   "screenshot [output-path]",
	Short: "Take a screenshot",
	Args:  cobra.MaximumNArgs(1),
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

var appShellCmd = &cobra.Command{
	Use:   "shell [command...]",
	Short: "Open ADB shell or run a command",
	Run: func(cmd *cobra.Command, args []string) {
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

func init() {
	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appInstallCmd)
	appCmd.AddCommand(appUninstallCmd)
	appCmd.AddCommand(appLaunchCmd)
	appCmd.AddCommand(appScreenshotCmd)
	appCmd.AddCommand(appShellCmd)
}
