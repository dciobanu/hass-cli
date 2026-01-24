package websocket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a WebSocket client for Home Assistant.
type Client struct {
	conn      *websocket.Conn
	token     string
	msgID     int
	msgIDLock sync.Mutex
	timeout   time.Duration
}

// NewClient creates a new WebSocket client.
func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	// Convert HTTP URL to WebSocket URL
	wsURL, err := httpToWS(baseURL)
	if err != nil {
		return nil, err
	}

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
	}

	conn, _, err := dialer.Dial(wsURL+"/api/websocket", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &Client{
		conn:    conn,
		token:   token,
		msgID:   0,
		timeout: timeout,
	}

	// Authenticate
	if err := client.authenticate(); err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

// httpToWS converts an HTTP(S) URL to a WebSocket URL.
func httpToWS(httpURL string) (string, error) {
	u, err := url.Parse(httpURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	return strings.TrimSuffix(u.String(), "/"), nil
}

// authenticate performs the authentication handshake.
func (c *Client) authenticate() error {
	// Set read deadline for auth
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))

	// Read auth_required message
	var authRequired AuthRequiredMessage
	if err := c.conn.ReadJSON(&authRequired); err != nil {
		return fmt.Errorf("failed to read auth_required: %w", err)
	}

	if authRequired.Type != "auth_required" {
		return fmt.Errorf("expected auth_required, got %s", authRequired.Type)
	}

	// Send auth message
	authMsg := AuthMessage{
		Type:        "auth",
		AccessToken: c.token,
	}
	if err := c.conn.WriteJSON(authMsg); err != nil {
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Read auth response
	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	// Check if auth succeeded
	var baseMsg Message
	if err := json.Unmarshal(msg, &baseMsg); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	switch baseMsg.Type {
	case "auth_ok":
		return nil
	case "auth_invalid":
		var authInvalid AuthInvalidMessage
		json.Unmarshal(msg, &authInvalid)
		return fmt.Errorf("authentication failed: %s", authInvalid.Message)
	default:
		return fmt.Errorf("unexpected auth response: %s", baseMsg.Type)
	}
}

// Close closes the WebSocket connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// nextID returns the next message ID.
func (c *Client) nextID() int {
	c.msgIDLock.Lock()
	defer c.msgIDLock.Unlock()
	c.msgID++
	return c.msgID
}

// SendCommand sends a command and waits for the result.
func (c *Client) SendCommand(msgType string, payload map[string]interface{}) (*ResultMessage, error) {
	id := c.nextID()

	// Build message
	msg := map[string]interface{}{
		"id":   id,
		"type": msgType,
	}
	for k, v := range payload {
		msg[k] = v
	}

	// Set write deadline
	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))

	// Send message
	if err := c.conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Set read deadline
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))

	// Read response(s) until we get the result for our ID
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var result ResultMessage
		if err := json.Unmarshal(data, &result); err != nil {
			continue // Skip messages we can't parse
		}

		if result.ID == id && result.Type == "result" {
			if !result.Success {
				if result.Error != nil {
					return nil, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
				}
				return nil, fmt.Errorf("command failed")
			}
			return &result, nil
		}
	}
}

// GetDevices retrieves all devices from the device registry.
func (c *Client) GetDevices() ([]Device, error) {
	result, err := c.SendCommand("config/device_registry/list", nil)
	if err != nil {
		return nil, err
	}

	var devices []Device
	if err := json.Unmarshal(result.Result, &devices); err != nil {
		return nil, fmt.Errorf("failed to parse devices: %w", err)
	}

	return devices, nil
}

// GetAreas retrieves all areas from the area registry.
func (c *Client) GetAreas() ([]Area, error) {
	result, err := c.SendCommand("config/area_registry/list", nil)
	if err != nil {
		return nil, err
	}

	var areas []Area
	if err := json.Unmarshal(result.Result, &areas); err != nil {
		return nil, fmt.Errorf("failed to parse areas: %w", err)
	}

	return areas, nil
}

// GetEntities retrieves all entities from the entity registry.
func (c *Client) GetEntities() ([]Entity, error) {
	result, err := c.SendCommand("config/entity_registry/list", nil)
	if err != nil {
		return nil, err
	}

	var entities []Entity
	if err := json.Unmarshal(result.Result, &entities); err != nil {
		return nil, fmt.Errorf("failed to parse entities: %w", err)
	}

	return entities, nil
}

// SubscribeEvents subscribes to events and returns the subscription ID.
// eventType can be empty to subscribe to all events, or a specific type like "state_changed".
func (c *Client) SubscribeEvents(eventType string) (int, error) {
	id := c.nextID()

	msg := map[string]interface{}{
		"id":   id,
		"type": "subscribe_events",
	}
	if eventType != "" {
		msg["event_type"] = eventType
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if err := c.conn.WriteJSON(msg); err != nil {
		return 0, fmt.Errorf("failed to subscribe: %w", err)
	}

	// Wait for result
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return 0, fmt.Errorf("failed to read subscription response: %w", err)
		}

		var result ResultMessage
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		if result.ID == id && result.Type == "result" {
			if !result.Success {
				if result.Error != nil {
					return 0, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
				}
				return 0, fmt.Errorf("subscription failed")
			}
			return id, nil
		}
	}
}

