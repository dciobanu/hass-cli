package cli

import "testing"

func TestJSONContainsValue(t *testing.T) {
	cfg := map[string]interface{}{
		"entities": []interface{}{
			map[string]interface{}{"entity": "light.kitchen"},
			map[string]interface{}{"entity": "light.kitchen_counter"},
		},
		"device_id": "abc123",
	}
	tests := []struct {
		target string
		want   bool
	}{
		{"light.kitchen", true},
		{"light.kitchen_counter", true},
		{"abc123", true},
		{"light.kitch", false},     // partial, not a full quoted value
		{"light.kitchen_c", false}, // partial
		{"light.bedroom", false},   // absent
	}
	for _, tt := range tests {
		if got := jsonContainsValue(cfg, tt.target); got != tt.want {
			t.Errorf("jsonContainsValue(%q) = %v, want %v", tt.target, got, tt.want)
		}
	}
}

func TestAttrID(t *testing.T) {
	if got := attrID(map[string]interface{}{"id": "1675304706068"}); got != "1675304706068" {
		t.Errorf("string id = %q", got)
	}
	if got := attrID(map[string]interface{}{"id": float64(1675304706068)}); got != "1675304706068" {
		t.Errorf("float id = %q, want 1675304706068", got)
	}
	if got := attrID(map[string]interface{}{}); got != "" {
		t.Errorf("missing id = %q, want empty", got)
	}
}
