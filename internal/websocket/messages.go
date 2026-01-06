// Package websocket provides WebSocket client functionality for Home Assistant API.
package websocket

import "encoding/json"

// Message represents a generic WebSocket message.
type Message struct {
	ID   int    `json:"id,omitempty"`
	Type string `json:"type"`
}

// AuthRequiredMessage is sent by the server when connection is established.
type AuthRequiredMessage struct {
	Type      string `json:"type"`
	HAVersion string `json:"ha_version"`
}

// AuthMessage is sent by the client to authenticate.
type AuthMessage struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token"`
}

// AuthOKMessage is sent by the server when authentication succeeds.
type AuthOKMessage struct {
	Type      string `json:"type"`
	HAVersion string `json:"ha_version"`
}

// AuthInvalidMessage is sent by the server when authentication fails.
type AuthInvalidMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ResultMessage is sent by the server in response to a command.
type ResultMessage struct {
	ID      int             `json:"id"`
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorResult    `json:"error,omitempty"`
}

// ErrorResult contains error details from a failed command.
type ErrorResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CommandMessage is a generic command sent to the server.
type CommandMessage struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

// Device represents a device from the device registry.
type Device struct {
	ID                 string              `json:"id"`
	AreaID             *string             `json:"area_id"`
	ConfigEntries      []string            `json:"config_entries"`
	Connections        [][]string          `json:"connections"`
	CreatedAt          float64             `json:"created_at"`
	DisabledBy         *string             `json:"disabled_by"`
	EntryType          *string             `json:"entry_type"`
	HWVersion          *string             `json:"hw_version"`
	Identifiers        [][]string          `json:"identifiers"`
	Labels             []string            `json:"labels"`
	Manufacturer       *string             `json:"manufacturer"`
	Model              *string             `json:"model"`
	ModelID            *string             `json:"model_id"`
	Name               *string             `json:"name"`
	NameByUser         *string             `json:"name_by_user"`
	PrimaryConfigEntry *string             `json:"primary_config_entry"`
	SerialNumber       *string             `json:"serial_number"`
	SWVersion          *string             `json:"sw_version"`
	ViaDeviceID        *string             `json:"via_device_id"`
	ConfigurationURL   *string             `json:"configuration_url"`
	ModifiedAt         float64             `json:"modified_at"`
}

// DisplayName returns the best available name for the device.
func (d *Device) DisplayName() string {
	if d.NameByUser != nil && *d.NameByUser != "" {
		return *d.NameByUser
	}
	if d.Name != nil && *d.Name != "" {
		return *d.Name
	}
	return d.ID
}

// DisplayManufacturer returns the manufacturer or "Unknown".
func (d *Device) DisplayManufacturer() string {
	if d.Manufacturer != nil && *d.Manufacturer != "" {
		return *d.Manufacturer
	}
	return "Unknown"
}

// DisplayModel returns the model or "Unknown".
func (d *Device) DisplayModel() string {
	if d.Model != nil && *d.Model != "" {
		return *d.Model
	}
	return "Unknown"
}

// Area represents an area from the area registry.
type Area struct {
	AreaID   string   `json:"area_id"`
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases"`
	FloorID  *string  `json:"floor_id"`
	Icon     *string  `json:"icon"`
	Labels   []string `json:"labels"`
	Picture  *string  `json:"picture"`
}

// Entity represents an entity from the entity registry.
type Entity struct {
	EntityID       string            `json:"entity_id"`
	AreaID         *string           `json:"area_id"`
	Categories     map[string]string `json:"categories"`
	ConfigEntryID  *string           `json:"config_entry_id"`
	DeviceID       *string           `json:"device_id"`
	DisabledBy     *string           `json:"disabled_by"`
	EntityCategory *string           `json:"entity_category"`
	HasEntityName  bool              `json:"has_entity_name"`
	HiddenBy       *string           `json:"hidden_by"`
	Icon           *string           `json:"icon"`
	ID             string            `json:"id"`
	Labels         []string          `json:"labels"`
	Name           *string           `json:"name"`
	OriginalName   interface{}       `json:"original_name"` // Can be string or null
	Platform       string            `json:"platform"`
	CreatedAt      float64           `json:"created_at"`
	ModifiedAt     float64           `json:"modified_at"`
}

// DisplayName returns the best available name for the entity.
func (e *Entity) DisplayName() string {
	if e.Name != nil && *e.Name != "" {
		return *e.Name
	}
	if origName, ok := e.OriginalName.(string); ok && origName != "" {
		return origName
	}
	return e.EntityID
}

// GetOriginalName returns the original name as a string pointer.
func (e *Entity) GetOriginalName() *string {
	if origName, ok := e.OriginalName.(string); ok {
		return &origName
	}
	return nil
}

// EventMessage represents an event message from a subscription.
type EventMessage struct {
	ID    int        `json:"id"`
	Type  string     `json:"type"`
	Event EventData  `json:"event"`
}

// EventData contains the event payload.
type EventData struct {
	EventType  string                 `json:"event_type"`
	Data       StateChangedData       `json:"data"`
	Origin     string                 `json:"origin"`
	TimeFired  string                 `json:"time_fired"`
	Context    EventContext           `json:"context"`
}

// StateChangedData contains state change information.
type StateChangedData struct {
	EntityID string       `json:"entity_id"`
	OldState *StateObject `json:"old_state"`
	NewState *StateObject `json:"new_state"`
}

// StateObject represents an entity state.
type StateObject struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged string                 `json:"last_changed"`
	LastUpdated string                 `json:"last_updated"`
	Context     EventContext           `json:"context"`
}

// EventContext contains context information about an event.
type EventContext struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	UserID   *string `json:"user_id"`
}
