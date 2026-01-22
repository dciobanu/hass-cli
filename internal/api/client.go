package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for the Home Assistant API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Home Assistant API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// doRequest performs an HTTP request and returns the response.
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// CheckConnection verifies that the API is accessible and the token is valid.
func (c *Client) CheckConnection() error {
	resp, err := c.doRequest("GET", "/api/", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// APIStatus represents the response from GET /api/
type APIStatus struct {
	Message string `json:"message"`
}

// GetStatus returns the API status message.
func (c *Client) GetStatus() (*APIStatus, error) {
	resp, err := c.doRequest("GET", "/api/", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var status APIStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// Config represents the Home Assistant configuration.
type Config struct {
	Components           []string   `json:"components"`
	ConfigDir            string     `json:"config_dir"`
	Elevation            float64    `json:"elevation"`
	Latitude             float64    `json:"latitude"`
	Longitude            float64    `json:"longitude"`
	LocationName         string     `json:"location_name"`
	TimeZone             string     `json:"time_zone"`
	UnitSystem           UnitSystem `json:"unit_system"`
	Version              string     `json:"version"`
	WhitelistExternalDir []string   `json:"whitelist_external_dirs"`
	State                string     `json:"state"`
	Currency             string     `json:"currency"`
	Country              string     `json:"country"`
	Language             string     `json:"language"`
}

// UnitSystem represents the unit system configuration.
type UnitSystem struct {
	Length      string `json:"length"`
	Mass        string `json:"mass"`
	Temperature string `json:"temperature"`
	Volume      string `json:"volume"`
}

// GetConfig returns the Home Assistant configuration.
func (c *Client) GetConfig() (*Config, error) {
	resp, err := c.doRequest("GET", "/api/config", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var config Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &config, nil
}

// State represents an entity state.
type State struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged string                 `json:"last_changed"`
	LastUpdated string                 `json:"last_updated"`
	Context     StateContext           `json:"context"`
}

// StateContext represents the context of a state change.
type StateContext struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	UserID   *string `json:"user_id"`
}

// GetStates returns all entity states.
func (c *Client) GetStates() ([]State, error) {
	resp, err := c.doRequest("GET", "/api/states", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var states []State
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return states, nil
}

// GetState returns the state of a specific entity.
func (c *Client) GetState(entityID string) (*State, error) {
	resp, err := c.doRequest("GET", "/api/states/"+entityID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var state State
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &state, nil
}

// SetState sets the state of an entity.
func (c *Client) SetState(entityID string, state string, attributes map[string]interface{}) (*State, error) {
	body := map[string]interface{}{
		"state": state,
	}
	if attributes != nil {
		body["attributes"] = attributes
	}

	resp, err := c.doRequest("POST", "/api/states/"+entityID, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	var resultState State
	if err := json.NewDecoder(resp.Body).Decode(&resultState); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resultState, nil
}

// Service represents a service domain with its services.
type Service struct {
	Domain   string                 `json:"domain"`
	Services map[string]ServiceInfo `json:"services"`
}

// ServiceInfo represents information about a service.
type ServiceInfo struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Fields      map[string]ServiceField   `json:"fields"`
	Target      *ServiceTarget            `json:"target"`
	Response    *ServiceResponseInfo      `json:"response,omitempty"`
}

// ServiceField represents a field in a service.
type ServiceField struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Example     interface{} `json:"example"`
	Selector    interface{} `json:"selector"`
}

// ServiceTarget represents the target configuration for a service.
type ServiceTarget struct {
	Entity []TargetEntity `json:"entity,omitempty"`
	Device []TargetDevice `json:"device,omitempty"`
	Area   []TargetArea   `json:"area,omitempty"`
}

// TargetEntity represents entity targeting info.
type TargetEntity struct {
	Domain string `json:"domain,omitempty"`
}

// TargetDevice represents device targeting info.
type TargetDevice struct {
	Integration string `json:"integration,omitempty"`
}

// TargetArea represents area targeting info.
type TargetArea struct{}

// ServiceResponseInfo represents response info for a service.
type ServiceResponseInfo struct {
	Optional bool `json:"optional"`
}

// GetServices returns all available services.
func (c *Client) GetServices() (map[string]map[string]ServiceInfo, error) {
	resp, err := c.doRequest("GET", "/api/services", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	// API returns array of {domain, services} objects
	var services []struct {
		Domain   string                     `json:"domain"`
		Services map[string]ServiceInfo     `json:"services"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to map
	result := make(map[string]map[string]ServiceInfo)
	for _, s := range services {
		result[s.Domain] = s.Services
	}

	return result, nil
}

// CallService calls a service.
func (c *Client) CallService(domain, service string, data map[string]interface{}) ([]State, error) {
	resp, err := c.doRequest("POST", "/api/services/"+domain+"/"+service, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode == 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var changedStates []State
	if err := json.NewDecoder(resp.Body).Decode(&changedStates); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return changedStates, nil
}

// SceneConfig represents a scene configuration.
type SceneConfig struct {
	ID       string                            `json:"id"`
	Name     string                            `json:"name"`
	Entities map[string]map[string]interface{} `json:"entities"`
	Icon     string                            `json:"icon,omitempty"`
}

// GetSceneConfig retrieves the configuration for a specific scene.
func (c *Client) GetSceneConfig(sceneID string) (*SceneConfig, error) {
	resp, err := c.doRequest("GET", "/api/config/scene/config/"+sceneID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var config SceneConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &config, nil
}

// CreateScene creates a new scene.
func (c *Client) CreateScene(sceneID string, config *SceneConfig) error {
	resp, err := c.doRequest("POST", "/api/config/scene/config/"+sceneID, config)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// UpdateScene updates an existing scene.
func (c *Client) UpdateScene(sceneID string, config *SceneConfig) error {
	return c.CreateScene(sceneID, config)
}

// DeleteScene deletes a scene.
func (c *Client) DeleteScene(sceneID string) error {
	resp, err := c.doRequest("DELETE", "/api/config/scene/config/"+sceneID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}

	if resp.StatusCode == 404 {
		return ErrNotFound
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// ScriptConfig represents a script configuration.
type ScriptConfig struct {
	Alias       string                   `json:"alias"`
	Description string                   `json:"description,omitempty"`
	Icon        string                   `json:"icon,omitempty"`
	Mode        string                   `json:"mode,omitempty"`
	Sequence    []map[string]interface{} `json:"sequence"`
	Fields      map[string]interface{}   `json:"fields,omitempty"`
	Variables   map[string]interface{}   `json:"variables,omitempty"`
	MaxExceeded string                   `json:"max_exceeded,omitempty"`
	Max         int                      `json:"max,omitempty"`
}

// GetScriptConfig retrieves the configuration for a specific script.
func (c *Client) GetScriptConfig(scriptID string) (*ScriptConfig, error) {
	resp, err := c.doRequest("GET", "/api/config/script/config/"+scriptID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var config ScriptConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &config, nil
}

// CreateScript creates a new script.
func (c *Client) CreateScript(scriptID string, config *ScriptConfig) error {
	resp, err := c.doRequest("POST", "/api/config/script/config/"+scriptID, config)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// UpdateScript updates an existing script.
func (c *Client) UpdateScript(scriptID string, config *ScriptConfig) error {
	return c.CreateScript(scriptID, config)
}

// DeleteScript deletes a script.
func (c *Client) DeleteScript(scriptID string) error {
	resp, err := c.doRequest("DELETE", "/api/config/script/config/"+scriptID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}

	if resp.StatusCode == 404 {
		return ErrNotFound
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}
