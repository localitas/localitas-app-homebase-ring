package ring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth_ReturnsManifest(t *testing.T) {
	req := httptest.NewRequest("GET", "/health.json", nil)
	w := httptest.NewRecorder()

	HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var health AppHealth
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Name != "homebase-ring" {
		t.Errorf("expected name=homebase-ring, got %q", health.Name)
	}
	if health.PluginType != "homebase-plugin" {
		t.Errorf("expected plugin_type=homebase-plugin, got %q", health.PluginType)
	}
	if health.PluginFor != "homebase" {
		t.Errorf("expected plugin_for=homebase, got %q", health.PluginFor)
	}
	if health.Icon != "bell" {
		t.Errorf("expected icon=bell, got %q", health.Icon)
	}
}
