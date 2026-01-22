# hass-cli

A command-line utility for controlling Home Assistant.

## Installation

### Using Homebrew

```bash
brew tap dciobanu/tap
brew install hass-cli
```

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

### Status

```bash
hass-cli status                         # Check API connectivity and HA version
hass-cli status --json                  # Output as JSON
```

### Devices

```bash
hass-cli devices                        # List all devices
hass-cli devices -m philips             # Filter by manufacturer
hass-cli devices -a "Living Room"       # Filter by area
hass-cli devices --json                 # Output as JSON
hass-cli devices inspect <id>           # Show full device JSON
hass-cli devices disable <id>           # Disable a device
hass-cli devices enable <id>            # Re-enable a disabled device
hass-cli devices remove <id>            # Remove orphaned device
```

### Entities

```bash
hass-cli entities                       # List all entities
hass-cli entities -d light              # Filter by domain
hass-cli entities -a kitchen            # Filter by area
hass-cli entities -D <device_id>        # Filter by device (prefix match)
hass-cli entities --json                # Output as JSON
hass-cli entities inspect <entity_id>   # Show full entity state + attributes
```

### Areas

```bash
hass-cli areas                          # List all areas with device/entity counts
hass-cli areas --json                   # Output as JSON
hass-cli areas inspect <area_id>        # Show area with all devices and entities
```

### Scenes

```bash
hass-cli scenes                         # List all scenes
hass-cli scenes --json                  # Output as JSON
hass-cli scenes inspect <scene_id>      # Show scene configuration with entities

# Create a scene capturing current entity states
hass-cli scenes create "Movie Night" -e light.living_room -e light.kitchen
hass-cli scenes create "Cozy Evening" -e light.bedroom --icon mdi:weather-sunset

# Modify existing scenes
hass-cli scenes add-entity <scene_id> <entity_id>     # Add entity to scene
hass-cli scenes remove-entity <scene_id> <entity_id>  # Remove entity from scene

# Delete a scene
hass-cli scenes delete <scene_id>

# Activate a scene
hass-cli call scene.turn_on -e scene.movie_night
```

### Scripts

```bash
hass-cli scripts                        # List all scripts
hass-cli scripts --json                 # Output as JSON
hass-cli scripts inspect <script_id>    # Show script configuration

# Run/trigger a script
hass-cli scripts run hello_world
hass-cli scripts trigger my_script      # 'trigger' is an alias for 'run'
hass-cli scripts run my_script --data '{"brightness": 128}'  # Pass variables

# Create a new script
hass-cli scripts create "Hello World" --description "A test script"
hass-cli scripts create "Turn Off Lights" \
  --icon mdi:lightbulb-off \
  --mode single \
  --sequence '[{"service":"light.turn_off","target":{"area_id":"living_room"}}]'

# Edit an existing script
hass-cli scripts edit hello_world --alias "Hello World Updated"
hass-cli scripts edit hello_world --description "Updated description"
hass-cli scripts edit hello_world --sequence '[{"service":"light.turn_on"}]'

# Rename a script
hass-cli scripts rename hello_world "Greeting Script"

# Debug a script (view execution traces)
hass-cli scripts debug hello_world                    # List all traces
hass-cli scripts debug hello_world --run-id <id>      # Show detailed trace
hass-cli scripts debug hello_world --json             # Output as JSON

# Delete a script
hass-cli scripts delete hello_world
```

### State

```bash
hass-cli state get <entity_id>          # Get current state of entity
hass-cli state get light.living_room --json
hass-cli state set <entity_id> <state>  # Set entity state directly
hass-cli state set sensor.custom 42 --attr unit_of_measurement=°C
```

### Services

```bash
hass-cli services                       # List all available services
hass-cli services -d light              # Filter by domain
hass-cli services inspect light.turn_on # Show service details and fields
```

### Call Service

```bash
hass-cli call <domain.service>          # Call a service
hass-cli call light.turn_on -e light.living_room
hass-cli call light.turn_off -a living_room        # Target by area
hass-cli call switch.toggle -e switch.fan
hass-cli call scene.turn_on -e scene.movie_night
hass-cli call homeassistant.restart

# Brightness (0-255)
hass-cli call light.turn_on -a living_room --data '{"brightness": 128}'

# Color (RGB)
hass-cli call light.turn_on -a living_room --data '{"rgb_color": [255, 0, 0]}'

# Color temperature (Kelvin: 2200=warm, 6500=cool)
hass-cli call light.turn_on -a living_room --data '{"color_temp_kelvin": 2700}'

# Effects (case-sensitive, check entity attributes for available effects)
hass-cli call light.turn_on -a living_room --data '{"effect": "Cozy"}'

# Combined
hass-cli call light.turn_on -a living_room --data '{"rgb_color": [255, 100, 50], "brightness": 200}'
```

### Watch

```bash
hass-cli watch                          # Watch all state changes (WebSocket)
hass-cli watch light.living_room        # Watch specific entity
hass-cli watch light.* sensor.*         # Watch multiple patterns
hass-cli watch --json                   # Output as JSON
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

## Later

API endpoints and features to explore for future development:

#### Device/Entity Management (Unofficial WebSocket API)
```
config/device_registry/update           - Rename device, assign area
config/entity_registry/update           - Rename entity, assign area, disable, hide
config/area_registry/create             - Create new area
config/area_registry/update             - Rename area, assign floor
config/area_registry/delete             - Delete area
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
