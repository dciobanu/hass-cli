package websocket

import (
	"fmt"
	"testing"
	"time"

	"github.com/dorinclisu/hass-cli/internal/testutil"
)

const wsTestToken = "ws-test-token-12345"

func TestHttpToWS(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "http to ws",
			input: "http://localhost:8123",
			want:  "ws://localhost:8123",
		},
		{
			name:  "https to wss",
			input: "https://ha.example.com",
			want:  "wss://ha.example.com",
		},
		{
			name:  "trailing slash stripped",
			input: "http://localhost:8123/",
			want:  "ws://localhost:8123",
		},
		{
			name:  "with port",
			input: "http://192.168.1.100:8123",
			want:  "ws://192.168.1.100:8123",
		},
		{
			name:    "unsupported scheme",
			input:   "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			input:   "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := httpToWS(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("httpToWS(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("httpToWS(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("httpToWS(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWSClient_AuthSuccess(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()
}

func TestWSClient_AuthFailure(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)

	_, err := NewClient(mock.URL(), "wrong-token", 5*time.Second)
	if err == nil {
		t.Error("NewClient() expected error for bad token")
	}
}

func TestWSClient_GetDevices(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/device_registry/list", func(msg map[string]interface{}) (interface{}, error) {
		return []map[string]interface{}{
			{
				"id":           "device1",
				"name":         "Living Room Light",
				"manufacturer": "Philips",
				"model":        "Hue Bulb",
			},
			{
				"id":           "device2",
				"name":         "Kitchen Sensor",
				"manufacturer": "Aqara",
				"model":        "Temperature",
			},
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	devices, err := client.GetDevices()
	if err != nil {
		t.Fatalf("GetDevices() error = %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("GetDevices() returned %d devices, want 2", len(devices))
	}
	if devices[0].ID != "device1" {
		t.Errorf("devices[0].ID = %q, want %q", devices[0].ID, "device1")
	}
	if *devices[0].Name != "Living Room Light" {
		t.Errorf("devices[0].Name = %q, want %q", *devices[0].Name, "Living Room Light")
	}
}

func TestWSClient_GetAreas(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/area_registry/list", func(msg map[string]interface{}) (interface{}, error) {
		return []map[string]interface{}{
			{
				"area_id": "living_room",
				"name":    "Living Room",
			},
			{
				"area_id": "kitchen",
				"name":    "Kitchen",
			},
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	areas, err := client.GetAreas()
	if err != nil {
		t.Fatalf("GetAreas() error = %v", err)
	}
	if len(areas) != 2 {
		t.Fatalf("GetAreas() returned %d areas, want 2", len(areas))
	}
	if areas[0].AreaID != "living_room" {
		t.Errorf("areas[0].AreaID = %q, want %q", areas[0].AreaID, "living_room")
	}
	if areas[0].Name != "Living Room" {
		t.Errorf("areas[0].Name = %q, want %q", areas[0].Name, "Living Room")
	}
}

func TestWSClient_GetEntities(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/entity_registry/list", func(msg map[string]interface{}) (interface{}, error) {
		return []map[string]interface{}{
			{
				"entity_id":     "light.living_room",
				"platform":      "hue",
				"original_name": "Living Room Light",
			},
			{
				"entity_id":     "sensor.temperature",
				"platform":      "aqara",
				"original_name": "Temperature",
			},
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	entities, err := client.GetEntities()
	if err != nil {
		t.Fatalf("GetEntities() error = %v", err)
	}
	if len(entities) != 2 {
		t.Fatalf("GetEntities() returned %d entities, want 2", len(entities))
	}
	if entities[0].EntityID != "light.living_room" {
		t.Errorf("entities[0].EntityID = %q, want %q", entities[0].EntityID, "light.living_room")
	}
	if entities[0].Platform != "hue" {
		t.Errorf("entities[0].Platform = %q, want %q", entities[0].Platform, "hue")
	}
}

func TestWSClient_UpdateDevice(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/device_registry/update", func(msg map[string]interface{}) (interface{}, error) {
		deviceID, _ := msg["device_id"].(string)
		nameByUser, _ := msg["name_by_user"].(string)

		if deviceID != "device1" {
			return nil, fmt.Errorf("unexpected device_id: %s", deviceID)
		}

		return map[string]interface{}{
			"id":           deviceID,
			"name_by_user": nameByUser,
			"name":         "Original Name",
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	device, err := client.UpdateDevice("device1", map[string]interface{}{
		"name_by_user": "Custom Name",
	})
	if err != nil {
		t.Fatalf("UpdateDevice() error = %v", err)
	}
	if device.ID != "device1" {
		t.Errorf("device.ID = %q, want %q", device.ID, "device1")
	}
}

func TestWSClient_UpdateEntity(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/entity_registry/update", func(msg map[string]interface{}) (interface{}, error) {
		entityID, _ := msg["entity_id"].(string)
		name, _ := msg["name"].(string)

		if entityID != "light.test" {
			return nil, fmt.Errorf("unexpected entity_id: %s", entityID)
		}

		return map[string]interface{}{
			"entity_id": entityID,
			"name":      name,
			"platform":  "test",
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	entity, err := client.UpdateEntity("light.test", map[string]interface{}{
		"name": "Renamed Light",
	})
	if err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}
	if entity.EntityID != "light.test" {
		t.Errorf("entity.EntityID = %q, want %q", entity.EntityID, "light.test")
	}
}

func TestWSClient_SendCommand_Error(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("test/fail", func(msg map[string]interface{}) (interface{}, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.SendCommand("test/fail", nil)
	if err == nil {
		t.Error("SendCommand() expected error")
	}
}

func TestWSClient_SendCommand_UnknownCommand(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.SendCommand("nonexistent/command", nil)
	if err == nil {
		t.Error("SendCommand() expected error for unknown command")
	}
}

func TestWSClient_DisableDevice(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/device_registry/update", func(msg map[string]interface{}) (interface{}, error) {
		disabledBy, _ := msg["disabled_by"].(string)
		if disabledBy != "user" {
			return nil, fmt.Errorf("expected disabled_by=user, got %v", msg["disabled_by"])
		}
		return map[string]interface{}{
			"id":          msg["device_id"],
			"disabled_by": "user",
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	device, err := client.DisableDevice("device1")
	if err != nil {
		t.Fatalf("DisableDevice() error = %v", err)
	}
	if device.ID != "device1" {
		t.Errorf("device.ID = %q, want %q", device.ID, "device1")
	}
}

func TestWSClient_EnableDevice(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/device_registry/update", func(msg map[string]interface{}) (interface{}, error) {
		// disabled_by should be nil for enabling
		if msg["disabled_by"] != nil {
			return nil, fmt.Errorf("expected disabled_by=nil, got %v", msg["disabled_by"])
		}
		return map[string]interface{}{
			"id":          msg["device_id"],
			"disabled_by": nil,
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	device, err := client.EnableDevice("device1")
	if err != nil {
		t.Fatalf("EnableDevice() error = %v", err)
	}
	if device.ID != "device1" {
		t.Errorf("device.ID = %q, want %q", device.ID, "device1")
	}
}

func TestWSClient_DeleteHelper(t *testing.T) {
	t.Run("input_select", func(t *testing.T) {
		mock := testutil.NewWSMock(t, wsTestToken)
		mock.Handle("input_select/delete", func(msg map[string]interface{}) (interface{}, error) {
			id, _ := msg["input_select_id"].(string)
			if id != "my_select" {
				return nil, fmt.Errorf("unexpected id: %s", id)
			}
			return nil, nil
		})

		client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		defer client.Close()

		err = client.DeleteHelper("input_select", "my_select")
		if err != nil {
			t.Errorf("DeleteHelper() error = %v", err)
		}
	})

	t.Run("input_boolean", func(t *testing.T) {
		mock := testutil.NewWSMock(t, wsTestToken)
		mock.Handle("input_boolean/delete", func(msg map[string]interface{}) (interface{}, error) {
			id, _ := msg["input_boolean_id"].(string)
			if id != "my_bool" {
				return nil, fmt.Errorf("unexpected id: %s", id)
			}
			return nil, nil
		})

		client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		defer client.Close()

		err = client.DeleteHelper("input_boolean", "my_bool")
		if err != nil {
			t.Errorf("DeleteHelper() error = %v", err)
		}
	})

	t.Run("unsupported domain", func(t *testing.T) {
		mock := testutil.NewWSMock(t, wsTestToken)

		client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		defer client.Close()

		err = client.DeleteHelper("unsupported_domain", "test")
		if err == nil {
			t.Error("DeleteHelper() expected error for unsupported domain")
		}
	})
}

func TestWSClient_CreateInputSelect(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("input_select/create", func(msg map[string]interface{}) (interface{}, error) {
		name, _ := msg["name"].(string)
		if name != "My Select" {
			return nil, fmt.Errorf("unexpected name: %s", name)
		}

		return map[string]interface{}{
			"id":   "generated_id",
			"name": name,
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	helper, err := client.CreateInputSelect("My Select", []string{"opt1", "opt2"}, "mdi:list")
	if err != nil {
		t.Fatalf("CreateInputSelect() error = %v", err)
	}
	if helper.ID != "generated_id" {
		t.Errorf("helper.ID = %q, want %q", helper.ID, "generated_id")
	}
}

func TestWSClient_CreateInputBoolean(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("input_boolean/create", func(msg map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"id":   "bool_id",
			"name": msg["name"],
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	helper, err := client.CreateInputBoolean("Toggle", "")
	if err != nil {
		t.Fatalf("CreateInputBoolean() error = %v", err)
	}
	if helper.ID != "bool_id" {
		t.Errorf("helper.ID = %q, want %q", helper.ID, "bool_id")
	}
}

func TestWSClient_CreateInputButton(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("input_button/create", func(msg map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"id":   "button_id",
			"name": msg["name"],
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	helper, err := client.CreateInputButton("Press Me", "mdi:gesture-tap")
	if err != nil {
		t.Fatalf("CreateInputButton() error = %v", err)
	}
	if helper.ID != "button_id" {
		t.Errorf("helper.ID = %q, want %q", helper.ID, "button_id")
	}
}

func TestWSClient_CreateInputNumber(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("input_number/create", func(msg map[string]interface{}) (interface{}, error) {
		min, _ := msg["min"].(float64)
		max, _ := msg["max"].(float64)
		step, _ := msg["step"].(float64)

		if min != 0 || max != 100 || step != 1 {
			return nil, fmt.Errorf("unexpected min/max/step: %v/%v/%v", min, max, step)
		}

		return map[string]interface{}{
			"id":   "number_id",
			"name": msg["name"],
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	helper, err := client.CreateInputNumber("Volume", 0, 100, 1, "slider", "", nil)
	if err != nil {
		t.Fatalf("CreateInputNumber() error = %v", err)
	}
	if helper.ID != "number_id" {
		t.Errorf("helper.ID = %q, want %q", helper.ID, "number_id")
	}
}

func TestWSClient_CreateInputText(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("input_text/create", func(msg map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"id":   "text_id",
			"name": msg["name"],
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	helper, err := client.CreateInputText("Note", 0, 255, "text", "", "")
	if err != nil {
		t.Fatalf("CreateInputText() error = %v", err)
	}
	if helper.ID != "text_id" {
		t.Errorf("helper.ID = %q, want %q", helper.ID, "text_id")
	}
}

func TestWSClient_ListTraces(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("trace/list", func(msg map[string]interface{}) (interface{}, error) {
		domain, _ := msg["domain"].(string)
		itemID, _ := msg["item_id"].(string)

		if domain != "script" || itemID != "hello_world" {
			return nil, fmt.Errorf("unexpected domain/item_id: %s/%s", domain, itemID)
		}

		return []map[string]interface{}{
			{
				"run_id":           "abc123",
				"state":            "stopped",
				"script_execution": "finished",
				"timestamp":        map[string]string{"start": "2024-01-15T10:00:00Z", "finish": "2024-01-15T10:00:01Z"},
			},
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	traces, err := client.ListTraces("script", "hello_world")
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("ListTraces() returned %d traces, want 1", len(traces))
	}
	if traces[0].RunID != "abc123" {
		t.Errorf("traces[0].RunID = %q, want %q", traces[0].RunID, "abc123")
	}
}

func TestWSClient_GetTrace(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("trace/get", func(msg map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"run_id":           "abc123",
			"state":            "stopped",
			"script_execution": "finished",
			"domain":           "script",
			"item_id":          "hello_world",
			"timestamp":        map[string]string{"start": "2024-01-15T10:00:00Z", "finish": "2024-01-15T10:00:01Z"},
			"trace":            map[string]interface{}{},
			"config":           map[string]interface{}{},
			"context":          map[string]interface{}{"id": "ctx1"},
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	trace, err := client.GetTrace("script", "hello_world", "abc123")
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if trace.RunID != "abc123" {
		t.Errorf("trace.RunID = %q, want %q", trace.RunID, "abc123")
	}
}

func TestWSClient_GetStates(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("get_states", func(msg map[string]interface{}) (interface{}, error) {
		return []map[string]interface{}{
			{
				"entity_id":    "light.test",
				"state":        "on",
				"attributes":   map[string]interface{}{"brightness": 255},
				"last_changed": "2024-01-15T10:00:00Z",
				"last_updated": "2024-01-15T10:00:00Z",
				"context":      map[string]interface{}{"id": "ctx1"},
			},
		}, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	states, err := client.GetStates()
	if err != nil {
		t.Fatalf("GetStates() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("GetStates() returned %d states, want 1", len(states))
	}
	if states[0].EntityID != "light.test" {
		t.Errorf("states[0].EntityID = %q, want %q", states[0].EntityID, "light.test")
	}
	if states[0].State != "on" {
		t.Errorf("states[0].State = %q, want %q", states[0].State, "on")
	}
}

func TestWSClient_RemoveConfigEntryFromDevice(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)
	mock.Handle("config/device_registry/remove_config_entry", func(msg map[string]interface{}) (interface{}, error) {
		deviceID, _ := msg["device_id"].(string)
		configEntryID, _ := msg["config_entry_id"].(string)

		if deviceID != "dev1" || configEntryID != "entry1" {
			return nil, fmt.Errorf("unexpected args: device_id=%s, config_entry_id=%s", deviceID, configEntryID)
		}

		return nil, nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	err = client.RemoveConfigEntryFromDevice("dev1", "entry1")
	if err != nil {
		t.Errorf("RemoveConfigEntryFromDevice() error = %v", err)
	}
}

func TestWSClient_MessageIDIncrement(t *testing.T) {
	mock := testutil.NewWSMock(t, wsTestToken)

	var receivedIDs []float64
	mock.Handle("test/ping", func(msg map[string]interface{}) (interface{}, error) {
		id, _ := msg["id"].(float64)
		receivedIDs = append(receivedIDs, id)
		return "pong", nil
	})

	client, err := NewClient(mock.URL(), wsTestToken, 5*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	// Send 3 commands
	for i := 0; i < 3; i++ {
		_, err := client.SendCommand("test/ping", nil)
		if err != nil {
			t.Fatalf("SendCommand() error = %v", err)
		}
	}

	// IDs should be sequential starting from 1
	if len(receivedIDs) != 3 {
		t.Fatalf("received %d messages, want 3", len(receivedIDs))
	}
	for i, id := range receivedIDs {
		expected := float64(i + 1)
		if id != expected {
			t.Errorf("message %d ID = %v, want %v", i, id, expected)
		}
	}
}
