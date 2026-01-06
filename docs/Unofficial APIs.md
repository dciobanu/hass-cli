# Unofficial Home Assistant APIs

These APIs are used by the Home Assistant frontend but are not documented in the official REST or WebSocket API documentation. They were discovered by analyzing browser network traces.

> **Warning**: These APIs are internal and may change without notice between Home Assistant versions.

---

## REST APIs

### Scene Configuration

Manage scene configurations directly.

#### Create/Update Scene

```
POST /api/config/scene/config/<scene_id>
```

Creates a new scene or updates an existing one.

**Request Body:**
```json
{
  "id": "1767672291452",
  "name": "Movie Night",
  "entities": {
    "light.living_room": {
      "state": "on",
      "brightness": 50
    },
    "light.kitchen": {
      "state": "off"
    }
  }
}
```

**Notes:**
- The `scene_id` in the URL should be a unique identifier (timestamp is commonly used)
- If `id` is omitted from the body, the scene is created; if included, it updates
- Entity states include all current attributes of the entity at capture time

#### Get Scene Configuration

```
GET /api/config/scene/config/<scene_id>
```

Retrieves the configuration for a specific scene.

#### Delete Scene

```
DELETE /api/config/scene/config/<scene_id>
```

Deletes a scene by its ID.

---

### Automation Configuration

#### Get Automation Configuration

```
GET /api/config/automation/config/<automation_id>
```

Retrieves the configuration for a specific automation.

---

## WebSocket APIs

### Authentication

#### Get Current User

```json
{
  "type": "auth/current_user",
  "id": 1
}
```

Returns information about the currently authenticated user.

---

### Entity Subscriptions

#### Subscribe to Entity Updates

```json
{
  "type": "subscribe_entities",
  "id": 1
}
```

Subscribes to entity state updates. More efficient than `subscribe_events` with `event_type: "state_changed"` as it sends compressed updates.

---

### Registry APIs

#### List Areas

```json
{
  "type": "config/area_registry/list",
  "id": 1
}
```

**Response:**
```json
{
  "id": 1,
  "type": "result",
  "success": true,
  "result": [
    {
      "aliases": [],
      "area_id": "living_room",
      "floor_id": null,
      "humidity_entity_id": null,
      "icon": null,
      "labels": [],
      "name": "Living Room",
      "picture": null,
      "temperature_entity_id": null,
      "created_at": 0.0,
      "modified_at": 0.0
    }
  ]
}
```

#### List Devices

```json
{
  "type": "config/device_registry/list",
  "id": 1
}
```

**Response:**
```json
{
  "id": 1,
  "type": "result",
  "success": true,
  "result": [
    {
      "area_id": null,
      "configuration_url": null,
      "config_entries": ["entry_id"],
      "connections": [],
      "created_at": 0.0,
      "disabled_by": null,
      "entry_type": null,
      "hw_version": null,
      "id": "device_id",
      "identifiers": [["integration", "unique_id"]],
      "labels": [],
      "manufacturer": "Manufacturer Name",
      "model": "Model",
      "model_id": null,
      "name": "Device Name",
      "name_by_user": null,
      "primary_config_entry": "entry_id",
      "serial_number": null,
      "sw_version": null,
      "via_device_id": null
    }
  ]
}
```

#### List Entities

```json
{
  "type": "config/entity_registry/list",
  "id": 1
}
```

**Response:**
```json
{
  "id": 1,
  "type": "result",
  "success": true,
  "result": [
    {
      "area_id": null,
      "categories": {},
      "config_entry_id": "entry_id",
      "config_subentry_id": null,
      "created_at": 0.0,
      "device_id": "device_id",
      "disabled_by": null,
      "entity_category": null,
      "entity_id": "light.living_room",
      "has_entity_name": true,
      "hidden_by": null,
      "icon": null,
      "id": "entity_registry_id",
      "labels": [],
      "modified_at": 0.0,
      "name": null,
      "options": {},
      "original_name": "Living Room Light",
      "platform": "hue"
    }
  ]
}
```

#### List Entities for Display (Compact)

```json
{
  "type": "config/entity_registry/list_for_display",
  "id": 1
}
```

Returns a compact representation of entities for UI display purposes. Uses abbreviated field names:
- `ei`: entity_id
- `pl`: platform
- `lb`: labels
- `di`: device_id
- `hn`: has_entity_name

#### Get Single Entity

```json
{
  "type": "config/entity_registry/get",
  "entity_id": "light.living_room",
  "id": 1
}
```

#### Update Entity