// ReadEvent reads the next event from the WebSocket.
// This blocks until an event is received or context is cancelled.
func (c *Client) ReadEvent() (*EventMessage, error) {
	// Clear deadline for long-running reads
	c.conn.SetReadDeadline(time.Time{})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("failed to read event: %w", err)
		}

		var msg EventMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue // Skip messages we can't parse
		}

		if msg.Type == "event" {
			return &msg, nil
		}
	}
}

// GetStates retrieves all current states via WebSocket.
func (c *Client) GetStates() ([]StateObject, error) {
	result, err := c.SendCommand("get_states", nil)
	if err != nil {
		return nil, err
	}

	var states []StateObject
	if err := json.Unmarshal(result.Result, &states); err != nil {
		return nil, fmt.Errorf("failed to parse states: %w", err)
	}

	return states, nil
}

// RemoveConfigEntryFromDevice removes a config entry from a device.
// When all config entries are removed, the device is automatically deleted.
func (c *Client) RemoveConfigEntryFromDevice(deviceID, configEntryID string) error {
	_, err := c.SendCommand("config/device_registry/remove_config_entry", map[string]interface{}{
		"device_id":       deviceID,
		"config_entry_id": configEntryID,
	})
	return err
}

// UpdateDevice updates a device in the device registry.
func (c *Client) UpdateDevice(deviceID string, updates map[string]interface{}) (*Device, error) {
	updates["device_id"] = deviceID
	result, err := c.SendCommand("config/device_registry/update", updates)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := json.Unmarshal(result.Result, &device); err != nil {
		return nil, fmt.Errorf("failed to parse device: %w", err)
	}

	return &device, nil
}

// DisableDevice disables a device.
func (c *Client) DisableDevice(deviceID string) (*Device, error) {
	return c.UpdateDevice(deviceID, map[string]interface{}{
		"disabled_by": "user",
	})
}

// EnableDevice enables a previously disabled device.
func (c *Client) EnableDevice(deviceID string) (*Device, error) {
	return c.UpdateDevice(deviceID, map[string]interface{}{
		"disabled_by": nil,
	})
}

// UpdateEntity updates an entity in the entity registry.
func (c *Client) UpdateEntity(entityID string, updates map[string]interface{}) (*Entity, error) {
	updates["entity_id"] = entityID

	result, err := c.SendCommand("config/entity_registry/update", updates)
	if err != nil {
		return nil, err
	}

	var entity Entity
	if err := json.Unmarshal(result.Result, &entity); err != nil {
		return nil, fmt.Errorf("failed to parse entity: %w", err)
	}

	return &entity, nil
}

// HelperItem represents the response returned by helper create/list commands.
type HelperItem struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Icon    string   `json:"icon,omitempty"`
	Options []string `json:"options,omitempty"`
}

// CreateInputSelect creates an input_select helper.
func (c *Client) CreateInputSelect(name string, options []string, icon string) (*HelperItem, error) {
	payload := map[string]interface{}{
		"name":    name,
		"options": options,
	}
	if icon != "" {
		payload["icon"] = icon
	}

	result, err := c.SendCommand("input_select/create", payload)
	if err != nil {
		return nil, err
	}

	var helper HelperItem
	if err := json.Unmarshal(result.Result, &helper); err != nil {
		return nil, fmt.Errorf("failed to parse helper: %w", err)
	}

	return &helper, nil
}

// CreateInputBoolean creates an input_boolean helper.
func (c *Client) CreateInputBoolean(name string, icon string) (*HelperItem, error) {
	payload := map[string]interface{}{
		"name": name,
	}
	if icon != "" {
		payload["icon"] = icon
	}

	result, err := c.SendCommand("input_boolean/create", payload)
	if err != nil {
		return nil, err
	}

	var helper HelperItem
	if err := json.Unmarshal(result.Result, &helper); err != nil {
		return nil, fmt.Errorf("failed to parse helper: %w", err)
	}

	return &helper, nil
}

// CreateInputButton creates an input_button helper.
func (c *Client) CreateInputButton(name string, icon string) (*HelperItem, error) {
	payload := map[string]interface{}{
		"name": name,
	}
	if icon != "" {
		payload["icon"] = icon
	}

	result, err := c.SendCommand("input_button/create", payload)
	if err != nil {
		return nil, err
	}

	var helper HelperItem
	if err := json.Unmarshal(result.Result, &helper); err != nil {
		return nil, fmt.Errorf("failed to parse helper: %w", err)
	}

	return &helper, nil
}

