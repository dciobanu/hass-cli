package cli

import (
	"testing"
)

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		wantRaw   bool // if true, expect raw timestamp returned
	}{
		{
			name:      "valid RFC3339",
			timestamp: "2024-01-15T10:30:00+00:00",
			wantRaw:   false,
		},
		{
			name:      "valid RFC3339 with Z",
			timestamp: "2024-01-15T10:30:00Z",
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
		{
			name:      "date only returns raw",
			timestamp: "2024-01-15",
			wantRaw:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.timestamp)
			if tt.wantRaw {
				if got != tt.timestamp {
					t.Errorf("formatTime(%q) = %q, want raw %q", tt.timestamp, got, tt.timestamp)
				}
			} else {
				// Should be formatted as YYYY-MM-DD HH:MM:SS
				if len(got) != 19 || got[4] != '-' || got[7] != '-' || got[10] != ' ' {
					t.Errorf("formatTime(%q) = %q, want YYYY-MM-DD HH:MM:SS format", tt.timestamp, got)
				}
			}
		})
	}
}
