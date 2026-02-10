package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/dorinclisu/hass-cli/internal/testutil"
)

const testToken = "test-token-abc123"

func TestCheckConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/", 200, map[string]string{"message": "API running."})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CheckConnection()
		if err != nil {
			t.Errorf("CheckConnection() error = %v, want nil", err)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/", 200, map[string]string{"message": "API running."})

		client := NewClient(mock.URL(), "wrong-token", 5*time.Second)
		err := client.CheckConnection()
		if !IsUnauthorized(err) {
			t.Errorf("CheckConnection() error = %v, want unauthorized", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("internal error"))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CheckConnection()
		if err == nil {
			t.Error("CheckConnection() expected error for 500")
		}
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/", 200, APIStatus{Message: "API running."})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		status, err := client.GetStatus()
		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}
		if status.Message != "API running." {
			t.Errorf("Message = %q, want %q", status.Message, "API running.")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)

		client := NewClient(mock.URL(), "bad-token", 5*time.Second)
		_, err := client.GetStatus()
		if !IsUnauthorized(err) {
			t.Errorf("GetStatus() error = %v, want unauthorized", err)
		}
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/config", 200, Config{
			LocationName: "Home",
			Version:      "2024.1.0",
			TimeZone:     "America/New_York",
			State:        "RUNNING",
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		config, err := client.GetConfig()
		if err != nil {
			t.Fatalf("GetConfig() error = %v", err)
		}
		if config.LocationName != "Home" {
			t.Errorf("LocationName = %q, want %q", config.LocationName, "Home")
		}
		if config.Version != "2024.1.0" {
			t.Errorf("Version = %q, want %q", config.Version, "2024.1.0")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)

		client := NewClient(mock.URL(), "bad-token", 5*time.Second)
		_, err := client.GetConfig()
		if !IsUnauthorized(err) {
			t.Errorf("GetConfig() error = %v, want unauthorized", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/config", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("internal error"))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		_, err := client.GetConfig()
		if err == nil {
			t.Error("GetConfig() expected error for 500")
		}
	})
}

func TestGetStates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		states := []State{
			{EntityID: "light.living_room", State: "on"},
			{EntityID: "sensor.temperature", State: "22.5"},
		}
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/states", 200, states)

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		got, err := client.GetStates()
		if err != nil {
			t.Fatalf("GetStates() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("GetStates() returned %d states, want 2", len(got))
		}
		if got[0].EntityID != "light.living_room" {
			t.Errorf("States[0].EntityID = %q, want %q", got[0].EntityID, "light.living_room")
		}
		if got[1].State != "22.5" {
			t.Errorf("States[1].State = %q, want %q", got[1].State, "22.5")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)

		client := NewClient(mock.URL(), "bad", 5*time.Second)
		_, err := client.GetStates()
		if !IsUnauthorized(err) {
			t.Errorf("GetStates() error = %v, want unauthorized", err)
		}
	})
}

func TestGetState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/states/light.living_room", 200, State{
			EntityID: "light.living_room",
			State:    "on",
			Attributes: map[string]interface{}{
				"brightness": float64(255),
			},
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		state, err := client.GetState("light.living_room")
		if err != nil {
			t.Fatalf("GetState() error = %v", err)
		}
		if state.EntityID != "light.living_room" {
			t.Errorf("EntityID = %q, want %q", state.EntityID, "light.living_room")
		}
		if state.State != "on" {
			t.Errorf("State = %q, want %q", state.State, "on")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/states/nonexistent.entity", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte("Not found"))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		_, err := client.GetState("nonexistent.entity")
		if !IsNotFound(err) {
			t.Errorf("GetState() error = %v, want not found", err)
		}
	})
}

func TestSetState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/states/sensor.test", func(w http.ResponseWriter, r *http.Request) {
			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			if payload["state"] != "42" {
				t.Errorf("request state = %v, want %q", payload["state"], "42")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(State{
				EntityID: "sensor.test",
				State:    "42",
			})
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		state, err := client.SetState("sensor.test", "42", nil)
		if err != nil {
			t.Fatalf("SetState() error = %v", err)
		}
		if state.State != "42" {
			t.Errorf("State = %q, want %q", state.State, "42")
		}
	})

	t.Run("with attributes", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/states/sensor.test", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			attrs, ok := payload["attributes"].(map[string]interface{})
			if !ok {
				t.Error("expected attributes in request body")
			}
			if attrs["unit_of_measurement"] != "°C" {
				t.Errorf("unit = %v, want °C", attrs["unit_of_measurement"])
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(State{EntityID: "sensor.test", State: "22"})
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		attrs := map[string]interface{}{"unit_of_measurement": "°C"}
		_, err := client.SetState("sensor.test", "22", attrs)
		if err != nil {
			t.Fatalf("SetState() error = %v", err)
		}
	})
}

