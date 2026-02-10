package cli

import (
	"testing"
)

func TestNormalizeScriptID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips script. prefix",
			input: "script.hello_world",
			want:  "hello_world",
		},
		{
			name:  "no prefix unchanged",
			input: "hello_world",
			want:  "hello_world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "just prefix",
			input: "script.",
			want:  "",
		},
		{
			name:  "different prefix unchanged",
			input: "automation.test",
			want:  "automation.test",
		},
		{
			name:  "numeric ID unchanged",
			input: "1761025981191",
			want:  "1761025981191",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeScriptID(tt.input)
			if got != tt.want {
				t.Errorf("normalizeScriptID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
