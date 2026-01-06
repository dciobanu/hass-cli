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

### Next (High Priority)

```
hass-cli status                         # Check API connectivity and HA version
hass-cli entities                       # List all entities
hass-cli entities -d light              # Filter by domain
hass-cli entities -a "Living Room"      # Filter by area
hass-cli entities inspect <entity_id>   # Show full entity state + attributes
hass-cli areas                          # List all areas
hass-cli areas inspect <area_id>        # Show area details with devices/entities
hass-cli state get <entity_id>          # Get current state of entity
hass-cli state set <entity_id> <state>  # Set entity state
hass-cli services                       # List available services
hass-cli services -d light              # Filter by domain
hass-cli call <domain>.<service>        # Call a service
hass-cli call light.turn_on -e light.living_room
hass-cli watch <entity_id>...           # Watch entity state changes (WebSocket)
```

### Later

API endpoints and features to explore for future development:

#### Device/Entity Management (Unofficial WebSocket API)
```
config/device_registry/update           - Rename device, assign area
config/entity_registry/update           - Rename entity, assign area, disable, hide
config/area_registry/create             - Create new area
config/area_registry/update             - Rename area, assign floor
config/area_registry/delete             - Delete area
```

#### Scene Management (Unofficial REST + WebSocket)
```
POST /api/config/scene/config/<id>      - Create/update scene
GET  /api/config/scene/config/<id>      - Get scene configuration
DELETE /api/config/scene/config/<id>    - Delete scene
scene.apply (service)                   - Preview scene without saving
```

#### Automation Management (Unofficial REST)
```
GET  /api/config/automation/config/<id> - Get automation configuration
POST /api/config/automation/config/<id> - Create/update automation
```

#### History & Logging
```
GET /api/history/period/<timestamp>     - Entity state history
GET /api/logbook/<timestamp>            - System event log
recorder/info (WebSocket)               - Database info
```

#### Calendar
```
GET /api/calendars                      - List calendars
GET /api/calendars/<id>?start=&end=     - Get calendar events
```

#### Camera
```
GET /api/camera_proxy/<entity_id>       - Download camera snapshot
```

#### Templates
```
POST /api/template                      - Render Jinja2 template
```

#### Events (WebSocket)
```
subscribe_events                        - Subscribe to event bus
fire_event                              - Fire custom event
subscribe_trigger                       - Subscribe to automation triggers
```

#### System
```
GET /api/config                         - Full system configuration
GET /api/error_log                      - Error log (plaintext)
POST /api/config/core/check_config      - Validate configuration.yaml
cloud/status (WebSocket)                - Nabu Casa cloud status
repairs/list_issues (WebSocket)         - System repair issues
```

## License

MIT
