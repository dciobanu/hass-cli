package cli

import (
	"reflect"
	"testing"
)

func TestIsEntityID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"light.living_room_ceiling_2_0_v2", true},
		{"sensor.temp_1", true},
		{"input_boolean.mb_scene_warm_white", true},
		{"light.", false},          // empty object_id
		{".kitchen", false},        // empty domain
		{"nodot", false},           // no dot
		{"two.dots.here", false},   // more than one dot
		{"Light.Kitchen", false},   // uppercase not allowed
		{"light.kitchen-1", false}, // hyphen not allowed
		{"mdi:home", false},        // icon, not an entity
		{"", false},
	}
	for _, tt := range tests {
		if got := isEntityID(tt.input); got != tt.want {
			t.Errorf("isEntityID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCollectEntityRefs(t *testing.T) {
	config := map[string]interface{}{
		"views": []interface{}{
			map[string]interface{}{
				"cards": []interface{}{
					map[string]interface{}{
						"type":  "entities",
						"icon":  "mdi:lightbulb", // not an entity
						"title": "Lights",
						"entities": []interface{}{
							map[string]interface{}{"entity": "light.a"},
							map[string]interface{}{"entity": "light.b"},
							"light.a", // duplicate, bare string form
						},
					},
				},
			},
		},
	}
	got := collectEntityRefs(config)
	want := map[string]int{"light.a": 2, "light.b": 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectEntityRefs() = %v, want %v", got, want)
	}
}

func TestReplaceEntityRefs(t *testing.T) {
	config := map[string]interface{}{
		"entities": []interface{}{
			map[string]interface{}{"entity": "light.old"},
			map[string]interface{}{"entity": "light.keep"},
			"light.old",
		},
	}
	count := 0
	out := replaceEntityRefs(config, "light.old", "light.new", &count)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	want := map[string]interface{}{
		"entities": []interface{}{
			map[string]interface{}{"entity": "light.new"},
			map[string]interface{}{"entity": "light.keep"},
			"light.new",
		},
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("replaceEntityRefs() = %v, want %v", out, want)
	}

	// Original must be untouched (replacement returns a copy).
	if config["entities"].([]interface{})[0].(map[string]interface{})["entity"] != "light.old" {
		t.Error("replaceEntityRefs mutated the original config")
	}
}
