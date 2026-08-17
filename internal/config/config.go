// Package config handles loading and saving hass-cli configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EnvConfigPath is the environment variable holding the full path to the
// config file. It takes precedence over XDG_CONFIG_HOME and the home
// directory, but not over the --config flag.
const EnvConfigPath = "HASS_CLI_CONFIG"

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
// Errors reporting a specific path wrap it, so use errors.Is to test for it.
var ErrNotConfigured = errors.New("hass-cli not configured. Run 'hass-cli login' first")

// NotConfiguredError reports a missing configuration along with the path that
// was searched, or the reason no path could be determined.
type NotConfiguredError struct {
	// Path is the config file that was looked for. Empty when the path
	// itself could not be resolved.
	Path string
	// Reason explains why Path could not be resolved. Only set when Path is empty.
	Reason string
}

func (e *NotConfiguredError) Error() string {
	where := fmt.Sprintf("no config at %s", e.Path)
	if e.Path == "" {
		where = e.Reason
	}
	return fmt.Sprintf("hass-cli not configured (%s). Run 'hass-cli login', pass --config, or set %s", where, EnvConfigPath)
}

// Unwrap makes errors.Is(err, ErrNotConfigured) report true.
func (e *NotConfiguredError) Unwrap() error { return ErrNotConfigured }

// ResolveConfigPath returns the config file path to use. Resolution order:
//
//  1. explicit (the --config flag), when non-empty
//  2. $HASS_CLI_CONFIG
//  3. $XDG_CONFIG_HOME/hass-cli/config.yaml
//  4. ~/.config/hass-cli/config.yaml, using os.UserHomeDir
//  5. ~/.config/hass-cli/config.yaml, using the current user's passwd entry
//     (works even when HOME is unset)
func ResolveConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if p := os.Getenv(EnvConfigPath); p != "" {
		return p, nil
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "hass-cli", "config.yaml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		u, userErr := user.Current()
		if userErr != nil || u.HomeDir == "" {
			return "", fmt.Errorf("could not determine home directory: %v", err)
		}
		home = u.HomeDir
	}
	return filepath.Join(home, ".config", "hass-cli", "config.yaml"), nil
}

// DefaultConfigPath returns the default configuration file path, or an empty
// string if it cannot be determined.
func DefaultConfigPath() string {
	path, err := ResolveConfigPath("")
	if err != nil {
		return ""
	}
	return path
}

// Load reads the configuration from the default path.
func Load() (*Config, error) {
	path, err := ResolveConfigPath("")
	if err != nil {
		return nil, &NotConfiguredError{Reason: err.Error()}
	}
	return LoadFrom(path)
}

// LoadFrom reads the configuration from the specified path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &NotConfiguredError{Path: path}
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
	path, err := ResolveConfigPath("")
	if err != nil {
		return err
	}
	return c.SaveTo(path)
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
	path, err := ResolveConfigPath("")
	if err != nil {
		return err
	}
	return DeleteFrom(path)
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
