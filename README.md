# APKless CLI

Cloud Android HTTPS capture — no device, no root, no setup.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/apkless-com/apkless-cli/main/install.sh | sh
```

Or download from [Releases](https://github.com/apkless-com/apkless-cli/releases).

## Quick Start

```bash
# Set your API key
export APKLESS_KEY=apkless_xxxxxxxx

# Create a cloud phone
apkless create

# Install an APK
apkless install ./app.apk

# Start capturing traffic
apkless capture start com.example.app

# View captured flows
apkless capture flows

# View flow detail
apkless capture flows <flow-id>

# Destroy when done
apkless rm <phone-id>
```

## Commands

### Device Management

```bash
apkless create [--region beijing] [--hours 1]  # Create a cloud phone
apkless ls                                      # List active phones
apkless ls --all                                # Include destroyed phones
apkless show <id>                               # Show phone details
apkless rm <id>                                 # Destroy a phone
apkless connect [id]                            # Connect local ADB
apkless open [id]                               # Open sandbox in browser
apkless use <id>                                # Set as default phone
apkless status [id]                             # Quick status overview
apkless restart <id>                            # Restart a phone
```

### ADB Operations

```bash
apkless shell                        # Interactive ADB shell
apkless shell <phone-id>             # Auto-connect + shell
apkless shell <command>              # Run a command
apkless apps                         # List installed apps
apkless install <apk-path>          # Install an APK
apkless uninstall <package>          # Uninstall an app
apkless launch <package>             # Launch an app
apkless screen [output.png]          # Take a screenshot
apkless push <local> <remote>        # Push a file
apkless pull <remote> [local]        # Pull a file
```

### Traffic Capture

```bash
apkless capture start <package>      # Start capturing
apkless capture stop                 # Stop capturing
apkless capture status               # Show capture status
apkless capture flows                # List captured flows
apkless capture flows <flow-id>      # Show flow detail
apkless capture watch                # Real-time traffic stream
apkless capture export [--output f]  # Export as HAR
apkless capture clear                # Clear all flows
```

### Flags

```
--key string   API key (overrides APKLESS_KEY env)
--help         Show help
```

## Authentication

Get your API key at [apkless.com/dashboard](https://apkless.com/dashboard).

Set it as an environment variable:

```bash
export APKLESS_KEY=apkless_xxxxxxxx
```

Or pass it directly:

```bash
apkless --key apkless_xxxxxxxx ls
```

## License

MIT
