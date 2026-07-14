package internal

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/mantonx/viewra/pkg/plugin/sdk"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestNewUpNextPlugin(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	if p == nil {
		t.Fatal("NewUpNextPlugin() returned nil")
	}
	if !p.enabled {
		t.Error("expected enabled to be true by default")
	}
}

func TestPlugin_Initialize(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	err := p.Initialize(context.Background(), t.TempDir(), nil, nil)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.data != nil {
		t.Error("expected data to be nil when services is nil")
	}

	// With services
	services := &sdk.HostServices{
		Data: &sdk.DataClient{},
	}
	err = p.Initialize(context.Background(), t.TempDir(), nil, services)
	if err != nil {
		t.Fatalf("Initialize() with services error = %v", err)
	}
	if p.data == nil {
		t.Error("expected data to be set when services.Data is non-nil")
	}
}

func TestPlugin_Shutdown(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	err := p.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestPlugin_IsConfigured(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	if !p.IsConfigured() {
		t.Error("IsConfigured() should return true")
	}
}

func TestPlugin_GetRoutes(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	routes := p.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Path != "/up-next" {
		t.Errorf("expected path /up-next, got %s", routes[0].Path)
	}
}

func TestPlugin_GetSettingsSchema(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	schema, err := p.GetSettingsSchema()
	if err != nil {
		t.Fatalf("GetSettingsSchema() error = %v", err)
	}
	if len(schema) == 0 {
		t.Fatal("GetSettingsSchema() returned empty schema")
	}
	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestPlugin_Configure(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	if err := p.Configure([]byte(`{"enabled":false}`)); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if p.enabled {
		t.Error("expected enabled to be false after configure")
	}

	if err := p.Configure([]byte(`{"enabled":true}`)); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if !p.enabled {
		t.Error("expected enabled to be true after configure")
	}

	if err := p.Configure([]byte(`invalid`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPlugin_HandleHTTP_UnknownRoute(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	resp, err := p.HandleHTTP(context.Background(), &sdk.HTTPRequest{
		Path:   "/unknown",
		Method: "GET",
	})
	if err != nil {
		t.Fatalf("HandleHTTP() error = %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.Unmarshal(resp.Body, &body)
	if body["error"] != "route not found" {
		t.Errorf("expected 'route not found', got '%s'", body["error"])
	}
}

func TestPlugin_HandleUpNext_NoServices(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	resp, err := p.HandleHTTP(context.Background(), &sdk.HTTPRequest{
		Path:   "/up-next",
		Method: "GET",
		UserID: "test-user",
	})
	if err != nil {
		t.Fatalf("HandleHTTP() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["title"] != "Up Next" {
		t.Errorf("expected title 'Up Next', got '%v'", body["title"])
	}
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("expected items to be an array")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestPlugin_HandleUpNext_Disabled(t *testing.T) {
	p := NewUpNextPlugin(newTestLogger())
	p.Configure([]byte(`{"enabled":false}`))

	resp, err := p.HandleHTTP(context.Background(), &sdk.HTTPRequest{
		Path:   "/up-next",
		Method: "GET",
		UserID: "test-user",
	})
	if err != nil {
		t.Fatalf("HandleHTTP() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.Unmarshal(resp.Body, &body)
	items, _ := body["items"].([]any)
	if len(items) != 0 {
		t.Errorf("expected 0 items when disabled, got %d", len(items))
	}
}

func TestSettingsSchema(t *testing.T) {
	schema := SettingsSchema()
	if schema == nil {
		t.Fatal("SettingsSchema() returned nil")
	}
}

func TestJSONResponse(t *testing.T) {
	resp, err := jsonResponse(200, map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("jsonResponse() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.ContentType != "application/json" {
		t.Errorf("expected application/json, got %s", resp.ContentType)
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["hello"] != "world" {
		t.Errorf("expected 'world', got '%s'", body["hello"])
	}
}

func TestJSONResponse_MarshalError(t *testing.T) {
	resp, err := jsonResponse(200, map[string]any{"ch": make(chan int)})
	if err != nil {
		t.Fatalf("jsonResponse() should not return error, got %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500 on marshal error, got %d", resp.StatusCode)
	}
}
