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
