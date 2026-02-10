// Package testutil provides test helpers for hass-cli tests.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// RESTMock wraps httptest.Server with route registration for testing the REST API client.
type RESTMock struct {
	Server *httptest.Server
	mu     sync.Mutex
	routes map[string]http.HandlerFunc
	Token  string
}

// NewRESTMock creates a new mock REST server with Bearer token validation.
// All routes return 404 by default. Use Handle/HandleJSON to register routes.
func NewRESTMock(t *testing.T, token string) *RESTMock {
	t.Helper()

	m := &RESTMock{
		routes: make(map[string]http.HandlerFunc),
		Token:  token,
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Bearer token
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+m.Token {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Invalid or missing access token",
			})
			return
		}

		// Look up route
		key := r.Method + " " + r.URL.Path
		m.mu.Lock()
		handler, ok := m.routes[key]
		m.mu.Unlock()

		if ok {
			handler(w, r)
			return
		}

		// Try prefix match for dynamic paths
		m.mu.Lock()
		for routeKey, h := range m.routes {
			if strings.HasSuffix(routeKey, "/*") {
				prefix := strings.TrimSuffix(routeKey, "*")
				if strings.HasPrefix(r.Method+" "+r.URL.Path, prefix) {
					m.mu.Unlock()
					h(w, r)
					return
				}
			}
		}
		m.mu.Unlock()

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Not found",
		})
	}))

	t.Cleanup(func() {
		m.Server.Close()
	})

	return m
}

// URL returns the base URL of the mock server.
func (m *RESTMock) URL() string {
	return m.Server.URL
}

// Handle registers a handler for a specific method and path.
func (m *RESTMock) Handle(method, path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes[method+" "+path] = handler
}

// HandleJSON registers a handler that returns a canned JSON response with status 200.
func (m *RESTMock) HandleJSON(method, path string, statusCode int, data interface{}) {
	m.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(data)
	})
}
