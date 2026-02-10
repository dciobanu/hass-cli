package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// WSHandler handles a WebSocket command message and returns a result.
// The handler receives the full message map and returns the result payload.
type WSHandler func(msg map[string]interface{}) (interface{}, error)

// WSMock wraps httptest.Server with WebSocket support for testing the WS client.
type WSMock struct {
	Server   *httptest.Server
	Token    string
	mu       sync.Mutex
	handlers map[string]WSHandler
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWSMock creates a new mock WebSocket server that handles the HA auth handshake.
// After authentication, it dispatches commands to registered handlers.
func NewWSMock(t *testing.T, token string) *WSMock {
	t.Helper()

	m := &WSMock{
		Token:    token,
		handlers: make(map[string]WSHandler),
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Step 1: Send auth_required
		conn.WriteJSON(map[string]interface{}{
			"type":       "auth_required",
			"ha_version": "2024.1.0",
		})

		// Step 2: Read auth message
		var authMsg map[string]interface{}
		if err := conn.ReadJSON(&authMsg); err != nil {
			t.Errorf("Failed to read auth message: %v", err)
			return
		}

		// Step 3: Validate token
		accessToken, _ := authMsg["access_token"].(string)
		if accessToken != m.Token {
			conn.WriteJSON(map[string]interface{}{
				"type":    "auth_invalid",
				"message": "Invalid access token",
			})
			return
		}

		// Step 4: Send auth_ok
		conn.WriteJSON(map[string]interface{}{
			"type":       "auth_ok",
			"ha_version": "2024.1.0",
		})

		// Step 5: Command loop
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				// Connection closed
				return
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			msgID, _ := msg["id"].(float64)
			msgType, _ := msg["type"].(string)

			m.mu.Lock()
			handler, ok := m.handlers[msgType]
			m.mu.Unlock()

			if !ok {
				// No handler, return error
				conn.WriteJSON(map[string]interface{}{
					"id":      int(msgID),
					"type":    "result",
					"success": false,
					"error": map[string]string{
						"code":    "unknown_command",
						"message": "Unknown command: " + msgType,
					},
				})
				continue
			}

			result, err := handler(msg)
			if err != nil {
				conn.WriteJSON(map[string]interface{}{
					"id":      int(msgID),
					"type":    "result",
					"success": false,
					"error": map[string]string{
						"code":    "command_error",
						"message": err.Error(),
					},
				})
				continue
			}

			// Marshal and re-unmarshal the result so it becomes json.RawMessage compatible
			resultJSON, _ := json.Marshal(result)
			var rawResult json.RawMessage = resultJSON

			conn.WriteJSON(map[string]interface{}{
				"id":      int(msgID),
				"type":    "result",
				"success": true,
				"result":  rawResult,
			})
		}
	}))

	t.Cleanup(func() {
		m.Server.Close()
	})

	return m
}

// URL returns the base URL of the mock server (http:// scheme, suitable for WS client).
func (m *WSMock) URL() string {
	return m.Server.URL
}

// Handle registers a command handler for a specific message type.
func (m *WSMock) Handle(msgType string, handler WSHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[msgType] = handler
}
