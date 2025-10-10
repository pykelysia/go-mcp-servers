package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dimasd-angga/go-mcp-servers/shared/testutil"
	"github.com/mark3labs/mcp-go/client"
)

// mockHA returns an httptest server that responds to the subset of the HA REST
// API exercised by our tools, with a tiny fixed dataset.
func mockHA(t *testing.T) (*httptest.Server, *capturedCalls) {
	t.Helper()
	calls := &capturedCalls{calls: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/states", func(w http.ResponseWriter, r *http.Request) {
		// Authorization header must be present.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, "no auth", 401)
			return
		}
		_, _ = w.Write([]byte(`[
		  {"entity_id":"light.kitchen","state":"on","attributes":{"brightness":200}},
		  {"entity_id":"switch.lamp","state":"off","attributes":{}},
		  {"entity_id":"automation.morning","state":"on","attributes":{}}
		]`))
	})
	mux.HandleFunc("/api/states/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/states/")
		if id == "" || id == "missing.entity" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"entity_id":"` + id + `","state":"on","attributes":{"foo":"bar"}}`))
	})
	mux.HandleFunc("/api/services/", func(w http.ResponseWriter, r *http.Request) {
		// /api/services/{domain}/{service}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/services/"), "/")
		body, _ := io.ReadAll(r.Body)
		calls.record(strings.Join(parts, "/"), string(body))
		_, _ = w.Write([]byte("[]"))
	})
	mux.HandleFunc("/api/history/period/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[[{"entity_id":"light.kitchen","state":"on"}]]`))
	})
	mux.HandleFunc("/api/events/", func(w http.ResponseWriter, r *http.Request) {
		evType := strings.TrimPrefix(r.URL.Path, "/api/events/")
		body, _ := io.ReadAll(r.Body)
		calls.record("event/"+evType, string(body))
		_, _ = w.Write([]byte(`{"message":"Event fired"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, calls
}

type capturedCalls struct {
	calls map[string]string
}

func (c *capturedCalls) record(key, body string) { c.calls[key] = body }

func newHAClient(t *testing.T) (*client.Client, *HAServer, *capturedCalls) {
	t.Helper()
	srv, calls := mockHA(t)
	t.Setenv("HA_URL", srv.URL)
	t.Setenv("HA_TOKEN", "test-token")
	t.Setenv("HA_TIMEOUT", "5")
	h, err := NewHAServer()
	if err != nil {
		t.Fatal(err)
	}
	return testutil.NewInProcessClient(t, h.MCP()), h, calls
}

func TestGetStates_All(t *testing.T) {
	c, _, _ := newHAClient(t)
	out := testutil.CallTool(t, c, "get_states", map[string]any{})
	for _, want := range []string{"light.kitchen", "switch.lamp", "automation.morning"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output: %s", want, out)
		}
	}
}

func TestGetStates_DomainFilter(t *testing.T) {
	c, _, _ := newHAClient(t)
	out := testutil.CallTool(t, c, "get_states", map[string]any{"domain": "light"})
	if !strings.Contains(out, "light.kitchen") {
		t.Errorf("missing light entity: %s", out)
	}
	if strings.Contains(out, "switch.lamp") {
		t.Errorf("switch leaked into light filter: %s", out)
	}
}

func TestGetState_Found(t *testing.T) {
	c, _, _ := newHAClient(t)
	out := testutil.CallTool(t, c, "get_state", map[string]any{"entity_id": "light.kitchen"})
	if !strings.Contains(out, `"light.kitchen"`) {
		t.Errorf("missing entity_id: %s", out)
	}
}

func TestGetState_NotFound(t *testing.T) {
	c, _, _ := newHAClient(t)
	r := testutil.CallToolRaw(t, c, "get_state", map[string]any{"entity_id": "missing.entity"})
	if !r.IsError {
		t.Error("missing entity should error")
	}
}

func TestGetAttributes(t *testing.T) {
	c, _, _ := newHAClient(t)
	out := testutil.CallTool(t, c, "get_attributes", map[string]any{"entity_id": "light.kitchen"})
	if !strings.Contains(out, `"foo"`) {
		t.Errorf("attributes wrong: %s", out)
	}
}

func TestCallService(t *testing.T) {
	c, _, calls := newHAClient(t)
	out := testutil.CallTool(t, c, "call_service", map[string]any{
		"domain":    "light",
		"service":   "turn_on",
		"entity_id": "light.kitchen",
		"data":      `{"brightness": 100}`,
	})
	if out != "[]" {
		t.Errorf("unexpected body: %s", out)
	}
	got, ok := calls.calls["light/turn_on"]
	if !ok {
		t.Fatalf("service not called, calls=%v", calls.calls)
	}
	if !strings.Contains(got, `"brightness":100`) {
		t.Errorf("missing extra data: %s", got)
	}
	if !strings.Contains(got, `"light.kitchen"`) {
		t.Errorf("missing entity_id: %s", got)
	}
}

func TestCallService_BadDataJSON(t *testing.T) {
	c, _, _ := newHAClient(t)
	r := testutil.CallToolRaw(t, c, "call_service", map[string]any{
		"domain":  "light",
		"service": "turn_on",
		"data":    "not json",
	})
	if !r.IsError {
		t.Error("bad data JSON should error")
	}
}

func TestGetHistory(t *testing.T) {
	c, _, _ := newHAClient(t)
	out := testutil.CallTool(t, c, "get_history", map[string]any{
		"entity_id": "light.kitchen",
		"hours":     12,
	})
	if !strings.Contains(out, "light.kitchen") {
		t.Errorf("history missing entity: %s", out)
	}
}

func TestListAutomations(t *testing.T) {
	c, _, _ := newHAClient(t)
	out := testutil.CallTool(t, c, "list_automations", map[string]any{})
	if !strings.Contains(out, "automation.morning") {
		t.Errorf("automation missing: %s", out)
	}
	if strings.Contains(out, "switch.lamp") {
		t.Errorf("switch leaked: %s", out)
	}
}

func TestToggleAutomation(t *testing.T) {
	c, _, calls := newHAClient(t)
	out := testutil.CallTool(t, c, "toggle_automation", map[string]any{
		"entity_id": "automation.morning",
		"enabled":   true,
	})
	var m map[string]string
	_ = json.Unmarshal([]byte(out), &m)
	if m["new_state"] != "turn_on" {
		t.Errorf("expected turn_on result: %s", out)
	}
	if _, ok := calls.calls["automation/turn_on"]; !ok {
		t.Errorf("turn_on not called, calls=%v", calls.calls)
	}
}

func TestFireEvent(t *testing.T) {
	c, _, calls := newHAClient(t)
	testutil.CallTool(t, c, "fire_event", map[string]any{
		"event_type": "custom_event",
		"data":       `{"k":"v"}`,
	})
	body, ok := calls.calls["event/custom_event"]
	if !ok {
		t.Fatalf("event not fired, calls=%v", calls.calls)
	}
	if !strings.Contains(body, `"k":"v"`) {
		t.Errorf("event body wrong: %s", body)
	}
}

func TestFireEvent_MissingType(t *testing.T) {
	c, _, _ := newHAClient(t)
	r := testutil.CallToolRaw(t, c, "fire_event", map[string]any{"event_type": ""})
	if !r.IsError {
		t.Error("empty event_type should error")
	}
}
