package cli

import "testing"

func TestReslugEntityID(t *testing.T) {
	tests := []struct {
		name    string
		entity  string
		oldBase string
		newBase string
		want    string
		wantOK  bool
	}{
		{"primary light exact", "light.living_room_ceiling_0_3_v2", "living_room_ceiling_0_3_v2", "living_room_ceiling_0_2_v2", "light.living_room_ceiling_0_2_v2", true},
		{"suffixed sensor", "sensor.living_room_ceiling_0_3_v2_instantaneous_demand", "living_room_ceiling_0_3_v2", "living_room_ceiling_0_2_v2", "sensor.living_room_ceiling_0_2_v2_instantaneous_demand", true},
		{"add v2 suffix", "light.living_room_ceiling_1_4", "living_room_ceiling_1_4", "living_room_ceiling_1_4_v2", "light.living_room_ceiling_1_4_v2", true},
		{"button identify", "button.living_room_ceiling_1_4_identify", "living_room_ceiling_1_4", "living_room_ceiling_1_4_v2", "button.living_room_ceiling_1_4_v2_identify", true},
		{"unrelated entity left alone", "sensor.sengled_e11_n1ea_lqi_5", "living_room_ceiling_0_3_v2", "living_room_ceiling_0_2_v2", "sensor.sengled_e11_n1ea_lqi_5", false},
		{"prefix-but-not-boundary not matched", "light.living_room_ceiling_1_40", "living_room_ceiling_1_4", "x", "light.living_room_ceiling_1_40", false},
		{"no dot", "nodomain", "a", "b", "nodomain", false},
	}
	for _, tt := range tests {
		got, ok := reslugEntityID(tt.entity, tt.oldBase, tt.newBase)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("%s: reslugEntityID(%q,%q,%q) = (%q,%v), want (%q,%v)",
				tt.name, tt.entity, tt.oldBase, tt.newBase, got, ok, tt.want, tt.wantOK)
		}
	}
}
