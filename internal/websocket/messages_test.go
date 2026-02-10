package websocket

import (
	"testing"
)

func TestDevice_DisplayName(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   string
	}{
		{
			name:   "user name takes priority",
			device: Device{ID: "abc", Name: strPtr("Device Name"), NameByUser: strPtr("Custom Name")},
			want:   "Custom Name",
		},
		{
			name:   "falls back to device name",
			device: Device{ID: "abc", Name: strPtr("Device Name"), NameByUser: nil},
			want:   "Device Name",
		},
		{
			name:   "falls back to ID when no names",
			device: Device{ID: "abc", Name: nil, NameByUser: nil},
			want:   "abc",
		},
		{
			name:   "skips empty user name",
			device: Device{ID: "abc", Name: strPtr("Device Name"), NameByUser: strPtr("")},
			want:   "Device Name",
		},
		{
			name:   "skips empty device name",
			device: Device{ID: "abc", Name: strPtr(""), NameByUser: nil},
			want:   "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.DisplayName()
			if got != tt.want {
				t.Errorf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDevice_DisplayManufacturer(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   string
	}{
		{
			name:   "returns manufacturer",
			device: Device{Manufacturer: strPtr("Philips")},
			want:   "Philips",
		},
		{
			name:   "returns Unknown for nil",
			device: Device{Manufacturer: nil},
			want:   "Unknown",
		},
		{
			name:   "returns Unknown for empty",
			device: Device{Manufacturer: strPtr("")},
			want:   "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.DisplayManufacturer()
			if got != tt.want {
				t.Errorf("DisplayManufacturer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDevice_DisplayModel(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   string
	}{
		{
			name:   "returns model",
			device: Device{Model: strPtr("Hue Bridge")},
			want:   "Hue Bridge",
		},
		{
			name:   "returns Unknown for nil",
			device: Device{Model: nil},
			want:   "Unknown",
		},
		{
			name:   "returns Unknown for empty",
			device: Device{Model: strPtr("")},
			want:   "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.DisplayModel()
			if got != tt.want {
				t.Errorf("DisplayModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEntity_DisplayName(t *testing.T) {
	tests := []struct {
		name   string
		entity Entity
		want   string
	}{
		{
			name:   "user-set name takes priority",
			entity: Entity{EntityID: "light.test", Name: strPtr("Custom"), OriginalName: "Original"},
			want:   "Custom",
		},
		{
			name:   "falls back to original name",
			entity: Entity{EntityID: "light.test", Name: nil, OriginalName: "Original"},
			want:   "Original",
		},
		{
			name:   "falls back to entity ID",
			entity: Entity{EntityID: "light.test", Name: nil, OriginalName: nil},
			want:   "light.test",
		},
		{
			name:   "skips empty user name",
			entity: Entity{EntityID: "light.test", Name: strPtr(""), OriginalName: "Original"},
			want:   "Original",
		},
		{
			name:   "skips empty original name",
			entity: Entity{EntityID: "light.test", Name: nil, OriginalName: ""},
			want:   "light.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entity.DisplayName()
			if got != tt.want {
				t.Errorf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEntity_GetOriginalName(t *testing.T) {
	tests := []struct {
		name         string
		originalName interface{}
		wantNil      bool
		wantValue    string
	}{
		{
			name:         "returns string pointer",
			originalName: "Temperature Sensor",
			wantNil:      false,
			wantValue:    "Temperature Sensor",
		},
		{
			name:         "returns nil for nil",
			originalName: nil,
			wantNil:      true,
		},
		{
			name:         "returns pointer for empty string",
			originalName: "",
			wantNil:      false,
			wantValue:    "",
		},
		{
			name:         "returns nil for non-string",
			originalName: 42,
			wantNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := Entity{OriginalName: tt.originalName}
			got := entity.GetOriginalName()

			if tt.wantNil {
				if got != nil {
					t.Errorf("GetOriginalName() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Error("GetOriginalName() = nil, want non-nil")
				} else if *got != tt.wantValue {
					t.Errorf("GetOriginalName() = %q, want %q", *got, tt.wantValue)
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
