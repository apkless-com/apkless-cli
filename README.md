# APKless CLI

Capture HTTP/HTTPS traffic from any Android app in the cloud.

## Install

```bash
curl -fsSL https://apkless.com/install.sh | sh
```

Or download from [Releases](https://github.com/apkless/apkless-cli/releases).

## Quick Start

```bash
# Set your API key
export APKLESS_KEY=apkless_xxxxxxxx

# Create a cloud phone
apkless phone create

# Install an APK
apkless app install <phone-id> ./app.apk

# Start capturing traffic
apkless capture start <phone-id> com.example.app

# View captured flows
apkless capture flows <phone-id>

# View flow detail
apkless capture flows <phone-id> <flow-id>

# Destroy when done
apkless phone destroy <phone-id>
```

## Commands

### Phone Management

```bash
apkless phone create [--region beijing]  # Create a cloud phone
apkless phone list                       # List your phones
apkless phone show <id>                  # Show phone details
apkless phone destroy <id>              # Destroy a phone
apkless phone connect <id> [--exec]     # Get ADB/scrcpy connection info
```

### App Management

```bash
apkless app list <id>                     # List installed apps
apkless app install <id> <apk-or-url>    # Install an APK
apkless app uninstall <id> <package>     # Uninstall an app
```

### Traffic Capture

```bash
apkless capture start <id> <package>     # Start capturing
apkless capture stop <id>                # Stop capturing
apkless capture status <id>              # Show capture status
apkless capture flows <id>              # List captured flows
apkless capture flows <id> <flow-id>    # Show flow detail
apkless capture clear <id>              # Clear all flows
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
apkless --key apkless_xxxxxxxx phone list
```

## License

MIT
