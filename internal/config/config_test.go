package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
		if err != ErrNotConfigured {
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