func TestCallService(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/services/light/turn_on", func(w http.ResponseWriter, r *http.Request) {
			// Verify content type
			ct := r.Header.Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			// Verify auth header
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+testToken {
				t.Errorf("Authorization = %q, want Bearer token", auth)
			}

			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			if payload["entity_id"] != "light.living_room" {
				t.Errorf("entity_id = %v, want light.living_room", payload["entity_id"])
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]State{
				{EntityID: "light.living_room", State: "on"},
			})
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		states, err := client.CallService("light", "turn_on", map[string]interface{}{
			"entity_id": "light.living_room",
		})
		if err != nil {
			t.Fatalf("CallService() error = %v", err)
		}
		if len(states) != 1 {
			t.Fatalf("CallService() returned %d states, want 1", len(states))
		}
	})

	t.Run("bad request", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/services/invalid/service", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			w.Write([]byte("Invalid service"))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		_, err := client.CallService("invalid", "service", nil)
		if err == nil {
			t.Error("CallService() expected error for 400")
		}
		var apiErr *APIError
		if ok := isAPIError(err, &apiErr); ok {
			if apiErr.StatusCode != 400 {
				t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
			}
		}
	})
}

func TestGetSceneConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/config/scene/config/12345", 200, SceneConfig{
			ID:   "12345",
			Name: "Movie Night",
			Entities: map[string]map[string]interface{}{
				"light.living_room": {"state": "on", "brightness": float64(50)},
			},
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		config, err := client.GetSceneConfig("12345")
		if err != nil {
			t.Fatalf("GetSceneConfig() error = %v", err)
		}
		if config.Name != "Movie Night" {
			t.Errorf("Name = %q, want %q", config.Name, "Movie Night")
		}
		if _, ok := config.Entities["light.living_room"]; !ok {
			t.Error("expected light.living_room in entities")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/config/scene/config/99999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		_, err := client.GetSceneConfig("99999")
		if !IsNotFound(err) {
			t.Errorf("GetSceneConfig() error = %v, want not found", err)
		}
	})
}

func TestCreateScene(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/config/scene/config/12345", func(w http.ResponseWriter, r *http.Request) {
			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var config SceneConfig
			json.Unmarshal(body, &config)

			if config.Name != "Test Scene" {
				t.Errorf("Name = %q, want %q", config.Name, "Test Scene")
			}

			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CreateScene("12345", &SceneConfig{
			ID:       "12345",
			Name:     "Test Scene",
			Entities: map[string]map[string]interface{}{},
		})
		if err != nil {
			t.Errorf("CreateScene() error = %v", err)
		}
	})
}

func TestDeleteScene(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/scene/config/12345", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteScene("12345")
		if err != nil {
			t.Errorf("DeleteScene() error = %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/scene/config/99999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteScene("99999")
		if !IsNotFound(err) {
			t.Errorf("DeleteScene() error = %v, want not found", err)
		}
	})
}

func TestGetScriptConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/config/script/config/hello_world", 200, ScriptConfig{
			Alias:       "Hello World",
			Description: "A test script",
			Mode:        "single",
			Sequence:    []map[string]interface{}{},
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		config, err := client.GetScriptConfig("hello_world")
		if err != nil {
			t.Fatalf("GetScriptConfig() error = %v", err)
		}
		if config.Alias != "Hello World" {
			t.Errorf("Alias = %q, want %q", config.Alias, "Hello World")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/config/script/config/nonexistent", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		_, err := client.GetScriptConfig("nonexistent")
		if !IsNotFound(err) {
			t.Errorf("GetScriptConfig() error = %v, want not found", err)
		}
	})
}

func TestCreateScript(t *testing.T) {
	t.Run("verifies request body and path", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/config/script/config/my_script", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Method = %q, want POST", r.Method)
			}

			body, _ := io.ReadAll(r.Body)
			var config ScriptConfig
			json.Unmarshal(body, &config)

			if config.Alias != "My Script" {
				t.Errorf("Alias = %q, want %q", config.Alias, "My Script")
			}
			if config.Mode != "single" {
				t.Errorf("Mode = %q, want %q", config.Mode, "single")
			}

			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CreateScript("my_script", &ScriptConfig{
			Alias:    "My Script",
			Mode:     "single",
			Sequence: []map[string]interface{}{},
		})
		if err != nil {
			t.Errorf("CreateScript() error = %v", err)
		}
	})
}

func TestDeleteScript(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/script/config/hello_world", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteScript("hello_world")
		if err != nil {
			t.Errorf("DeleteScript() error = %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/script/config/missing", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteScript("missing")
		if !IsNotFound(err) {
			t.Errorf("DeleteScript() error = %v, want not found", err)
		}
	})
}

func TestGetAutomationConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.HandleJSON("GET", "/api/config/automation/config/12345", 200, AutomationConfig{
			ID:    "12345",
			Alias: "Motion Light",
			Mode:  "single",
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		config, err := client.GetAutomationConfig("12345")
		if err != nil {
			t.Fatalf("GetAutomationConfig() error = %v", err)
		}
		if config.Alias != "Motion Light" {
			t.Errorf("Alias = %q, want %q", config.Alias, "Motion Light")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/config/automation/config/99999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		_, err := client.GetAutomationConfig("99999")
		if !IsNotFound(err) {
			t.Errorf("GetAutomationConfig() error = %v, want not found", err)
		}
	})
}

func TestCreateAutomation(t *testing.T) {
	t.Run("verifies request body", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/config/automation/config/12345", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var config AutomationConfig
			json.Unmarshal(body, &config)

			if config.Alias != "Test Automation" {
				t.Errorf("Alias = %q, want %q", config.Alias, "Test Automation")
			}

			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CreateAutomation("12345", &AutomationConfig{
			ID:    "12345",
			Alias: "Test Automation",
			Mode:  "single",
		})
		if err != nil {
			t.Errorf("CreateAutomation() error = %v", err)
		}
	})
}

func TestDeleteAutomation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/automation/config/12345", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteAutomation("12345")
		if err != nil {
			t.Errorf("DeleteAutomation() error = %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/automation/config/99999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteAutomation("99999")
		if !IsNotFound(err) {
			t.Errorf("DeleteAutomation() error = %v, want not found", err)
		}
	})
}

func TestCreateInputSelect(t *testing.T) {
	t.Run("success with icon", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/config/input_select/config/my_select", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			if payload["name"] != "My Select" {
				t.Errorf("name = %v, want My Select", payload["name"])
			}
			if payload["icon"] != "mdi:format-list-bulleted" {
				t.Errorf("icon = %v, want mdi:format-list-bulleted", payload["icon"])
			}

			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CreateInputSelect("my_select", "My Select", []string{"a", "b"}, "mdi:format-list-bulleted")
		if err != nil {
			t.Errorf("CreateInputSelect() error = %v", err)
		}
	})

	t.Run("success without icon", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/config/input_select/config/my_select", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			if _, ok := payload["icon"]; ok {
				t.Error("icon should not be present when empty")
			}

			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CreateInputSelect("my_select", "My Select", []string{"a"}, "")
		if err != nil {
			t.Errorf("CreateInputSelect() error = %v", err)
		}
	})
}

func TestCreateInputBoolean(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("POST", "/api/config/input_boolean/config/my_bool", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			if payload["name"] != "My Bool" {
				t.Errorf("name = %v, want My Bool", payload["name"])
			}

			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.CreateInputBoolean("my_bool", "My Bool", "")
		if err != nil {
			t.Errorf("CreateInputBoolean() error = %v", err)
		}
	})
}

func TestDeleteHelper(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/input_select/config/my_select", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"result": "ok"}`))
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteHelper("input_select", "my_select")
		if err != nil {
			t.Errorf("DeleteHelper() error = %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("DELETE", "/api/config/input_select/config/missing", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		err := client.DeleteHelper("input_select", "missing")
		if !IsNotFound(err) {
			t.Errorf("DeleteHelper() error = %v, want not found", err)
		}
	})
}

func TestGetServices(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := testutil.NewRESTMock(t, testToken)
		mock.Handle("GET", "/api/services", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"domain": "light",
					"services": map[string]interface{}{
						"turn_on": map[string]interface{}{
							"name":        "Turn on",
							"description": "Turn on a light",
						},
					},
				},
			})
		})

		client := NewClient(mock.URL(), testToken, 5*time.Second)
		services, err := client.GetServices()
		if err != nil {
			t.Fatalf("GetServices() error = %v", err)
		}
		if _, ok := services["light"]; !ok {
			t.Error("expected light domain in services")
		}
		if _, ok := services["light"]["turn_on"]; !ok {
			t.Error("expected turn_on service under light domain")
		}
	})
}

func TestAuthorizationHeader(t *testing.T) {
	mock := testutil.NewRESTMock(t, testToken)
	mock.Handle("GET", "/api/", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+testToken {
			t.Errorf("Authorization header = %q, want Bearer %s", auth, testToken)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type header = %q, want application/json", ct)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	})

	client := NewClient(mock.URL(), testToken, 5*time.Second)
	client.CheckConnection()
}

// isAPIError is a test helper that extracts an APIError from an error chain.
func isAPIError(err error, target **APIError) bool {
	var apiErr *APIError
	if ok := err != nil; ok {
		if ae, ok := err.(*APIError); ok {
			*target = ae
			return true
		}
	}
	_ = apiErr
	return false
}
