package config

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	t.Run("flag wins over everything", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "/env/config.yaml")
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		t.Setenv("HOME", "/home/user")

		got, err := ResolveConfigPath("/flag/config.yaml")
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v", err)
		}
		if got != "/flag/config.yaml" {
			t.Errorf("path = %q, want %q", got, "/flag/config.yaml")
		}
	})

	t.Run("HASS_CLI_CONFIG wins over XDG and HOME", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "/env/config.yaml")
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		t.Setenv("HOME", "/home/user")

		got, err := ResolveConfigPath("")
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v", err)
		}
		if got != "/env/config.yaml" {
			t.Errorf("path = %q, want %q", got, "/env/config.yaml")
		}
	})

	t.Run("XDG_CONFIG_HOME wins over HOME", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "")
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		t.Setenv("HOME", "/home/user")

		got, err := ResolveConfigPath("")
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v", err)
		}
		want := filepath.Join("/xdg", "hass-cli", "config.yaml")
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("falls back to HOME", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/user")

		got, err := ResolveConfigPath("")
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v", err)
		}
		want := filepath.Join("/home/user", ".config", "hass-cli", "config.yaml")
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("falls back to passwd entry when HOME is unset", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "") // registers restoration of HOME after the test
		os.Unsetenv("HOME")

		got, err := ResolveConfigPath("")
		u, userErr := user.Current()
		if userErr != nil || u.HomeDir == "" {
			// No passwd entry available: resolution must fail loudly.
			if err == nil {
				t.Fatalf("ResolveConfigPath() = %q, want error when home is unresolvable", got)
			}
			return
		}
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v", err)
		}
		want := filepath.Join(u.HomeDir, ".config", "hass-cli", "config.yaml")
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})
}

func TestLoadUsesResolvedPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("server:\n  url: http://ha\n  token: tok\n"), 0600)

	t.Setenv(EnvConfigPath, path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.URL != "http://ha" {
		t.Errorf("URL = %q, want %q", cfg.Server.URL, "http://ha")
	}
}

func TestNotConfiguredErrorMentionsPath(t *testing.T) {
	t.Run("names the path it looked at", func(t *testing.T) {
		_, err := LoadFrom("/nonexistent/path/config.yaml")
		if !errors.Is(err, ErrNotConfigured) {
			t.Fatalf("LoadFrom() error = %v, want ErrNotConfigured", err)
		}
		msg := err.Error()
		for _, want := range []string{"/nonexistent/path/config.yaml", "--config", EnvConfigPath} {
			if !strings.Contains(msg, want) {
				t.Errorf("error %q does not mention %q", msg, want)
			}
		}
	})

	t.Run("explains an unresolvable home", func(t *testing.T) {
		err := &NotConfiguredError{Reason: "could not determine home directory: $HOME is not defined"}
		if !errors.Is(err, ErrNotConfigured) {
			t.Errorf("errors.Is(err, ErrNotConfigured) = false, want true")
		}
		if !strings.Contains(err.Error(), "could not determine home directory") {
			t.Errorf("error %q does not explain the home failure", err.Error())
		}
	})
}

