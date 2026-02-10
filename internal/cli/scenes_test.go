package cli

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple lowercase",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "uppercase to lowercase",
			input: "Hello World",
			want:  "hello_world",
		},
		{
			name:  "special characters replaced",
			input: "Movie Night!",
			want:  "movie_night",
		},
		{
			name:  "multiple spaces",
			input: "good   morning",
			want:  "good_morning",
		},
		{
			name:  "leading and trailing spaces",
			input: " hello world ",
			want:  "hello_world",
		},
		{
			name:  "hyphens replaced",
			input: "my-scene-name",
			want:  "my_scene_name",
		},
		{
			name:  "mixed special characters",
			input: "Room #1 (Main)",
			want:  "room_1_main",
		},
		{
			name:  "numbers preserved",
			input: "scene123",
			want:  "scene123",
		},
		{
			name:  "already a slug",
			input: "already_slug",
			want:  "already_slug",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "!!!",
			want:  "",
		},
		{
			name:  "unicode characters",
			input: "café résumé",
			want:  "caf_r_sum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
