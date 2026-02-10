package cli

import (
	"testing"
)

func TestMatchesPatterns(t *testing.T) {
	tests := []struct {
		name     string
		entityID string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match",
			entityID: "light.living_room",
			patterns: []string{"light.living_room"},
			want:     true,
		},
		{
			name:     "no match",
			entityID: "light.living_room",
			patterns: []string{"light.bedroom"},
			want:     false,
		},
		{
			name:     "wildcard prefix match",
			entityID: "light.living_room",
			patterns: []string{"light.*"},
			want:     true,
		},
		{
			name:     "wildcard no match",
			entityID: "sensor.temperature",
			patterns: []string{"light.*"},
			want:     false,
		},
		{
			name:     "multiple patterns first matches",
			entityID: "light.living_room",
			patterns: []string{"light.*", "sensor.*"},
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			entityID: "sensor.temperature",
			patterns: []string{"light.*", "sensor.*"},
			want:     true,
		},
		{
			name:     "multiple patterns none match",
			entityID: "switch.outlet",
			patterns: []string{"light.*", "sensor.*"},
			want:     false,
		},
		{
			name:     "case insensitive",
			entityID: "Light.Living_Room",
			patterns: []string{"light.living_room"},
			want:     true,
		},
		{
			name:     "empty patterns",
			entityID: "light.living_room",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "wildcard all match",
			entityID: "anything.at_all",
			patterns: []string{"*"},
			want:     true,
		},
		{
			name:     "partial domain wildcard",
			entityID: "light.living_room",
			patterns: []string{"li*"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPatterns(tt.entityID, tt.patterns)
			if got != tt.want {
				t.Errorf("matchesPatterns(%q, %v) = %v, want %v", tt.entityID, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestFormatEventTime(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		wantRaw   bool // if true, expect the raw timestamp back (parse failure)
	}{
		{
			name:      "RFC3339Nano",
			timestamp: "2024-01-15T10:30:00.123456+00:00",
			wantRaw:   false,
		},
		{
			name:      "RFC3339",
			timestamp: "2024-01-15T10:30:00+00:00",
			wantRaw:   false,
		},
		{
			name:      "invalid timestamp returns raw",
			timestamp: "not-a-timestamp",
			wantRaw:   true,
		},
		{
			name:      "empty string returns raw",
			timestamp: "",
			wantRaw:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatEventTime(tt.timestamp)
			if tt.wantRaw {
				if got != tt.timestamp {
					t.Errorf("formatEventTime(%q) = %q, want raw %q", tt.timestamp, got, tt.timestamp)
				}
			} else {
				// Should be formatted as HH:MM:SS
				if len(got) != 8 || got[2] != ':' || got[5] != ':' {
					t.Errorf("formatEventTime(%q) = %q, want HH:MM:SS format", tt.timestamp, got)
				}
			}
		})
	}
}