func TestLoadFrom(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, []byte(`
server:
  url: http://localhost:8123
  token: test-token-12345678
defaults:
  output: json
  timeout: 60
`), 0600)

		cfg, err := LoadFrom(path)
		if err != nil {
			t.Fatalf("LoadFrom() error = %v", err)
		}

		if cfg.Server.URL != "http://localhost:8123" {
			t.Errorf("URL = %q, want %q", cfg.Server.URL, "http://localhost:8123")
		}
		if cfg.Server.Token != "test-token-12345678" {
			t.Errorf("Token = %q, want %q", cfg.Server.Token, "test-token-12345678")
		}
		if cfg.Defaults.Output != "json" {
			t.Errorf("Output = %q, want %q", cfg.Defaults.Output, "json")
		}
		if cfg.Defaults.Timeout != 60 {
			t.Errorf("Timeout = %d, want %d", cfg.Defaults.Timeout, 60)
		}
	})

	t.Run("sets defaults for missing fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, []byte(`
server:
  url: http://localhost:8123
  token: abc
`), 0600)

		cfg, err := LoadFrom(path)
		if err != nil {
			t.Fatalf("LoadFrom() error = %v", err)
		}

		if cfg.Defaults.Output != "human" {
			t.Errorf("Output default = %q, want %q", cfg.Defaults.Output, "human")
		}
		if cfg.Defaults.Timeout != 30 {
			t.Errorf("Timeout default = %d, want %d", cfg.Defaults.Timeout, 30)
		}
	})

	t.Run("returns ErrNotConfigured for missing file", func(t *testing.T) {
		_, err := LoadFrom("/nonexistent/path/config.yaml")
		if !errors.Is(err, ErrNotConfigured) {
			t.Errorf("LoadFrom() error = %v, want ErrNotConfigured", err)
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, []byte(`{invalid yaml`), 0600)

		_, err := LoadFrom(path)
		if err == nil {
			t.Error("LoadFrom() expected error for invalid YAML")
		}
	})
}

func TestSaveTo(t *testing.T) {
	t.Run("saves and reloads config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")

		cfg := &Config{
			Server: ServerConfig{
				URL:   "http://ha.local:8123",
				Token: "my-token",
			},
			Defaults: DefaultsConfig{
				Output:  "json",
				Timeout: 45,
			},
		}

		if err := cfg.SaveTo(path); err != nil {
			t.Fatalf("SaveTo() error = %v", err)
		}

		loaded, err := LoadFrom(path)
		if err != nil {
			t.Fatalf("LoadFrom() error = %v", err)
		}

		if loaded.Server.URL != cfg.Server.URL {
			t.Errorf("URL = %q, want %q", loaded.Server.URL, cfg.Server.URL)
		}
		if loaded.Server.Token != cfg.Server.Token {
			t.Errorf("Token = %q, want %q", loaded.Server.Token, cfg.Server.Token)
		}
	})

	t.Run("creates directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "dir", "config.yaml")

		cfg := &Config{
			Server: ServerConfig{URL: "http://test", Token: "tok"},
		}

		if err := cfg.SaveTo(path); err != nil {
			t.Fatalf("SaveTo() error = %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("SaveTo() did not create file")
		}
	})

	t.Run("file has restricted permissions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")

		cfg := &Config{
			Server: ServerConfig{URL: "http://test", Token: "tok"},
		}

		cfg.SaveTo(path)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}

		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	})
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "fully configured",
			cfg: &Config{
				Server: ServerConfig{URL: "http://localhost:8123", Token: "abc"},
			},
			want: true,
		},
		{
			name: "missing URL",
			cfg: &Config{
				Server: ServerConfig{URL: "", Token: "abc"},
			},
			want: false,
		},
		{
			name: "missing token",
			cfg: &Config{
				Server: ServerConfig{URL: "http://localhost:8123", Token: ""},
			},
			want: false,
		},
		{
			name: "nil config",
			cfg:  nil,
			want: false,
		},
		{
			name: "both empty",
			cfg: &Config{
				Server: ServerConfig{URL: "", Token: ""},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedactedToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "long token is redacted",
			token: "abcdefghijklmnop",
			want:  "abcd...mnop",
		},
		{
			name:  "short token returns ***",
			token: "short",
			want:  "***",
		},
		{
			name:  "exactly 8 chars returns ***",
			token: "12345678",
			want:  "***",
		},
		{
			name:  "9 chars is redacted",
			token: "123456789",
			want:  "1234...6789",
		},
		{
			name:  "empty token returns ***",
			token: "",
			want:  "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Token: tt.token},
			}
			got := cfg.RedactedToken()
			if got != tt.want {
				t.Errorf("RedactedToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeleteFrom(t *testing.T) {
	t.Run("deletes existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, []byte("test"), 0600)

		err := DeleteFrom(path)
		if err != nil {
			t.Fatalf("DeleteFrom() error = %v", err)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("DeleteFrom() did not remove file")
		}
	})

	t.Run("no error for nonexistent file", func(t *testing.T) {
		err := DeleteFrom("/nonexistent/config.yaml")
		if err != nil {
			t.Errorf("DeleteFrom() error = %v, want nil", err)
		}
	})
}