// CreateInputNumber creates an input_number helper.
func (c *Client) CreateInputNumber(name string, min, max, step float64, mode, icon string, initial *float64) (*HelperItem, error) {
	payload := map[string]interface{}{
		"name": name,
		"min":  min,
		"max":  max,
		"step": step,
	}
	if mode != "" {
		payload["mode"] = mode
	}
	if icon != "" {
		payload["icon"] = icon
	}
	if initial != nil {
		payload["initial"] = *initial
	}

	result, err := c.SendCommand("input_number/create", payload)
	if err != nil {
		return nil, err
	}

	var helper HelperItem
	if err := json.Unmarshal(result.Result, &helper); err != nil {
		return nil, fmt.Errorf("failed to parse helper: %w", err)
	}

	return &helper, nil
}

// CreateInputText creates an input_text helper.
func (c *Client) CreateInputText(name string, min, max int, mode, pattern, icon string) (*HelperItem, error) {
	payload := map[string]interface{}{
		"name": name,
		"min":  min,
		"max":  max,
	}
	if mode != "" {
		payload["mode"] = mode
	}
	if pattern != "" {
		payload["pattern"] = pattern
	}
	if icon != "" {
		payload["icon"] = icon
	}

	result, err := c.SendCommand("input_text/create", payload)
	if err != nil {
		return nil, err
	}

	var helper HelperItem
	if err := json.Unmarshal(result.Result, &helper); err != nil {
		return nil, fmt.Errorf("failed to parse helper: %w", err)
	}

	return &helper, nil
}

type helperCommandInfo struct {
	command string
	idField string
}

var helperDeleteCommands = map[string]helperCommandInfo{
	"input_boolean":  {command: "input_boolean/delete", idField: "input_boolean_id"},
	"input_button":   {command: "input_button/delete", idField: "input_button_id"},
	"input_datetime": {command: "input_datetime/delete", idField: "input_datetime_id"},
	"input_number":   {command: "input_number/delete", idField: "input_number_id"},
	"input_select":   {command: "input_select/delete", idField: "input_select_id"},
	"input_text":     {command: "input_text/delete", idField: "input_text_id"},
}

// DeleteHelper removes a helper entity using the WebSocket API.
func (c *Client) DeleteHelper(domain, objectID string) error {
	info, ok := helperDeleteCommands[domain]
	if !ok {
		return fmt.Errorf("unsupported helper domain: %s", domain)
	}

	payload := map[string]interface{}{
		info.idField: objectID,
	}

	_, err := c.SendCommand(info.command, payload)
	return err
}

// TraceSummary represents a summary of a script/automation trace.
type TraceSummary struct {
	LastStep        string         `json:"last_step"`
	RunID           string         `json:"run_id"`
	State           string         `json:"state"`
	ScriptExecution string         `json:"script_execution"`
	Timestamp       TraceTimestamp `json:"timestamp"`
	Domain          string         `json:"domain"`
	ItemID          string         `json:"item_id"`
}

// TraceTimestamp represents timing information for a trace.
type TraceTimestamp struct {
	Start  string `json:"start"`
	Finish string `json:"finish"`
}

// TraceDetail represents detailed trace information.
type TraceDetail struct {
	LastStep        string                 `json:"last_step"`
	RunID           string                 `json:"run_id"`
	State           string                 `json:"state"`
	ScriptExecution string                 `json:"script_execution"`
	Timestamp       TraceTimestamp         `json:"timestamp"`
	Domain          string                 `json:"domain"`
	ItemID          string                 `json:"item_id"`
	Trace           map[string][]TraceStep `json:"trace"`
	Config          map[string]interface{} `json:"config"`
	BlueprintInputs map[string]interface{} `json:"blueprint_inputs,omitempty"`
	Context         TraceContext           `json:"context"`
}

// TraceStep represents a single step in a trace.
type TraceStep struct {
	Path             string                 `json:"path"`
	Timestamp        string                 `json:"timestamp"`
	ChangedVariables map[string]interface{} `json:"changed_variables,omitempty"`
	Result           map[string]interface{} `json:"result,omitempty"`
	Error            string                 `json:"error,omitempty"`
}

// TraceContext represents the context of a trace execution.
type TraceContext struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	UserID   *string `json:"user_id"`
}

// ListTraces retrieves all traces for a script or automation.
func (c *Client) ListTraces(domain, itemID string) ([]TraceSummary, error) {
	result, err := c.SendCommand("trace/list", map[string]interface{}{
		"domain":  domain,
		"item_id": itemID,
	})
	if err != nil {
		return nil, err
	}

	var traces []TraceSummary
	if err := json.Unmarshal(result.Result, &traces); err != nil {
		return nil, fmt.Errorf("failed to parse traces: %w", err)
	}

	return traces, nil
}

// GetTrace retrieves detailed trace information for a specific run.
func (c *Client) GetTrace(domain, itemID, runID string) (*TraceDetail, error) {
	result, err := c.SendCommand("trace/get", map[string]interface{}{
		"domain":  domain,
		"item_id": itemID,
		"run_id":  runID,
	})
	if err != nil {
		return nil, err
	}

	var trace TraceDetail
	if err := json.Unmarshal(result.Result, &trace); err != nil {
		return nil, fmt.Errorf("failed to parse trace: %w", err)
	}

	return &trace, nil
}