```json
{
  "type": "config/entity_registry/update",
  "entity_id": "light.living_room",
  "area_id": "living_room",
  "name": "Main Light",
  "icon": "mdi:ceiling-light",
  "disabled_by": null,
  "hidden_by": null,
  "id": 1
}
```

Updates entity registry properties. Possible fields:
- `area_id` - Assign entity to an area
- `name` - Custom name override
- `icon` - Custom icon (mdi format)
- `disabled_by` - Disable entity ("user" or null)
- `hidden_by` - Hide entity ("user" or null)
- `labels` - Array of label IDs

#### List Floors

```json
{
  "type": "config/floor_registry/list",
  "id": 1
}
```

#### List Labels

```json
{
  "type": "config/label_registry/list",
  "id": 1
}
```

#### List Categories

```json
{
  "type": "config/category_registry/list",
  "scope": "automation",
  "id": 1
}
```

Scope can be: `automation`, `scene`, etc.

---

### Config Entries

#### Get Config Entries

```json
{
  "type": "config_entries/get",
  "id": 1
}
```

Returns all configuration entries (integrations).

---

### Device Automation

#### List Device Triggers

```json
{
  "type": "device_automation/trigger/list",
  "device_id": "device_id_here",
  "id": 1
}
```

Returns available triggers for a device.

#### Get Trigger Capabilities

```json
{
  "type": "device_automation/trigger/capabilities",
  "trigger": {
    "device_id": "device_id",
    "domain": "zha",
    "type": "remote_button_short_press",
    "subtype": "button",
    "trigger": "device"
  },
  "id": 1
}
```

#### List Device Actions

```json
{
  "type": "device_automation/action/list",
  "device_id": "device_id_here",
  "id": 1
}
```

Returns available actions for a device.

#### Get Action Capabilities

```json
{
  "type": "device_automation/action/capabilities",
  "action": {
    "type": "turn_off",
    "device_id": "device_id",
    "entity_id": "switch.my_switch",
    "domain": "switch"
  },
  "id": 1
}
```

---

### Blueprints

#### List Blueprints

```json
{
  "type": "blueprint/list",
  "domain": "automation",
  "id": 1
}
```

Returns available blueprints for the specified domain (`automation` or `script`).

---

### Frontend

#### Get Themes

```json
{
  "type": "frontend/get_themes",
  "id": 1
}
```

#### Get Translations

```json
{
  "type": "frontend/get_translations",
  "language": "en",
  "category": "entity_component",
  "id": 1
}
```

Categories include: `entity_component`, `state`, `config`, etc.

#### Get Icons

```json
{
  "type": "frontend/get_icons",
  "category": "services",
  "integration": "light",
  "id": 1
}
```

#### Subscribe to User Data

```json
{
  "type": "frontend/subscribe_user_data",
  "key": "core",
  "id": 1
}
```

---

### Integration Manifest

#### Get Manifest

```json
{
  "type": "manifest/get",
  "integration": "light",
  "id": 1
}
```

Returns the manifest for an integration, including version, dependencies, etc.

---

### Notifications

#### Subscribe to Persistent Notifications

```json
{
  "type": "persistent_notification/subscribe",
  "id": 1
}
```

---

### System Information

#### Get Cloud Status

```json
{
  "type": "cloud/status",
  "id": 1
}
```

Returns Nabu Casa cloud connection status.

#### Get Recorder Info

```json
{
  "type": "recorder/info",
  "id": 1
}
```

Returns recorder/database information.

#### List Repair Issues

```json
{
  "type": "repairs/list_issues",
  "id": 1
}
```

Returns list of issues detected by the repairs integration.

#### Get Numeric Device Classes

```json
{
  "type": "sensor/numeric_device_classes",
  "id": 1
}
```

Returns list of numeric sensor device classes.

---

## Scene Management via Services

While scenes are configured via REST API, they can be controlled via the standard WebSocket `call_service`:

### Apply Scene (Preview)

Apply entity states without saving as a scene:

```json
{
  "type": "call_service",
  "domain": "scene",
  "service": "apply",
  "return_response": false,
  "service_data": {
    "entities": {
      "light.living_room": {
        "state": "on",
        "brightness": 128,
        "color_temp": 370
      },
      "light.kitchen": {
        "state": "off"
      }
    }
  },
  "id": 1
}
```

### Turn On Scene

```json
{
  "type": "call_service",
  "domain": "scene",
  "service": "turn_on",
  "target": {
    "entity_id": "scene.movie_night"
  },
  "id": 1
}
```
