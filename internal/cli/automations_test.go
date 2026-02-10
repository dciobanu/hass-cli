package cli

import (
	"testing"
)

func TestNormalizeAutomationID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips automation. prefix",
			input: "automation.brightness_change",
			want:  "brightness_change",
		},
		{
			name:  "no prefix unchanged",
			input: "brightness_change",
			want:  "brightness_change",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "just prefix",
			input: "automation.",
			want:  "",
		},
		{
			name:  "different prefix unchanged",
			input: "script.test",
			want:  "script.test",
		},
		{
			name:  "numeric config ID unchanged",
			input: "1761025981191",
			want:  "1761025981191",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAutomationID(tt.input)
			if got != tt.want {
				t.Errorf("normalizeAutomationID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
