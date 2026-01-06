# hass-cli

A command-line utility for controlling Home Assistant.

## Installation

### Building from source

```bash
# Build
make build

# Install to /usr/local/bin
make install
```

## Usage

### Authentication

```bash
hass-cli login                          # Configure server URL and token (interactive)
hass-cli login --url http://ha:8123 --token TOKEN  # Non-interactive
hass-cli logout                         # Remove saved credentials
```

### Devices

```bash
hass-cli devices                        # List all devices
hass-cli devices -m philips             # Filter by manufacturer
hass-cli devices -a "Living Room"       # Filter by area
hass-cli devices --json                 # Output as JSON
hass-cli devices inspect <id>           # Show full device JSON
```

### Global Flags

```bash
--json, -j          # Output in JSON format
--url <url>         # Override server URL
--token <token>     # Override access token
--timeout <secs>    # Request timeout (default: 30)
--verbose, -v       # Verbose output
```

## Configuration

Credentials are stored in `~/.config/hass-cli/config.yaml`

## Development

```bash
make build      # Build binary
make test       # Run tests
make build-all  # Cross-compile for all platforms
make clean      # Remove build artifacts
```

## Planned Features

- `hass-cli status` - Check API connectivity
- `hass-cli entities` - List/inspect entities
- `hass-cli areas` - List/manage areas
- `hass-cli services` - List/call services
- `hass-cli state get/set` - Get/set entity states
- `hass-cli events` - Subscribe to events (WebSocket)
- `hass-cli watch` - Watch entity state changes

## License

MIT
