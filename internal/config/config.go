// Package config handles loading and saving hass-cli configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the hass-cli configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Defaults DefaultsConfig `yaml:"defaults"`
}

// ServerConfig contains Home Assistant server connection details.
type ServerConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

// DefaultsConfig contains default settings.
type DefaultsConfig struct {
	Output  string `yaml:"output"`
	Timeout int    `yaml:"timeout"`
}

// ErrNotConfigured is returned when the config file doesn't exist or is incomplete.
var ErrNotConfigured = errors.New("hass-cli not configured. Run 'hass-cli login' first")

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "hass-cli", "config.yaml")
}

// Load reads the configuration from the default path.
func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

// LoadFrom reads the configuration from the specified path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotConfigured
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Defaults.Output == "" {
		cfg.Defaults.Output = "human"
	}
	if cfg.Defaults.Timeout == 0 {
		cfg.Defaults.Timeout = 30
	}

	return &cfg, nil
}

// Save writes the configuration to the default path.
func (c *Config) Save() error {
	return c.SaveTo(DefaultConfigPath())
}

// SaveTo writes the configuration to the specified path.
func (c *Config) SaveTo(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsConfigured returns true if the config has server URL and token set.
func (c *Config) IsConfigured() bool {
	return c != nil && c.Server.URL != "" && c.Server.Token != ""
}

// Delete removes the configuration file.
func Delete() error {
	return DeleteFrom(DefaultConfigPath())
}

// DeleteFrom removes the configuration file at the specified path.
func DeleteFrom(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %w", err)
	}
	return nil
}

// RedactedToken returns the token with most characters replaced by asterisks.
func (c *Config) RedactedToken() string {
	if len(c.Server.Token) <= 8 {
		return "***"
	}
	return c.Server.Token[:4] + "..." + c.Server.Token[len(c.Server.Token)-4:]
}
